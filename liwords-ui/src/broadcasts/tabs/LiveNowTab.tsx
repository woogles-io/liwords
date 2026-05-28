import React from "react";
import { Table, Tag, Typography } from "antd";
import { PlayCircleOutlined } from "@ant-design/icons";
import type { BroadcastGameStat } from "../../gen/api/proto/broadcast_service/broadcast_service_pb";
import { GameLink } from "./GameLink";

const { Text } = Typography;

type Props = {
  stats: BroadcastGameStat[];
};

export const LiveNowTab: React.FC<Props> = ({ stats }) => {
  const live = stats.filter((s) => !s.completedAt && s.gameUuid);

  if (live.length === 0) {
    return (
      <Text type="secondary" style={{ display: "block", marginTop: 24 }}>
        No games currently being annotated live.
      </Text>
    );
  }

  return (
    <Table
      style={{ marginTop: 16 }}
      rowKey="gameUuid"
      dataSource={live}
      pagination={false}
      size="small"
      columns={[
        {
          title: "",
          key: "live",
          width: 60,
          render: () => (
            <Tag color="red" icon={<PlayCircleOutlined />}>
              LIVE
            </Tag>
          ),
        },
        {
          title: "Rd",
          dataIndex: "round",
          key: "round",
          width: 45,
        },
        {
          title: "Table",
          dataIndex: "tableNumber",
          key: "tableNumber",
          width: 55,
          render: (n: number) => <Text strong>#{n}</Text>,
        },
        {
          title: "Players",
          key: "players",
          render: (_, r) => (
            <span>
              {r.player1Name} <Text type="secondary">vs</Text> {r.player2Name}
            </span>
          ),
        },
        {
          title: "Score",
          key: "score",
          width: 110,
          render: (_, r) => {
            if (!r.currentScore1 && !r.currentScore2) {
              return <Text type="secondary">—</Text>;
            }
            return `${r.currentScore1} – ${r.currentScore2}`;
          },
        },
        {
          title: "",
          key: "link",
          width: 90,
          render: (_, r) =>
            r.gameUuid ? <GameLink gameUuid={r.gameUuid} /> : null,
        },
      ]}
    />
  );
};
