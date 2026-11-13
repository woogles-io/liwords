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
import { whatIfPass, whatIfPlay } from "./what_if";

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

describe("whatIfPass", () => {
  it("passes, the score as it was", () => {
    expect(whatIfPass(StandardEnglishAlphabet, side())[0]).toMatchObject({
      type: GameEvent_Type.PASS,
      playerIndex: 1,
      cumulative: 100,
    });
  });
});
