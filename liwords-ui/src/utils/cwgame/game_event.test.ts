import { EphemeralTile } from "./common";
import { create } from "@bufbuild/protobuf";
import {
  GameEvent_Type,
  GameEventSchema,
} from "../../gen/api/proto/vendored/macondo/macondo_pb";
import {
  computeLeave,
  retainedLeave,
  tilesetToMoveDisplay,
  tilesetToMoveEvent,
} from "./game_event";
import { Board } from "./board";
import {
  StandardEnglishAlphabet,
  englishLetterToML,
} from "../../constants/alphabets";

const oxyTilesLayout = [
  " PACIFYING     ",
  " IS            ",
  "YE             ",
  " REQUALIFIED   ",
  "H L            ",
  "EDS            ",
  "NO   T         ",
  " RAINWASHING   ",
  "UM   O         ",
  "T  E O         ",
  " WAKEnERS      ",
  " OnETIME       ",
  "OOT  E B       ",
  "N      U       ",
  " JACULATING    ",
];

it("tests complex event", () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 0,
    col: 0,
    letter: englishLetterToML("O"),
  });
  placedTiles.add({
    row: 1,
    col: 0,
    letter: englishLetterToML("X"),
  });
  placedTiles.add({
    row: 3,
    col: 0,
    letter: englishLetterToML("P"),
  });
  placedTiles.add({
    row: 7,
    col: 0,
    letter: englishLetterToML("B"),
  });
  placedTiles.add({
    row: 10,
    col: 0,
    letter: englishLetterToML("A"),
  });
  placedTiles.add({
    row: 11,
    col: 0,
    letter: englishLetterToML("Z"),
  });
  placedTiles.add({
    row: 14,
    col: 0,
    letter: englishLetterToML("E"),
  });
  const board = new Board();
  board.setTileLayout(oxyTilesLayout);
  const evt = tilesetToMoveEvent(placedTiles, board, "");
  expect(evt).not.toBeNull();
  expect(evt?.positionCoords).toEqual("A1");
  expect(evt?.machineLetters).toEqual(
    Uint8Array.from([15, 24, 0, 16, 0, 0, 0, 2, 0, 0, 1, 26, 0, 0, 5]),
  ); // 'OX.P...B..AZ..E'
});

it("writes a play for the confirm before playing it", () => {
  const board = new Board();
  board.setTileLayout(oxyTilesLayout);
  const tiles = (placed: Array<[number, number, string]>) =>
    new Set<EphemeralTile>(
      placed.map(([row, col, letter]) => ({
        row,
        col,
        letter: englishLetterToML(letter),
      })),
    );
  expect(
    tilesetToMoveDisplay(
      tiles([
        [0, 0, "O"],
        [1, 0, "X"],
        [3, 0, "P"],
        [7, 0, "B"],
        [10, 0, "A"],
        [11, 0, "Z"],
        [14, 0, "E"],
      ]),
      board,
      StandardEnglishAlphabet,
    ),
  ).toEqual("A1 OX(Y)P(HEN)B(UT)AZ(ON)E");
  // one tile on the end of a word across
  expect(
    tilesetToMoveDisplay(tiles([[7, 12, "S"]]), board, StandardEnglishAlphabet),
  ).toEqual("8B (RAINWASHING)S");
  // not in one line: no play
  expect(
    tilesetToMoveDisplay(
      tiles([
        [0, 0, "O"],
        [7, 0, "B"],
      ]),
      board,
      StandardEnglishAlphabet,
    ),
  ).toBeNull();
});

it("tests invalid play", () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 0,
    col: 0,
    letter: englishLetterToML("O"),
  });
  placedTiles.add({
    row: 1,
    col: 0,
    letter: englishLetterToML("X"),
  });
  // Not contiguous; missing the Y.
  placedTiles.add({
    row: 7,
    col: 0,
    letter: englishLetterToML("B"),
  });
  placedTiles.add({
    row: 10,
    col: 0,
    letter: englishLetterToML("A"),
  });
  placedTiles.add({
    row: 11,
    col: 0,
    letter: englishLetterToML("Z"),
  });
  placedTiles.add({
    row: 14,
    col: 0,
    letter: englishLetterToML("E"),
  });
  const board = new Board();
  board.setTileLayout(oxyTilesLayout);
  const evt = tilesetToMoveEvent(placedTiles, board, "");
  expect(evt).toBeNull();
});

it("should not commit undesignated blank", () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 4,
    col: 3,
    letter: englishLetterToML("I"),
  });
  placedTiles.add({
    row: 4,
    col: 4,
    letter: englishLetterToML("?"),
  });
  placedTiles.add({
    row: 4,
    col: 5,
    letter: englishLetterToML("B"),
  });
  const board = new Board();
  board.setTileLayout(oxyTilesLayout);
  const evt = tilesetToMoveEvent(placedTiles, board, "");
  expect(evt).toBeNull();
});

it("tests event with blank", () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 4,
    col: 3,
    letter: englishLetterToML("I"),
  });
  placedTiles.add({
    row: 4,
    col: 4,
    letter: englishLetterToML("m"),
  });
  placedTiles.add({
    row: 4,
    col: 5,
    letter: englishLetterToML("B"),
  });
  const board = new Board();
  board.setTileLayout(oxyTilesLayout);
  const evt = tilesetToMoveEvent(placedTiles, board, "");
  expect(evt).not.toBeNull();
  expect(evt?.positionCoords).toEqual("5C");
  expect(evt?.machineLetters).toEqual(Uint8Array.from([0, 9, 0x80 | 13, 2]));
});

it("tests computeLeave", () => {
  expect(computeLeave("DOGS", "GOURDES")).toBe("ERU");
  expect(computeLeave("DOgS", "?OURDES")).toBe("ERU");
});

describe("retainedLeave", () => {
  const evt = (fields: Parameters<typeof create<typeof GameEventSchema>>[1]) =>
    create(GameEventSchema, fields);
  const turns = [
    evt({
      playerIndex: 0,
      type: GameEvent_Type.TILE_PLACEMENT_MOVE,
      rack: "AEINRST",
      playedTiles: "RAN.",
    }),
    evt({
      playerIndex: 1,
      type: GameEvent_Type.EXCHANGE,
      rack: "?DEIUUV",
      exchanged: "UUV",
    }),
  ];

  it("returns the on-turn player's leave from their last move", () => {
    expect(retainedLeave(turns, 0)).toBe("EIST");
    expect(retainedLeave(turns, 1)).toBe("?DEI");
  });

  it("keeps the whole rack after a phony is returned", () => {
    const withPhony = [
      ...turns,
      evt({
        playerIndex: 0,
        type: GameEvent_Type.TILE_PLACEMENT_MOVE,
        rack: "EFISTXZ",
        playedTiles: "ZEX",
      }),
      evt({ playerIndex: 0, type: GameEvent_Type.PHONY_TILES_RETURNED }),
    ];
    expect(retainedLeave(withPhony, 0)).toBe("EFISTXZ");
  });

  it("is empty before the player's first turn", () => {
    expect(retainedLeave(turns.slice(0, 1), 1)).toBe("");
  });
});
