import { create } from "@bufbuild/protobuf";
import {
  GameEndedEvent,
  GameEndedEventSchema,
  GameEndReason,
  GameInfoResponse,
} from "../gen/api/proto/ipc/omgwords_pb";
import { getEnumLabel } from "../utils/protobuf";

export const endGameMessage = (gee: GameEndedEvent): string => {
  const scores = gee.scores;
  const ratings = gee.newRatings;
  // const ratings = gee.getNewRatingsMap();
  const reason = gee.endReason;
  let winner = gee.winner;
  let loser = gee.loser;
  const tie = gee.tie;
  let summary = [""];
  // const message = `Game is over. Scores: ${JSON.stringify(
  //   scores
  // )}, new ratings: ${JSON.stringify(ratings)}`;
  let summaryReason = "";
  let summaryAddendum = "";
  if (tie) {
    // Doesn't matter who we call the "winner" and "loser" here.
    const wlArray = Object.keys(scores);
    winner = wlArray[0];
    loser = wlArray[1];
  }
  const winscore = scores[winner];
  const losescore = scores[loser];
  const winrating = ratings[winner];
  const loserating = ratings[loser];
  let properEnding = false;
  switch (reason) {
    case GameEndReason.STANDARD:
      properEnding = true;
      summaryAddendum = `Final score: ${winscore} - ${losescore}`;
      break;
    case GameEndReason.TIME:
      // timed out.
      properEnding = true;
      summaryReason = ` (${loser} timed out!)`;
      break;
    case GameEndReason.CONSECUTIVE_ZEROES:
      properEnding = true;
      summaryReason = " (six consecutive scores of zero)";
      summaryAddendum = `Final score: ${winscore} - ${losescore}`;
      break;

    case GameEndReason.RESIGNED:
      properEnding = true;
      summaryReason = ` (${loser} resigned)`;
      break;
    case GameEndReason.TRIPLE_CHALLENGE:
      properEnding = true;
      summaryReason = " (triple challenge!)";
      break;
    case GameEndReason.ABORTED:
      summaryReason = "Game was cancelled.";
      break;
    case GameEndReason.CANCELLED:
      summaryReason = "Game was cancelled.";
      break;
    case GameEndReason.FORCE_FORFEIT:
      properEnding = true;
      summaryReason = " by forfeit";
      break;
    case GameEndReason.ADJUDICATED:
      properEnding = true;
      summaryReason = " (adjudicated)";
      summaryAddendum = `Final score: ${winscore} - ${losescore}`;
      break;
  }
  if (!properEnding) {
    summary.push(summaryReason);
  } else {
    if (!tie) {
      summary = [`${winner} wins${summaryReason}. ${summaryAddendum}`];
    } else {
      summary = [`Tie game! ${summaryAddendum}`];
    }
    if (winrating || loserating) {
      summary.push(
        `New ratings: ${winner}: ${winrating}, ${loser}: ${loserating}`,
      );
    }
  }
  return summary.join("\n");
};

export const endGameMessageFromGameInfo = (info: GameInfoResponse): string => {
  // construct an artificial GameEndedEvent

  const gee = create(GameEndedEventSchema, {});
  const scores = gee.scores;
  if (
    info.gameEndReason === GameEndReason.ABORTED ||
    info.gameEndReason === GameEndReason.CANCELLED
  ) {
    const endReasonText = getEnumLabel(
      GameEndReason,
      info.gameEndReason,
    )?.toLowerCase();
    return `Game was ${endReasonText}.`;
  }
  if (info.scores) {
    scores[info.players[0].nickname] = info.scores[0];
    scores[info.players[1].nickname] = info.scores[1];
  }
  if (info.winner === -1) {
    gee.tie = true;
  } else {
    gee.winner = info.players[info.winner ?? 0].nickname;
    gee.loser = info.players[1 - (info.winner ?? 0)].nickname;
  }

  gee.endReason = info.gameEndReason;

  return endGameMessage(gee);
};
