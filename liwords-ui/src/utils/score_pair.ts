// Two scores as "428-386". A negative score, possible after time penalties or
// a game ended early on scoreless turns, is bracketed, so that "-10--10"
// reads "(-10)-(-10)".
export const scorePair = (a: number, b: number): string => {
  const side = (n: number) => (n < 0 ? `(${n})` : `${n}`);
  return `${side(a)}-${side(b)}`;
};
