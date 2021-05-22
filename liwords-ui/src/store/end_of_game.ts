import { GameMetadata } from '../gameroom/game_info';
import {
  GameEndedEvent,
  GameEndReason,
} from '../gen/api/proto/realtime/realtime_pb';

export const endGameMessage = (gee: GameEndedEvent): string => {
  const scores = gee.getScoresMap();
  const ratings = gee.getNewRatingsMap();
  // const ratings = gee.getNewRatingsMap();
  const reason = gee.getEndReason();
  let winner = gee.getWinner();
  let loser = gee.getLoser();
  const tie = gee.getTie();
  let summary = [''];
  // const message = `Game is over. Scores: ${JSON.stringify(
  //   scores
  // )}, new ratings: ${JSON.stringify(ratings)}`;
  let summaryReason = '';
  let summaryAddendum = '';
  if (tie) {
    // Doesn't matter who we call the "winner" and "loser" here.
    const wlArray = scores.toArray();
    winner = wlArray[0][0];
    loser = wlArray[1][0];
  }
  const winscore = scores.get(winner);
  const losescore = scores.get(loser);
  const winrating = ratings.get(winner);
  const loserating = ratings.get(loser);
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
      summaryReason = ' (six consecutive scores of zero)';
      summaryAddendum = `Final score: ${winscore} - ${losescore}`;
      break;

    case GameEndReason.RESIGNED:
      properEnding = true;
      summaryReason = ` (${loser} resigned)`;
      break;
    case GameEndReason.TRIPLE_CHALLENGE:
      properEnding = true;
      summaryReason = ' (triple challenge!)';
      break;
    case GameEndReason.ABORTED:
      summaryReason = 'Game was aborted.';
      break;
    case GameEndReason.CANCELLED:
      summaryReason = 'Game was cancelled.';
      break;
    case GameEndReason.FORCE_FORFEIT:
      properEnding = true;
      summaryReason = ' by forfeit.';
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
        `New ratings: ${winner}: ${winrating}, ${loser}: ${loserating}`
      );
    }
  }
  return summary.join('\n');
};

export const endGameMessageFromGameInfo = (info: GameMetadata): string => {
  // construct an artificial GameEndedEvent

  const gee = new GameEndedEvent();
  const scores = gee.getScoresMap();
  if (
    info.game_end_reason === 'ABORTED' ||
    info.game_end_reason === 'CANCELLED'
  ) {
    return `Game was ${info.game_end_reason.toLowerCase()}.`;
  }
  if (info.scores) {
    scores.set(info.players[0].nickname, info.scores[0]);
    scores.set(info.players[1].nickname, info.scores[1]);
  }
  if (info.winner === -1) {
    gee.setTie(true);
  } else {
    gee.setWinner(info.players[info.winner ?? 0].nickname);
    gee.setLoser(info.players[1 - (info.winner ?? 0)].nickname);
  }

  const ger = info.game_end_reason as
    | 'NONE'
    | 'STANDARD'
    | 'TIME'
    | 'CONSECUTIVE_ZEROES'
    | 'RESIGNED'
    | 'TRIPLE_CHALLENGE';

  gee.setEndReason(GameEndReason[ger]);

  return endGameMessage(gee);
};
