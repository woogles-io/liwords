import {
  StandardEnglishAlphabet,
  runesToMachineWord,
} from '../../constants/alphabets';
import { CrosswordGameGridLayout } from '../../constants/board_layout';
import { EmptySpace, MachineLetter } from './common';

export type Tile = {
  row: number;
  col: number;
  ml: MachineLetter;
};

function blankLayout(gridlayout: string[]) {
  const layout = [];
  for (let i = 0; i < gridlayout.length * gridlayout[0].length; i++) {
    layout.push(0);
  }
  return Array.from(layout);
}

function setLetterAt(
  letters: Array<MachineLetter>,
  index: number,
  ml: MachineLetter
) {
  if (index > letters.length - 1) {
    return letters;
  }
  letters[index] = ml;
}

export class Board {
  letters: Array<MachineLetter>; // The letters on the board

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
          // assume english; this is only for tests!
          const temp = runesToMachineWord(letter, StandardEnglishAlphabet);
          setLetterAt(this.letters, row * this.dim + col, temp[0]);
        }
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
    setLetterAt(this.letters, t.row * this.dim + t.col, t.ml);
    this.isEmpty = false;
  }

  removeTile(t: Tile) {
    setLetterAt(this.letters, t.row * this.dim + t.col, 0);
    // don't know how else to check, annoyingly
    this.isEmpty = true;
    for (let i = 0; i < this.letters.length; i++) {
      if (this.letters[i] !== 0) {
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
