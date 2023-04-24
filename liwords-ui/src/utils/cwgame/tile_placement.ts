/** @fileoverview business logic for placing tiles on a board */

import {
  EphemeralTile,
  uniqueTileIdx,
  MachineLetter,
  makeBlank,
  EmptyBoardSpaceMachineLetter,
} from './common';
import { calculateTemporaryScore } from './scoring';
import { Board } from './board';
import { Alphabet, machineLetterToRune } from '../../constants/alphabets';
import { isDesignatedBlankMachineLetter } from './common';
import { BlankMachineLetter } from './common';
import { EmptyMachineLetter } from './common';

const NormalizedBackspace = 'BACKSPACE';
const NormalizedSpace = ' ';

export type PlacementArrow = {
  row: number;
  col: number;
  horizontal: boolean;
  show: boolean;
};

export const nextArrowPropertyState = (
  props: PlacementArrow,
  row: number,
  col: number
): PlacementArrow => {
  if (row !== props.row || col !== props.col) {
    // start over
    return {
      row,
      col,
      show: true,
      horizontal: true,
    };
  }

  let nextHoriz = false;
  let nextShow = false;

  if (props.show) {
    if (props.horizontal) {
      nextHoriz = false;
      nextShow = true;
    } else {
      nextShow = false;
    }
  } else {
    nextShow = true;
    nextHoriz = true;
  }

  // Return the next arrow click state given the current one.
  return {
    row,
    col,
    show: nextShow,
    horizontal: nextHoriz,
  };
};

type PlacementHandlerReturn = {
  newPlacedTiles: Set<EphemeralTile>;
  newDisplayedRack: Array<MachineLetter>;
  playScore: number | undefined; // undefined for illegal plays
  isUndesignated?: boolean;
};

interface KeypressHandlerReturn extends PlacementHandlerReturn {
  newArrow: PlacementArrow;
}

export const handleTileDeletion = (
  arrowProperty: PlacementArrow,
  unplacedTiles: Array<MachineLetter>, // tiles currently still on rack
  currentlyPlacedTiles: Set<EphemeralTile>,
  board: Board,
  alphabet: Alphabet
): KeypressHandlerReturn => {
  // Remove any tiles.
  const newUnplacedTiles = [...unplacedTiles];
  const newPlacedTiles = new Set(currentlyPlacedTiles);

  currentlyPlacedTiles.forEach((t) => {
    if (t.col === arrowProperty.col && t.row === arrowProperty.row) {
      // Remove this tile.
      newPlacedTiles.delete(t);
      // can't exit early but w/e, this is fast
      let { letter } = t;
      if (isDesignatedBlankMachineLetter(letter)) {
        // unassign the blank
        letter = BlankMachineLetter;
      }
      const emptyIndex = newUnplacedTiles.indexOf(EmptyMachineLetter);
      if (emptyIndex >= 0) {
        newUnplacedTiles[emptyIndex] = letter;
      } else {
        newUnplacedTiles.push(letter);
      }
    }
  });

  return {
    newArrow: arrowProperty,
    newPlacedTiles,
    newDisplayedRack: newUnplacedTiles,
    playScore: calculateTemporaryScore(newPlacedTiles, board, alphabet),
  };
};

const getMachineLetterFor = (
  key: string,
  alphabet: Alphabet
): MachineLetter | null => {
  let foundML = alphabet.machineLetterMap[key];
  if (foundML == null) {
    foundML = alphabet.shortcutMap[key];
  }
  return foundML;
};

/**
 * This is a fairly important function for placing tiles with the keyboard.
 * It handles a keypress, and takes in the current direction of the placement
 * arrow, as well as the board tiles, etc.
 * XXX: The logic in this function is very ugly. We need to write some tests
 * and try to clean up the logic a bit. There's a lot of fiddly special cases.
 */
export const handleKeyPress = (
  arrowProperty: PlacementArrow,
  board: Board,
  key: string,
  unplacedTiles: Array<MachineLetter>, // tiles currently still on rack
  currentlyPlacedTiles: Set<EphemeralTile>,
  alphabet: Alphabet
): KeypressHandlerReturn | null => {
  const normalizedKey = key.toUpperCase();

  const newPlacedTiles = new Set(currentlyPlacedTiles);

  // Create an ephemeral tile map with unique keys.
  const ephTileMap: { [tileIdx: number]: EphemeralTile } = {};
  currentlyPlacedTiles.forEach((t) => {
    ephTileMap[uniqueTileIdx(t.row, t.col)] = t;
  });

  if (
    !Object.prototype.hasOwnProperty.call(alphabet.letterMap, normalizedKey) &&
    !Object.prototype.hasOwnProperty.call(
      alphabet.shortcutMap,
      normalizedKey
    ) &&
    normalizedKey !== NormalizedBackspace &&
    normalizedKey !== NormalizedSpace
  ) {
    // Return with no changes.
    return null;
  }

  // Make sure we're not trying to type off the edge of the board.
  if (
    !arrowProperty.show ||
    arrowProperty.row >= board.dim ||
    arrowProperty.col >= board.dim ||
    arrowProperty.row < 0 ||
    arrowProperty.col < 0
  ) {
    if (
      normalizedKey !== NormalizedBackspace &&
      normalizedKey !== NormalizedSpace
    ) {
      return null;
    }
  }

  let increment = 1;
  // Check the backspace and unplay any tiles if necessary.
  if (normalizedKey === NormalizedBackspace) {
    increment = -1;
  }

  let newrow = arrowProperty.row;
  let newcol = arrowProperty.col;
  const newUnplacedTiles = [...unplacedTiles];

  // First figure out where to put the arrow, no matter what.
  if (arrowProperty.horizontal) {
    do {
      newcol += increment;
    } while (
      newcol < board.dim &&
      newcol >= 0 &&
      (board.letterAt(newrow, newcol) !== EmptyMachineLetter ||
        (increment === 1 &&
          ephTileMap[uniqueTileIdx(newrow, newcol)] !== undefined))
    );
  } else {
    do {
      newrow += increment;
    } while (
      newrow < board.dim &&
      newrow >= 0 &&
      (board.letterAt(newrow, newcol) !== EmptyMachineLetter ||
        (increment === 1 &&
          ephTileMap[uniqueTileIdx(newrow, newcol)] !== undefined))
    );
  }

  if (normalizedKey === NormalizedBackspace) {
    // Don't allow the arrow to go off-screen when backspacing.
    if (newrow < 0) {
      newrow = arrowProperty.row;
    }
    if (newcol < 0) {
      newcol = arrowProperty.col;
    }
    return handleTileDeletion(
      {
        row: newrow,
        col: newcol,
        horizontal: arrowProperty.horizontal,
        show: true,
      },
      unplacedTiles,
      currentlyPlacedTiles,
      board,
      alphabet
    );
  }

  if (normalizedKey === NormalizedSpace) {
    if (newrow > board.dim - 1) {
      newrow = arrowProperty.row;
    }
    if (newcol > board.dim - 1) {
      newcol = arrowProperty.col;
    }
    return {
      newArrow: {
        row: newrow,
        col: newcol,
        horizontal: arrowProperty.horizontal,
        show: true,
      },
      newPlacedTiles,
      newDisplayedRack: newUnplacedTiles,
      playScore: calculateTemporaryScore(newPlacedTiles, board, alphabet),
    };
  }

  const blankIdx = unplacedTiles.indexOf(BlankMachineLetter);
  let existed = false;

  if (blankIdx !== -1 && normalizedKey === key) {
    // If there is a blank, and the user specifically requested to use it
    // (by typing the letter with a Shift)
    const foundML = getMachineLetterFor(normalizedKey, alphabet);
    if (foundML == null) {
      return null;
    }
    existed = true;
    newPlacedTiles.add({
      row: arrowProperty.row,
      col: arrowProperty.col,
      // Specifically designate it as a blanked letter.
      letter: makeBlank(foundML),
    });
    newUnplacedTiles[blankIdx] = EmptyMachineLetter;
  } else {
    // check if the key is in the unplaced tiles.
    for (let i = 0; i < unplacedTiles.length; i++) {
      const ml = getMachineLetterFor(normalizedKey, alphabet);

      if (ml != null) {
        // Only use the blank in one of two situations:
        // - the original letter was uppercase (typed with a Shift)
        // - last-case scenario (all tiles have been scanned first)

        newPlacedTiles.add({
          row: arrowProperty.row,
          col: arrowProperty.col,
          letter: ml,
        });

        newUnplacedTiles[i] = EmptyMachineLetter;
        existed = true;
        break;
      }
    }
  }
  if (!existed) {
    // tile did not exist on rack. Check if there's a blank we can use.
    if (blankIdx !== -1) {
      const ml = getMachineLetterFor(normalizedKey, alphabet);
      if (ml != null) {
        newPlacedTiles.add({
          row: arrowProperty.row,
          col: arrowProperty.col,
          letter: makeBlank(ml),
        });

        newUnplacedTiles[blankIdx] = EmptyMachineLetter;
      } else {
        return null;
      }
    } else {
      // Can't place this tile at all.
      return null;
    }
  }

  return {
    newArrow: {
      row: newrow,
      col: newcol,
      horizontal: arrowProperty.horizontal,
      show: true,
    },
    newPlacedTiles,
    newDisplayedRack: newUnplacedTiles,
    playScore: calculateTemporaryScore(newPlacedTiles, board, alphabet),
  };
};

// Insert a MachineLetter into unplacedTiles at the preferred position.
// Remove a nearby gap if any to keep length the same.
// Assume 0 <= rackIndex <= unplacedTiles.length.
export const stableInsertRack = (
  unplacedTiles: Array<MachineLetter>,
  rackIndex: number,
  letter?: MachineLetter
): Array<MachineLetter> => {
  if (letter == null) {
    return unplacedTiles;
  }
  let newUnplacedTilesLeft = unplacedTiles.slice(0, rackIndex);
  let newUnplacedTilesRight = unplacedTiles.slice(rackIndex);
  let emptyIndexLeft = newUnplacedTilesLeft.lastIndexOf(EmptyMachineLetter);
  let emptyIndexRight = newUnplacedTilesRight.indexOf(EmptyMachineLetter);
  if (emptyIndexLeft >= 0 && emptyIndexRight >= 0) {
    // Determine which gap to recover.
    // Right has an advantage because it starts from 0, Left starts from 1.
    if (newUnplacedTilesLeft.length - emptyIndexLeft < emptyIndexRight) {
      // Left wins anyway.
      emptyIndexRight = -1;
    } else {
      emptyIndexLeft = -1;
    }
  }
  if (emptyIndexLeft >= 0) {
    newUnplacedTilesLeft = newUnplacedTilesLeft
      .slice(0, emptyIndexLeft)
      .concat(newUnplacedTilesLeft.slice(emptyIndexLeft + 1));
    // Keep Left's length the same if possible.
    if (newUnplacedTilesRight.length > 0) {
      newUnplacedTilesLeft = newUnplacedTilesLeft.concat(
        newUnplacedTilesRight[0]
      );
      newUnplacedTilesRight = newUnplacedTilesRight.slice(1);
    }
  } else if (emptyIndexRight >= 0) {
    newUnplacedTilesRight = newUnplacedTilesRight
      .slice(0, emptyIndexRight)
      .concat(newUnplacedTilesRight.slice(emptyIndexRight + 1));
  }
  // It is also possible there are no gaps left, just insert in that case.
  return newUnplacedTilesLeft.concat(letter).concat(newUnplacedTilesRight);
};

export const returnTileToRack = (
  board: Board,
  unplacedTiles: Array<MachineLetter>,
  currentlyPlacedTiles: Set<EphemeralTile>,
  alphabet: Alphabet,
  rackIndex = -1,
  tileIndex = -1
): PlacementHandlerReturn | null => {
  // Create an ephemeral tile map with unique keys.
  const ephTileMap: { [tileIdx: number]: EphemeralTile } = {};
  currentlyPlacedTiles.forEach((t) => {
    ephTileMap[uniqueTileIdx(t.row, t.col)] = t;
  });
  const newPlacedTiles = new Set(currentlyPlacedTiles);
  let letter;
  if (tileIndex > -1) {
    letter = ephTileMap[tileIndex] ? ephTileMap[tileIndex].letter : undefined;
    if (letter != null && isDesignatedBlankMachineLetter(letter)) {
      letter = BlankMachineLetter;
    }
    newPlacedTiles.delete(ephTileMap[tileIndex]);
  } else {
    return null;
  }
  return {
    newPlacedTiles,
    newDisplayedRack: stableInsertRack(unplacedTiles, rackIndex, letter),
    playScore: calculateTemporaryScore(newPlacedTiles, board, alphabet),
  };
};

export const handleDroppedTile = (
  row: number,
  col: number,
  board: Board,
  unplacedTiles: Array<MachineLetter>,
  currentlyPlacedTiles: Set<EphemeralTile>,
  rackIndex: number,
  tileIndex: number,
  alphabet: Alphabet
): PlacementHandlerReturn | null => {
  // Create an ephemeral tile map with unique keys.
  const ephTileMap: { [tileIdx: number]: EphemeralTile } = {};
  let targetSquare: EphemeralTile | undefined;
  let sourceSquare: EphemeralTile | undefined;
  currentlyPlacedTiles.forEach((t) => {
    if (t.row === row && t.col === col) {
      targetSquare = t;
    } else {
      const thisTileIndex = uniqueTileIdx(t.row, t.col);
      if (!(rackIndex >= 0) && tileIndex === thisTileIndex) {
        sourceSquare = t;
      } else {
        ephTileMap[thisTileIndex] = t;
      }
    }
  });
  let newUnplacedTiles = unplacedTiles;
  const newPlacedTiles = new Set(Object.values(ephTileMap));
  let letter;
  if (rackIndex >= 0) {
    letter = unplacedTiles[rackIndex];
    if (targetSquare) {
      newUnplacedTiles = unplacedTiles
        .slice(0, rackIndex)
        .concat(
          isDesignatedBlankMachineLetter(targetSquare.letter)
            ? BlankMachineLetter
            : targetSquare.letter
        )
        .concat(unplacedTiles.slice(rackIndex + 1));
    } else {
      newUnplacedTiles = unplacedTiles
        .slice(0, rackIndex)
        .concat(EmptyMachineLetter)
        .concat(unplacedTiles.slice(rackIndex + 1));
    }
  } else {
    if (!sourceSquare) {
      // Dragged tile no longer at source, likely because opponent moved there.
      // Also the case if dragging to the same spot.
      return null;
    }
    letter = sourceSquare.letter;
    if (targetSquare) {
      // Behold this prestidigitation!
      newPlacedTiles.add({
        ...sourceSquare,
        letter: targetSquare.letter,
      });
    }
  }

  if (isDesignatedBlankMachineLetter(letter)) {
    // reset moved blanks
    letter = BlankMachineLetter;
  }

  newPlacedTiles.add({
    row: row,
    col: col,
    letter: letter,
  });

  return {
    newPlacedTiles,
    newDisplayedRack: newUnplacedTiles,
    playScore: calculateTemporaryScore(newPlacedTiles, board, alphabet),
    isUndesignated: letter === BlankMachineLetter,
  };
};

export const designateBlank = (
  board: Board,
  currentlyPlacedTiles: Set<EphemeralTile>,
  displayedRack: Array<MachineLetter>,
  letter: MachineLetter,
  alphabet: Alphabet
): PlacementHandlerReturn | null => {
  // Find the undesignated blank
  const newPlacedTiles = new Set(currentlyPlacedTiles);
  newPlacedTiles.forEach((t) => {
    if (t.letter === BlankMachineLetter) {
      t.letter = makeBlank(letter);
    }
  });
  return {
    newPlacedTiles,
    newDisplayedRack: displayedRack,
    playScore: calculateTemporaryScore(newPlacedTiles, board, alphabet),
  };
};

// Return an array of words on the board.
// If tiles are included, results will be limited to ones formed by them
export const getWordsFormed = (
  board: Board,
  tiles: Set<EphemeralTile> | undefined,
  alphabet: Alphabet
): string[] => {
  const tentativeTiles = tiles ? Array.from(tiles.values()) : [];
  const tilesLayout = board.letters;
  tentativeTiles.sort((a, b) => {
    if (a.col === b.col) {
      return a.row - b.row;
    }
    return a.col - b.col;
  });
  const boardSize = board.gridLayout.length;
  const tentativeBoard = Array.from(new Array(boardSize), (_, y) =>
    Array.from(new Array(boardSize), (_, x) => tilesLayout[y * boardSize + x])
  );
  const newTilesPlaced = Array.from(new Array(boardSize), (_, y) =>
    Array.from(new Array(boardSize), (_, x) => EmptyBoardSpaceMachineLetter)
  );
  for (const { row, col, letter } of tentativeTiles) {
    tentativeBoard[row][col] = letter;
    newTilesPlaced[row][col] = letter;
  }
  const wordsFormed = new Set<string>();
  for (let y = 0; y < boardSize; y += 1) {
    for (let x = 0; x < boardSize; x += 1) {
      let usesTentativeTile = false;
      let sh = '';
      {
        let i = x;
        while (
          i > 0 &&
          tentativeBoard[y][i - 1] !== EmptyBoardSpaceMachineLetter
        )
          --i;
        for (
          ;
          i < boardSize &&
          tentativeBoard[y][i] !== EmptyBoardSpaceMachineLetter;
          ++i
        ) {
          if (newTilesPlaced[y][i] !== EmptyBoardSpaceMachineLetter) {
            usesTentativeTile = true;
          }
          sh += machineLetterToRune(tentativeBoard[y][i], alphabet);
        }
      }
      //Ignore if it's not a new word and new tiles were placed.
      if (tiles && !usesTentativeTile) {
        sh = '';
      }
      usesTentativeTile = false;
      let sv = '';
      {
        let i = y;
        while (
          i > 0 &&
          tentativeBoard[i - 1][x] !== EmptyBoardSpaceMachineLetter
        )
          --i;
        for (
          ;
          i < boardSize &&
          tentativeBoard[i][x] !== EmptyBoardSpaceMachineLetter;
          ++i
        ) {
          sv += machineLetterToRune(tentativeBoard[i][x], alphabet);
          if (newTilesPlaced[i][x] !== EmptyBoardSpaceMachineLetter) {
            usesTentativeTile = true;
          }
        }
      }
      if (tiles && !usesTentativeTile) {
        sv = '';
      }
      const tempWords = [sh, sv].filter((word) => word.length >= 2);
      tempWords.forEach(wordsFormed.add, wordsFormed);
    }
  }
  return Array.from(wordsFormed.values());
};
