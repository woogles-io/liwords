import React from "react";
import { Table, Typography } from "antd";
import type { BroadcastPlayer } from "../../gen/api/proto/broadcast_service/broadcast_service_pb";

const { Text } = Typography;

type Props = {
  players: BroadcastPlayer[];
};

export const StandingsTab: React.FC<Props> = ({ players }) => {
  if (players.length === 0) {
    return (
      <Text type="secondary" style={{ display: "block", marginTop: 24 }}>
        No standings available yet.
      </Text>
    );
  }

  const dataSource = players.map((p, i) => ({
    key: p.playerId,
    rank: i + 1,
    name: p.name,
    rating: p.rating,
    wins: p.wins,
    losses: p.losses,
    spread: p.spread,
    gamesPlayed: p.gamesPlayed,
  }));

  return (
    <Table
      style={{ marginTop: 16 }}
      dataSource={dataSource}
      pagination={false}
      size="small"
      columns={[
        {
          title: "#",
          dataIndex: "rank",
          key: "rank",
          width: 40,
        },
        {
          title: "Player",
          dataIndex: "name",
          key: "name",
          sorter: (a, b) => a.name.localeCompare(b.name),
        },
        {
          title: "Rating",
          dataIndex: "rating",
          key: "rating",
          width: 70,
          sorter: (a, b) => a.rating - b.rating,
          render: (r: number) => r || "—",
        },
        {
          title: "W-L",
          key: "record",
          width: 70,
          sorter: (a, b) => {
            const ap = a.wins * 2 - a.losses * 2;
            const bp = b.wins * 2 - b.losses * 2;
            return ap !== bp ? ap - bp : a.spread - b.spread;
          },
          render: (_, r) =>
            `${r.wins % 1 === 0 ? r.wins : r.wins.toFixed(1)}-${r.losses % 1 === 0 ? r.losses : r.losses.toFixed(1)}`,
        },
        {
          title: "Spread",
          dataIndex: "spread",
          key: "spread",
          width: 70,
          sorter: (a, b) => a.spread - b.spread,
          render: (s: number) => (s > 0 ? `+${s}` : s),
        },
        {
          title: "GP",
          dataIndex: "gamesPlayed",
          key: "gamesPlayed",
          width: 50,
        },
      ]}
    />
  );
};
