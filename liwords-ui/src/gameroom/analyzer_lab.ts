import {
  GameEvent,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { TileDistribution } from "../store/reducers/game_reducer";
import { MachineLetter, MachineWord } from "../utils/cwgame/common";

// Turns the analyzer lab on.
export const ANALYZER_LAB_KEY = "enableAnalyzerLab";

// The scoreless turns in a row at the end of the turns, counted as the server
// counts them again when a phony comes off (pkg/cwgame): a pass, an exchange,
// a turn lost to a challenge, and a play whose tiles came back.
export const scorelessTurnsAtEnd = (turns: Array<GameEvent>) => {
  let count = 0;
  let sawReturnedPhony = false;
  for (let i = turns.length - 1; i >= 0; i--) {
    switch (turns[i].type) {
      case GameEvent_Type.TILE_PLACEMENT_MOVE:
        if (!sawReturnedPhony) return count;
        sawReturnedPhony = false;
        count++;
        break;
      case GameEvent_Type.PHONY_TILES_RETURNED:
        sawReturnedPhony = true;
        break;
      case GameEvent_Type.EXCHANGE:
      case GameEvent_Type.PASS:
      case GameEvent_Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
        count++;
        sawReturnedPhony = false;
        break;
      default:
        sawReturnedPhony = false;
    }
  }
  return count;
};

// The passes in a row at the end, which end a game under some rules (German,
// Polish, and Spanish); a challenge and its bonus are not turns.
export const passesAtEnd = (turns: Array<GameEvent>) => {
  let count = 0;
  for (let i = turns.length - 1; i >= 0; i--) {
    switch (turns[i].type) {
      case GameEvent_Type.PASS:
        count++;
        break;
      case GameEvent_Type.CHALLENGE:
      case GameEvent_Type.CHALLENGE_BONUS:
        break;
      default:
        return count;
    }
  }
  return count;
};

// The tiles the player to move cannot see. The pool holds every tile off the
// board, that player's own rack too.
export const unseenBy = (pool: TileDistribution, rack: MachineWord) => {
  const left: TileDistribution = { ...pool };
  for (const t of rack) {
    left[t] = (left[t] ?? 0) - 1;
  }
  const unseen: MachineWord = [];
  for (const [t, n] of Object.entries(left)) {
    for (let i = 0; i < n; i++) unseen.push(Number(t) as MachineLetter);
  }
  return unseen.sort((a, b) => a - b);
};

export type SimStat = { mean: number; variance: number };

export type SimCandidate = {
  play: unknown;
  state: "running" | "paused" | "pruned";
  n: number;
  win: SimStat;
  equity: SimStat;
};

// The order a sim's candidates are shown in: by the chance to win, then by
// equity.
export const simOrder = (a: SimCandidate, b: SimCandidate) =>
  b.win.mean - a.win.mean || b.equity.mean - a.equity.mean;

// The page's clock, in milliseconds, which times how long a sim or an endgame
// has run.
export const labNow = () => performance.now();
