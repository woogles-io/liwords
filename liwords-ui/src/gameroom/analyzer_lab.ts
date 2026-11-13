import {
  GameEvent,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { TileDistribution } from "../store/reducers/game_reducer";
import { MachineLetter, MachineWord } from "../utils/cwgame/common";

// Turns the analyzer lab on.
export const ANALYZER_LAB_KEY = "enableAnalyzerLab";
// Shows the lab's what-if row, and lets the rack under the board be set.
export const ANALYZER_LAB_WHAT_IF_KEY = "analyzerLabWhatIf";

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

// Each seat's exchanges so far, by seat, which an endgame with two or more
// tiles in the bag is told of.
export const exchangesBySeat = (turns: Array<GameEvent>) => {
  const exchanges = [0, 0];
  for (const t of turns) {
    if (t.type === GameEvent_Type.EXCHANGE && t.playerIndex in exchanges) {
      exchanges[t.playerIndex]++;
    }
  }
  return exchanges;
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

// How many tiles the bag holds, the other player having a full rack of the
// rest; or null when the position cannot say: no rack, or a rack short of
// full with tiles still in the bag (a partial rack in an annotated game).
export const endgameBag = (unseenCount: number, rackLength: number) => {
  if (rackLength === 0) return null;
  const bag = Math.max(0, unseenCount - 7);
  if (bag > 0 && rackLength < 7) return null;
  return bag;
};

// The most memory an endgame may use, in MiB, by what the device says it has;
// a browser that says nothing gets the least of them.
export const endgameMemoryMb = () => {
  const gb = (navigator as Navigator & { deviceMemory?: number }).deviceMemory;
  if (gb === undefined) return 256;
  if (gb >= 8) return 1024;
  if (gb >= 4) return 512;
  return 256;
};

export type SimStat = { mean: number; variance: number };

// A statistic over fewer iterations than the candidate's rollouts, its own
// n with it.
export type SimCountedStat = SimStat & { n: number };

// A sim's candidate: where it stands, its rollouts, their chance to win and
// equity (the change in spread, the leaves left counted), that equity's two
// parts (spread, in points alone; leaves, the mover's left less the
// opponent's), and the objective the sim prunes on; with a play to compare
// against, each less that play's on the iterations both ran.
export type SimCandidate = {
  play: unknown;
  state: "running" | "paused" | "pruned";
  n: number;
  win: SimStat;
  equity: SimStat;
  spread?: SimStat;
  leaves?: SimStat;
  objective?: SimStat;
  against?: {
    win: SimCountedStat;
    equity: SimCountedStat;
    objective: SimCountedStat;
  };
};

// The order a sim's candidates are shown in: by the chance to win, then by
// equity.
export const simOrder = (a: SimCandidate, b: SimCandidate) =>
  b.win.mean - a.win.mean || b.equity.mean - a.equity.mean;

// The page's clock, in milliseconds, which times how long a sim or an endgame
// has run.
export const labNow = () => performance.now();
