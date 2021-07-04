import { BonusType } from '../../constants/board_layout';

import {
  EphemeralTile,
  Direction,
  EmptySpace,
  isBlank,
  uniqueTileIdx,
} from './common';

import { Board } from './board';
import { Alphabet, runeToValues } from '../../constants/alphabets';

export type simpletile = {
  fresh: boolean;
  letter: string;
  row: number;
  col: number;
};

const genContiguousTiles = (
  sorted: Array<EphemeralTile>,
  board: Board
): [Array<simpletile>, Direction] => {
  // build an object of contiguous tiles that includes `sorted`,
  // return the direction of play on the board.
  const contiguous: { [tileIdx: number]: simpletile } = {};

  let wordDir = Direction.Vertical;
  if (sorted.length > 1 && sorted[0].col !== sorted[1].col) {
    wordDir = Direction.Horizontal;
  } else if (sorted.length === 1) {
    // If we are placing down just one single tile, we need to determine
    // whether the word is vertical or horizontal, based on what tiles are
    // bordered.
    const tile = sorted[0];
    if (
      tileOnBoard(tile.row, tile.col + 1, board) ||
      tileOnBoard(tile.row, tile.col - 1, board)
    ) {
      wordDir = Direction.Horizontal;
    }
  }

  // Add all the tiles in sorted to the map:
  sorted.forEach((t) => {
    contiguous[uniqueTileIdx(t.row, t.col)] = {
      fresh: true,
      letter: t.letter,
      row: t.row,
      col: t.col,
    };
  });
  // Now look in the board for the remainder of the tiles.
  let lastSeenTile;
  let newRow = sorted[0].row;
  let newCol = sorted[0].col;
  while (lastSeenTile !== EmptySpace && lastSeenTile !== null) {
    if (wordDir === Direction.Horizontal) {
      newCol -= 1;
    } else {
      newRow -= 1;
    }
    lastSeenTile = board.letterAt(newRow, newCol);
    if (lastSeenTile !== EmptySpace && lastSeenTile !== null) {
      const u = uniqueTileIdx(newRow, newCol);
      contiguous[u] = {
        fresh: false,
        letter: lastSeenTile,
        row: newRow,
        col: newCol,
      };
    }
  }

  // Go back to the first tile, and begin looking to the bottom/right.
  newRow = sorted[0].row;
  newCol = sorted[0].col;
  lastSeenTile = undefined;
  while (lastSeenTile !== EmptySpace && lastSeenTile !== null) {
    if (wordDir === Direction.Horizontal) {
      newCol += 1;
    } else {
      newRow += 1;
    }
    lastSeenTile = board.letterAt(newRow, newCol);
    const u = uniqueTileIdx(newRow, newCol);
    if (lastSeenTile !== EmptySpace && lastSeenTile !== null) {
      contiguous[u] = {
        fresh: false,
        letter: lastSeenTile,
        row: newRow,
        col: newCol,
      };
    } else if (u in contiguous) {
      // Otherwise, we saw an empty space. If there is an ephemeral tile
      // on this space, set lastSeenTile to it so the loop continues.
      lastSeenTile = contiguous[u].letter;
    }
  }

  // Turn the contiguous dict into a simple sorted array of contiguous tiles.
  const retArr = Object.values(contiguous);
  retArr.sort((a, b) => {
    if (a.col === b.col) {
      return a.row - b.row;
    }
    return a.col - b.col;
  });

  return [retArr, wordDir];
};

const getCrossScore = (
  row: number,
  col: number,
  crossDir: Direction,
  board: Board,
  alphabet: Alphabet
): [number, boolean] => {
  // Traverse in both directions from (row, col) in the crossDir axis.
  let lastSeenTile;
  let crossScore = 0;
  let newRow = row;
  let newCol = col;
  let actualCrossWord = false;
  while (lastSeenTile !== EmptySpace && lastSeenTile !== null) {
    if (crossDir === Direction.Horizontal) {
      newCol -= 1;
    } else {
      newRow -= 1;
    }
    lastSeenTile = board.letterAt(newRow, newCol);
    if (lastSeenTile !== null && lastSeenTile !== EmptySpace) {
      actualCrossWord = true;
    }
    crossScore += runeToValues(alphabet, lastSeenTile);
  }
  // Now go in the other direction:
  newCol = col;
  newRow = row;
  lastSeenTile = undefined;
  while (lastSeenTile !== EmptySpace && lastSeenTile !== null) {
    if (crossDir === Direction.Horizontal) {
      newCol += 1;
    } else {
      newRow += 1;
    }
    lastSeenTile = board.letterAt(newRow, newCol);
    if (lastSeenTile !== null && lastSeenTile !== EmptySpace) {
      actualCrossWord = true;
    }
    crossScore += runeToValues(alphabet, lastSeenTile);
  }
  return [crossScore, actualCrossWord];
};

const tileOnBoard = (row: number, col: number, board: Board): boolean => {
  const letter = board.letterAt(row, col);
  return letter !== EmptySpace && letter !== null;
};

export const borders = (
  t1: EphemeralTile,
  t2: EphemeralTile,
  board: Board
): boolean => {
  // Do the two tiles touch each other either directly or across board tiles?
  if (t1.col !== t2.col && t1.row !== t2.row) {
    return false;
  }
  if (t1.col === t2.col) {
    if (Math.abs(t1.row - t2.row) === 1) {
      return true;
    }
  } else if (t1.row === t2.row) {
    if (Math.abs(t1.col - t2.col) === 1) {
      return true;
    }
  }
  // Otherwise, the tiles do not touch directly.
  if (t1.col === t2.col) {
    for (
      let i = Math.min(t1.row, t2.row) + 1;
      i < Math.max(t1.row, t2.row);
      i++
    ) {
      if (board.letterAt(i, t1.col) === EmptySpace) {
        return false;
      }
    }
  } else if (t1.row === t2.row) {
    for (
      let i = Math.min(t1.col, t2.col) + 1;
      i < Math.max(t1.col, t2.col);
      i++
    ) {
      if (board.letterAt(t1.row, i) === EmptySpace) {
        return false;
      }
    }
  }
  return true;
};

export const touchesBoardTile = (t1: EphemeralTile, board: Board): boolean => {
  // Does the tile touch any tiles on the board?
  const dirsToLook = [
    [1, 0],
    [-1, 0],
    [0, 1],
    [0, -1],
  ];

  for (let i = 0; i < dirsToLook.length; i++) {
    const row = t1.row + dirsToLook[i][0];
    const col = t1.col + dirsToLook[i][1];
    const letter = board.letterAt(row, col);
    if (letter !== EmptySpace) {
      return true;
    }
  }
  return false;
};

const isLegalPlay = (
  currentlyPlacedTiles: Array<EphemeralTile>,
  board: Board
): boolean => {
  // Check that all tiles are colinear
  const rows = new Set<number>();
  const cols = new Set<number>();
  currentlyPlacedTiles.forEach((t) => {
    rows.add(t.row);
    cols.add(t.col);
  });

  if (
    (rows.size > 1 && cols.size !== 1) ||
    (cols.size > 1 && rows.size !== 1)
  ) {
    return false;
  }

  // Play must have contiguous tiles (each placed tile must border the next one
  // either directly, or "through" tiles already on the board):
  for (let i = 0; i < currentlyPlacedTiles.length - 1; i++) {
    const t1 = currentlyPlacedTiles[i];
    const t2 = currentlyPlacedTiles[i + 1];
    if (!borders(t1, t2, board)) {
      return false;
    }
  }

  let touches = false;
  // Play must touch some tile already on the board unless board is empty.
  for (let i = 0; i < currentlyPlacedTiles.length; i++) {
    if (touchesBoardTile(currentlyPlacedTiles[i], board)) {
      touches = true;
      break;
    }
  }

  if (!touches && !board.isEmpty) {
    return false;
  }

  if (board.isEmpty) {
    // Must touch center square
    if (!(rows.has(7) && cols.has(7))) {
      return false;
    }
  }

  return true;
};

export const contiguousTilesFromTileSet = (
  tiles: Set<EphemeralTile>,
  board: Board
): [Array<simpletile>, Direction] | null => {
  const sorted = Array.from(tiles.values());
  sorted.sort((a, b) => {
    if (a.col === b.col) {
      return a.row - b.row;
    }
    return a.col - b.col;
  });

  if (!isLegalPlay(sorted, board)) {
    return null;
  }

  return genContiguousTiles(sorted, board);
};

export const calculateTemporaryScore = (
  currentlyPlacedTiles: Set<EphemeralTile>,
  board: Board,
  alphabet: Alphabet
): number | undefined => {
  const ret = contiguousTilesFromTileSet(currentlyPlacedTiles, board);
  if (ret === null) {
    return undefined;
  }
  const [wordTiles, wordDir] = ret;

  const crossDir =
    wordDir === Direction.Horizontal
      ? Direction.Vertical
      : Direction.Horizontal;

  // Ok - the play is technically legal (it may form invalid words, but
  // we won't worry about that here):
  // a lot of this code is from board.go in the macondo repo.
  let mainWordScore = 0;
  let crossScores = 0;
  let bingoBonus = 0;
  let wordMultiplier = 1;
  if (currentlyPlacedTiles.size === 7) {
    bingoBonus = 50;
  }

  wordTiles.forEach((st) => {
    const bonusSq = board.gridLayout[st.row][st.col];
    let letterMultiplier = 1;
    let crossWordMultiplier = 1; // the multiplier for the orthogonal word.
    if (st.fresh) {
      // Bonus only counts if we are putting a fresh tile on it!
      switch (bonusSq) {
        case BonusType.DoubleLetter:
          letterMultiplier = 2;
          break;
        case BonusType.TripleLetter:
          letterMultiplier = 3;
          break;
        case BonusType.DoubleWord:
          wordMultiplier *= 2;
          crossWordMultiplier = 2;
          break;
        case BonusType.TripleWord:
          wordMultiplier *= 3;
          crossWordMultiplier = 3;
          break;
      }
    }
    const [cs, realcs] = getCrossScore(
      st.row,
      st.col,
      crossDir,
      board,
      alphabet
    );
    let ls;
    if (isBlank(st.letter)) {
      ls = 0;
    } else {
      ls = runeToValues(alphabet, st.letter);
    }
    mainWordScore += ls * letterMultiplier;
    if (realcs && st.fresh) {
      // It's not enough to check that the cross-score is 0 (because of blanks)
      // we must actually check that we're forming a real cross word to add
      // any bonuses.
      crossScores +=
        ls * letterMultiplier * crossWordMultiplier + cs * crossWordMultiplier;
    }
  });

  return mainWordScore * wordMultiplier + crossScores + bingoBonus;
};
