import { calculateTemporaryScore, borders, touchesBoardTile } from './scoring';
import { EphemeralTile } from './common';
import { Board } from './board';

const someTileLayout = [
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

it('tests borders', () => {
  // Check the R SI (row 3) scenario
  // The actual letters don't matter here.
  const t1 = {
    row: 2,
    col: 5,
    letter: 'R',
  };
  const t2 = {
    row: 2,
    col: 7,
    letter: 'F',
  };
  const t3 = {
    row: 2,
    col: 10,
    letter: 'U',
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
    letter: 'R',
  };
  const t2 = {
    row: 2,
    col: 4,
    letter: 'T',
  };
  const t3 = {
    row: 0,
    col: 14,
    letter: 'S',
  };
  const t4 = {
    row: 5,
    col: 11,
    letter: 'Q',
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
    letter: 'Q',
  });
  placedTiles.add({
    row: 9,
    col: 2,
    letter: 'u',
  });
  placedTiles.add({
    row: 9,
    col: 3,
    letter: 'A',
  });
  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(calculateTemporaryScore(placedTiles, board)).toEqual(32);
  placedTiles.add({
    row: 9,
    col: 5,
    letter: 'L',
  });
  expect(calculateTemporaryScore(placedTiles, board)).toEqual(35);
});

it('tests more scores', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 10,
    col: 2,
    letter: 'Q',
  });
  placedTiles.add({
    row: 10,
    col: 3,
    letter: 'u',
  });
  placedTiles.add({
    row: 10,
    col: 5,
    letter: 'D',
  });

  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(calculateTemporaryScore(placedTiles, board)).toEqual(13);
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
    letter: 'O',
  });
  placedTiles.add({
    row: 1,
    col: 0,
    letter: 'X',
  });
  placedTiles.add({
    row: 3,
    col: 0,
    letter: 'P',
  });
  placedTiles.add({
    row: 7,
    col: 0,
    letter: 'B',
  });
  placedTiles.add({
    row: 10,
    col: 0,
    letter: 'A',
  });
  placedTiles.add({
    row: 11,
    col: 0,
    letter: 'Z',
  });
  placedTiles.add({
    row: 14,
    col: 0,
    letter: 'E',
  });
  const board = new Board();
  board.setTileLayout(oxyTilesLayout);
  expect(calculateTemporaryScore(placedTiles, board)).toEqual(1780);
});

it('tests opening score', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({ row: 7, col: 7, letter: 'Q' });
  placedTiles.add({ row: 7, col: 8, letter: 'I' });
  const board = new Board();
  expect(calculateTemporaryScore(placedTiles, board)).toEqual(22);
});

it('tests scores vertical', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 6,
    col: 7,
    letter: 'M',
  });
  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(calculateTemporaryScore(placedTiles, board)).toEqual(4);
  placedTiles.add({
    row: 8,
    col: 7,
    letter: 'L',
  });
  expect(calculateTemporaryScore(placedTiles, board)).toEqual(5);
});

it('tests scores horizontal', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 10,
    col: 3,
    letter: 'M',
  });
  const board = new Board();
  board.setTileLayout(someTileLayout);

  expect(calculateTemporaryScore(placedTiles, board)).toEqual(4);
  placedTiles.add({
    row: 10,
    col: 5,
    letter: 'L',
  });
  expect(calculateTemporaryScore(placedTiles, board)).toEqual(5);
});
