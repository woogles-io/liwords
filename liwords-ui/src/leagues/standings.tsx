import React, { useState } from "react";
import { Table, Tag } from "antd";
import { Division, PromotionFormula } from "../gen/api/proto/ipc/league_pb";
import { StandingResult } from "../gen/api/proto/ipc/league_pb";
import { PlayerGameHistoryModal } from "./player_game_history_modal";

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
};

export const DivisionStandings: React.FC<DivisionStandingsProps> = ({
  division,
  totalDivisions,
  seasonId,
  seasonNumber,
  currentUserId,
  promotionFormula = PromotionFormula.PROMO_N_DIV_6,
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

  const getRowClassName = (_record: unknown, index: number) => {
    const rank = index + 1;
    if (promotionCount > 0 && rank <= promotionCount) {
      return "promotion-row";
    }
    if (relegationCount > 0 && rank > divisionSize - relegationCount) {
      return "relegation-row";
    }
    return "";
  };
  const columns = [
    {
      title: "Rank",
      dataIndex: "rank",
      key: "rank",
      width: 70,
    },
    {
      title: "Player",
      dataIndex: "username",
      key: "username",
      render: (username: string, record: { userId: string }) => {
        const isCurrentUser = record.userId === currentUserId;
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
          </span>
        );
      },
    },
    {
      title: "Points",
      key: "points",
      className: "points-column",
      render: (_: unknown, record: { wins: number; draws: number }) =>
        record.wins * 2 + record.draws,
    },
    {
      title: "W-L-D",
      key: "record",
      render: (
        _: unknown,
        record: { wins: number; losses: number; draws: number },
      ) => `${record.wins}-${record.losses}-${record.draws}`,
    },
    {
      title: "Spread",
      dataIndex: "spread",
      key: "spread",
      render: (spread: number) => (spread > 0 ? `+${spread}` : spread),
    },
    {
      title: "Games",
      key: "games",
      render: (
        _: unknown,
        record: { gamesPlayed: number; gamesRemaining: number },
      ) =>
        `${record.gamesPlayed} / ${record.gamesPlayed + record.gamesRemaining}`,
    },
    {
      title: "Result",
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
  }));

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
