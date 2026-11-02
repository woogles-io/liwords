// One score of a pair, a negative one bracketed, as scorePair writes it.
export const scoreSide = (n: number) => (n < 0 ? `(${n})` : `${n}`);

// Two scores as "428-386". A negative score, possible after time penalties or
// a game ended early on scoreless turns, is bracketed, so that "-10--10"
// reads "(-10)-(-10)".
export const scorePair = (a: number, b: number): string =>
  `${scoreSide(a)}-${scoreSide(b)}`;
