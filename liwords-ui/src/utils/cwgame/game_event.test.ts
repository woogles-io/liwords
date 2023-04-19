import { EphemeralTile } from './common';
import { computeLeave, tilesetToMoveEvent } from './game_event';
import { Board } from './board';
import { StandardEnglishAlphabet } from '../../constants/alphabets';

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

it('tests complex event', () => {
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
  const evt = tilesetToMoveEvent(
    placedTiles,
    board,
    '',
    StandardEnglishAlphabet
  );
  expect(evt).not.toBeNull();
  expect(evt?.positionCoords).toEqual('A1');
  expect(evt?.machineLetters).toEqual(
    Uint8Array.from([15, 24, 0, 16, 0, 0, 0, 2, 0, 0, 1, 26, 0, 0, 5])
  ); // 'OX.P...B..AZ..E'
});

it('tests invalid play', () => {
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
  // Not contiguous; missing the Y.
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
  const evt = tilesetToMoveEvent(
    placedTiles,
    board,
    '',
    StandardEnglishAlphabet
  );
  expect(evt).toBeNull();
});

it('should not commit undesignated blank', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 4,
    col: 3,
    letter: 'I',
  });
  placedTiles.add({
    row: 4,
    col: 4,
    letter: '?',
  });
  placedTiles.add({
    row: 4,
    col: 5,
    letter: 'B',
  });
  const board = new Board();
  board.setTileLayout(oxyTilesLayout);
  const evt = tilesetToMoveEvent(
    placedTiles,
    board,
    '',
    StandardEnglishAlphabet
  );
  expect(evt).toBeNull();
});

it('tests event with blank', () => {
  const placedTiles = new Set<EphemeralTile>();
  placedTiles.add({
    row: 4,
    col: 3,
    letter: 'I',
  });
  placedTiles.add({
    row: 4,
    col: 4,
    letter: 'm',
  });
  placedTiles.add({
    row: 4,
    col: 5,
    letter: 'B',
  });
  const board = new Board();
  board.setTileLayout(oxyTilesLayout);
  const evt = tilesetToMoveEvent(
    placedTiles,
    board,
    '',
    StandardEnglishAlphabet
  );
  expect(evt).not.toBeNull();
  expect(evt?.positionCoords).toEqual('5C');
  expect(evt?.machineLetters).toEqual(Uint8Array.from([0, 9, 0x80 | 13, 2]));
});

it('tests computeLeave', () => {
  expect(computeLeave('DOGS', 'GOURDES')).toBe('ERU');
  expect(computeLeave('DOgS', '?OURDES')).toBe('ERU');
});
