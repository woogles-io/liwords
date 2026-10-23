import React from "react";
import { Spin, Table, type TableColumnsType, Tooltip, Typography } from "antd";
import { useQuery } from "@connectrpc/connect-query";
import { Division } from "../gen/api/proto/ipc/league_pb";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { getDivisionCrosstable } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { CrosstableGame } from "../gen/api/proto/league_service/league_service_pb";

const { Text } = Typography;

// One game as seen by the row player.
export type CrosstableCell = {
  gameId: string;
  myScore: number;
  theirScore: number;
  // "win", "loss" or "draw" once finished; "turn" (the row player's) or
  // "waiting" (the opponent's) while in progress.
  result: "win" | "loss" | "draw" | "turn" | "waiting";
};

const cellKey = (rowUserId: string, colUserId: string) =>
  `${rowUserId}|${colUserId}`;

// Maps each (row player, column player) pair to their games, from the row
// player's side. Each pair should meet once a season, but every game found is
// kept so bad data shows up instead of breaking the table; a game naming a
// player outside the division is skipped.
export const buildCrosstable = (
  userIds: Array<string>,
  games: Array<CrosstableGame>,
): Map<string, Array<CrosstableCell>> => {
  const members = new Set(userIds);
  const cells = new Map<string, Array<CrosstableCell>>();
  for (const g of games) {
    const sides: Array<[string, string, number]> = [
      [g.player0UserId, g.player1UserId, 0],
      [g.player1UserId, g.player0UserId, 1],
    ];
    for (const [me, them, idx] of sides) {
      if (!members.has(me) || !members.has(them)) continue;
      const live = g.gameEndReason === GameEndReason.NONE;
      let result: CrosstableCell["result"];
      if (live) {
        result = g.playerOnTurn === idx ? "turn" : "waiting";
      } else if (g.winnerIndex === -1) {
        result = "draw";
      } else {
        result = g.winnerIndex === idx ? "win" : "loss";
      }
      const cell: CrosstableCell = {
        gameId: g.gameId,
        myScore: idx === 0 ? g.player0Score : g.player1Score,
        theirScore: idx === 0 ? g.player1Score : g.player0Score,
        result,
      };
      const key = cellKey(me, them);
      const list = cells.get(key);
      if (list) {
        list.push(cell);
      } else {
        cells.set(key, [cell]);
      }
    }
  }
  return cells;
};

const resultType = (
  result: CrosstableCell["result"],
): "success" | "danger" | "secondary" | undefined => {
  switch (result) {
    case "win":
      return "success";
    case "loss":
      return "danger";
    case "draw":
      return undefined;
    default:
      return "secondary";
  }
};

const resultLabel = (result: CrosstableCell["result"]): string => {
  switch (result) {
    case "turn":
      return "in progress, row player on turn";
    case "waiting":
      return "in progress, column player on turn";
    default:
      return result;
  }
};

type Row = {
  key: string;
  userId: string;
  username: string;
  rank: number;
};

// Every pairing of a division in one grid: row player's score first, green for
// a win, red for a loss, gray (with * when the row player is on turn) for a
// game in progress. A cell opens its game.
export const DivisionCrosstable = (props: { division: Division }) => {
  const { data, isLoading, error } = useQuery(getDivisionCrosstable, {
    divisionId: props.division.uuid,
  });

  const rows: Array<Row> = props.division.standings.map((s, i) => ({
    key: s.userId,
    userId: s.userId,
    username: s.username,
    rank: i + 1,
  }));
  const cells = buildCrosstable(
    rows.map((r) => r.userId),
    data?.games ?? [],
  );

  if (isLoading) {
    return <Spin />;
  }
  if (error) {
    return (
      <Text type="danger">Failed to load crosstable: {error.message}</Text>
    );
  }

  const columns: TableColumnsType<Row> = [
    { title: "#", dataIndex: "rank", key: "rank", fixed: "left" as const },
    {
      title: "Player",
      dataIndex: "username",
      key: "username",
      fixed: "left" as const,
    },
    ...rows.map((col) => ({
      title: (
        <Tooltip title={col.username}>
          <span style={{ cursor: "help" }}>{col.rank}</span>
        </Tooltip>
      ),
      key: col.userId,
      align: "center" as const,
      render: (_: unknown, row: Row) => {
        if (row.userId === col.userId) {
          return <Text type="secondary">-</Text>;
        }
        const games = cells.get(cellKey(row.userId, col.userId)) ?? [];
        if (games.length === 0) {
          return null;
        }
        return (
          <React.Fragment>
            {games.map((g) => (
              <div key={g.gameId}>
                <Tooltip
                  title={`${row.username} vs ${col.username}: ${resultLabel(g.result)}`}
                >
                  <a
                    href={`/game/${g.gameId}`}
                    target="_blank"
                    rel="noreferrer"
                    style={{ whiteSpace: "nowrap" }}
                  >
                    <Text type={resultType(g.result)}>
                      {g.myScore}-{g.theirScore}
                      {g.result === "turn" ? "*" : ""}
                    </Text>
                  </a>
                </Tooltip>
              </div>
            ))}
          </React.Fragment>
        );
      },
    })),
  ];

  return (
    <Table
      className="division-crosstable"
      columns={columns}
      dataSource={rows}
      pagination={false}
      size="small"
      scroll={{ x: "max-content" }}
    />
  );
};
