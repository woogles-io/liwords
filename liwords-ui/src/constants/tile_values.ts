// Turn the rune into a point value.
// Note: This should be part of its own Alphabet or Bag or similar
// class. This is for a quick MVP. We use these keys for blank designation.
const CrosswordGameTileValues = {
  A: 1,
  B: 3,
  C: 3,
  D: 2,
  E: 1,
  F: 4,
  G: 2,
  H: 4,
  I: 1,
  J: 8,
  K: 5,
  L: 1,
  M: 3,
  N: 1,
  O: 1,
  P: 3,
  Q: 10,
  R: 1,
  S: 1,
  T: 1,
  U: 1,
  V: 4,
  W: 4,
  X: 8,
  Y: 4,
  Z: 10,
};

function runeToValues(rune: string | null, values: any): number {
  if (rune === null) {
    return 0;
  }
  if (values[rune]) {
    return values[rune];
  }
  return 0;
}

export { CrosswordGameTileValues, runeToValues };
