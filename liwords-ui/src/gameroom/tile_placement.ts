/** @fileoverview business logic for placing tiles on a board */
import { Blank } from './tile';
import { EnglishCrosswordGameDistribution } from '../constants/tile_distributions';

const EmptySpace = ' ';
const NormalizedBackspace = 'BACKSPACE';

export type PlacementArrow = {
  row: number;
  col: number;
  horizontal: boolean;
  show: boolean;
};

export type EphemeralTile = {
  // ephemeron <3 you are missed
  row: number;
  col: number;
  letter: string; // lowercase for blank
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

type KeypressHandlerReturn = {
  newArrow: PlacementArrow;
  justPlayedTile: EphemeralTile | null;
  justUnplayedTile: EphemeralTile | null;
  newDisplayedRack: string;
  playScore: number;
};

const handleTileDeletion = (
  arrowProperty: PlacementArrow,
  unplacedTiles: string, // tiles currently still on rack
  currentlyPlacedTiles: Set<EphemeralTile>
): KeypressHandlerReturn => {
  // Remove any tiles.
  let justUnplayedTile = null;
  let newUnplacedTiles = unplacedTiles;

  currentlyPlacedTiles.forEach((t) => {
    if (t.col === arrowProperty.col && t.row === arrowProperty.row) {
      // Remove this tile.
      justUnplayedTile = t;
      // can't exit early but w/e, this is fast
      let { letter } = t;
      if (letter.toLowerCase() === letter) {
        // unassign the blank
        letter = Blank;
      }
      newUnplacedTiles += letter;
    }
  });

  return {
    newArrow: arrowProperty,
    justPlayedTile: null,
    justUnplayedTile,
    newDisplayedRack: newUnplacedTiles,
    playScore: 0,
  };
};

/**
 * This is a fairly important function for placing tiles with the keyboard.
 * It handles a keypress, and takes in the current direction of the placement
 * arrow, as well as the board tiles, etc.
 * @param arrowProperty
 * @param boardTiles
 * @param key
 * @param unplacedTiles
 * @param currentlyPlacedTiles
 */
export const handleKeyPress = (
  arrowProperty: PlacementArrow,
  boardTiles: Array<string>,
  key: string,
  unplacedTiles: string, // tiles currently still on rack
  currentlyPlacedTiles: Set<EphemeralTile>
): KeypressHandlerReturn | null => {
  const normalizedKey = key.toUpperCase();

  if (
    !Object.prototype.hasOwnProperty.call(
      EnglishCrosswordGameDistribution,
      normalizedKey
    ) &&
    normalizedKey !== NormalizedBackspace
  ) {
    // Return with no changes.
    return null;
  }

  // Make sure we're not trying to type off the edge of the board.
  if (
    arrowProperty.row >= boardTiles.length ||
    arrowProperty.col >= boardTiles[0].length ||
    arrowProperty.row < 0 ||
    arrowProperty.col < 0
  ) {
    if (normalizedKey !== NormalizedBackspace) {
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
  let justPlayedTile = null;

  // First figure out where to put the arrow, no matter what.
  if (arrowProperty.horizontal) {
    do {
      newcol += increment;
    } while (
      newcol < boardTiles[newrow].length &&
      newcol >= 0 &&
      boardTiles[newrow][newcol] !== EmptySpace
    );
  } else {
    do {
      newrow += increment;
    } while (
      newrow < boardTiles.length &&
      newrow >= 0 &&
      boardTiles[newrow][newcol] !== EmptySpace
    );
  }

  if (normalizedKey === NormalizedBackspace) {
    return handleTileDeletion(
      {
        row: newrow,
        col: newcol,
        horizontal: arrowProperty.horizontal,
        show: true,
      },
      unplacedTiles,
      currentlyPlacedTiles
    );
  }

  const blankIdx = unplacedTiles.indexOf(Blank);
  let existed = false;

  if (blankIdx !== -1 && normalizedKey === key) {
    // If there is a blank, and the user specifically requested to use it
    // (by typing the letter with a Shift)
    existed = true;
    justPlayedTile = {
      row: arrowProperty.row,
      col: arrowProperty.col,
      // Specifically designate it as a blanked letter by lower-casing it.
      letter: normalizedKey.toLowerCase(),
    };
    newUnplacedTiles =
      unplacedTiles.substring(0, blankIdx) +
      unplacedTiles.substring(blankIdx + 1);
  } else {
    // check if the key is in the unplaced tiles.
    for (let i = 0; i < unplacedTiles.length; i++) {
      if (unplacedTiles[i] === normalizedKey) {
        // Only use the blank in one of two situations:
        // - the original letter was uppercase (typed with a Shift)
        // - last-case scenario (all tiles have been scanned first)

        justPlayedTile = {
          row: arrowProperty.row,
          col: arrowProperty.col,
          letter: normalizedKey,
        };

        newUnplacedTiles =
          unplacedTiles.substring(0, i) + unplacedTiles.substring(i + 1);
        existed = true;
        break;
      }
    }
  }
  if (!existed) {
    // tile did not exist on rack. Check if there's a blank we can use.
    if (blankIdx !== -1) {
      justPlayedTile = {
        row: arrowProperty.row,
        col: arrowProperty.col,
        letter: normalizedKey.toLowerCase(),
      };

      newUnplacedTiles =
        unplacedTiles.substring(0, blankIdx) +
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
    justPlayedTile,
    justUnplayedTile: null,
    newDisplayedRack: newUnplacedTiles,
    playScore: 0,
  };
};
