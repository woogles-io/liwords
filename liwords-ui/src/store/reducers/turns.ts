import { GameEvent } from '../../gen/macondo/api/proto/macondo/macondo_pb';

export type Turn = Array<GameEvent>;

export const gameEventsToTurns = (evts: Array<GameEvent>) => {
  // Compute the turns based on the game events.
  const turns = new Array<Turn>();
  let lastTurn: Turn = new Array<GameEvent>();
  evts.forEach((evt) => {
    if (
      lastTurn.length !== 0 &&
      lastTurn[0].getNickname() !== evt.getNickname()
    ) {
      // time to add a new turn.
      turns.push(lastTurn);
      lastTurn = new Array<GameEvent>();
    }
    lastTurn.push(evt);
  });
  if (lastTurn.length > 0) {
    turns.push(lastTurn);
  }
  return turns;
};
