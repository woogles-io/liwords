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

export const uniqueTileIdx = (row: number, col: number): number => {
  // Just a unique number to identify a row,col coordinate.
  return row * 100 + col;
};

export const isBlank = (letter: string): boolean => {
  if (letter.toLowerCase() === letter) {
    return true;
  }
  return false;
};
