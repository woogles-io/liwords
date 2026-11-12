import { describe, expect, it } from "vitest";
import {
  ThreatsAnswer,
  leaveAfter,
  playSquares,
  threatHeat,
  threatsSummary,
} from "./lab_threats";

const answer = (more: Partial<ThreatsAnswer> = {}): ThreatsAnswer => ({
  racks: 10,
  count: 4,
  expected_best: 38.44,
  bingo: 0.27,
  covered: 1,
  samples: 0,
  list_ms: 3,
  exact: true,
  plays: [],
  squares: [],
  ...more,
});

describe("threatHeat", () => {
  const squares = [
    [null, { best: 40, chance: 0.5, expected: 10 }],
    [
      { best: 0, chance: 0, expected: 0 },
      { best: 10, chance: 0.25, expected: 2 },
    ],
  ];

  it("tints by the best score, scaled to the board's best", () => {
    expect(threatHeat(squares, "score")).toEqual([
      undefined,
      { level: 1, label: "40" },
      undefined,
      { level: 0.25, label: "10" },
    ]);
  });

  it("tints by the chance, as a percent", () => {
    expect(threatHeat(squares, "chance")).toEqual([
      undefined,
      { level: 0.5, label: "50" },
      undefined,
      { level: 0.25, label: "25" },
    ]);
  });
});

describe("playSquares", () => {
  it("gives the squares a play places a tile on, not those played through", () => {
    expect(
      playSquares({ down: false, lane: 7, idx: 6, word: [1, 0, 3] }, 15),
    ).toEqual([7 * 15 + 6, 7 * 15 + 8]);
    expect(
      playSquares({ down: true, lane: 2, idx: 0, word: [1, 2] }, 15),
    ).toEqual([2, 17]);
  });
});

describe("leaveAfter", () => {
  it("takes the tiles placed off the rack, a blank as 0", () => {
    expect(leaveAfter([1, 1, 0, 5], [1, 0])).toEqual([1, 5]);
  });

  it("is null when a tile placed is not on the rack", () => {
    expect(leaveAfter([1, 5], [2])).toBeNull();
  });
});

describe("threatsSummary", () => {
  it("says the expected best, the bingo chance and the tiles unseen", () => {
    expect(threatsSummary(answer(), 25)).toBe(
      "Opponent's best 38.4 on average, bingo 27.0%, from 25 unseen",
    );
  });

  it("says no bingo when the opponent holds fewer tiles than a rack", () => {
    expect(threatsSummary(answer({ bingo: null }), 5)).toBe(
      "Opponent's best 38.4 on average, from 5 unseen",
    );
  });

  it("marks an answer cut short, with how much it counted", () => {
    expect(
      threatsSummary(
        answer({ exact: false, covered: 0.625, samples: 300 }),
        60,
      ),
    ).toBe(
      "Opponent's best ~38.4 on average, bingo ~27.0%, from 60 unseen; time ran out: 62.5% of racks counted, 300 more drawn",
    );
  });
});
