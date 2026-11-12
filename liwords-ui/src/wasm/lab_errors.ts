// What the lab's worker says when a solve has no room left: its memory cap
// counts every byte the instance holds, the lexicon too, so a part given too
// small a share of the device's memory can search nothing.
export const OUT_OF_MEMORY =
  "Out of memory: the solve has no room to search. Use fewer workers.";
