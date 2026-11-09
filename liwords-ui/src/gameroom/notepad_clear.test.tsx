import React from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { Board } from "../utils/cwgame/board";
import { Notepad, NotepadContextProvider } from "./notepad";

const state = vi.hoisted(() => ({
  mobile: false,
  game: null as unknown,
}));
vi.mock("../utils/cwgame/common", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../utils/cwgame/common")>();
  return { ...actual, isMobile: () => state.mobile };
});
vi.mock("../store/store", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../store/store")>();
  return {
    ...actual,
    useGameContextStoreContext: () => ({ gameContext: state.game }),
    useExaminableGameContextStoreContext: () => ({ gameContext: state.game }),
    useTentativeTileContext: () => ({
      displayedRack: [],
      placedTiles: new Set(),
      placedTilesTempScore: 0,
    }),
  };
});

afterEach(cleanup);

// A notepad with a note in it; then its button that empties it.
const written = () => {
  state.game = { board: new Board(), alphabet: StandardEnglishAlphabet };
  render(
    <NotepadContextProvider feRackInfo gameID="g1">
      <Notepad />
    </NotepadContextProvider>,
  );
  fireEvent.change(screen.getByRole("textbox"), { target: { value: "note" } });
  return document.querySelector(".anticon-delete")!.closest("button")!;
};

it("focuses the notepad it empties", () => {
  state.mobile = false;
  fireEvent.click(written());
  expect(screen.getByRole("textbox")).toHaveValue("");
  expect(screen.getByRole("textbox")).toHaveFocus();
});

it("leaves the focus where it is on mobile", () => {
  state.mobile = true;
  fireEvent.click(written());
  expect(screen.getByRole("textbox")).toHaveValue("");
  expect(screen.getByRole("textbox")).not.toHaveFocus();
});
