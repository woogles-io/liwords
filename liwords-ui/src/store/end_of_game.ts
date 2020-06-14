import {
  GameEndedEvent,
  GameEndReason,
} from '../gen/api/proto/game_service_pb';

export const endGameMessage = (gee: GameEndedEvent) => {
  const scores = gee.getScoresMap();
  console.log('scores are', scores);
  const ratings = gee.getNewRatingsMap();
  const reason = gee.getEndReason();
  const winner = gee.getWinner();
  const loser = gee.getLoser();
  const tie = gee.getTie();
  let summary = '';
  // const message = `Game is over. Scores: ${JSON.stringify(
  //   scores
  // )}, new ratings: ${JSON.stringify(ratings)}`;
  let summaryReason = '';
  let summaryAddendum = '';
  const winscore = scores.get(winner);
  const losescore = scores.get(loser);

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
  return summary;
};
