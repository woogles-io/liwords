import React, { useMemo, useState } from "react";
import { Input, Select, Space, Table, Typography } from "antd";
import type { BroadcastGameStat } from "../../gen/api/proto/broadcast_service/broadcast_service_pb";
import { GameLink } from "./GameLink";

const { Text } = Typography;

type Props = {
  stats: BroadcastGameStat[];
  totalRounds: number;
};

export const ArchiveTab: React.FC<Props> = ({ stats, totalRounds }) => {
  const [roundFilter, setRoundFilter] = useState<number>(0);
  const [playerFilter, setPlayerFilter] = useState("");

  const filtered = useMemo(() => {
    let rows = stats;
    if (roundFilter > 0) rows = rows.filter((r) => r.round === roundFilter);
    if (playerFilter.trim()) {
      const q = playerFilter.trim().toLowerCase();
      rows = rows.filter(
        (r) =>
          r.player1Name.toLowerCase().includes(q) ||
          r.player2Name.toLowerCase().includes(q),
      );
    }
    return rows;
  }, [stats, roundFilter, playerFilter]);

  const roundOptions = [
    { value: 0, label: "All rounds" },
    ...Array.from({ length: totalRounds }, (_, i) => ({
      value: i + 1,
      label: `Round ${i + 1}`,
    })),
  ];

  return (
    <div>
      <Space style={{ marginTop: 16, marginBottom: 8 }} wrap>
        <Select
          value={roundFilter}
          onChange={setRoundFilter}
          options={roundOptions}
          style={{ width: 140 }}
        />
        <Input.Search
          placeholder="Filter by player"
          value={playerFilter}
          onChange={(e) => setPlayerFilter(e.target.value)}
          onSearch={setPlayerFilter}
          allowClear
          style={{ width: 200 }}
        />
        <Text type="secondary">{filtered.length} games</Text>
      </Space>
      <Table
        rowKey="gameUuid"
        dataSource={filtered}
        pagination={{
          pageSize: 50,
          showSizeChanger: false,
          hideOnSinglePage: true,
        }}
        size="small"
        columns={[
          {
            title: "Rd",
            dataIndex: "round",
            key: "round",
            width: 45,
            sorter: (a, b) =>
              a.round !== b.round
                ? a.round - b.round
                : a.tableNumber - b.tableNumber,
            defaultSortOrder: "ascend",
          },
          {
            title: "Tbl",
            dataIndex: "tableNumber",
            key: "tableNumber",
            width: 45,
            render: (n: number) => `#${n}`,
          },
          {
            title: "Went First",
            key: "p1",
            render: (_, r) => {
              const name = r.player1GoesFirst ? r.player1Name : r.player2Name;
              const won = r.player1GoesFirst ? r.winner === 0 : r.winner === 1;
              return <Text strong={won}>{name}</Text>;
            },
          },
          {
            title: "Went Second",
            key: "p2",
            render: (_, r) => {
              const name = r.player1GoesFirst ? r.player2Name : r.player1Name;
              const won = r.player1GoesFirst ? r.winner === 1 : r.winner === 0;
              return <Text strong={won}>{name}</Text>;
            },
          },
          {
            title: "Score",
            key: "score",
            width: 120,
            render: (_, r) => `${r.player1Score} – ${r.player2Score}`,
            sorter: (a, b) =>
              Math.max(a.player1Score, a.player2Score) -
              Math.max(b.player1Score, b.player2Score),
          },
          {
            title: "Spread",
            key: "spread",
            width: 65,
            sorter: (a, b) =>
              Math.abs(a.player1Score - a.player2Score) -
              Math.abs(b.player1Score - b.player2Score),
            render: (_, r) => r.player1Score - r.player2Score,
          },
          {
            title: "Combined",
            key: "combined",
            width: 80,
            sorter: (a, b) =>
              a.player1Score +
              a.player2Score -
              (b.player1Score + b.player2Score),
            render: (_, r) => r.player1Score + r.player2Score,
          },
          {
            title: "Bingos",
            key: "bingos",
            width: 65,
            sorter: (a, b) =>
              a.player1Bingos +
              a.player2Bingos -
              (b.player1Bingos + b.player2Bingos),
            render: (_, r) => r.player1Bingos + r.player2Bingos || "—",
          },
          {
            title: "Moves",
            dataIndex: "moveCount",
            key: "moves",
            width: 60,
            sorter: (a, b) => a.moveCount - b.moveCount,
            render: (n: number) => n || "—",
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
    </div>
  );
};
