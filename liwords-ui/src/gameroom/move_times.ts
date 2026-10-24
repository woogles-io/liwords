import {
  GameEvent,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";

// The clock settings a game was played with, in milliseconds.
export type ClockRules = {
  timeStarted: number;
  initialMs: number;
  incrementMs: number;
  // correspondence: each turn's clock resets to the increment (a time bank
  // only pays for overruns, so it does not change the clock).
  resetToIncrement: boolean;
  maxOvertimeMs: number;
};

// When each event happened, replayed from the start of the game the way the
// server keeps the clocks: an event records how much time its player had left
// at that moment, so the time since their turn began is what they had minus
// that. Moves end the turn (the clock then gets its increment, or resets);
// a challenge is timed on the challenger's clock but leaves their turn
// running. Events with no time of their own (end-of-game rack points and
// penalties) are left undefined. Approximate: a server restart or an undo
// can throw it off.
export const moveTimes = (
  events: Array<GameEvent>,
  rules: ClockRules,
): Array<number | undefined> => {
  const clock = [rules.initialMs, rules.initialMs];
  let turnStart = rules.timeStarted;
  const at = (p: number, remaining: number) => turnStart + clock[p] - remaining;

  const endTurn = (p: number, remaining: number, t: number) => {
    if (rules.resetToIncrement) {
      clock[p] = rules.incrementMs;
    } else {
      clock[p] = Math.max(remaining + rules.incrementMs, -rules.maxOvertimeMs);
    }
    turnStart = t;
  };

  return events.map((e) => {
    const p = e.playerIndex;
    switch (e.type) {
      case GameEvent_Type.TILE_PLACEMENT_MOVE:
      case GameEvent_Type.PASS:
      case GameEvent_Type.EXCHANGE:
      case GameEvent_Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS: {
        const t = at(p, e.millisRemaining);
        endTurn(p, e.millisRemaining, t);
        return t;
      }
      case GameEvent_Type.CHALLENGE:
        // made by the challenger.
        return at(p, e.millisRemaining);
      case GameEvent_Type.PHONY_TILES_RETURNED:
      case GameEvent_Type.CHALLENGE_BONUS:
        // these name the challenged player but carry the challenger's clock.
        return at(1 - p, e.millisRemaining);
      default:
        return undefined;
    }
  });
};
