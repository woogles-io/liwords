const ordinalSuffix = (n: number) => {
  if (n < 0) n = -n;
  if (Math.floor(n / 10) % 10 === 1) return "th";
  n %= 10;
  return n === 1 ? "st" : n === 2 ? "nd" : n === 3 ? "rd" : "th";
};

export const ordinal = (n: number) => `${n}${ordinalSuffix(n)}`;
