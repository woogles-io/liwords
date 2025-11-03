import React from "react";
import { Table, Tag } from "antd";
import { Division } from "../gen/api/proto/ipc/league_pb";
import { StandingResult } from "../gen/api/proto/ipc/league_pb";

type DivisionStandingsProps = {
  division: Division;
};

export const DivisionStandings: React.FC<DivisionStandingsProps> = ({
  division,
}) => {
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
      render: (username: string) => <strong>{username}</strong>,
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
        {division.divisionName && ` - ${division.divisionName}`}
      </h4>
      <Table
        columns={columns}
        dataSource={dataSource}
        pagination={false}
        size="small"
      />
    </div>
  );
};
