import React, { useCallback, useEffect, useState } from "react";
import { Spin, Table, type TableColumnsType, Tooltip, Typography } from "antd";
import { useQuery } from "@connectrpc/connect-query";
import { Division } from "../gen/api/proto/ipc/league_pb";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { getDivisionCrosstable } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { CrosstableGame } from "../gen/api/proto/league_service/league_service_pb";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { trackMenuItem } from "./follow_bar";
import { endReasonLabel } from "./player_game_history_modal";

const { Text } = Typography;

// One game as seen by the row player.
export type CrosstableCell = {
  gameId: string;
  opponentId: string;
  endReason: GameEndReason;
  myScore: number;
  theirScore: number;
  // "win", "loss" or "draw" once finished; "turn" (the row player's) or
  // "waiting" (the opponent's) while in progress.
  result: "win" | "loss" | "draw" | "turn" | "waiting";
};

// A game still in progress.
export const isLive = (g: CrosstableCell) =>
  g.result === "turn" || g.result === "waiting";

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
        opponentId: them,
        endReason: g.gameEndReason,
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

// Green ahead or won, red behind or lost, gray when level (as elsewhere on
// the site). A finished game goes by its result, so a win on time while
// behind is still green; a live one by its current score.
export const scoreType = (
  g: CrosstableCell,
): "success" | "danger" | "secondary" => {
  if (g.result === "win") return "success";
  if (g.result === "loss") return "danger";
  if (g.result === "draw") return "secondary";
  if (g.myScore > g.theirScore) return "success";
  if (g.myScore < g.theirScore) return "danger";
  return "secondary";
};

// A letter after the score when a game did not end by being played out, so a
// win on time while behind on score reads as what it was.
export const endMark = (reason: GameEndReason): string => {
  switch (reason) {
    case GameEndReason.TIME:
      return "T";
    case GameEndReason.RESIGNED:
      return "R";
    case GameEndReason.FORCE_FORFEIT:
      return "F";
    case GameEndReason.TRIPLE_CHALLENGE:
      // Not "3": a digit after the score would read as part of it.
      return "C";
    case GameEndReason.ADJUDICATED:
      return "A";
    case GameEndReason.CONSECUTIVE_ZEROES:
      return "Z";
    default:
      return "";
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

// One game's score as the row player sees it, linked to the game. The hover
// says who played, how it ended and by how much.
const GameScore = (props: { g: CrosstableCell; me: string; them: string }) => {
  const { g } = props;
  const spread = g.myScore - g.theirScore;
  const how = endReasonLabel(g.endReason);
  const live = isLive(g);
  const title = `${props.me} vs ${props.them}: ${resultLabel(g.result)}${
    !live && how ? ` (${how})` : ""
  }, ${spread >= 0 ? "+" : ""}${spread}`;
  return (
    <Tooltip title={title}>
      <a
        href={`/game/${g.gameId}`}
        target="_blank"
        rel="noreferrer"
        style={{ whiteSpace: "nowrap" }}
      >
        <Text type={scoreType(g)} italic={live}>
          {`${g.myScore}-${g.theirScore}`}
          {live ? "" : endMark(g.endReason)}
          {g.result === "turn" ? "*" : ""}
        </Text>
      </a>
    </Tooltip>
  );
};

// The cell under the pointer, by its row and column players.
export type CrossHover = { row: string; col: string } | null;

// How a grid cell shows the hovered game: its own cell and the mirror cell
// (the same game from the other player's side) stand out, and the rows and
// columns of both players are shaded, so the pair is easy to find.
export const crossClass = (
  row: string,
  col: string,
  hover: CrossHover,
): string => {
  if (!hover) return "";
  if (
    (row === hover.row && col === hover.col) ||
    (row === hover.col && col === hover.row)
  ) {
    return "xt-game";
  }
  const pair = [hover.row, hover.col];
  return pair.includes(row) || pair.includes(col) ? "xt-cross" : "";
};

// A player's official standing: finished games only, as the standings show it.
type Official = {
  userId: string;
  wins: number;
  losses: number;
  draws: number;
  spread: number;
};

// A player's record, spread and rank if every game in progress ended now at
// its current score: the leader wins and level scores draw, which is how a
// season's unfinished games are decided when it closes. Finished games count
// as the official standings record them (forfeits and time losses included).
export type IfEndedNow = {
  wins: number;
  losses: number;
  draws: number;
  spread: number;
  rank: number;
  // shares its rank with another player
  tied: boolean;
  // no game in progress, so the record and spread are already the official
  // ones
  final: boolean;
};

// Whether anyone's record if every game ended now has a draw. The record's
// hover names draws only then, as the standings' does.
export const anyDraws = (now: Map<string, IfEndedNow>) =>
  [...now.values()].some((n) => n.draws > 0);

// Starts from each player's official standing and adds only their games in
// progress from the cross-table, then ranks the players by points (two for a
// win, one for a draw) and then spread. Tied players share a rank.
export const ifEndedNow = (
  standings: Array<Official>,
  cells: Map<string, Array<CrosstableCell>>,
): Map<string, IfEndedNow> => {
  const userIds = standings.map((s) => s.userId);
  const totals = standings.map((s) => {
    const t = {
      wins: s.wins,
      losses: s.losses,
      draws: s.draws,
      spread: s.spread,
      final: true,
    };
    for (const them of userIds) {
      for (const g of cells.get(cellKey(s.userId, them)) ?? []) {
        if (!isLive(g)) continue;
        t.final = false;
        const margin = g.myScore - g.theirScore;
        if (margin > 0) t.wins++;
        else if (margin < 0) t.losses++;
        else t.draws++;
        t.spread += margin;
      }
    }
    return t;
  });
  const points = (t: { wins: number; draws: number }) => t.wins * 2 + t.draws;
  const out = new Map<string, IfEndedNow>();
  totals.forEach((t, i) => {
    const above = totals.filter(
      (o) =>
        points(o) > points(t) ||
        (points(o) === points(t) && o.spread > t.spread),
    ).length;
    const tied = totals.some(
      (o) => o !== t && points(o) === points(t) && o.spread === t.spread,
    );
    out.set(userIds[i], { ...t, rank: above + 1, tied });
  });
  return out;
};

// The width the cross-table needs before its totals stay in view at the right:
// the names and the totals take about 320px, which leaves room for two or more
// games between them. It goes by the table, not the screen, since the page's
// side panels can leave the table narrow on a wide screen.
const PIN_TOTALS_WIDTH = 480;

// Every pairing of a division: row player's score first, colored by scoreType;
// a game in progress is in italics (with * when the row player is on turn); a letter after the score says how a game ended when
// it was not played out. Always a grid, however wide: a player's games as a
// list are in the standings' game history. While any game is in progress, the
// grid ends with each player's totals if every game ended now (ifEndedNow).
export const DivisionCrosstable = (props: {
  division: Division;
  onChat?: (uuid: string, username: string) => void;
  // called when the cross-table cannot be loaded, e.g. from a server that
  // does not have it yet, so the page can drop it.
  onError?: () => void;
  // tracks a player through their seasons, from their name's menu.
  onTrack?: (userId: string) => void;
}) => {
  const { data, isLoading, error } = useQuery(
    getDivisionCrosstable,
    { divisionId: props.division.uuid },
    // a missing endpoint will not appear on a retry; drop the panel at once.
    { retry: false },
  );
  const [hover, setHover] = useState<CrossHover>(null);
  // whether the cross-table itself is wide enough to keep the totals in view
  // at the right while the grid scrolls (PIN_TOTALS_WIDTH).
  const [wide, setWide] = useState(false);
  const watchWidth = useCallback((el: HTMLDivElement | null) => {
    if (!el) return;
    const ro = new ResizeObserver(([entry]) =>
      setWide(entry.contentRect.width >= PIN_TOTALS_WIDTH),
    );
    ro.observe(el);
    return () => ro.disconnect();
  }, []);
  const { onError } = props;
  useEffect(() => {
    if (error) onError?.();
  }, [error, onError]);

  const rows: Array<Row> = props.division.standings.map((s, i) => ({
    key: s.userId,
    userId: s.userId,
    username: s.username,
    rank: i + 1,
  }));
  const userIds = rows.map((r) => r.userId);
  const cells = buildCrosstable(userIds, data?.games ?? []);
  const now = ifEndedNow(props.division.standings, cells);
  // with nothing in progress the totals would only repeat the standings.
  const anyLive = [...cells.values()].some((list) => list.some(isLive));

  if (isLoading) {
    return <Spin />;
  }
  if (error) {
    return null;
  }

  // A hovered game's two players, picked out in the fixed columns.
  const playerCell = (row: Row) => ({
    className:
      hover && (row.userId === hover.row || row.userId === hover.col)
        ? "xt-player"
        : "",
  });

  const fixedColumns: TableColumnsType<Row> = [
    {
      title: "#",
      dataIndex: "rank",
      key: "rank",
      fixed: "left" as const,
      onCell: playerCell,
    },
    {
      title: "Player",
      key: "username",
      fixed: "left" as const,
      onCell: playerCell,
      render: (_: unknown, row: Row) => (
        <UsernameWithContext
          username={row.username}
          userID={row.userId}
          sendMessage={props.onChat}
          omitSendMessage={!props.onChat}
          extraItems={
            props.onTrack
              ? [trackMenuItem(() => props.onTrack?.(row.userId))]
              : undefined
          }
        />
      ),
    },
  ];

  // Each player's totals if every game in progress ended now, in italics like
  // the live scores they count. A player with no game in progress has a final
  // record and spread, shown upright; their rank stays italic, since the
  // others' games can still move it.
  const nowFixed = wide ? ("right" as const) : undefined;
  const nowColumn = (
    key: string,
    title: string,
    tip: string,
    show: (n: IfEndedNow) => string,
    settled: (n: IfEndedNow) => boolean,
  ) => ({
    title: (
      <Tooltip title={tip}>
        <span style={{ cursor: "help" }}>{title}</span>
      </Tooltip>
    ),
    key,
    align: "center" as const,
    fixed: nowFixed,
    onCell: playerCell,
    render: (_: unknown, row: Row) => {
      const n = now.get(row.userId);
      return n && <Text italic={!settled(n)}>{show(n)}</Text>;
    },
  });
  const nowColumns: TableColumnsType<Row> = anyLive
    ? [
        {
          title: (
            <Tooltip title="The standings with every game in progress counted as if it ended now at its current score.">
              <span style={{ cursor: "help" }}>If ended now</span>
            </Tooltip>
          ),
          key: "now",
          fixed: nowFixed,
          children: [
            nowColumn(
              "nowRecord",
              "Rec",
              anyDraws(now) ? "Win-Loss-Draw record" : "Win-Loss record",
              (n) => `${n.wins}-${n.losses}${n.draws ? `-${n.draws}` : ""}`,
              (n) => n.final,
            ),
            nowColumn(
              "nowSpread",
              "Spr",
              "Cumulative spread",
              (n) => (n.spread > 0 ? `+${n.spread}` : `${n.spread}`),
              (n) => n.final,
            ),
            nowColumn(
              "nowRank",
              "#",
              "Rank in division",
              (n) => `${n.rank}${n.tied ? "=" : ""}`,
              () => false,
            ),
          ],
        },
      ]
    : [];

  const columns: TableColumnsType<Row> = [
    ...fixedColumns,
    ...rows.map((col) => ({
      title: (
        <Tooltip title={col.username}>
          <span style={{ cursor: "help" }}>{col.rank}</span>
        </Tooltip>
      ),
      key: col.userId,
      align: "center" as const,
      onHeaderCell: () => ({
        className:
          hover && (col.userId === hover.row || col.userId === hover.col)
            ? "xt-player"
            : "",
      }),
      onCell: (row: Row) => ({
        className: crossClass(row.userId, col.userId, hover),
        onMouseEnter: () =>
          row.userId !== col.userId &&
          setHover({ row: row.userId, col: col.userId }),
        onMouseLeave: () => setHover(null),
      }),
      render: (_: unknown, row: Row) => {
        if (row.userId === col.userId) {
          return <Text type="secondary">-</Text>;
        }
        return (cells.get(cellKey(row.userId, col.userId)) ?? []).map((g) => (
          <div key={g.gameId}>
            <GameScore g={g} me={row.username} them={col.username} />
          </div>
        ));
      },
    })),
    ...nowColumns,
  ];

  return (
    <div ref={watchWidth}>
      <Table
        className="division-crosstable"
        columns={columns}
        dataSource={rows}
        pagination={false}
        size="small"
        scroll={{ x: "max-content" }}
      />
    </div>
  );
};
