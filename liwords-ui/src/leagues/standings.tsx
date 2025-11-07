import React, { useState } from "react";
import { Table, Tag } from "antd";
import { StarFilled } from "@ant-design/icons";
import { Division } from "../gen/api/proto/ipc/league_pb";
import { StandingResult } from "../gen/api/proto/ipc/league_pb";
import { PlayerGameHistoryModal } from "./player_game_history_modal";

type DivisionStandingsProps = {
  division: Division;
  totalDivisions: number;
  seasonId: string;
  seasonNumber: number;
  currentUserId?: string;
};

export const DivisionStandings: React.FC<DivisionStandingsProps> = ({
  division,
  totalDivisions,
  seasonId,
  seasonNumber,
  currentUserId,
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

  // Calculate promotion/relegation zones
  const divisionSize = division.standings.length;
  const promotionCount =
    division.divisionNumber === 1 ? 0 : Math.ceil(divisionSize / 6);
  const relegationCount =
    division.divisionNumber === totalDivisions
      ? 0
      : Math.ceil(divisionSize / 6);

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
      title: "",
      key: "isCurrentUser",
      width: 40,
      render: (_: unknown, record: { userId: string }) =>
        record.userId === currentUserId ? (
          <StarFilled style={{ color: "#faad14" }} />
        ) : null,
    },
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
      render: (username: string, record: { userId: string }) => (
        <span
          className="clickable-player"
          onClick={() => handlePlayerClick(record.userId, username)}
        >
          <strong>{username}</strong>
        </span>
      ),
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
      <h4>
        Division {division.divisionNumber}
        {division.divisionName &&
          division.divisionName !== `Division ${division.divisionNumber}` &&
          ` - ${division.divisionName}`}
      </h4>
      <Table
        columns={columns}
        dataSource={dataSource}
        pagination={false}
        size="small"
        rowClassName={getRowClassName}
      />
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
