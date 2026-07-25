import React, { useMemo } from "react";
import { Modal, Table, Tag, Typography } from "antd";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { Division, SinglePairing } from "../store/reducers/tournament_reducer";
import { TournamentGameResult } from "../gen/api/proto/ipc/tournament_pb";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { signed } from "./format";
import { competitionRanks, rankLabel } from "./ranks";

const { Text } = Typography;

type Props = {
  division: Division;
  // The full "uuid:username" key the standings and the player index map use.
  playerId: string;
  // Zero-indexed, and never past the round that is open: the last row then
  // matches the standings row this was opened from.
  throughRound: number;
  irlMode: boolean;
  onClose: () => void;
};

type ScorecardRow = {
  key: number;
  round: number;
  wentFirst: boolean;
  opponentId?: string;
  opponentName?: string;
  opponentRemoved: boolean;
  outcome: TournamentGameResult;
  gameEndReason: GameEndReason;
  playerScore?: number;
  opponentScore?: number;
  // Running totals after this round, with a draw counted as half of each, the
  // way the standings table counts them.
  wins?: number;
  losses?: number;
  spread?: number;
  rank?: string;
};

const splitPlayerId = (id: string) => {
  const [uuid, username] = id.split(":");
  return { uuid, username };
};

// Only the reasons worth a parenthetical. In-real-life results are entered by a
// director and carry no end reason at all, so NONE stays silent rather than
// tagging every row of an in-real-life event.
const endReasonLabel = (reason: GameEndReason): string => {
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
      return "canceled";
    default:
      return "";
  }
};

// A game that two players actually finished. A forfeit or a void round carries
// scores of 0-0, which is a placeholder rather than a result, so those are out.
const isFinishedGame = (outcome: TournamentGameResult) =>
  outcome === TournamentGameResult.WIN ||
  outcome === TournamentGameResult.LOSS ||
  outcome === TournamentGameResult.DRAW;

// A round the players are meant to sit down for -- finished, or still running.
const isRealGame = (outcome: TournamentGameResult) =>
  isFinishedGame(outcome) || outcome === TournamentGameResult.NO_RESULT;

// Only finished games count toward the averages.
const isPlayedGame = (row: ScorecardRow) =>
  row.playerScore !== undefined && isFinishedGame(row.outcome);

export const buildScorecard = (
  division: Division,
  playerId: string,
  throughRound: number,
): ScorecardRow[] => {
  const playerIndex = division.playerIndexMap[playerId];
  const rows: ScorecardRow[] = [];

  for (let round = 0; round <= throughRound; round++) {
    const pairing: SinglePairing | undefined =
      division.pairings[round]?.roundPairings[playerIndex];
    const standing = division.standingsMap[round]?.standings;
    const rankIndex = standing?.findIndex((s) => s.playerId === playerId) ?? -1;
    const mine = rankIndex >= 0 ? standing[rankIndex] : undefined;

    const row: ScorecardRow = {
      key: round,
      round,
      wentFirst: false,
      opponentRemoved: false,
      outcome: TournamentGameResult.NO_RESULT,
      gameEndReason: GameEndReason.NONE,
      wins: mine ? mine.wins + mine.draws / 2 : undefined,
      losses: mine ? mine.losses + mine.draws / 2 : undefined,
      spread: mine?.spread,
      rank:
        rankIndex >= 0
          ? rankLabel(
              competitionRanks(standing)[rankIndex],
              rankIndex + 1,
              // The column is right-aligned, so the marker leads.
              "leading",
            )
          : undefined,
    };

    if (pairing?.players?.length) {
      // Each pairing is stored under both players' indexes, so the player is
      // one of the two -- except for a bye, where they are both of them.
      const seat = pairing.players[0].id === playerId ? 0 : 1;
      const opponent = pairing.players[1 - seat];
      const isBye = opponent.id === playerId;

      row.outcome = pairing.outcomes[seat];
      row.gameEndReason = pairing.games[0]?.gameEndReason ?? GameEndReason.NONE;

      // Going first only means something for a round somebody sits down for.
      // It survives an unfinished game, where it is the useful half of the row.
      row.wentFirst = seat === 0 && !isBye && isRealGame(row.outcome);

      if (!isBye) {
        row.opponentId = opponent.id;
        row.opponentName = splitPlayerId(opponent.id).username;
        row.opponentRemoved =
          division.players[division.playerIndexMap[opponent.id]]?.suspended ??
          false;
      }

      // A score only means something once two players have finished a game.
      // A bye, a forfeit and a void round all carry 0-0 or a stand-in spread,
      // none of which anybody played.
      const scores = pairing.games[0]?.scores;
      if (!isBye && scores?.length === 2 && isFinishedGame(row.outcome)) {
        row.playerScore = scores[seat];
        row.opponentScore = scores[1 - seat];
      }
    }

    rows.push(row);
  }

  return rows;
};

const resultTag = (row: ScorecardRow): React.ReactNode => {
  switch (row.outcome) {
    case TournamentGameResult.WIN:
      return <Tag className="ant-tag-win">Win</Tag>;
    case TournamentGameResult.LOSS:
      return <Tag className="ant-tag-loss">Loss</Tag>;
    case TournamentGameResult.DRAW:
      return <Tag className="ant-tag-tie">Draw</Tag>;
    case TournamentGameResult.BYE:
      return <Tag className="ant-tag-bye">Bye</Tag>;
    case TournamentGameResult.FORFEIT_WIN:
      return <Tag className="ant-tag-forfeit">Forfeit win</Tag>;
    case TournamentGameResult.FORFEIT_LOSS:
      return <Tag className="ant-tag-forfeit">Forfeit loss</Tag>;
    case TournamentGameResult.ELIMINATED:
      return <Tag className="ant-tag-forfeit">Eliminated</Tag>;
    case TournamentGameResult.VOID:
      return <Tag className="ant-tag-bye">Not playing</Tag>;
    default:
      // Paired, but the game has not finished. Say nothing rather than claim a
      // state the round has not reached.
      return null;
  }
};

export const PlayerScorecardModal = (props: Props) => {
  const { division, playerId, throughRound, irlMode } = props;
  const { username } = splitPlayerId(playerId);

  const rows = useMemo(
    () => buildScorecard(division, playerId, throughRound),
    [division, playerId, throughRound],
  );

  const averages = useMemo(() => {
    const played = rows.filter(isPlayedGame);
    if (!played.length) {
      return undefined;
    }
    const total = (pick: (row: ScorecardRow) => number) =>
      played.reduce((sum, row) => sum + pick(row), 0) / played.length;
    return {
      games: played.length,
      score: total((row) => row.playerScore ?? 0),
      opponentScore: total((row) => row.opponentScore ?? 0),
    };
  }, [rows]);

  const columns = [
    {
      title: "Round",
      key: "round",
      render: (_: unknown, row: ScorecardRow) => (
        <span style={{ whiteSpace: "nowrap" }}>
          {row.round + 1}{" "}
          {irlMode && row.wentFirst && <Tag className="ant-tag-first">1st</Tag>}
        </span>
      ),
    },
    {
      title: "Opponent",
      key: "opponent",
      render: (_: unknown, row: ScorecardRow) =>
        row.opponentId ? (
          <span style={{ whiteSpace: "nowrap" }}>
            <UsernameWithContext
              username={row.opponentName ?? ""}
              userID={splitPlayerId(row.opponentId).uuid}
              omitSendMessage
              omitBlock
            />{" "}
            {row.opponentRemoved && (
              <Tag className="ant-tag-removed">Removed</Tag>
            )}
          </span>
        ) : null,
    },
    {
      title: "Result",
      key: "result",
      render: (_: unknown, row: ScorecardRow) => {
        const tag = resultTag(row);
        const reason = endReasonLabel(row.gameEndReason);
        if (!tag) {
          return null;
        }
        return (
          <span style={{ whiteSpace: "nowrap" }}>
            {tag}
            {reason ? ` (${reason})` : ""}
          </span>
        );
      },
    },
    {
      title: "Score",
      key: "score",
      render: (_: unknown, row: ScorecardRow) => {
        if (row.playerScore === undefined || row.opponentScore === undefined) {
          return null;
        }
        // Same shape as the league game history: the round's own margin lives
        // inline here, so it needs no column of its own.
        const spread = row.playerScore - row.opponentScore;
        return (
          <span style={{ whiteSpace: "nowrap" }}>
            {row.playerScore}-{row.opponentScore}{" "}
            <Text
              type={
                spread > 0 ? "success" : spread < 0 ? "danger" : "secondary"
              }
            >
              ({signed(spread)})
            </Text>
          </span>
        );
      },
    },
    // The running totals, split and right-aligned the way the standings table
    // this opens from lays them out. A drawn round puts a half in the record,
    // and 1.5 only lines up under 0 if the two are their own columns.
    {
      title: "W",
      key: "wins",
      align: "right" as const,
      render: (_: unknown, row: ScorecardRow) => row.wins ?? null,
    },
    {
      title: "L",
      key: "losses",
      align: "right" as const,
      render: (_: unknown, row: ScorecardRow) => row.losses ?? null,
    },
    {
      // Signed, so a minus sign cannot be mistaken for the start of a digit.
      // The spread inline in the Score cell is signed for the same reason.
      title: "Spread",
      key: "spread",
      align: "right" as const,
      render: (_: unknown, row: ScorecardRow) =>
        row.spread === undefined ? null : signed(row.spread),
    },
    {
      title: "#",
      key: "rank",
      align: "right" as const,
      render: (_: unknown, row: ScorecardRow) => row.rank ?? null,
    },
  ];

  return (
    <Modal
      className="custom-tags"
      title={
        <>
          {username}
          <Text type="secondary" style={{ fontWeight: "normal" }}>
            {" "}
            -- rounds 1 to {throughRound + 1}
          </Text>
        </>
      }
      open
      onCancel={props.onClose}
      footer={null}
      width={720}
    >
      <div style={{ overflowX: "auto" }}>
        <Table
          className="player-scorecard"
          columns={columns}
          dataSource={rows}
          pagination={false}
          size="small"
          rowKey="key"
          locale={{ emptyText: "No rounds have been played yet." }}
        />
      </div>
      {averages && (
        <p>
          <Text type="secondary">
            Average score {averages.score.toFixed(2)}, average opponent score{" "}
            {averages.opponentScore.toFixed(2)}, over {averages.games}{" "}
            {averages.games === 1 ? "game" : "games"}.
          </Text>
        </p>
      )}
    </Modal>
  );
};
