import { CrosswordGameGridLayout } from '../../constants/board_layout';
import { EmptySpace } from './common';

/* TODO: should be dependent on board dimensions in future.  */
export function blankLayout() {
  return new Array(225).fill(' ');
}

export class Board {
  private letters: Array<string>; // The letters on the board

  gridLayout: Array<string>; // the bonus squares.

  isEmpty: boolean;

  dim: number;

  constructor() {
    this.letters = blankLayout();
    this.isEmpty = true;
    this.gridLayout = CrosswordGameGridLayout;
    this.dim = this.gridLayout.length;
  }

  tilesLayout() {
    const layout = [];
    for (let j = 0; j < 15; j += 1) {
      // row by row
      const x = j * 15;
      const sl = this.letters.slice(x, x + 15);
      layout.push(sl.join(''));
    }
    return layout;
  }

  /** take in a 2D board array */
  setTileLayout(layout: Array<string>) {
    this.isEmpty = true;
    for (let row = 0; row < 15; row += 1) {
      for (let col = 0; col < 15; col += 1) {
        const letter = layout[row][col];
        if (letter !== EmptySpace) {
          this.isEmpty = false;
        }
        this.letters[row * 15 + col] = letter;
      }
    }
  }

  /**
   * Return the letter at the given row, col. Returns null if out of bounds.
   */
  letterAt(row: number, col: number) {
    if (row > this.dim - 1 || row < 0 || col > this.dim - 1 || col < 0) {
      return null;
    }
    return this.letters[row * 15 + col];
  }

  addLetter(row: number, col: number, letter: string) {
    this.letters[row * 15 + col] = letter;
    this.isEmpty = false;
  }

  removeLetter(row: number, col: number, letter: string) {
    this.letters[row * 15 + col] = ' ';
    // don't know how else to check, annoyingly
    this.isEmpty = true;
    for (let i = 0; i < 225; i++) {
      if (this.letters[i] !== ' ') {
        this.isEmpty = false;
        break;
      }
    }
  }

  deepCopy() {
    const newBoard = new Board();
    newBoard.letters = [...this.letters];
    newBoard.gridLayout = [...this.gridLayout];
    newBoard.isEmpty = this.isEmpty;
    newBoard.dim = this.dim;
    return newBoard;
  }
}
