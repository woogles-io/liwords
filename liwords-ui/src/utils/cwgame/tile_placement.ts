/** @fileoverview business logic for placing tiles on a board */
import { EnglishCrosswordGameDistribution } from '../../constants/tile_distributions';

import {
  EphemeralTile,
  EmptySpace,
  isBlank,
  uniqueTileIdx,
  Blank,
} from './common';
import { calculateTemporaryScore } from './scoring';
import { Board } from './board';

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
  newDisplayedRack: string;
  playScore: number | undefined; // undefined for illegal plays
  isUndesignated?: boolean;
};

interface KeypressHandlerReturn extends PlacementHandlerReturn {
  newArrow: PlacementArrow;
}

const handleTileDeletion = (
  arrowProperty: PlacementArrow,
  unplacedTiles: string, // tiles currently still on rack
  currentlyPlacedTiles: Set<EphemeralTile>,
  board: Board
): KeypressHandlerReturn => {
  // Remove any tiles.
  let newUnplacedTiles = unplacedTiles;
  const newPlacedTiles = new Set(currentlyPlacedTiles);

  currentlyPlacedTiles.forEach((t) => {
    if (t.col === arrowProperty.col && t.row === arrowProperty.row) {
      // Remove this tile.
      newPlacedTiles.delete(t);
      // can't exit early but w/e, this is fast
      let { letter } = t;
      if (isBlank(letter)) {
        // unassign the blank
        letter = Blank;
      }
      const emptyIndex = newUnplacedTiles.indexOf(EmptySpace);
      if (emptyIndex >= 0) {
        newUnplacedTiles =
          newUnplacedTiles.substring(0, emptyIndex) +
          letter +
          newUnplacedTiles.substring(emptyIndex + 1);
      } else {
        newUnplacedTiles += letter;
      }
    }
  });

  return {
    newArrow: arrowProperty,
    newPlacedTiles,
    newDisplayedRack: newUnplacedTiles,
    playScore: calculateTemporaryScore(newPlacedTiles, board),
  };
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
  unplacedTiles: string, // tiles currently still on rack
  currentlyPlacedTiles: Set<EphemeralTile>
): KeypressHandlerReturn | null => {
  const normalizedKey = key.toUpperCase();

  const newPlacedTiles = new Set(currentlyPlacedTiles);

  // Create an ephemeral tile map with unique keys.
  const ephTileMap: { [tileIdx: number]: EphemeralTile } = {};
  currentlyPlacedTiles.forEach((t) => {
    ephTileMap[uniqueTileIdx(t.row, t.col)] = t;
  });

  if (
    !Object.prototype.hasOwnProperty.call(
      EnglishCrosswordGameDistribution,
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
  let newUnplacedTiles = unplacedTiles;

  // First figure out where to put the arrow, no matter what.
  if (arrowProperty.horizontal) {
    do {
      newcol += increment;
    } while (
      newcol < board.dim &&
      newcol >= 0 &&
      (board.letterAt(newrow, newcol) !== EmptySpace ||
        (increment === 1 &&
          ephTileMap[uniqueTileIdx(newrow, newcol)] !== undefined))
    );
  } else {
    do {
      newrow += increment;
    } while (
      newrow < board.dim &&
      newrow >= 0 &&
      (board.letterAt(newrow, newcol) !== EmptySpace ||
        (increment === 1 &&
          ephTileMap[uniqueTileIdx(newrow, newcol)] !== undefined))
    );
  }

  if (normalizedKey === NormalizedBackspace) {
    // Don't allow the arrow to go off-screen when backspacing.
    if (newrow < 0) {
      newrow = 0;
    }
    if (newcol < 0) {
      newcol = 0;
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
      board
    );
  }

  if (normalizedKey === NormalizedSpace) {
    if (newrow > board.dim - 1) {
      newrow = board.dim - 1;
    }
    if (newcol > board.dim - 1) {
      newcol = board.dim - 1;
    }
    if (ephTileMap[uniqueTileIdx(newrow, newcol)] !== undefined) {
      // Just after placing a tile at the right or bottom edge,
      // pressing Space should not place the newArrow on the tile.
      return null;
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
      playScore: calculateTemporaryScore(newPlacedTiles, board),
    };
  }

  const blankIdx = unplacedTiles.indexOf(Blank);
  let existed = false;

  if (blankIdx !== -1 && normalizedKey === key) {
    // If there is a blank, and the user specifically requested to use it
    // (by typing the letter with a Shift)
    existed = true;
    newPlacedTiles.add({
      row: arrowProperty.row,
      col: arrowProperty.col,
      // Specifically designate it as a blanked letter by lower-casing it.
      letter: normalizedKey.toLowerCase(),
    });
    newUnplacedTiles =
      unplacedTiles.substring(0, blankIdx) +
      EmptySpace +
      unplacedTiles.substring(blankIdx + 1);
  } else {
    // check if the key is in the unplaced tiles.
    for (let i = 0; i < unplacedTiles.length; i++) {
      if (unplacedTiles[i] === normalizedKey) {
        // Only use the blank in one of two situations:
        // - the original letter was uppercase (typed with a Shift)
        // - last-case scenario (all tiles have been scanned first)

        newPlacedTiles.add({
          row: arrowProperty.row,
          col: arrowProperty.col,
          letter: normalizedKey,
        });

        newUnplacedTiles =
          unplacedTiles.substring(0, i) +
          EmptySpace +
          unplacedTiles.substring(i + 1);
        existed = true;
        break;
      }
    }
  }
  if (!existed) {
    // tile did not exist on rack. Check if there's a blank we can use.
    if (blankIdx !== -1) {
      newPlacedTiles.add({
        row: arrowProperty.row,
        col: arrowProperty.col,
        letter: normalizedKey.toLowerCase(),
      });

      newUnplacedTiles =
        unplacedTiles.substring(0, blankIdx) +
        EmptySpace +
        unplacedTiles.substring(blankIdx + 1);
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
    playScore: calculateTemporaryScore(newPlacedTiles, board),
  };
};

// Insert a rune to unplacedTiles at the preferred position.
// Remove a nearby gap if any to keep length the same.
// Assume 0 <= rackIndex <= unplacedTiles.length and rune.length === 1.
export const stableInsertRack = (
  unplacedTiles: string,
  rackIndex: number,
  rune: string
): string => {
  let newUnplacedTilesLeft = unplacedTiles.substring(0, rackIndex);
  let newUnplacedTilesRight = unplacedTiles.substring(rackIndex);
  let emptyIndexLeft = newUnplacedTilesLeft.lastIndexOf(EmptySpace);
  let emptyIndexRight = newUnplacedTilesRight.indexOf(EmptySpace);
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
    newUnplacedTilesLeft =
      newUnplacedTilesLeft.substring(0, emptyIndexLeft) +
      newUnplacedTilesLeft.substring(emptyIndexLeft + 1);
    // Keep Left's length the same if possible.
    if (newUnplacedTilesRight.length > 0) {
      newUnplacedTilesLeft += newUnplacedTilesRight[0];
      newUnplacedTilesRight = newUnplacedTilesRight.substring(1);
    }
  } else if (emptyIndexRight >= 0) {
    newUnplacedTilesRight =
      newUnplacedTilesRight.substring(0, emptyIndexRight) +
      newUnplacedTilesRight.substring(emptyIndexRight + 1);
  }
  // It is also possible there are no gaps left, just insert in that case.
  return newUnplacedTilesLeft + rune + newUnplacedTilesRight;
};

export const returnTileToRack = (
  board: Board,
  unplacedTiles: string,
  currentlyPlacedTiles: Set<EphemeralTile>,
  rackIndex: number = -1,
  tileIndex: number = -1
): PlacementHandlerReturn | null => {
  // Create an ephemeral tile map with unique keys.
  const ephTileMap: { [tileIdx: number]: EphemeralTile } = {};
  currentlyPlacedTiles.forEach((t) => {
    ephTileMap[uniqueTileIdx(t.row, t.col)] = t;
  });
  const newPlacedTiles = new Set(currentlyPlacedTiles);
  let rune;
  if (tileIndex > -1) {
    rune = ephTileMap[tileIndex] ? ephTileMap[tileIndex].letter : '';
    // Reset blank
    if (isBlank(rune)) {
      rune = Blank;
    }
    newPlacedTiles.delete(ephTileMap[tileIndex]);
  } else {
    return null;
  }
  return {
    newPlacedTiles,
    newDisplayedRack: stableInsertRack(unplacedTiles, rackIndex, rune),
    playScore: calculateTemporaryScore(newPlacedTiles, board),
  };
};

export const handleDroppedTile = (
  row: number,
  col: number,
  board: Board,
  unplacedTiles: string,
  currentlyPlacedTiles: Set<EphemeralTile>,
  rackIndex: number,
  tileIndex: number
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
  let rune;
  if (rackIndex >= 0) {
    rune = unplacedTiles[rackIndex];
    if (targetSquare) {
      newUnplacedTiles =
        unplacedTiles.substring(0, rackIndex) +
        (isBlank(targetSquare.letter) ? Blank : targetSquare.letter) +
        unplacedTiles.substring(rackIndex + 1);
    } else {
      newUnplacedTiles =
        unplacedTiles.substring(0, rackIndex) +
        EmptySpace +
        unplacedTiles.substring(rackIndex + 1);
    }
  } else {
    if (!sourceSquare) {
      // Dragged tile no longer at source, likely because opponent moved there.
      // Also the case if dragging to the same spot.
      return null;
    }
    rune = sourceSquare.letter;
    if (targetSquare) {
      // Behold this prestidigitation!
      newPlacedTiles.add({
        ...sourceSquare,
        letter: targetSquare.letter,
      });
    }
  }

  if (isBlank(rune)) {
    // reset moved blanks
    rune = Blank;
  }

  newPlacedTiles.add({
    row: row,
    col: col,
    letter: rune,
  });

  return {
    newPlacedTiles,
    newDisplayedRack: newUnplacedTiles,
    playScore: calculateTemporaryScore(newPlacedTiles, board),
    isUndesignated: rune === Blank,
  };
};

export const designateBlank = (
  board: Board,
  currentlyPlacedTiles: Set<EphemeralTile>,
  displayedRack: string,
  rune: string
): PlacementHandlerReturn | null => {
  // Find the undesignated blank
  const newPlacedTiles = new Set(currentlyPlacedTiles);
  newPlacedTiles.forEach((t) => {
    if (t.letter === Blank) {
      t.letter = rune.toLowerCase();
    }
  });
  return {
    newPlacedTiles,
    newDisplayedRack: displayedRack,
    playScore: calculateTemporaryScore(newPlacedTiles, board),
  };
};
