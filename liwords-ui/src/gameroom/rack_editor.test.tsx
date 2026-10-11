import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import {
  runesToMachineWord,
  StandardSpanishAlphabet,
} from "../constants/alphabets";
import { RackEditor } from "./rack_editor";

// Enter sends the rack the editor holds, so a rack it opens with is capped to
// seven tiles like typed input (#1776). A digraph tile counts once.
it("sends at most seven tiles on Enter, a digraph counting as one", () => {
  const rack = runesToMachineWord("[CH][LL][RR]AEIOU", StandardSpanishAlphabet);
  expect(rack.length).toBe(8);
  const rackCallback = vi.fn();
  render(
    <RackEditor
      rackCallback={rackCallback}
      cancelCallback={() => {}}
      currentRack={rack}
      alphabet={StandardSpanishAlphabet}
    />,
  );

  fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

  expect(rackCallback).toHaveBeenCalledWith(rack.slice(0, 7));
});

it("sends a full rack it opens with unchanged", () => {
  const rack = runesToMachineWord("[CH]AEIOUS", StandardSpanishAlphabet);
  expect(rack.length).toBe(7);
  const rackCallback = vi.fn();
  render(
    <RackEditor
      rackCallback={rackCallback}
      cancelCallback={() => {}}
      currentRack={rack}
      alphabet={StandardSpanishAlphabet}
    />,
  );

  fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

  expect(rackCallback).toHaveBeenCalledWith(rack);
});
