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
import { timestampDate } from "@bufbuild/protobuf/wkt";
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
  onChat?: (uuid: string, username: string) => void;
};

// One row of the season game list. Finished games carry scores, a result and an
// optional mistake index (absent until the game has been analyzed); in-progress
// games carry the live-clock anchor (lastUpdateMs / incrementSecs /
// onTurnTimeBankMs) so the clock can tick. gameDate is the last-updated time for
// both, and is what the list sorts by, by default.
type GameRow = {
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

export const PlayerGameHistoryModal: React.FC<PlayerGameHistoryModalProps> = ({
  visible,
  onClose,
  userId,
  username,
  seasonId,
  seasonNumber,
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
    },
    ...(hasLiveClocks || hasMistakeData ? [timeMistakeColumn] : []),
    {
      title: "Date",
      dataIndex: "gameDate",
      key: "date",
      render: (date?: Date) => {
        if (!date) return "—";
        return (
          <Tooltip title={date.toLocaleString()}>
            {date.toLocaleDateString()}
          </Tooltip>
        );
      },
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
