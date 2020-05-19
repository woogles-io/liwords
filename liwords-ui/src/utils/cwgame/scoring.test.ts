import { calculateTemporaryScore, borders, touchesBoardTile } from './scoring';
import { EphemeralTile } from './common';

const gridLayout = [
  "=  '   =   '  =",
  ' -   "   "   - ',
  "  -   ' '   -  ",
  "'  -   '   -  '",
  '    -     -    ',
  ' "   "   "   " ',
  "  '   ' '   '  ",
  "=  '   -   '  =",
  "  '   ' '   '  ",
  ' "   "   "   " ',
  '    -     -    ',
  "'  -   '   -  '",
  "  -   ' '   -  ",
  ' -   "   "   - ',
  "=  '   =   '  =",
];

const tilesLayout = [
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

  expect(borders(t1, t2, tilesLayout)).toBeTruthy();
  expect(borders(t2, t3, tilesLayout)).toBeTruthy();
  expect(borders(t1, t3, tilesLayout)).toBeFalsy();
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
  expect(touchesBoardTile(t1, tilesLayout)).toBeTruthy();
  expect(touchesBoardTile(t2, tilesLayout)).toBeFalsy();
  expect(touchesBoardTile(t3, tilesLayout)).toBeTruthy();
  expect(touchesBoardTile(t4, tilesLayout)).toBeFalsy();
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
  expect(calculateTemporaryScore(placedTiles, tilesLayout, gridLayout)).toEqual(
    32
  );
  placedTiles.add({
    row: 9,
    col: 5,
    letter: 'L',
  });
  expect(calculateTemporaryScore(placedTiles, tilesLayout, gridLayout)).toEqual(
    35
  );
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
  expect(calculateTemporaryScore(placedTiles, tilesLayout, gridLayout)).toEqual(
    13
  );
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
  expect(
    calculateTemporaryScore(placedTiles, oxyTilesLayout, gridLayout)
  ).toEqual(1780);
});
