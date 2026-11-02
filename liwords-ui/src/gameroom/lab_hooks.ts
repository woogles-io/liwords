import { useSyncExternalStore } from "react";

// An empty square's hooks, as the wasm gives them: the letters that make the
// run of tiles across it a word, and those that make the run down it one;
// null where no tile is beside it that way.
export type HookSquare = {
  across: Array<string> | null;
  down: Array<string> | null;
};

// The hooks the analyzer shows, for the board to draw over the squares. The
// analyzer publishes them while its lab is open, so they never outlive it:
// where no analyzer is open, as in a player's own game in play, none show.
export type LabHooks = {
  dim: number;
  // by square, row by row; undefined where no tile is beside the square
  squares: Array<HookSquare | undefined>;
};

// A square's line of hook letters: all of them if they fit, else as many
// whole tiles as fit beside the count of the rest. A tile of several letters
// comes bracketed and takes its width. fits says whether a line fits the
// square; where nothing measures it, every line does.
export const hookLine = (
  letters: Array<string>,
  fits: (line: string) => boolean = () => true,
) => {
  const all = letters.join("");
  if (fits(all)) return all;
  let shown = "";
  let n = 0;
  while (
    n < letters.length &&
    fits(`${shown}${letters[n]}+${letters.length - n - 1}`)
  ) {
    shown += letters[n];
    n += 1;
  }
  return `${shown}+${letters.length - n}`;
};

const publishedHooks = new Map<symbol, LabHooks>();
let currentHooks: LabHooks | null = null;
const hooksSubscribers = new Set<() => void>();

export const publishLabHooks = (publisher: symbol, hooks: LabHooks | null) => {
  if (hooks) publishedHooks.set(publisher, hooks);
  else publishedHooks.delete(publisher);
  currentHooks = publishedHooks.values().next().value ?? null;
  hooksSubscribers.forEach((notify) => notify());
};

const subscribeHooks = (notify: () => void) => {
  hooksSubscribers.add(notify);
  return () => {
    hooksSubscribers.delete(notify);
  };
};

export const useLabHooks = () =>
  useSyncExternalStore(subscribeHooks, () => currentHooks);
