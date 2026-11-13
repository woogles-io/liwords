import { describe, expect, it } from "vitest";
import { EndgamePartAnswer, mergeEndgameAnswers } from "./lab_endgame_merge";

const ba = { action: "play", word: [2, 1] };
const pass = { action: "exchange", tiles: [] };

const part = (
  k: number,
  more: Partial<EndgamePartAnswer>,
): EndgamePartAnswer => ({
  bag: 2,
  steps: 3,
  nodes: 100,
  exact: true,
  play: ba,
  value: 0,
  win: 0,
  spread: 10,
  depth: 127,
  endgames: 20,
  moves: [9, 9],
  part: [k, 2],
  ...more,
});

describe("a pre-endgame's parts merged", () => {
  it("puts each play's rows together, weighs its win and value by deals, and picks the best", () => {
    const merged = mergeEndgameAnswers([
      part(0, {
        // this part's best is BA, from its own sets only
        play: ba,
        value: 4,
        win: 1,
        spread: 14,
        candidates: [
          {
            play: ba,
            value: 4,
            win: 1,
            exact: true,
            draws: [{ tiles: [5, 9], count: 3, value: 4, win: 1 }],
          },
          {
            play: pass,
            value: 1,
            win: 0.5,
            exact: true,
            draws: [{ tiles: [5, 9], count: 3, value: 1, win: 0.5 }],
          },
        ],
      }),
      part(1, {
        play: pass,
        value: 2,
        win: 1,
        spread: 12,
        nodes: 50,
        steps: 2,
        candidates: [
          {
            play: pass,
            value: 2,
            win: 1,
            exact: true,
            draws: [{ tiles: [9, 9], count: 1, value: 2, win: 1 }],
          },
          {
            play: ba,
            value: -20,
            win: 0,
            exact: true,
            draws: [{ tiles: [9, 9], count: 1, value: -20, win: 0 }],
          },
        ],
      }),
    ]);
    // BA: (3 * 1 + 1 * 0) / 4 wins, (3 * 4 - 20) / 4 value; the pass:
    // (3 * 0.5 + 1) / 4 wins, (3 + 2) / 4 value
    expect(merged.candidates?.map((c) => [c.play, c.win, c.value])).toEqual([
      [ba, 0.75, -2],
      [pass, 0.625, 1.25],
    ]);
    expect(merged).toMatchObject({
      exact: true,
      play: ba,
      value: -2,
      win: 0.75,
      // the lead now 10
      spread: 8,
      steps: 2,
      nodes: 150,
      endgames: 40,
      draws: [
        { tiles: [5, 9], count: 3, value: 4, win: 1 },
        { tiles: [9, 9], count: 1, value: -20, win: 0 },
      ],
    });
  });

  it("is exact only once every part is, a play estimated in any part estimated", () => {
    const merged = mergeEndgameAnswers([
      part(0, {
        exact: false,
        depth: 2,
        candidates: [{ play: ba, value: 0, win: 0, exact: false, draws: [] }],
      }),
      part(1, {
        candidates: [{ play: ba, value: 0, win: 0, exact: true, draws: [] }],
      }),
    ]);
    expect(merged.exact).toBe(false);
    expect(merged.depth).toBe(2);
    expect(merged.candidates?.[0].exact).toBe(false);
  });

  it("has no play while a part has valued none, nor plays only some parts list", () => {
    const early = mergeEndgameAnswers([
      part(0, {
        exact: false,
        play: null,
        value: undefined,
        win: undefined,
        depth: 0,
        moves: [0, 9],
        candidates: [],
      }),
      part(1, {
        exact: false,
        moves: [4, 9],
        candidates: [{ play: ba, value: 3, win: 1, draws: [] }],
      }),
    ]);
    expect(early).toMatchObject({
      exact: false,
      play: null,
      depth: 0,
      moves: [0, 9],
      candidates: [],
      draws: [],
    });
    expect(early.value).toBeUndefined();
    expect(early.win).toBeUndefined();
    const some = mergeEndgameAnswers([
      part(0, {
        candidates: [
          { play: ba, value: 3, win: 1, draws: [] },
          { play: pass, value: 1, win: 1, draws: [] },
        ],
      }),
      part(1, { candidates: [{ play: ba, value: 3, win: 1, draws: [] }] }),
    ]);
    expect(some.candidates?.map((c) => c.play)).toEqual([ba]);
  });
});
