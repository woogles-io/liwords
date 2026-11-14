import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import {
  runesToMachineWord,
  StandardGermanAlphabet,
  StandardSpanishAlphabet,
} from "../constants/alphabets";
import { DndProvider } from "react-dnd";
import { TouchBackend } from "react-dnd-touch-backend";
import { RackEditor } from "./rack_editor";

// Enter sends the rack the editor holds, so a rack it opens with is capped to
// seven tiles like typed input (#1776). A digraph tile counts once.
it("sends at most seven tiles on Enter, a digraph counting as one", () => {
  const rack = runesToMachineWord("[CH][LL][RR]AEIOU", StandardSpanishAlphabet);
  expect(rack.length).toBe(8);
  const rackCallback = vi.fn();
  render(
    <DndProvider backend={TouchBackend}>
      <RackEditor
        rackCallback={rackCallback}
        cancelCallback={() => {}}
        currentRack={rack}
        alphabet={StandardSpanishAlphabet}
      />
    </DndProvider>,
  );

  fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

  expect(rackCallback).toHaveBeenCalledWith(rack.slice(0, 7));
});

it("sends a full rack it opens with unchanged", () => {
  const rack = runesToMachineWord("[CH]AEIOUS", StandardSpanishAlphabet);
  expect(rack.length).toBe(7);
  const rackCallback = vi.fn();
  render(
    <DndProvider backend={TouchBackend}>
      <RackEditor
        rackCallback={rackCallback}
        cancelCallback={() => {}}
        currentRack={rack}
        alphabet={StandardSpanishAlphabet}
      />
    </DndProvider>,
  );

  fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

  expect(rackCallback).toHaveBeenCalledWith(rack);
});

// A tile clicked joins the rack, so a letter the keyboard cannot type, or a
// digraph, is a click away; a full rack takes no more.
it("adds a clicked tile to the rack, up to seven", () => {
  const rackCallback = vi.fn();
  const { container } = render(
    <DndProvider backend={TouchBackend}>
      <RackEditor
        rackCallback={rackCallback}
        cancelCallback={() => {}}
        currentRack={runesToMachineWord("AEIOUS", StandardSpanishAlphabet)}
        alphabet={StandardSpanishAlphabet}
      />
    </DndProvider>,
  );
  const tiles = container.querySelectorAll(".rack-editor-tiles .tile");
  const ch = runesToMachineWord("[CH]", StandardSpanishAlphabet)[0];
  fireEvent.click(tiles[ch]);
  fireEvent.click(tiles[ch]);
  expect(screen.getByRole("textbox")).toHaveValue("AEIOUS[CH]");
  fireEvent.click(screen.getByRole("button", { name: "Use this rack" }));
  expect(rackCallback).toHaveBeenCalledWith(
    runesToMachineWord("AEIOUS[CH]", StandardSpanishAlphabet),
  );
});

// What Enter and Esc do, a pointer can do with the editor's two buttons.
it("uses the rack typed, or cancels, by its buttons", () => {
  const rack = runesToMachineWord("[CH]AEIOUS", StandardSpanishAlphabet);
  const rackCallback = vi.fn();
  const cancelCallback = vi.fn();
  render(
    <DndProvider backend={TouchBackend}>
      <RackEditor
        rackCallback={rackCallback}
        cancelCallback={cancelCallback}
        currentRack={rack}
        alphabet={StandardSpanishAlphabet}
      />
    </DndProvider>,
  );

  fireEvent.click(screen.getByRole("button", { name: "Use this rack" }));
  expect(rackCallback).toHaveBeenCalledWith(rack);

  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(cancelCallback).toHaveBeenCalledTimes(1);
});

// A rack it opens with full can be emptied, or a tile taken out, by a click.
it("takes a tile out of the rack by a click on it, or clears it", () => {
  const rackCallback = vi.fn();
  const { container } = render(
    <DndProvider backend={TouchBackend}>
      <RackEditor
        rackCallback={rackCallback}
        cancelCallback={() => {}}
        currentRack={runesToMachineWord("AEIOUSR", StandardSpanishAlphabet)}
        alphabet={StandardSpanishAlphabet}
      />
    </DndProvider>,
  );
  const rackTiles = () => container.querySelectorAll(".rack-editor-rack .tile");
  expect(rackTiles()).toHaveLength(7);
  fireEvent.click(rackTiles()[1]);
  expect(screen.getByRole("textbox")).toHaveValue("AIOUSR");
  fireEvent.click(screen.getByRole("button", { name: "Clear" }));
  expect(screen.getByRole("textbox")).toHaveValue("");
  expect(rackTiles()).toHaveLength(0);
  const tiles = container.querySelectorAll(".rack-editor-tiles .tile");
  fireEvent.click(tiles[runesToMachineWord("Ñ", StandardSpanishAlphabet)[0]]);
  fireEvent.click(screen.getByRole("button", { name: "Use this rack" }));
  expect(rackCallback).toHaveBeenCalledWith(
    runesToMachineWord("Ñ", StandardSpanishAlphabet),
  );
});

// A dead key's accent stays in the box until its letter makes the letter.
it("keeps a composition's accent until the letter is made", () => {
  const rackCallback = vi.fn();
  render(
    <DndProvider backend={TouchBackend}>
      <RackEditor
        rackCallback={rackCallback}
        cancelCallback={() => {}}
        currentRack={[]}
        alphabet={StandardGermanAlphabet}
      />
    </DndProvider>,
  );
  const box = screen.getByRole("textbox");
  fireEvent.compositionStart(box);
  fireEvent.change(box, { target: { value: "\u00a8" } });
  // the accent is not thrown away mid-composition
  expect(box).toHaveValue("\u00a8");
  fireEvent.change(box, { target: { value: "\u00f6" } });
  fireEvent.compositionEnd(box, { data: "\u00f6" });
  expect(box).toHaveValue("\u00d6");
  fireEvent.keyDown(box, { key: "Enter" });
  expect(rackCallback).toHaveBeenCalledWith(
    runesToMachineWord("\u00d6", StandardGermanAlphabet),
  );
});

// Each grid tile says how many are left: the game's, less those on the board
// and in the rack being set; one with none left is dimmed and adds nothing.
it("counts the tiles left, and adds none of a tile used up", () => {
  const alphabet = StandardSpanishAlphabet;
  const ch = runesToMachineWord("[CH]", alphabet)[0];
  const board = new Array(225).fill(0);
  // the one CH on the board, and a blank played as an A
  board[0] = ch;
  board[1] = runesToMachineWord("A", alphabet)[0] | 0x80;
  const { container } = render(
    <DndProvider backend={TouchBackend}>
      <RackEditor
        rackCallback={() => {}}
        cancelCallback={() => {}}
        currentRack={runesToMachineWord("?", alphabet)}
        alphabet={alphabet}
        boardLetters={board}
      />
    </DndProvider>,
  );
  const cells = container.querySelectorAll(
    ".rack-editor-tiles .rack-editor-tile",
  );
  // both blanks gone: one on the board, one in the rack
  expect(cells[0].className).toContain("rack-editor-none");
  expect(cells[0].textContent).toContain("0");
  expect(cells[ch].className).toContain("rack-editor-none");
  fireEvent.click(cells[ch].querySelector(".tile")!);
  expect(screen.getByRole("textbox")).toHaveValue("?");
});
