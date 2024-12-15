import { someTileLayout } from "./scoring.test";
import { getWordsFormed, handleKeyPress } from "./tile_placement";
import { Board } from "./board";
import { EphemeralTile } from "./common";
import {
  StandardEnglishAlphabet,
  englishLetterToML,
} from "../../constants/alphabets";

it("getWordsFormed lists all words on board when no tiles are passed", () => {
  const board = new Board();
  board.setTileLayout(someTileLayout);
  const wordsFormed = getWordsFormed(board, undefined, StandardEnglishAlphabet);
  expect(wordsFormed).toHaveLength(8);
  expect(wordsFormed).toContain("ZoNATION");
  expect(wordsFormed).toContain("SI");
});

it("getWordsFormed lists only new words when tiles are passed", () => {
  const board = new Board();
  board.setTileLayout(someTileLayout);
  const horizontalPlay = new Set<EphemeralTile>();
  horizontalPlay.add({
    row: 5,
    col: 9,
    letter: englishLetterToML("E"),
  });
  horizontalPlay.add({
    row: 5,
    col: 10,
    letter: englishLetterToML("S"),
  });
  horizontalPlay.add({
    row: 5,
    col: 11,
    letter: englishLetterToML("T"),
  });
  const wordsFormedHoriz = getWordsFormed(
    board,
    horizontalPlay,
    StandardEnglishAlphabet,
  );
  expect(wordsFormedHoriz).toHaveLength(3);
  expect(wordsFormedHoriz).toContain("TEST");
  expect(wordsFormedHoriz).toContain("RE");
  expect(wordsFormedHoriz).toContain("OS");

  const verticalPlay = new Set<EphemeralTile>();
  verticalPlay.add({
    row: 10,
    col: 3,
    letter: englishLetterToML("T"),
  });
  verticalPlay.add({
    row: 11,
    col: 3,
    letter: englishLetterToML("E"),
  });
  verticalPlay.add({
    row: 12,
    col: 3,
    letter: englishLetterToML("s"),
  });
  verticalPlay.add({
    row: 13,
    col: 3,
    letter: englishLetterToML("T"),
  });

  const wordsFormedVertical = getWordsFormed(
    board,
    verticalPlay,
    StandardEnglishAlphabet,
  );
  expect(wordsFormedVertical).toHaveLength(3);
  expect(wordsFormedVertical).toContain("TEsT");
  expect(wordsFormedVertical).toContain("TO");
  expect(wordsFormedVertical).toContain("EN");
});
/*
  '         RADIOS',
  '         E     ',
  '      R SI     ',
  '      U E      ',
  '    ZINGARO    ',
  '    o   T      ',
  '    N          ',
  '   WASTE       ',
  '    T          ',
  '    I          ',
  '    O          ',
  '    N          ',
  '               ',
  '               ',
  '               ',
  */

describe("handleKeyPress test suite", () => {
  it("does nothing if arrow is not showing", () => {
    const board = new Board();
    board.setTileLayout(someTileLayout);
    const arrow = { row: 7, col: 8, horizontal: true, show: false };
    // rack is empty, empty, O, L, T, U, V
    const resp = handleKeyPress(
      arrow,
      board,
      "L",
      [0x80, 0x80, 15, 12, 20, 21, 22],
      new Set(),
      StandardEnglishAlphabet,
    );
    expect(resp).toBe(null);
  });

  it("places letter on board properly", () => {
    const board = new Board();
    board.setTileLayout(someTileLayout);
    const arrow = { row: 7, col: 8, horizontal: true, show: true };
    const resp = handleKeyPress(
      arrow,
      board,
      "l",
      [0x80, 0x80, 15, 12, 20, 21, 22],
      new Set(),
      StandardEnglishAlphabet,
    );
    // arrow should also advance
    expect(resp).toStrictEqual({
      newPlacedTiles: new Set([{ row: 7, col: 8, letter: 12 }]),
      newDisplayedRack: [0x80, 0x80, 15, 0x80, 20, 21, 22],
      playScore: 9,
      newArrow: { row: 7, col: 9, horizontal: true, show: true },
    });
  });

  it("places blank letter on board properly", () => {
    const board = new Board();
    board.setTileLayout(someTileLayout);
    const arrow = { row: 7, col: 8, horizontal: true, show: true };
    const resp = handleKeyPress(
      arrow,
      board,
      "L",
      [0x80, 0x80, 15, 0, 20, 21, 22],
      new Set(),
      StandardEnglishAlphabet,
    );
    expect(resp).toStrictEqual({
      newPlacedTiles: new Set([{ row: 7, col: 8, letter: 12 | 0x80 }]),
      newDisplayedRack: [0x80, 0x80, 15, 0x80, 20, 21, 22],
      playScore: 8,
      newArrow: { row: 7, col: 9, horizontal: true, show: true },
    });
  });

  it("uses blank as last resort", () => {
    const board = new Board();
    board.setTileLayout(someTileLayout);
    const arrow = { row: 7, col: 8, horizontal: true, show: true };
    const resp = handleKeyPress(
      arrow,
      board,
      "l",
      // l isn't on rack, and wasn't capitalized. but since we have a blank it should
      // still be used.
      [0x80, 0x80, 15, 0, 20, 21, 22],
      new Set(),
      StandardEnglishAlphabet,
    );
    expect(resp).toStrictEqual({
      newPlacedTiles: new Set([{ row: 7, col: 8, letter: 12 | 0x80 }]),
      newDisplayedRack: [0x80, 0x80, 15, 0x80, 20, 21, 22],
      playScore: 8,
      newArrow: { row: 7, col: 9, horizontal: true, show: true },
    });
  });

  it("does nothing if letter is not on rack", () => {
    const board = new Board();
    board.setTileLayout(someTileLayout);
    const arrow = { row: 7, col: 8, horizontal: true, show: true };
    // Try blank L
    let resp = handleKeyPress(
      arrow,
      board,
      "L",
      [0x80, 0x80, 15, 0x80, 20, 21, 22],
      new Set(),
      StandardEnglishAlphabet,
    );
    expect(resp).toBe(null);
    // Try natural L, which we also don't have
    resp = handleKeyPress(
      arrow,
      board,
      "l",
      [0x80, 0x80, 15, 0x80, 20, 21, 22],
      new Set(),
      StandardEnglishAlphabet,
    );
    expect(resp).toBe(null);
  });

  it("does not place arrow under tile", () => {
    const board = new Board();
    board.setTileLayout(someTileLayout);
    // Put the arrow immediately after WASTE.
    const arrow = { row: 7, col: 8, horizontal: true, show: true };
    // then pretend we just dragged a tile to row 7 col 9
    board.addTile({ row: 7, col: 9, ml: 15 });
    // arrow should jump after the O once we type in a tile
    const resp = handleKeyPress(
      arrow,
      board,
      "L",
      [0x80, 0x80, 15, 12, 20, 21, 22],
      new Set(),
      StandardEnglishAlphabet,
    );
    expect(resp).toStrictEqual({
      newPlacedTiles: new Set([{ row: 7, col: 8, letter: 12 }]),
      newDisplayedRack: [0x80, 0x80, 15, 0x80, 20, 21, 22],
      playScore: 10,
      // arrow skips
      newArrow: { row: 7, col: 10, horizontal: true, show: true },
    });
  });

  it("places arrow off edge", () => {
    const board = new Board();
    board.setTileLayout(someTileLayout);
    const arrow = { row: 0, col: 8, horizontal: true, show: true };

    const resp = handleKeyPress(
      arrow,
      board,
      "e",
      [0x80, 0x80, 15, 5, 20, 21, 22],
      new Set(),
      StandardEnglishAlphabet,
    );
    expect(resp).toStrictEqual({
      newPlacedTiles: new Set([{ row: 0, col: 8, letter: 5 }]),
      newDisplayedRack: [0x80, 0x80, 15, 0x80, 20, 21, 22],
      playScore: 8,
      // arrow goes all the way off the board. oh well.
      newArrow: { row: 0, col: 15, horizontal: true, show: true },
    });
  });
});
