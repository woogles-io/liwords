import React from "react";
import {
  Modal,
  Spin,
  Table,
  type TableColumnsType,
  Tag,
  Tooltip,
  Typography,
} from "antd";
import { useQuery } from "@connectrpc/connect-query";
import { getPlayerSeasonGames } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { timestampDate, type Timestamp } from "@bufbuild/protobuf/wkt";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { CorrespondenceTurnIndicator } from "../shared/corres_turn_indicator";
import { UsernameWithContext } from "../shared/usernameWithContext";

const { Text } = Typography;

export const endReasonLabel = (reason: GameEndReason): string => {
  switch (reason) {
    case GameEndReason.TIME:
      return "time";
    case GameEndReason.CONSECUTIVE_ZEROES:
      return "zeroes";
    case GameEndReason.RESIGNED:
      return "resigned";
    case GameEndReason.TRIPLE_CHALLENGE:
      return "triple";
    case GameEndReason.FORCE_FORFEIT:
      return "forfeit";
    case GameEndReason.ADJUDICATED:
      return "adjudicated";
    case GameEndReason.ABORTED:
      return "aborted";
    case GameEndReason.CANCELLED:
      return "cancelled";
    case GameEndReason.STANDARD:
      return "";
    case GameEndReason.NONE:
    default:
      return "other";
  }
};

type PlayerGameHistoryModalProps = {
  visible: boolean;
  onClose: () => void;
  userId: string;
  username: string;
  seasonId: string;
  seasonNumber: number;
  // The season's start (all league games are created then). Passed in from the
  // league page, which already has it, so the title's date range does not
  // refetch it. When absent the range falls back to the games' own span.
  seasonStartDate?: Timestamp;
  onChat?: (uuid: string, username: string) => void;
};

// One row of the season game list. Finished games carry scores, a result and an
// optional mistake index (absent until the game has been analyzed); in-progress
// games carry the live-clock anchor (lastUpdateMs / incrementSecs /
// onTurnTimeBankMs) so the clock can tick. gameDate is the last-updated time for
// both, and is what the list sorts by, by default.
export type GameRow = {
  key: string;
  gameId: string;
  opponentUsername: string;
  opponentUserId: string;
  result: string;
  playerScore: number;
  opponentScore: number;
  mistakeIndex?: number;
  gameDate?: Date;
  gameEndReason: GameEndReason;
  lastUpdateMs?: number;
  incrementSecs: number;
  onTurnTimeBankMs: number;
};

// Sort rank for the Result column: live games first (they are the actionable
// ones), then wins, draws, losses. The sorter breaks ties by spread.
export const resultRank = (result: string): number => {
  switch (result) {
    case "turn":
      return 0;
    case "in_progress":
      return 1;
    case "win":
      return 2;
    case "draw":
      return 3;
    case "loss":
      return 4;
    default:
      return 5;
  }
};

// Composite band for the combined Time/Mistakes column: not-done games first
// (your turn, then opponent's), then done games (just-finished, then analyzed).
// The band groups not-done vs done; the sorter orders within each band.
export const timeMistakeBand = (record: GameRow): number => {
  if (record.result === "turn") return 0;
  if (record.result === "in_progress") return 1;
  if (record.mistakeIndex === undefined) return 2; // finished, unanalyzed
  return 3; // finished, analyzed
};

// Milliseconds until this player's on-turn game forfeits (per-turn allowance +
// time bank - elapsed), matching the deadline the clock ticks toward. Only
// meaningful for a live game with a clock anchor; unknown clocks sort last.
export const clockRemainingMs = (record: GameRow): number => {
  if (record.lastUpdateMs === undefined) return Number.MAX_SAFE_INTEGER;
  return (
    record.incrementSecs * 1000 +
    record.onTurnTimeBankMs -
    (Date.now() - record.lastUpdateMs)
  );
};

// Column sorters, pulled out as named functions so they can be unit-tested --
// the modal has no local season data to exercise them by hand. All sort
// ascending; antd's header toggle reverses. The direction each encodes (Result
// live-first, timeMistake not-done-first) is the useful default.
export const opponentSorter = (a: GameRow, b: GameRow): number =>
  a.opponentUsername.localeCompare(b.opponentUsername);

// Result: live games first (actionable), then win/draw/loss, ties by spread.
export const resultSorter = (a: GameRow, b: GameRow): number =>
  resultRank(a.result) - resultRank(b.result) ||
  b.playerScore - b.opponentScore - (a.playerScore - a.opponentScore);

// Score: by spread (player - opponent).
export const scoreSorter = (a: GameRow, b: GameRow): number =>
  a.playerScore - a.opponentScore - (b.playerScore - b.opponentScore);

// Date: by last-updated timestamp.
export const dateSorter = (a: GameRow, b: GameRow): number =>
  (a.gameDate?.getTime() ?? 0) - (b.gameDate?.getTime() ?? 0);

// Time/Mistakes: not-done games first (your turn by soonest deadline, then
// opponent's by recency), then done games (just-finished by recency, then
// analyzed by worst mistake first) -- so one click surfaces what needs a move,
// and the reverse toggle surfaces the cleanest games.
export const timeMistakeSorter = (a: GameRow, b: GameRow): number => {
  const ba = timeMistakeBand(a);
  const bb = timeMistakeBand(b);
  if (ba !== bb) return ba - bb;
  switch (ba) {
    case 0:
      return clockRemainingMs(a) - clockRemainingMs(b);
    case 1:
    case 2:
      return (b.gameDate?.getTime() ?? 0) - (a.gameDate?.getTime() ?? 0);
    default:
      return (b.mistakeIndex ?? 0) - (a.mistakeIndex ?? 0);
  }
};

// Formats the span of the season's games for the modal title. Intl's
// formatRange collapses the shared parts locale-appropriately (e.g.
// "16-30 Jul 2026", "30 Jul - 14 Aug 2026", "30 Dec 2025 - 14 Jan 2026"); a
// single day shows once. The year lives here so the per-row Date cells can drop
// it without a screenshot losing which season it was.
const seasonRangeFormat = new Intl.DateTimeFormat(undefined, {
  day: "numeric",
  month: "short",
  year: "numeric",
});
export const formatSeasonRange = (start: Date, end: Date): string =>
  start.toDateString() === end.toDateString()
    ? seasonRangeFormat.format(start)
    : seasonRangeFormat.formatRange(start, end);

export const PlayerGameHistoryModal: React.FC<PlayerGameHistoryModalProps> = ({
  visible,
  onClose,
  userId,
  username,
  seasonId,
  seasonNumber,
  seasonStartDate,
  onChat,
}) => {
  const { data, isLoading, error } = useQuery(getPlayerSeasonGames, {
    userId,
    seasonId,
  });

  // These gate the combined "Time / Mistakes" column and pick its header. A
  // season with live games has ticking clocks; a season with any analyzed game
  // has mistake scores. The column is hidden only when the list has neither.
  const hasLiveClocks = (data?.games ?? []).some(
    (g) => (g.result === "turn" || g.result === "in_progress") && g.lastUpdate,
  );
  const hasMistakeData = (data?.games ?? []).some(
    (g) => g.mistakeIndex !== undefined,
  );
  // Time and Mistakes never apply to the same row -- a game is either live (show
  // its clock) or finished (show its mistake score once analyzed) -- so they
  // share one column. The just-finished-but-unanalyzed gap shows "-". The header
  // names whichever kind(s) the current list actually has.
  // Header label plus one explanatory tooltip on the header itself (reachable
  // even when every row is "-"), rather than repeating the note on each cell.
  const timeMistakeLabel = hasLiveClocks
    ? hasMistakeData
      ? "Time / Mistakes"
      : "Time"
    : "Mistakes";
  const timeMistakeHint = hasLiveClocks
    ? hasMistakeData
      ? "Time until the on-turn player's clock forfeits for a live game; its BestBot Mistake Score once finished and analyzed (lower is better; 0 is a perfect game; - until analyzed)."
      : "Time until the on-turn player's clock forfeits."
    : "BestBot Mistake Score (lower is better; 0 is a perfect game; - until the game has been analyzed).";
  const timeMistakeColumn: TableColumnsType<GameRow>[number] = {
    title: (
      <Tooltip title={timeMistakeHint}>
        <span style={{ cursor: "help" }}>{timeMistakeLabel}</span>
      </Tooltip>
    ),
    key: "timeMistake",
    align: "right" as const,
    render: (_, record) => {
      if (
        (record.result === "turn" || record.result === "in_progress") &&
        record.lastUpdateMs !== undefined
      ) {
        // Bare view: just the ticking d:hh:mm:ss (the Result column already
        // shows whose turn it is). The tooltip still names the on-turn player --
        // this player when it is their turn, otherwise the opponent.
        const onTurnName =
          record.result === "turn" ? username : record.opponentUsername;
        return (
          <CorrespondenceTurnIndicator
            perspective={{ kind: "bare", playerName: onTurnName }}
            lastUpdateMs={record.lastUpdateMs}
            incrementMs={record.incrementSecs * 1000}
            bankMs={record.onTurnTimeBankMs}
          />
        );
      }
      // Finished games: the mistake score once analyzed, else "-". Absent (not
      // 0) means unanalyzed, so 0 renders as a genuine perfect game. The column
      // header carries the explanation, so the cell is just the number.
      if (record.mistakeIndex === undefined) {
        return "-";
      }
      return record.mistakeIndex.toFixed(1);
    },
    // Click-sort by urgency then learnability (see timeMistakeSorter).
    sorter: timeMistakeSorter,
  };

  const columns: TableColumnsType<GameRow> = [
    {
      title: "Opponent",
      key: "opponent",
      fixed: "left" as const,
      render: (
        _: unknown,
        record: { opponentUsername: string; opponentUserId: string },
      ) => (
        // Same menu helper the modal header uses: it renders the M/IM title
        // badge and the player context menu. Stop propagation so opening the
        // menu does not also fire the row's navigate-to-game click.
        <strong
          onClick={(e) => e.stopPropagation()}
          style={{ cursor: "default" }}
        >
          <UsernameWithContext
            username={record.opponentUsername}
            userID={record.opponentUserId}
            sendMessage={onChat}
            omitSendMessage={!onChat}
          />
        </strong>
      ),
      sorter: opponentSorter,
    },
    {
      title: "Result",
      key: "result",
      render: (
        _: unknown,
        record: { result: string; gameEndReason: GameEndReason },
      ) => {
        const reason = endReasonLabel(record.gameEndReason);
        let tag: React.ReactNode = null;
        if (record.result === "win") {
          tag = <Tag color="green">Win</Tag>;
        } else if (record.result === "loss") {
          tag = <Tag color="red">Loss</Tag>;
        } else if (record.result === "draw") {
          tag = <Tag color="blue">Draw</Tag>;
        } else if (record.result === "turn") {
          return <Tag color="gold">On Turn</Tag>;
        } else if (record.result === "in_progress") {
          return <Tag color="orange">In Progress</Tag>;
        }
        if (!tag) return null;
        return reason ? (
          <span style={{ whiteSpace: "nowrap" }}>
            {tag} ({reason})
          </span>
        ) : (
          tag
        );
      },
      sorter: resultSorter,
    },
    {
      title: "Score",
      key: "score",
      render: (_, record) => {
        const spread = record.playerScore - record.opponentScore;
        // Spread inline (was hover-only): green ahead, red behind, muted at
        // level. The running margin while a game is live, the final one once
        // it has finished.
        return (
          <span style={{ whiteSpace: "nowrap" }}>
            {record.playerScore}-{record.opponentScore}{" "}
            <Text
              type={
                spread > 0 ? "success" : spread < 0 ? "danger" : "secondary"
              }
            >
              ({spread >= 0 ? "+" : ""}
              {spread})
            </Text>
          </span>
        );
      },
      sorter: scoreSorter,
    },
    ...(hasLiveClocks || hasMistakeData ? [timeMistakeColumn] : []),
    {
      title: "Date",
      dataIndex: "gameDate",
      key: "date",
      render: (date?: Date) => {
        if (!date) return "—";
        // Locale short date without the year (the season's year lives in the
        // modal title) plus the time -- the part that actually varies row to
        // row, previously buried in the hover. Full timestamp stays on hover.
        return (
          <Tooltip title={date.toLocaleString()}>
            {date.toLocaleString(undefined, {
              month: "short",
              day: "numeric",
              hour: "2-digit",
              minute: "2-digit",
            })}
          </Tooltip>
        );
      },
      sorter: dateSorter,
    },
  ];

  const dataSource: GameRow[] =
    data?.games.map((game) => ({
      key: game.gameId,
      gameId: game.gameId,
      opponentUsername: game.opponentUsername,
      opponentUserId: game.opponentUserId,
      result: game.result,
      playerScore: game.playerScore,
      opponentScore: game.opponentScore,
      mistakeIndex: game.mistakeIndex,
      gameDate: game.gameDate ? timestampDate(game.gameDate) : undefined,
      gameEndReason: game.gameEndReason,
      lastUpdateMs: game.lastUpdate
        ? timestampDate(game.lastUpdate).getTime()
        : undefined,
      incrementSecs: game.incrementSecs,
      onTurnTimeBankMs: Number(game.onTurnTimeBankMs),
    })) || [];

  // Title range, shown once so the per-row Date cells can drop the year: from
  // the season start (all league games are created then) to this player's
  // latest activity. Min/max over the games' last-updated times plus the start,
  // so it spans the whole season even if the earliest game was last touched
  // well after it began; falls back to the games' own span if no start is given.
  const rangeMs = dataSource
    .map((row) => row.gameDate?.getTime())
    .filter((t): t is number => t !== undefined);
  if (seasonStartDate) {
    rangeMs.push(timestampDate(seasonStartDate).getTime());
  }
  const seasonRange =
    rangeMs.length === 0
      ? null
      : formatSeasonRange(
          new Date(Math.min(...rangeMs)),
          new Date(Math.max(...rangeMs)),
        );

  const handleRowClick = (record: { gameId: string }) => {
    window.open(`/game/${record.gameId}`, "_blank");
  };

  return (
    <Modal
      className="league-game-modal"
      title={
        <React.Fragment>
          <UsernameWithContext
            username={username}
            userID={userId}
            sendMessage={onChat}
            omitSendMessage={!onChat}
          />
          's Season {seasonNumber} Games
          {seasonRange ? ` (${seasonRange})` : null}
        </React.Fragment>
      }
      open={visible}
      onCancel={onClose}
      footer={null}
      width={700}
      zIndex={2000}
    >
      {isLoading && (
        <div style={{ textAlign: "center", padding: "40px" }}>
          <Spin size="large" />
        </div>
      )}
      {error && (
        <div style={{ color: "red", padding: "20px", textAlign: "center" }}>
          Failed to load game history: {error.message}
        </div>
      )}
      {!isLoading && !error && (
        <>
          {dataSource.length === 0 ? (
            <div
              style={{ textAlign: "center", padding: "40px" }}
              className="league-color-999"
            >
              No games found for this season.
            </div>
          ) : (
            <Table
              columns={columns}
              dataSource={dataSource}
              pagination={false}
              size="small"
              showSorterTooltip={false}
              scroll={{ x: "max-content" }}
              onRow={(record) => ({
                onClick: () => handleRowClick(record),
                style: { cursor: "pointer" },
              })}
            />
          )}
        </>
      )}
    </Modal>
  );
};
