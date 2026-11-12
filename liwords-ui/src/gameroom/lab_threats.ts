import { useSyncExternalStore } from "react";

// What the opponent may play next, by score alone, as the wasm's threats
// answers: each empty square's best score of a play placing a tile on it, the
// chance the opponent holds a play doing that, and the expected score of
// their best play through it (a rack with none there scoring 0); null where
// a tile is.
export type ThreatSquare = { best: number; chance: number; expected: number };

// A play the opponent may make, as the wasm writes one (its score in
// points), the tiles it places (0 for a blank), and the chance their rack
// holds them.
export type ThreatPlay = {
  action: "play";
  down: boolean;
  lane: number;
  idx: number;
  word: Array<number>;
  score: number;
  tiles: Array<number>;
  chance: number;
};

// The answer: how many racks and plays the opponent may have, their best
// play's expected score, the chance of a play of a whole rack (null when they
// hold fewer tiles than a rack), whether every rack was counted (exact), else
// the share of rack chance counted exactly (covered) and how many racks were
// drawn for the rest (samples), how long listing every play took, the best
// plays first, and the board's squares, row by row.
export type ThreatsAnswer = {
  racks: number;
  count: number;
  expected_best: number;
  bingo: number | null;
  covered: number;
  samples: number;
  list_ms: number;
  exact: boolean;
  plays: Array<ThreatPlay>;
  squares: Array<Array<ThreatSquare | null>>;
};

// What the board shows over its squares: by square, row by row, how hot it
// is (0 to 1) and the number written on it; the letters of the play picked,
// which the opponent would place, by square; the squares of the play pointed
// at; and the square the list is narrowed to, if any. While picking, a click
// on a square narrows the list to it.
export type LabThreats = {
  dim: number;
  heat: Array<{ level: number; label: string } | undefined>;
  ghost: Map<number, string>;
  outline: Set<number>;
  square: number | null;
  picking: boolean;
  pick: (square: number | null) => void;
};

// How the squares are tinted: by the best score through each, scaled to the
// board's best, or by the chance the opponent can score that there.
export type ThreatTint = "score" | "chance";

// Each square's tint and its number, by the way asked.
export const threatHeat = (
  squares: ThreatsAnswer["squares"],
  tint: ThreatTint,
): LabThreats["heat"] => {
  const flat = squares.flat();
  const top = Math.max(0, ...flat.map((s) => s?.best ?? 0));
  return flat.map((s) => {
    if (!s || s.best <= 0) return undefined;
    return tint === "score"
      ? { level: top ? s.best / top : 0, label: String(s.best) }
      : { level: s.chance, label: `${Math.round(s.chance * 100)}` };
  });
};

// The squares a play places a tile on, by index row by row.
export const playSquares = (
  play: Pick<ThreatPlay, "down" | "lane" | "idx" | "word">,
  dim: number,
) =>
  play.word.flatMap((tile, i) => {
    if (tile === 0) return [];
    const row = play.down ? play.idx + i : play.lane;
    const col = play.down ? play.lane : play.idx + i;
    return [row * dim + col];
  });

// The tiles the player to move holds after placing the tiles given, the rest
// of what the opponent may hold staying unseen: the tiles the player would
// draw after the play are not theirs yet. Null when a tile placed is not on
// the rack.
export const leaveAfter = (rack: Array<number>, placed: Array<number>) => {
  const left = [...rack];
  for (const t of placed) {
    const at = left.indexOf(t);
    if (at < 0) return null;
    left.splice(at, 1);
  }
  return left;
};

// A share as a percent, to a tenth.
export const percentText = (x: number) => `${(x * 100).toFixed(1)}%`;

// The status line for an answer: the opponent's best on average, the chance
// of a whole rack, from how many tiles unseen; "~" and how much was counted
// exactly when time ran out.
export const threatsSummary = (answer: ThreatsAnswer, unseen: number) => {
  const about = answer.exact ? "" : "~";
  const bingo =
    answer.bingo === null ? "" : `, bingo ${about}${percentText(answer.bingo)}`;
  const cut = answer.exact
    ? ""
    : `; time ran out: ${percentText(answer.covered)} of racks counted, ${answer.samples} more drawn`;
  return `Opponent's best ${about}${answer.expected_best.toFixed(1)} on average${bingo}, from ${unseen} unseen${cut}`;
};

const published = new Map<symbol, LabThreats>();
let current: LabThreats | null = null;
const subscribers = new Set<() => void>();

// The threats the analyzer shows, for the board to draw over the squares,
// while its panel is open.
export const publishLabThreats = (
  publisher: symbol,
  threats: LabThreats | null,
) => {
  if (threats) published.set(publisher, threats);
  else published.delete(publisher);
  current = published.values().next().value ?? null;
  subscribers.forEach((notify) => notify());
};

const subscribe = (notify: () => void) => {
  subscribers.add(notify);
  return () => {
    subscribers.delete(notify);
  };
};

export const useLabThreats = () =>
  useSyncExternalStore(subscribe, () => current);
