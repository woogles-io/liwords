export const EmptySpace = ' ';

export type EphemeralTile = {
  // ephemeron <3 you are missed
  row: number;
  col: number;
  letter: string; // lowercase for blank
};

export enum Direction {
  Horizontal,
  Vertical,
}

/**
 * Looks up the tile in the given row, col. If this is out of bounds,
 * return null.
 * @param row
 * @param col
 * @param boardTiles
 */
export const safeBoardLookup = (
  row: number,
  col: number,
  boardTiles: Array<string>
): string | null => {
  if (
    row > boardTiles.length - 1 ||
    row < 0 ||
    col > boardTiles[0].length - 1 ||
    col < 0
  ) {
    return null;
  }

  return boardTiles[row][col];
};

export const isBlank = (letter: string): boolean => {
  if (letter.toLowerCase() === letter) {
    return true;
  }
  return false;
};
