import React from "react";
import { Table, Tag, Typography } from "antd";
import type { BroadcastGameStat } from "../../gen/api/proto/broadcast_service/broadcast_service_pb";
import { GameLink } from "./GameLink";

const { Text } = Typography;

function spreadLabel(p1: number, p2: number, winner: number): string {
  const spread = Math.abs(p1 - p2);
  return winner === 0 ? `+${spread}` : winner === 1 ? `-${spread}` : "±0";
}

type Props = {
  stats: BroadcastGameStat[];
};

export const RecentlyCompletedTab: React.FC<Props> = ({ stats }) => {
  const completed = stats
    .filter((s) => s.completedAt)
    .sort(
      (a, b) => Number(b.completedAt!.seconds) - Number(a.completedAt!.seconds),
    )
    .slice(0, 20);

  if (completed.length === 0) {
    return (
      <Text type="secondary" style={{ display: "block", marginTop: 24 }}>
        No completed games yet.
      </Text>
    );
  }

  return (
    <Table
      style={{ marginTop: 16 }}
      rowKey="gameUuid"
      dataSource={completed}
      pagination={false}
      size="small"
      columns={[
        {
          title: "Rd",
          dataIndex: "round",
          key: "round",
          width: 45,
        },
        {
          title: "Players",
          key: "players",
          render: (_, r) => {
            const p1Won = r.winner === 0;
            const p2Won = r.winner === 1;
            return (
              <span>
                <Text strong={p1Won}>{r.player1Name}</Text>
                <Text type="secondary"> vs </Text>
                <Text strong={p2Won}>{r.player2Name}</Text>
              </span>
            );
          },
        },
        {
          title: "Score",
          key: "score",
          width: 120,
          render: (_, r) => `${r.player1Score} – ${r.player2Score}`,
        },
        {
          title: "Spread",
          key: "spread",
          width: 70,
          render: (_, r) =>
            spreadLabel(r.player1Score, r.player2Score, r.winner),
        },
        {
          title: "Bingos",
          key: "bingos",
          width: 65,
          render: (_, r) => {
            const total = r.player1Bingos + r.player2Bingos;
            return total > 0 ? total : <Text type="secondary">—</Text>;
          },
        },
        {
          title: "",
          key: "flags",
          width: 60,
          render: (_, r) =>
            r.walkOffBingo ? (
              <Tag color="gold" title="Walk-off bingo">
                WOB
              </Tag>
            ) : null,
        },
        {
          title: "",
          key: "link",
          width: 80,
          render: (_, r) =>
            r.gameUuid ? (
              <GameLink gameUuid={r.gameUuid} label="Review" />
            ) : null,
        },
      ]}
    />
  );
};
