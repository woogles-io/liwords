/** @fileoverview business logic for placing tiles on a board */

import {
  EphemeralTile,
  uniqueTileIdx,
  MachineLetter,
  makeBlank,
  EmptyBoardSpaceMachineLetter,
  EmptyRackSpaceMachineLetter,
} from "./common";
import { calculateTemporaryScore } from "./scoring";
import { Board } from "./board";
import {
  Alphabet,
  getMachineLetterForKey,
  machineLetterToRune,
} from "../../constants/alphabets";
import { isDesignatedBlankMachineLetter } from "./common";
import { BlankMachineLetter } from "./common";

const NormalizedBackspace = "BACKSPACE";
const NormalizedSpace = " ";

export type PlacementArrow = {
  row: number;
  col: number;
  horizontal: boolean;
  show: boolean;
};

export const nextArrowPropertyState = (
  props: PlacementArrow,
  row: number,
  col: number,
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
  alphabet: Alphabet,
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
      const emptyIndex = newUnplacedTiles.indexOf(EmptyRackSpaceMachineLetter);
      if (emptyIndex >= 0) {
        // There's an empty slot, so this tile came from the rack - return it
        newUnplacedTiles[emptyIndex] = letter;
      }
      // If no empty slot exists, the tile came from the pool in editor mode
      // Don't add it back to the rack
    }
  });

  return {
    newArrow: arrowProperty,
    newPlacedTiles,
    newDisplayedRack: newUnplacedTiles,
    playScore: calculateTemporaryScore(newPlacedTiles, board, alphabet),
  };
};

export const nextArrowStateAfterTilePlacement = (
  arrowProperty: PlacementArrow,
  ephTileMap: { [tileIdx: number]: EphemeralTile },
  increment: 1 | -1,
  board: Board,
) => {
  let { col, row } = arrowProperty;
  if (arrowProperty.horizontal) {
    do {
      col += increment;
    } while (
      col < board.dim &&
      col >= 0 &&
      (board.letterAt(row, col) !== EmptyBoardSpaceMachineLetter ||
        (increment === 1 && ephTileMap[uniqueTileIdx(row, col)] !== undefined))
    );
  } else {
    do {
      row += increment;
    } while (
      row < board.dim &&
      row >= 0 &&
      (board.letterAt(row, col) !== EmptyBoardSpaceMachineLetter ||
        (increment === 1 && ephTileMap[uniqueTileIdx(row, col)] !== undefined))
    );
  }
  return {
    row,
    col,
    horizontal: arrowProperty.horizontal,
    show: true,
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
  unplacedTiles: Array<MachineLetter>, // tiles currently still on rack
  currentlyPlacedTiles: Set<EphemeralTile>,
  alphabet: Alphabet,
  boardEditingMode?: boolean, // If true, allow placing tiles from bag when rack is empty
  pool?: { [ml: MachineLetter]: number }, // Tile bag/pool for editor mode validation
): KeypressHandlerReturn | null => {
  const normalizedKey = key.toUpperCase();

  const newPlacedTiles = new Set(currentlyPlacedTiles);

  if (
    !Object.prototype.hasOwnProperty.call(alphabet.letterMap, normalizedKey) &&
    !Object.prototype.hasOwnProperty.call(
      alphabet.shortcutMap,
      normalizedKey,
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

  // Create an ephemeral tile map with unique keys.
  const ephTileMap: { [tileIdx: number]: EphemeralTile } = {};
  currentlyPlacedTiles.forEach((t) => {
    ephTileMap[uniqueTileIdx(t.row, t.col)] = t;
  });

  let increment: 1 | -1 = 1;
  // Check the backspace and unplay any tiles if necessary.
  if (normalizedKey === NormalizedBackspace) {
    increment = -1;
  }

  const newUnplacedTiles = [...unplacedTiles];

  // First figure out where to put the arrow, no matter what.

  const newArrowProperty = nextArrowStateAfterTilePlacement(
    arrowProperty,
    ephTileMap,
    increment,
    board,
  );

  if (normalizedKey === NormalizedBackspace) {
    // Don't allow the arrow to go off-screen when backspacing.
    if (newArrowProperty.row < 0) {
      newArrowProperty.row = arrowProperty.row;
    }
    if (newArrowProperty.col < 0) {
      newArrowProperty.col = arrowProperty.col;
    }
    return handleTileDeletion(
      {
        row: newArrowProperty.row,
        col: newArrowProperty.col,
        horizontal: arrowProperty.horizontal,
        show: true,
      },
      unplacedTiles,
      currentlyPlacedTiles,
      board,
      alphabet,
    );
  }

  if (normalizedKey === NormalizedSpace) {
    if (newArrowProperty.row > board.dim - 1) {
      newArrowProperty.row = arrowProperty.row;
    }
    if (newArrowProperty.col > board.dim - 1) {
      newArrowProperty.col = arrowProperty.col;
    }
    return {
      newArrow: newArrowProperty,
      newPlacedTiles,
      newDisplayedRack: newUnplacedTiles,
      playScore: calculateTemporaryScore(newPlacedTiles, board, alphabet),
    };
  }

  const blankIdx = unplacedTiles.indexOf(BlankMachineLetter);
  let existed = false;
  const typedML = getMachineLetterForKey(normalizedKey, alphabet);
  const wantsBlank = normalizedKey === key; // User typed uppercase (Shift+letter)

  if (wantsBlank && typedML != null) {
    // User specifically requested a blank (by typing with Shift)
    if (blankIdx !== -1) {
      // There's a blank in the rack, use it
      newPlacedTiles.add({
        row: arrowProperty.row,
        col: arrowProperty.col,
        letter: makeBlank(typedML),
      });
      newUnplacedTiles[blankIdx] = EmptyRackSpaceMachineLetter;
      existed = true;
    } else if (boardEditingMode && pool) {
      // Editor mode: check if blank exists in the bag/pool
      const blankCount = pool[BlankMachineLetter] || 0;
      if (blankCount > 0) {
        // Blank is available in the bag, allow placing it
        newPlacedTiles.add({
          row: arrowProperty.row,
          col: arrowProperty.col,
          letter: makeBlank(typedML),
        });
        existed = true;
      }
    }
  } else {
    // Not requesting a blank, check if the key is in the unplaced tiles
    for (let i = 0; i < unplacedTiles.length; i++) {
      if (unplacedTiles[i] === typedML) {
        newPlacedTiles.add({
          row: arrowProperty.row,
          col: arrowProperty.col,
          letter: typedML,
        });
        newUnplacedTiles[i] = EmptyRackSpaceMachineLetter;
        existed = true;
        break;
      }
    }
  }

  if (!existed) {
    // Tile did not exist on rack. Try alternatives.
    if (blankIdx !== -1 && typedML != null) {
      // Use blank as fallback if available
      newPlacedTiles.add({
        row: arrowProperty.row,
        col: arrowProperty.col,
        letter: makeBlank(typedML),
      });
      newUnplacedTiles[blankIdx] = EmptyRackSpaceMachineLetter;
    } else if (boardEditingMode && pool && typedML != null) {
      // Editor mode: check if tile exists in the bag/pool
      const tileCount = pool[typedML] || 0;
      if (tileCount > 0) {
        // Tile is available in the bag, allow placing it
        newPlacedTiles.add({
          row: arrowProperty.row,
          col: arrowProperty.col,
          letter: typedML,
        });
      } else {
        // Tile not in bag either
        return null;
      }
    } else {
      // Can't place this tile at all
      return null;
    }
  }

  return {
    newArrow: newArrowProperty,
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
  letter?: MachineLetter,
): Array<MachineLetter> => {
  if (letter == null) {
    return unplacedTiles;
  }
  let newUnplacedTilesLeft = unplacedTiles.slice(0, rackIndex);
  let newUnplacedTilesRight = unplacedTiles.slice(rackIndex);
  let emptyIndexLeft = newUnplacedTilesLeft.lastIndexOf(
    EmptyRackSpaceMachineLetter,
  );
  let emptyIndexRight = newUnplacedTilesRight.indexOf(
    EmptyRackSpaceMachineLetter,
  );
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
        newUnplacedTilesRight[0],
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
  tileIndex = -1,
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
  alphabet: Alphabet,
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
  let newUnplacedTiles = [...unplacedTiles];
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
            : targetSquare.letter,
        )
        .concat(unplacedTiles.slice(rackIndex + 1));
    } else {
      newUnplacedTiles = unplacedTiles
        .slice(0, rackIndex)
        .concat(EmptyRackSpaceMachineLetter)
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
  alphabet: Alphabet,
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
  alphabet: Alphabet,
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
    Array.from(new Array(boardSize), (_, x) => tilesLayout[y * boardSize + x]),
  );
  const newTilesPlaced = Array.from(new Array(boardSize), (_, y) =>
    Array.from(new Array(boardSize), (_, x) => EmptyBoardSpaceMachineLetter),
  );
  for (const { row, col, letter } of tentativeTiles) {
    tentativeBoard[row][col] = letter;
    newTilesPlaced[row][col] = letter;
  }
  const wordsFormed = new Set<string>();
  for (let y = 0; y < boardSize; y += 1) {
    for (let x = 0; x < boardSize; x += 1) {
      let usesTentativeTile = false;
      let sh = "";
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
        sh = "";
      }
      usesTentativeTile = false;
      let sv = "";
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
        sv = "";
      }
      const tempWords = [sh, sv].filter((word) => word.length >= 2);
      tempWords.forEach(wordsFormed.add, wordsFormed);
    }
  }
  return Array.from(wordsFormed.values());
};
