// A spread reads as a margin rather than a count only when it carries its sign,
// and a signed column cannot mistake a leading minus for the start of a digit.
export const signed = (n: number) => `${n >= 0 ? "+" : ""}${n}`;
