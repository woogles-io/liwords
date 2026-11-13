import { describe, expect, it } from "vitest";
import {
  PartCandidate,
  behindLeader,
  leaderOf,
  mergeAnswers,
  mergeStats,
  stitchedZ,
} from "./lab_merge";

// a sample's mean and sample variance, as the wasm reports a statistic
const statOf = (xs: Array<number>) => {
  const mean = xs.reduce((a, b) => a + b, 0) / xs.length;
  const variance =
    xs.length > 1
      ? xs.reduce((a, x) => a + (x - mean) ** 2, 0) / (xs.length - 1)
      : 0;
  return { mean, variance };
};

const close = (a: number, b: number) => expect(a).toBeCloseTo(b, 10);

describe("parts merged", () => {
  it("merges parts' statistics as the one sample they make", () => {
    const a = [1, 4, 9, 2];
    const b = [7, -3];
    const c = [5];
    const merged = mergeStats([
      { n: a.length, stat: statOf(a) },
      { n: b.length, stat: statOf(b) },
      { n: c.length, stat: statOf(c) },
      { n: 0, stat: { mean: 0, variance: 0 } },
    ]);
    const all = statOf([...a, ...b, ...c]);
    close(merged.mean, all.mean);
    close(merged.variance, all.variance);
    expect(mergeStats([])).toEqual({ mean: 0, variance: 0 });
  });

  const cand = (play: string, n: number, xs: Array<number>, more = {}) =>
    ({
      play,
      state: "running",
      n,
      win: statOf(xs.map((x) => (x > 0 ? 1 : 0))),
      equity: statOf(xs),
      plies: [{ n, score: statOf(xs), bingo: 0.5 }],
      ...more,
    }) as PartCandidate;

  it("merges answers candidate by candidate, their iterations and plies added", () => {
    const merged = mergeAnswers([
      { iterations: 2, candidates: [cand("A", 2, [1, 3])] },
      {
        iterations: 3,
        candidates: [
          {
            ...cand("A", 3, [-1, 2, 5]),
            plies: [
              { n: 3, score: statOf([-1, 2, 5]), bingo: 0 },
              { n: 1, score: statOf([4]), bingo: 1 },
            ],
          },
        ],
      },
    ])!;
    expect(merged.iterations).toBe(5);
    const [c] = merged.candidates;
    expect(c.n).toBe(5);
    close(c.equity.mean, statOf([1, 3, -1, 2, 5]).mean);
    close(c.equity.variance, statOf([1, 3, -1, 2, 5]).variance);
    expect(c.plies).toHaveLength(2);
    close(c.plies![0].bingo, (0.5 * 2 + 0 * 3) / 5);
    expect(c.plies![1]).toMatchObject({ n: 1, bingo: 1 });
  });

  it("waits while the parts do not hold the same candidates, and runs a candidate only if every part does", () => {
    const one = { iterations: 1, candidates: [cand("A", 1, [1])] };
    expect(
      mergeAnswers([
        one,
        { iterations: 1, candidates: [cand("A", 1, [1]), cand("B", 1, [1])] },
      ]),
    ).toBeNull();
    expect(
      mergeAnswers([one, { iterations: 1, candidates: [cand("B", 1, [1])] }]),
    ).toBeNull();
    const paused = mergeAnswers([
      one,
      { iterations: 1, candidates: [cand("A", 1, [1], { state: "paused" })] },
    ])!;
    expect(paused.candidates[0].state).toBe("paused");
  });

  it("works out the stitched boundary's z as wolges does", () => {
    // wolges's move_picker test's values
    for (const [n, alpha, z] of [
      [16, 0.05, 3.6408929961462624],
      [1000, 0.05, 3.940271585734935],
      [87057, 0.05, 4.106291666114815],
      [1000, 0.01, 4.344497505758632],
    ]) {
      expect(Math.abs(stitchedZ(n, alpha) - z)).toBeLessThan(1e-12);
    }
  });
});

describe("the candidates behind the leader", () => {
  const c = (win: number, equity: number, against?: object) =>
    ({
      play: "",
      state: "running",
      n: 100,
      win: { mean: win, variance: 0 },
      equity: { mean: equity, variance: 0 },
      against,
    }) as PartCandidate;
  const all = () => true;

  it("leads by the chance to win, or by equity among games won, or all lost", () => {
    expect(leaderOf([c(0.4, 9), c(0.6, 1), c(0.5, 5)], all, 0.001)).toBe(1);
    expect(leaderOf([c(1, 9), c(1, 12), c(0.6, 30)], all, 0.001)).toBe(1);
    expect(leaderOf([c(0, 3), c(0, -2)], all, 0.001)).toBe(0);
    expect(leaderOf([c(0.4, 9), c(0.6, 1)], (i) => i === 0, 0.001)).toBe(0);
    expect(leaderOf([c(0.4, 9)], () => false, 0.001)).toBe(-1);
  });

  it("is behind when the difference is below 0 at the top of its interval", () => {
    const d = (n: number, mean: number, variance: number) => ({
      n,
      mean,
      variance,
    });
    const zero = d(100, 0, 0);
    const candidates = [
      c(0.6, 5, { win: zero, equity: zero, objective: zero }),
      // well behind: -0.2 + 3 * sqrt(0.04 / 100) = -0.14
      c(0.4, 5, { win: d(100, -0.2, 0.04), equity: zero, objective: zero }),
      // within reach: -0.01 + 3 * 0.02 > 0
      c(0.59, 5, { win: d(100, -0.01, 0.04), equity: zero, objective: zero }),
      // too few iterations
      c(0.1, 5, { win: d(1, -0.5, 0), equity: zero, objective: zero }),
    ];
    expect(behindLeader(candidates, 0, all, 3, 0.001)).toEqual([1]);
  });

  it("compares by equity where both games are won", () => {
    const d = (mean: number) => ({ n: 50, mean, variance: 1 });
    const candidates = [
      c(1, 20, { win: d(0), equity: d(0), objective: d(0) }),
      // the chance to win cannot tell them apart; equity can
      c(1, 10, { win: d(0), equity: d(-10), objective: d(-10) }),
    ];
    expect(behindLeader(candidates, 0, all, 3, 0.001)).toEqual([1]);
  });
});
