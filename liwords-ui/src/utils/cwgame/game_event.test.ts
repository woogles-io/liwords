import { EphemeralTile } from './common';
import { tilesetToMoveEvent } from './game_event';
import { Board } from './game';

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
  const evt = tilesetToMoveEvent(placedTiles, board);
  expect(evt).not.toBeNull();
  expect(evt?.getPositionCoords()).toEqual('A1');
  expect(evt?.getTiles()).toEqual('OX.P...B..AZ..E');
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
  const evt = tilesetToMoveEvent(placedTiles, board);
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
  const evt = tilesetToMoveEvent(placedTiles, board);
  expect(evt).not.toBeNull();
  expect(evt?.getPositionCoords()).toEqual('5C');
  expect(evt?.getTiles()).toEqual('.ImB');
});
