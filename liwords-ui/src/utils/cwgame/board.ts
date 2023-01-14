import { CrosswordGameGridLayout } from '../../constants/board_layout';
import { EmptySpace } from './common';

export type Tile = {
  row: number;
  col: number;
  rune: string; // why doesn't Javascript have runes.
};

export function blankLayout(gridlayout: string[]) {
  return repeatChar(gridlayout.length * gridlayout.length, EmptySpace);
}

function repeatChar(count: number, ch: string) {
  let txt = '';
  for (let i = 0; i < count; i++) {
    txt += ch;
  }
  return txt;
}

export function setCharAt(str: string, index: number, chr: string) {
  if (index > str.length - 1) {
    return str;
  }
  return str.substr(0, index) + chr + str.substr(index + 1);
}

export class Board {
  letters: string; // The letters on the board

  gridLayout: Array<string>; // the bonus squares.

  isEmpty: boolean;

  dim: number;

  constructor(gridLayout = CrosswordGameGridLayout) {
    this.letters = blankLayout(gridLayout);
    this.isEmpty = true;
    this.gridLayout = gridLayout;
    this.dim = this.gridLayout.length;
  }

  /** take in a 2D board array. Used for tests and board previews only,
   * do not use for interactive boards */
  setTileLayout(layout: Array<string>) {
    this.isEmpty = true;
    for (let row = 0; row < this.dim; row += 1) {
      for (let col = 0; col < this.dim; col += 1) {
        const letter = layout[row][col];
        if (letter !== EmptySpace) {
          this.isEmpty = false;
        }
        this.letters = setCharAt(this.letters, row * this.dim + col, letter);
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
    return this.letters[row * this.dim + col];
  }

  addTile(t: Tile) {
    this.letters = setCharAt(this.letters, t.row * this.dim + t.col, t.rune);
    this.isEmpty = false;
  }

  removeTile(t: Tile) {
    this.letters = setCharAt(
      this.letters,
      t.row * this.dim + t.col,
      EmptySpace
    );
    // don't know how else to check, annoyingly
    this.isEmpty = true;
    for (let i = 0; i < this.letters.length; i++) {
      if (this.letters[i] !== EmptySpace) {
        this.isEmpty = false;
        break;
      }
    }
  }

  deepCopy() {
    const newBoard = new Board();
    newBoard.letters = this.letters;
    newBoard.gridLayout = [...this.gridLayout];
    newBoard.isEmpty = this.isEmpty;
    newBoard.dim = this.dim;
    return newBoard;
  }
}
