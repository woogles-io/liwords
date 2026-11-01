import React from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { Board } from "../utils/cwgame/board";
import { Notepad, NotepadContextProvider } from "./notepad";

const A = 1;
const B = 2;
const C = 3;
const D = 4;
const E = 5;
const Z = 26;

// The game being played has a Z where the position examined has an A.
const boardWith = (ml: number) => {
  const board = new Board();
  board.addTile({ row: 7, col: 7, ml });
  return board;
};
const store = vi.hoisted(() => ({
  examined: null as unknown,
  live: null as unknown,
}));
vi.mock("../store/store", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../store/store")>();
  return {
    ...actual,
    useGameContextStoreContext: () => ({ gameContext: store.live }),
    useExaminableGameContextStoreContext: () => ({
      gameContext: store.examined,
    }),
    useTentativeTileContext: () => ({
      displayedRack: [C, D, E],
      placedTiles: new Set([{ row: 7, col: 8, letter: B }]),
      placedTilesTempScore: 4,
    }),
  };
});

afterEach(cleanup);

it("notes the move placed on the position examined", () => {
  store.live = { board: boardWith(Z), alphabet: StandardEnglishAlphabet };
  store.examined = { board: boardWith(A), alphabet: StandardEnglishAlphabet };
  render(
    <NotepadContextProvider feRackInfo gameID="g1">
      <Notepad />
    </NotepadContextProvider>,
  );
  fireEvent.click(screen.getByRole("button"));
  expect(screen.getByRole("textbox")).toHaveValue("8H (A)B 4 CDE");
});
