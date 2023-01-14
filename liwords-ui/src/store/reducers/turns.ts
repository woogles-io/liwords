import {
  GameEvent,
  GameEvent_Type,
} from '../../gen/api/proto/macondo/macondo_pb';

export type Turn = Array<GameEvent>;

export const gameEventsToTurns = (evts: Array<GameEvent>) => {
  // Compute the turns based on the game events.
  const turns = new Array<Turn>();
  let lastTurn: Turn = new Array<GameEvent>();
  evts.forEach((evt) => {
    let playersDiffer = false;
    if (lastTurn.length !== 0) {
      playersDiffer = lastTurn[0].playerIndex !== evt.playerIndex;
    }

    if (
      (lastTurn.length !== 0 && playersDiffer) ||
      evt.type === GameEvent_Type.TIME_PENALTY ||
      evt.type === GameEvent_Type.END_RACK_PENALTY
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
