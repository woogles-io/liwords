import React, { useState } from "react";
import { Table, Tag, Tooltip } from "antd";
import {
  StarOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  MinusOutlined,
  HistoryOutlined,
  ClockCircleOutlined,
} from "@ant-design/icons";
import {
  Division,
  PromotionFormula,
  StandingResult,
  PlacementStatus,
} from "../gen/api/proto/ipc/league_pb";
import { PlayerGameHistoryModal } from "./player_game_history_modal";

// Get placement status icon and tooltip
const getPlacementIndicator = (
  status: PlacementStatus,
): { icon: React.ReactNode; tooltip: string } | null => {
  switch (status) {
    case PlacementStatus.PLACEMENT_NEW:
      return {
        icon: <StarOutlined style={{ fontSize: 10, color: "#1890ff" }} />,
        tooltip: "New to this league",
      };
    case PlacementStatus.PLACEMENT_PROMOTED:
      return {
        icon: <ArrowUpOutlined style={{ fontSize: 10, color: "#52c41a" }} />,
        tooltip: "Promoted from lower division",
      };
    case PlacementStatus.PLACEMENT_RELEGATED:
      return {
        icon: <ArrowDownOutlined style={{ fontSize: 10, color: "#ff4d4f" }} />,
        tooltip: "Relegated from higher division",
      };
    case PlacementStatus.PLACEMENT_STAYED:
      return {
        icon: <MinusOutlined style={{ fontSize: 10, color: "#8c8c8c" }} />,
        tooltip: "Stayed in same division",
      };
    case PlacementStatus.PLACEMENT_SHORT_HIATUS_RETURNING:
      return {
        icon: <HistoryOutlined style={{ fontSize: 10, color: "#faad14" }} />,
        tooltip: "Returning after short break (1-3 seasons)",
      };
    case PlacementStatus.PLACEMENT_LONG_HIATUS_RETURNING:
      return {
        icon: <HistoryOutlined style={{ fontSize: 10, color: "#d48806" }} />,
        tooltip: "Returning after long break (4+ seasons)",
      };
    default:
      return null;
  }
};

// Column header with tooltip
const ColHeader: React.FC<{ title: string; tooltip: string }> = ({
  title,
  tooltip,
}) => (
  <Tooltip title={tooltip}>
    <span style={{ cursor: "help", whiteSpace: "nowrap" }}>{title}</span>
  </Tooltip>
);

// Calculate promotion/relegation count based on the formula
function calculatePromotionCount(
  divSize: number,
  formula: PromotionFormula,
): number {
  if (divSize === 0) return 0;
  switch (formula) {
    case PromotionFormula.PROMO_N_PLUS_1_DIV_5:
      // ceil((N+1)/5): 13->3, 15->4, 17->4, 20->5
      return Math.ceil((divSize + 1) / 5);
    case PromotionFormula.PROMO_N_DIV_5:
      // ceil(N/5): 13->3, 15->3, 17->4, 20->4
      return Math.ceil(divSize / 5);
    case PromotionFormula.PROMO_N_DIV_3:
      // ceil(N/3): 13->5, 15->5, 17->6, 20->7
      return Math.ceil(divSize / 3);
    default:
      // PROMO_N_DIV_6 (default): ceil(N/6): 13->3, 15->3, 17->3, 20->4
      return Math.ceil(divSize / 6);
  }
}

type DivisionStandingsProps = {
  division: Division;
  totalDivisions: number;
  seasonId: string;
  seasonNumber: number;
  currentUserId?: string;
  promotionFormula?: PromotionFormula;
  timeBankWarnings?: Map<string, number>; // Map of userId to count of low timebank games
};

export const DivisionStandings: React.FC<DivisionStandingsProps> = ({
  division,
  totalDivisions,
  seasonId,
  seasonNumber,
  currentUserId,
  promotionFormula = PromotionFormula.PROMO_N_DIV_6,
  timeBankWarnings,
}) => {
  const [selectedPlayer, setSelectedPlayer] = useState<{
    userId: string;
    username: string;
  } | null>(null);

  const handlePlayerClick = (userId: string, username: string) => {
    setSelectedPlayer({ userId, username });
  };

  const handleModalClose = () => {
    setSelectedPlayer(null);
  };

  // Calculate promotion/relegation zones using the season's formula
  const divisionSize = division.standings.length;
  const baseCount = calculatePromotionCount(divisionSize, promotionFormula);
  const promotionCount = division.divisionNumber === 1 ? 0 : baseCount;
  const relegationCount =
    division.divisionNumber === totalDivisions ? 0 : baseCount;

  const getRowClassName = (record: { rank: number }) => {
    const rank = record.rank;
    if (promotionCount > 0 && rank <= promotionCount) {
      return "promotion-row";
    }
    if (relegationCount > 0 && rank > divisionSize - relegationCount) {
      return "relegation-row";
    }
    return "";
  };

  // Helper function to calculate average (returns "-" if no games played)
  const formatAvg = (total: number, gamesPlayed: number, decimals = 1) => {
    if (gamesPlayed === 0) return "-";
    return (total / gamesPlayed).toFixed(decimals);
  };

  const dataSource = division.standings.map((standing) => ({
    key: standing.userId,
    userId: standing.userId,
    rank: standing.rank,
    username: standing.username,
    wins: standing.wins,
    losses: standing.losses,
    draws: standing.draws,
    spread: standing.spread,
    gamesPlayed: standing.gamesPlayed,
    gamesRemaining: standing.gamesRemaining,
    result: standing.result,
    totalScore: standing.totalScore,
    totalOpponentScore: standing.totalOpponentScore,
    totalBingos: standing.totalBingos,
    totalOpponentBingos: standing.totalOpponentBingos,
    totalTurns: standing.totalTurns,
    highTurn: standing.highTurn,
    highGame: standing.highGame,
    timeouts: standing.timeouts,
    blanksPlayed: standing.blanksPlayed,
    totalTilesPlayed: standing.totalTilesPlayed,
    totalOpponentTilesPlayed: standing.totalOpponentTilesPlayed,
    placementStatus: standing.placementStatus,
  }));

  // Define the record type for sorter functions
  type StandingRecord = (typeof dataSource)[number];

  // Hide sort icons while keeping sort functionality
  const noSortIcon = () => null;

  const columns = [
    {
      title: <ColHeader title="#" tooltip="Rank in division" />,
      dataIndex: "rank",
      key: "rank",
      width: 40,
      sorter: (a: StandingRecord, b: StandingRecord) => a.rank - b.rank,
      sortIcon: noSortIcon,
    },
    {
      title: <ColHeader title="Player" tooltip="Player username" />,
      dataIndex: "username",
      key: "username",
      sorter: (a: StandingRecord, b: StandingRecord) =>
        a.username.localeCompare(b.username),
      sortIcon: noSortIcon,
      render: (
        username: string,
        record: { userId: string; placementStatus: PlacementStatus },
      ) => {
        const isCurrentUser = record.userId === currentUserId;
        const placementIndicator = getPlacementIndicator(
          record.placementStatus,
        );
        const lowTimebankGameCount = timeBankWarnings?.get(record.userId);
        const hasLowTimebank = lowTimebankGameCount && lowTimebankGameCount > 0;

        return (
          <span
            className="clickable-player"
            onClick={() => handlePlayerClick(record.userId, username)}
            style={
              isCurrentUser
                ? { color: "#d4af37", fontWeight: "bold" }
                : undefined
            }
          >
            <strong>{username}</strong>
            {placementIndicator && (
              <Tooltip title={placementIndicator.tooltip}>
                <span style={{ marginLeft: 4, opacity: 0.7 }}>
                  {placementIndicator.icon}
                </span>
              </Tooltip>
            )}
            {hasLowTimebank && (
              <Tooltip
                title={`${lowTimebankGameCount} ${lowTimebankGameCount === 1 ? "game" : "games"} with < 1 day remaining`}
              >
                <ClockCircleOutlined
                  style={{ marginLeft: 4, color: "#ff4d4f", fontSize: 12 }}
                />
              </Tooltip>
            )}
          </span>
        );
      },
    },
    {
      title: <ColHeader title="PTS" tooltip="Points (2 per win, 1 per draw)" />,
      key: "points",
      className: "points-column",
      width: 45,
      sorter: (a: StandingRecord, b: StandingRecord) =>
        a.wins * 2 + a.draws - (b.wins * 2 + b.draws),
      sortIcon: noSortIcon,
      render: (_: unknown, record: { wins: number; draws: number }) =>
        record.wins * 2 + record.draws,
    },
    {
      title: <ColHeader title="W" tooltip="Wins" />,
      dataIndex: "wins",
      key: "wins",
      width: 35,
      sorter: (a: StandingRecord, b: StandingRecord) => a.wins - b.wins,
      sortIcon: noSortIcon,
    },
    {
      title: <ColHeader title="L" tooltip="Losses" />,
      dataIndex: "losses",
      key: "losses",
      width: 35,
      sorter: (a: StandingRecord, b: StandingRecord) => a.losses - b.losses,
      sortIcon: noSortIcon,
    },
    {
      title: <ColHeader title="D" tooltip="Draws" />,
      dataIndex: "draws",
      key: "draws",
      width: 35,
      sorter: (a: StandingRecord, b: StandingRecord) => a.draws - b.draws,
      sortIcon: noSortIcon,
    },
    {
      title: <ColHeader title="CUM" tooltip="Cumulative spread" />,
      dataIndex: "spread",
      key: "spread",
      width: 55,
      sorter: (a: StandingRecord, b: StandingRecord) => a.spread - b.spread,
      sortIcon: noSortIcon,
      render: (spread: number) => (spread > 0 ? `+${spread}` : spread),
    },
    {
      title: <ColHeader title="GP" tooltip="Games played / total games" />,
      key: "games",
      width: 55,
      sorter: (a: StandingRecord, b: StandingRecord) =>
        a.gamesPlayed - b.gamesPlayed,
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { gamesPlayed: number; gamesRemaining: number },
      ) =>
        `${record.gamesPlayed}/${record.gamesPlayed + record.gamesRemaining}`,
    },
    {
      title: <ColHeader title="ScAV" tooltip="Average score per game" />,
      key: "avgScore",
      width: 50,
      sorter: (a: StandingRecord, b: StandingRecord) => {
        const avgA = a.gamesPlayed > 0 ? a.totalScore / a.gamesPlayed : 0;
        const avgB = b.gamesPlayed > 0 ? b.totalScore / b.gamesPlayed : 0;
        return avgA - avgB;
      },
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { totalScore: number; gamesPlayed: number },
      ) => formatAvg(record.totalScore, record.gamesPlayed),
    },
    {
      title: (
        <ColHeader title="OScAV" tooltip="Average opponent score per game" />
      ),
      key: "avgOppScore",
      width: 55,
      sorter: (a: StandingRecord, b: StandingRecord) => {
        const avgA =
          a.gamesPlayed > 0 ? a.totalOpponentScore / a.gamesPlayed : 0;
        const avgB =
          b.gamesPlayed > 0 ? b.totalOpponentScore / b.gamesPlayed : 0;
        return avgA - avgB;
      },
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { totalOpponentScore: number; gamesPlayed: number },
      ) => formatAvg(record.totalOpponentScore, record.gamesPlayed),
    },
    {
      title: <ColHeader title="BAV" tooltip="Average bingos per game" />,
      key: "avgBingos",
      width: 45,
      sorter: (a: StandingRecord, b: StandingRecord) => {
        const avgA = a.gamesPlayed > 0 ? a.totalBingos / a.gamesPlayed : 0;
        const avgB = b.gamesPlayed > 0 ? b.totalBingos / b.gamesPlayed : 0;
        return avgA - avgB;
      },
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { totalBingos: number; gamesPlayed: number },
      ) => formatAvg(record.totalBingos, record.gamesPlayed),
    },
    {
      title: (
        <ColHeader title="OBAV" tooltip="Average opponent bingos per game" />
      ),
      key: "avgOppBingos",
      width: 50,
      sorter: (a: StandingRecord, b: StandingRecord) => {
        const avgA =
          a.gamesPlayed > 0 ? a.totalOpponentBingos / a.gamesPlayed : 0;
        const avgB =
          b.gamesPlayed > 0 ? b.totalOpponentBingos / b.gamesPlayed : 0;
        return avgA - avgB;
      },
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { totalOpponentBingos: number; gamesPlayed: number },
      ) => formatAvg(record.totalOpponentBingos, record.gamesPlayed),
    },
    {
      title: <ColHeader title="TAV" tooltip="Turn average (points per turn)" />,
      key: "turnAvg",
      width: 45,
      sorter: (a: StandingRecord, b: StandingRecord) => {
        const avgA = a.totalTurns > 0 ? a.totalScore / a.totalTurns : 0;
        const avgB = b.totalTurns > 0 ? b.totalScore / b.totalTurns : 0;
        return avgA - avgB;
      },
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { totalScore: number; totalTurns: number; gamesPlayed: number },
      ) =>
        record.gamesPlayed > 0 && record.totalTurns > 0
          ? (record.totalScore / record.totalTurns).toFixed(1)
          : "-",
    },
    {
      title: <ColHeader title="HG" tooltip="High game score" />,
      key: "highGame",
      width: 45,
      sorter: (a: StandingRecord, b: StandingRecord) => a.highGame - b.highGame,
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { highGame: number; gamesPlayed: number },
      ) => (record.gamesPlayed > 0 ? record.highGame : "-"),
    },
    {
      title: <ColHeader title="HT" tooltip="High turn score" />,
      key: "highTurn",
      width: 45,
      sorter: (a: StandingRecord, b: StandingRecord) => a.highTurn - b.highTurn,
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { highTurn: number; gamesPlayed: number },
      ) => (record.gamesPlayed > 0 ? record.highTurn : "-"),
    },
    {
      title: <ColHeader title="TiAV" tooltip="Average tiles played per game" />,
      key: "avgTiles",
      width: 45,
      sorter: (a: StandingRecord, b: StandingRecord) => {
        const avgA = a.gamesPlayed > 0 ? a.totalTilesPlayed / a.gamesPlayed : 0;
        const avgB = b.gamesPlayed > 0 ? b.totalTilesPlayed / b.gamesPlayed : 0;
        return avgA - avgB;
      },
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { totalTilesPlayed: number; gamesPlayed: number },
      ) => formatAvg(record.totalTilesPlayed, record.gamesPlayed),
    },
    {
      title: (
        <ColHeader
          title="OTiAV"
          tooltip="Average opponent tiles played per game"
        />
      ),
      key: "avgOppTiles",
      width: 50,
      sorter: (a: StandingRecord, b: StandingRecord) => {
        const avgA =
          a.gamesPlayed > 0 ? a.totalOpponentTilesPlayed / a.gamesPlayed : 0;
        const avgB =
          b.gamesPlayed > 0 ? b.totalOpponentTilesPlayed / b.gamesPlayed : 0;
        return avgA - avgB;
      },
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { totalOpponentTilesPlayed: number; gamesPlayed: number },
      ) => formatAvg(record.totalOpponentTilesPlayed, record.gamesPlayed),
    },
    {
      title: (
        <ColHeader title="#TAV" tooltip="Average number of turns per game" />
      ),
      key: "avgTurns",
      width: 50,
      sorter: (a: StandingRecord, b: StandingRecord) => {
        const avgA = a.gamesPlayed > 0 ? a.totalTurns / a.gamesPlayed : 0;
        const avgB = b.gamesPlayed > 0 ? b.totalTurns / b.gamesPlayed : 0;
        return avgA - avgB;
      },
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { totalTurns: number; gamesPlayed: number },
      ) => formatAvg(record.totalTurns, record.gamesPlayed),
    },
    {
      title: <ColHeader title="?AV" tooltip="Average blanks played per game" />,
      key: "avgBlanks",
      width: 40,
      sorter: (a: StandingRecord, b: StandingRecord) => {
        const avgA = a.gamesPlayed > 0 ? a.blanksPlayed / a.gamesPlayed : 0;
        const avgB = b.gamesPlayed > 0 ? b.blanksPlayed / b.gamesPlayed : 0;
        return avgA - avgB;
      },
      sortIcon: noSortIcon,
      render: (
        _: unknown,
        record: { blanksPlayed: number; gamesPlayed: number },
      ) => formatAvg(record.blanksPlayed, record.gamesPlayed),
    },
    {
      title: <ColHeader title="#TO" tooltip="Number of timeouts" />,
      dataIndex: "timeouts",
      key: "timeouts",
      width: 40,
      sorter: (a: StandingRecord, b: StandingRecord) => a.timeouts - b.timeouts,
      sortIcon: noSortIcon,
    },
    {
      title: <ColHeader title="Result" tooltip="Season result" />,
      dataIndex: "result",
      key: "result",
      render: (result: StandingResult) => {
        if (result === StandingResult.RESULT_PROMOTED) {
          return <Tag color="green">Promoted</Tag>;
        } else if (result === StandingResult.RESULT_RELEGATED) {
          return <Tag color="red">Relegated</Tag>;
        } else if (result === StandingResult.RESULT_CHAMPION) {
          return <Tag color="gold">Champion</Tag>;
        } else if (result === StandingResult.RESULT_STAYED) {
          return <Tag>Stayed</Tag>;
        }
        return null;
      },
    },
  ];

  return (
    <div className="division-standings">
      <div style={{ overflowX: "auto" }}>
        <Table
          columns={columns}
          dataSource={dataSource}
          pagination={false}
          size="small"
          rowClassName={getRowClassName}
          scroll={{ x: "max-content" }}
          showSorterTooltip={false}
        />
      </div>
      {selectedPlayer && (
        <PlayerGameHistoryModal
          visible={true}
          onClose={handleModalClose}
          userId={selectedPlayer.userId}
          username={selectedPlayer.username}
          seasonId={seasonId}
          seasonNumber={seasonNumber}
        />
      )}
    </div>
  );
};
