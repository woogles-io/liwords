import { describe, expect, it } from "vitest";
import {
  StandardEnglishAlphabet,
  runesToMachineWord,
} from "../constants/alphabets";
import {
  GameEvent_Direction,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { Board } from "../utils/cwgame/board";
import { EphemeralTile } from "../utils/cwgame/common";
import {
  EXCHANGE_MIN_BAG,
  exchangeMinBag,
  placedOfPlay,
  whatIfDraw,
  whatIfExchange,
  whatIfPass,
  whatIfPlay,
} from "./what_if";

const ml = (s: string) => runesToMachineWord(s, StandardEnglishAlphabet);

// AB across from the center square, a double word: (1 + 3) * 2
const placedAB = () =>
  new Set<EphemeralTile>([
    { row: 7, col: 7, letter: ml("A")[0] },
    { row: 7, col: 8, letter: ml("B")[0] },
  ]);

const side = (more = {}) => ({
  playerIndex: 1,
  score: 100,
  rack: ml("ABE"),
  otherScore: 90,
  otherRack: ml("QE"),
  bag: 0,
  unseen: ml("ABEQE"),
  ...more,
});

describe("whatIfPlay", () => {
  it("scores the move placed by the page alone, as an event", () => {
    const events = whatIfPlay(
      placedAB(),
      new Board(),
      StandardEnglishAlphabet,
      side(),
    )!;
    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({
      type: GameEvent_Type.TILE_PLACEMENT_MOVE,
      playerIndex: 1,
      rack: "ABE",
      row: 7,
      column: 7,
      direction: GameEvent_Direction.HORIZONTAL,
      playedTiles: "AB",
      score: 8,
      cumulative: 108,
    });
  });

  it("adds twice the other rack for going out with the bag empty", () => {
    const events = whatIfPlay(
      placedAB(),
      new Board(),
      StandardEnglishAlphabet,
      side({ rack: ml("AB") }),
    )!;
    // Q 10 and E 1, twice
    expect(events[1]).toMatchObject({
      type: GameEvent_Type.END_RACK_PTS,
      playerIndex: 1,
      endRackPoints: 22,
      cumulative: 130,
    });
  });

  it("does not go out while the bag holds tiles", () => {
    expect(
      whatIfPlay(
        placedAB(),
        new Board(),
        StandardEnglishAlphabet,
        side({ rack: ml("AB"), bag: 3 }),
      ),
    ).toHaveLength(1);
  });

  it("is null for tiles that make no play", () => {
    const apart = new Set<EphemeralTile>([
      { row: 7, col: 7, letter: ml("A")[0] },
      { row: 9, col: 9, letter: ml("B")[0] },
    ]);
    expect(
      whatIfPlay(apart, new Board(), StandardEnglishAlphabet, side()),
    ).toBeNull();
  });
});

describe("whatIfPlay's tiles", () => {
  it("refuses a tile the mover cannot hold", () => {
    // a Q, the other player's only one
    const q = new Set<EphemeralTile>([
      { row: 7, col: 7, letter: ml("Q")[0] },
      { row: 7, col: 8, letter: ml("A")[0] },
    ]);
    expect(
      whatIfPlay(q, new Board(), StandardEnglishAlphabet, side()),
    ).toBeNull();
  });

  it("takes a tile from the bag for the part of the rack not known", () => {
    const za = new Set<EphemeralTile>([
      { row: 7, col: 7, letter: ml("Z")[0] },
      { row: 7, col: 8, letter: ml("A")[0] },
    ]);
    const known = side({ rack: ml("A"), bag: 3, unseen: ml("AZQEXY") });
    expect(
      whatIfPlay(za, new Board(), StandardEnglishAlphabet, known),
    ).toHaveLength(1);
    // the rack known in full: no room for a tile not on it
    const full = side({
      rack: ml("AEEIIOU"),
      bag: 3,
      unseen: ml("AEEIIOUZQX"),
    });
    expect(
      whatIfPlay(za, new Board(), StandardEnglishAlphabet, full),
    ).toBeNull();
  });
});

describe("whatIfPass", () => {
  it("passes, the score as it was", () => {
    expect(whatIfPass(StandardEnglishAlphabet, side())[0]).toMatchObject({
      type: GameEvent_Type.PASS,
      playerIndex: 1,
      cumulative: 100,
    });
  });
});

describe("whatIfDraw", () => {
  // the mover holds ABE and places AB; unseen: their rack and the rest
  const unseen = ml("ABECCDQ");

  it("takes as many as the move places, from the tiles unseen", () => {
    expect(whatIfDraw(ml("CD"), 2, ml("ABE"), unseen, 5)).toEqual(ml("CD"));
  });

  it("takes only what the bag holds", () => {
    expect(whatIfDraw(ml("C"), 2, ml("ABE"), unseen, 1)).toEqual(ml("C"));
    expect(whatIfDraw(ml("CD"), 2, ml("ABE"), unseen, 1)).toBeNull();
  });

  it("refuses the mover's own tiles, or more of a tile than is unseen", () => {
    expect(whatIfDraw(ml("AC"), 2, ml("ABE"), unseen, 5)).toBeNull();
    expect(whatIfDraw(ml("DD"), 2, ml("ABE"), unseen, 5)).toBeNull();
  });
});

describe("whatIfExchange", () => {
  it("puts back tiles of the rack, the score as it was", () => {
    expect(
      whatIfExchange(
        ml("AE"),
        StandardEnglishAlphabet,
        side({ bag: EXCHANGE_MIN_BAG }),
        EXCHANGE_MIN_BAG,
      )![0],
    ).toMatchObject({
      type: GameEvent_Type.EXCHANGE,
      playerIndex: 1,
      rack: "ABE",
      exchanged: "AE",
      cumulative: 100,
    });
  });

  it("refuses tiles not on the rack, none, or too few in the bag", () => {
    const full = side({ bag: EXCHANGE_MIN_BAG });
    const al = StandardEnglishAlphabet;
    expect(whatIfExchange(ml("AA"), al, full, EXCHANGE_MIN_BAG)).toBeNull();
    expect(whatIfExchange([], al, full, EXCHANGE_MIN_BAG)).toBeNull();
    expect(
      whatIfExchange(
        ml("A"),
        al,
        side({ bag: EXCHANGE_MIN_BAG - 1 }),
        EXCHANGE_MIN_BAG,
      ),
    ).toBeNull();
  });

  it("lets Spanish exchange with a tile in the bag", () => {
    expect(exchangeMinBag("FILE2017")).toBe(1);
    expect(exchangeMinBag("CSW24")).toBe(EXCHANGE_MIN_BAG);
  });
});

describe("placedOfPlay", () => {
  it("places a play's own tiles, a wasm blank as a designated one", () => {
    expect([
      ...placedOfPlay({ down: true, lane: 3, idx: 5, word: [1, 0, -2] }),
    ]).toEqual([
      { row: 5, col: 3, letter: 1 },
      { row: 7, col: 3, letter: 2 | 0x80 },
    ]);
  });
});
