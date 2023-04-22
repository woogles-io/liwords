import { calculateTemporaryScore, borders, touchesBoardTile } from './scoring';
import { EphemeralTile, MachineLetter } from './common';
import { Board } from './board';
import {
  StandardEnglishAlphabet,
  runesToUint8Array,
} from '../../constants/alphabets';

export const someTileLayout = [
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
];

export const englishLetterToML = (letter: string): MachineLetter => {
  const alphabet = StandardEnglishAlphabet;
  const arr = runesToUint8Array(letter, alphabet);
  return arr[0];
};

it('tests borders', () => {
  // Check the R SI (row 3) scenario
  // The actual letters don't matter here.
  const t1 = {
    row: 2,
    col: 5,
    letter: englishLetterToML('R'),
  };
  const t2 = {
    row: 2,
    col: 7,
    letter: englishLetterToML('F'),
  };
  const t3 = {
    row: 2,
    col: 10,
    letter: englishLetterToML('U'),
  };
  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(borders(t1, t2, board)).toBeTruthy();
  expect(borders(t2, t3, board)).toBeTruthy();
  expect(borders(t1, t3, board)).toBeFalsy();
});

it('testTouches', () => {
  const t1 = {
    row: 2,
    col: 5,
    letter: englishLetterToML('R'),
  };
  const t2 = {
    row: 2,
    col: 4,
    letter: englishLetterToML('T'),
  };
  const t3 = {
    row: 0,
    col: 14,
    letter: englishLetterToML('S'),
  };
  const t4 = {
    row: 5,
    col: 11,
    letter: englishLetterToML('Q'),
  };

  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(touchesBoardTile(t1, board)).toBeTruthy();
  expect(touchesBoardTile(t2, board)).toBeFalsy();
  expect(touchesBoardTile(t3, board)).toBeTruthy();
  expect(touchesBoardTile(t4, board)).toBeFalsy();
});

it('tests scores', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 9,
    col: 1,
    letter: englishLetterToML('Q'),
  });
  placedTiles.add({
    row: 9,
    col: 2,
    letter: englishLetterToML('u'),
  });
  placedTiles.add({
    row: 9,
    col: 3,
    letter: englishLetterToML('A'),
  });
  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(
    calculateTemporaryScore(placedTiles, board, StandardEnglishAlphabet)
  ).toEqual(32);
  placedTiles.add({
    row: 9,
    col: 5,
    letter: englishLetterToML('L'),
  });
  expect(
    calculateTemporaryScore(placedTiles, board, StandardEnglishAlphabet)
  ).toEqual(35);
});

it('tests more scores', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 10,
    col: 2,
    letter: englishLetterToML('Q'),
  });
  placedTiles.add({
    row: 10,
    col: 3,
    letter: englishLetterToML('u'),
  });
  placedTiles.add({
    row: 10,
    col: 5,
    letter: englishLetterToML('D'),
  });

  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(
    calculateTemporaryScore(placedTiles, board, StandardEnglishAlphabet)
  ).toEqual(13);
});

const oxyTilesLayout = [
  ' PACIFYING     ',
  ' IS            ',
  'YE             ',
  ' REQUALIFIED   ',
  'H L            ',
  'EDS            ',
  'NO   T         ',
  ' RAINWASHING   ',
  'UM   O         ',
  'T  E O         ',
  ' WAKEnERS      ',
  ' OnETIME       ',
  'OOT  E B       ',
  'N      U       ',
  ' JACULATING    ',
];

it('tests more complex scores', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 0,
    col: 0,
    letter: englishLetterToML('O'),
  });
  placedTiles.add({
    row: 1,
    col: 0,
    letter: englishLetterToML('X'),
  });
  placedTiles.add({
    row: 3,
    col: 0,
    letter: englishLetterToML('P'),
  });
  placedTiles.add({
    row: 7,
    col: 0,
    letter: englishLetterToML('B'),
  });
  placedTiles.add({
    row: 10,
    col: 0,
    letter: englishLetterToML('A'),
  });
  placedTiles.add({
    row: 11,
    col: 0,
    letter: englishLetterToML('Z'),
  });
  placedTiles.add({
    row: 14,
    col: 0,
    letter: englishLetterToML('E'),
  });
  const board = new Board();
  board.setTileLayout(oxyTilesLayout);
  expect(
    calculateTemporaryScore(placedTiles, board, StandardEnglishAlphabet)
  ).toEqual(1780);
});

it('tests opening score', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({ row: 7, col: 7, letter: englishLetterToML('Q') });
  placedTiles.add({ row: 7, col: 8, letter: englishLetterToML('I') });
  const board = new Board();
  expect(
    calculateTemporaryScore(placedTiles, board, StandardEnglishAlphabet)
  ).toEqual(22);
});

it('tests scores vertical', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 6,
    col: 7,
    letter: englishLetterToML('M'),
  });
  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(
    calculateTemporaryScore(placedTiles, board, StandardEnglishAlphabet)
  ).toEqual(4);
  placedTiles.add({
    row: 8,
    col: 7,
    letter: englishLetterToML('L'),
  });
  expect(
    calculateTemporaryScore(placedTiles, board, StandardEnglishAlphabet)
  ).toEqual(5);
});

it('tests scores horizontal', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 10,
    col: 3,
    letter: englishLetterToML('M'),
  });
  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(
    calculateTemporaryScore(placedTiles, board, StandardEnglishAlphabet)
  ).toEqual(4);
  placedTiles.add({
    row: 10,
    col: 5,
    letter: englishLetterToML('L'),
  });
  expect(
    calculateTemporaryScore(placedTiles, board, StandardEnglishAlphabet)
  ).toEqual(5);
});
