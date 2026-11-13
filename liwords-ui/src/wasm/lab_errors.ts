// What the lab's worker says when a solve has no room left: its memory cap
// counts every byte the instance holds, the lexicon too, so a part given too
// small a share of the device's memory can search nothing.
export const OUT_OF_MEMORY =
  "Out of memory: the solve has no room to search. Use fewer workers.";

// The same, on one worker already: the device's allowance holds the lexicon
// and the solve's own graph with no room left to search.
export const OUT_OF_MEMORY_ALONE =
  "Out of memory: the solve has no room to search, even on one worker.";

// A wasm error names a tile by its number ("too many tile 17"): the tile's
// letter in its place, as the page shows tiles.
export const namedTiles = (text: string, rune: (tile: number) => string) =>
  text.replace(/\btile (\d+)\b/g, (_, n: string) => rune(Number(n)));
