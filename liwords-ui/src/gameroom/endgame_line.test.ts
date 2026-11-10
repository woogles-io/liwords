import { expect, it } from "vitest";
import { AnalyzerMove } from "./analyzer";
import { lineEndsGame } from "./endgame_line";

// A move laying the given wolges tiles; 0 marks a square played through.
const play = (tiles: Array<number>): AnalyzerMove => ({
  jsonKey: "",
  displayMove: "",
  coordinates: "",
  leave: [],
  leaveWithGaps: [],
  score: 0,
  equity: 0,
  row: 0,
  col: 0,
  vertical: false,
  tiles,
  isExchange: false,
});

const pass: AnalyzerMove = { ...play([]), isExchange: true };

// The line that prompted this: 7 tiles against 5, cut off at the 7-ply search
// depth with the mover still holding a tile.
//   K7 N.  O7 F.ET  G9 T..N  D2 B...  4H .....d  (Pass)  3D .IF
const truncated = [
  play([14, 0]),
  play([6, 0, 5, 20]),
  play([20, 0, 0, 14]),
  play([2, 0, 0, 0]),
  play([0, 0, 0, 0, 0, -4]),
  pass,
  play([0, 9, 6]),
];

it("flags a line that stops with tiles on both racks", () => {
  expect(lineEndsGame(truncated, 7, 5)).toBe(false);
});

it("accepts a line where the mover plays out", () => {
  expect(lineEndsGame([...truncated, pass, play([0, 9])], 7, 5)).toBe(true);
});

it("accepts a line where the opponent plays out", () => {
  // Mover plays one tile, opponent goes out with both of theirs.
  expect(lineEndsGame([play([1]), play([2, 0, 3])], 3, 2)).toBe(true);
});

it("accepts a line that ends on two passes in a row", () => {
  expect(lineEndsGame([play([1]), pass, pass], 3, 2)).toBe(true);
});

it("does not end the game on passes that are not consecutive", () => {
  expect(lineEndsGame([pass, play([1]), pass], 3, 2)).toBe(false);
});
