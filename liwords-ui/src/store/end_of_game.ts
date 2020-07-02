import {
  GameEndedEvent,
  GameEndReason,
} from '../gen/api/proto/realtime/realtime_pb';

export const endGameMessage = (gee: GameEndedEvent) => {
  const scores = gee.getScoresMap();
  const ratings = gee.getNewRatingsMap();
  console.log('scores are', scores);
  // const ratings = gee.getNewRatingsMap();
  const reason = gee.getEndReason();
  let winner = gee.getWinner();
  let loser = gee.getLoser();
  const tie = gee.getTie();
  let summary = '';
  // const message = `Game is over. Scores: ${JSON.stringify(
  //   scores
  // )}, new ratings: ${JSON.stringify(ratings)}`;
  let summaryReason = '';
  let summaryAddendum = '';
  if (tie) {
    [winner, loser] = scores.keys();
  }
  const winscore = scores.get(winner);
  const losescore = scores.get(loser);
  const winrating = ratings.get(winner);
  const loserating = ratings.get(loser);

  switch (reason) {
    case GameEndReason.STANDARD:
      summaryAddendum = `Final score: ${winscore} - ${losescore}`;
      break;
    case GameEndReason.TIME:
      // timed out.
      summaryReason = ` (${loser} timed out!)`;
      break;
    case GameEndReason.CONSECUTIVE_ZEROES:
      summaryReason = ' (six consecutive scores of zero)';
      summaryAddendum = `Final score: ${winscore} - ${losescore}`;
      break;

    case GameEndReason.RESIGNED:
      summaryReason = ` (${loser} resigned...)`;
      break;
  }
  summary = 'Game is over - ';

  if (!tie) {
    summary = `${summary}${winner} wins${summaryReason}. ${summaryAddendum}`;
  } else {
    summary = `${summary}tie game! ${summaryAddendum}`;
  }
  if (winrating || loserating) {
    summary += ` New ratings: ${winner}: ${winrating}, ${loser}: ${loserating}`;
  }
  return summary;
};
