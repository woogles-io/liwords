import { FullResetInputs, needsFullReset } from "./board_panel_utils";
import { EphemeralTile, MachineLetter } from "../utils/cwgame/common";

const dim = 15;

const emptyBoard = () => new Array<MachineLetter>(dim * dim).fill(0);

// Place a word on a copy of the given board, going across.
const withWord = (
  letters: Array<MachineLetter>,
  row: number,
  col: number,
  word: string,
) => {
  const newLetters = [...letters];
  for (let i = 0; i < word.length; ++i) {
    newLetters[row * dim + col + i] = word.charCodeAt(i) - 64;
  }
  return newLetters;
};

// CAT tentatively placed at the star, ESIN still on the rack.
const placedCat = () =>
  new Set<EphemeralTile>([
    { row: 7, col: 7, letter: 3 },
    { row: 7, col: 8, letter: 1 },
    { row: 7, col: 9, letter: 20 },
  ]);

const inputs = (overrides: Partial<FullResetInputs> = {}): FullResetInputs => ({
  lastLetters: emptyBoard(),
  letters: emptyBoard(),
  dim,
  currentRack: [3, 1, 20, 5, 19, 9, 14],
  displayedRack: [5, 19, 9, 14],
  placedTiles: placedCat(),
  arrowProperties: { row: 0, col: 0, show: false },
  isMyTurn: true,
  isExamining: false,
  puzzleMode: false,
  boardEditingMode: false,
  numEventsChanged: false,
  justCommitted: false,
  ...overrides,
});

it("resets on first load", () => {
  expect(needsFullReset(inputs({ lastLetters: undefined }))).toBe(true);
});

it("resets when the tiles do not add up to the rack", () => {
  expect(needsFullReset(inputs({ currentRack: [1, 2, 3, 4, 5, 6, 7] }))).toBe(
    true,
  );
});

it("resets while examining", () => {
  expect(needsFullReset(inputs({ isExamining: true }))).toBe(true);
});

// A GameHistoryRefresher for a clock change adds no events and leaves the
// board alone, so a premove placed while off turn must survive it.
it("keeps placed tiles when clock time is added off turn", () => {
  expect(needsFullReset(inputs({ isMyTurn: false }))).toBe(false);
});

// Our play and the challenge that took it off can arrive in the same packet,
// leaving the board and the rack exactly as they were before we played.
it("resets when our play is challenged off without the board ever changing", () => {
  expect(
    needsFullReset(
      inputs({ isMyTurn: false, numEventsChanged: true, justCommitted: true }),
    ),
  ).toBe(true);
});

it("resets when we are off turn because the game gained events", () => {
  expect(
    needsFullReset(inputs({ isMyTurn: false, numEventsChanged: true })),
  ).toBe(true);
});

// Same packet again, this time also carrying the opponent's reply, so we are
// back on turn with our committed tiles still sitting there.
it("resets when the game moves on from a move we committed", () => {
  expect(
    needsFullReset(inputs({ numEventsChanged: true, justCommitted: true })),
  ).toBe(true);
});

// The premove backup exists precisely to survive this, so leave it be.
it("keeps a premove when a play elsewhere is challenged off", () => {
  expect(
    needsFullReset(
      inputs({
        lastLetters: withWord(emptyBoard(), 0, 0, "QIS"),
        numEventsChanged: true,
      }),
    ),
  ).toBe(false);
});

it("keeps a premove the opponent's play does not touch", () => {
  expect(
    needsFullReset(
      inputs({
        letters: withWord(emptyBoard(), 0, 0, "QIS"),
        numEventsChanged: true,
      }),
    ),
  ).toBe(false);
});

it("resets a premove the opponent's play hooks onto", () => {
  expect(
    needsFullReset(
      inputs({
        letters: withWord(emptyBoard(), 6, 7, "QIS"),
        numEventsChanged: true,
      }),
    ),
  ).toBe(true);
});

it("resets when the opponent plays where the placement arrow is", () => {
  expect(
    needsFullReset(
      inputs({
        currentRack: [5, 19, 9, 14],
        placedTiles: new Set<EphemeralTile>(),
        arrowProperties: { row: 7, col: 7, show: true },
        letters: withWord(emptyBoard(), 7, 7, "QIS"),
        numEventsChanged: true,
      }),
    ),
  ).toBe(true);
});

// The premove backed up when the opponent's phony displaced it (the hooking
// case above) is reinstated by the caller once the board and the rack are back
// at their backed up values. That only happens while we say no reset is
// needed, so the rest of the sequence has to keep saying no: clock changes
// either way, then challenging the phony off.
describe("reinstating a premove the opponent's phony displaced", () => {
  const displaced = () =>
    inputs({
      lastLetters: withWord(emptyBoard(), 6, 7, "QIS"),
      letters: withWord(emptyBoard(), 6, 7, "QIS"),
      placedTiles: new Set<EphemeralTile>(),
      displayedRack: [3, 1, 20, 5, 19, 9, 14],
    });

  it("keeps out of the way when either player is given time", () => {
    expect(needsFullReset(displaced())).toBe(false);
  });

  it("keeps out of the way when the phony is challenged off", () => {
    expect(
      needsFullReset({
        ...displaced(),
        letters: emptyBoard(),
        numEventsChanged: true,
      }),
    ).toBe(false);
  });
});
