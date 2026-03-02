export type Board3DData = {
  // 2D grid of display letters. Uppercase = normal, lowercase = blank, "" = empty.
  boardArray: string[][];
  // Bonus square layout — one string per row, each char is a bonus code.
  // Codes: '=' triple word, '-' double word, '"' triple letter, "'" double letter,
  //        '~' quad word, '^' quad letter, '*' starting square, ' ' no bonus.
  gridLayout: string[];
  boardDimension: number;
  // Current player's rack as display strings (e.g. ["A","B","CH",...])
  rack: string[];
  // Unseen tiles (bag + opponent rack): display-string → count
  remainingTiles: Record<string, number>;
  // Letter → point value (uppercase key, e.g. "A" → 1, "?" → 0)
  alphabetScores: Record<string, number>;
  player0Name: string;
  player1Name: string;
  player0Score: number;
  player1Score: number;
  // 0 = player0's turn, 1 = player1's turn
  playerOnTurn: number;
  // Human-readable description of the last move
  lastPlay: string;
  tileColor: string;
  boardColor: string;
};
