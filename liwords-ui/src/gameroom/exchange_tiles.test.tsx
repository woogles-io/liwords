import React from "react";
import { cleanup, render } from "@testing-library/react";

import { ExchangeTiles } from "./exchange_tiles";
import { MachineLetter, MachineWord } from "../utils/cwgame/common";
import {
  StandardCatalanAlphabet,
  StandardEnglishAlphabet,
  Alphabet,
} from "../constants/alphabets";
import { DndProvider } from "react-dnd";
import { TouchBackend } from "react-dnd-touch-backend";

afterEach(cleanup);

// These drive the component with real DOM events and real timers. They used to
// use fake timers plus waitFor, which deadlock each other, and had been skipped
// for it.

const pressKey = (key: string) =>
  window.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true }));
const click = (el: Element) =>
  el.dispatchEvent(
    new MouseEvent("click", { bubbles: true, cancelable: true }),
  );
const settle = () => new Promise((r) => setTimeout(r, 0));

const exchangeButton = () =>
  Array.from(document.querySelectorAll("button")).find(
    (b) => b.textContent === "Exchange",
  )!;

async function renderExchangeTiles(
  callback: (t: MachineWord) => void,
  rack: MachineWord = [1, 2, 5, 8, 12, 12, 12], // ABEHLLL
  alphabet: Alphabet = StandardEnglishAlphabet,
) {
  const ret = render(
    <DndProvider backend={TouchBackend}>
      <ExchangeTiles
        tileColorId={1}
        rack={rack}
        alphabet={alphabet}
        onOk={callback}
        onCancel={() => {}}
        modalVisible={true}
      />
    </DndProvider>,
  );
  // the modal ignores keystrokes for a moment after opening so that the key
  // used to open it doesn't preselect a tile.
  await new Promise((r) => setTimeout(r, 200));
  return ret;
}

it("exchanges the right tiles", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(cb);
  expect(exchangeButton()).toBeDisabled();

  pressKey("B");
  pressKey("E");
  await settle();

  expect(exchangeButton()).toBeEnabled();
  click(exchangeButton());
  await settle();
  expect(cb).toHaveBeenCalledWith(new Array<MachineLetter>(2, 5));
});

it("exchanges repeated tile", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(cb);

  pressKey("L");
  pressKey("L");
  await settle();

  expect(exchangeButton()).toBeEnabled();
  click(exchangeButton());
  await settle();
  expect(cb).toHaveBeenCalledWith(new Array<MachineLetter>(12, 12));
});

it("deselects every instance of a repeated tile at once", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(cb);

  pressKey("L");
  pressKey("L");
  pressKey("L");
  await settle();
  expect(document.querySelectorAll(".rack .tile.selected").length).toBe(3);

  pressKey("L"); // all three are selected, so this clears them
  await settle();
  expect(document.querySelectorAll(".rack .tile.selected").length).toBe(0);
  expect(exchangeButton()).toBeDisabled();
});

it("selects and clears the whole rack with -", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(cb);

  pressKey("-");
  await settle();
  expect(document.querySelectorAll(".rack .tile.selected").length).toBe(7);

  click(exchangeButton());
  await settle();
  expect(cb).toHaveBeenCalledWith([1, 2, 5, 8, 12, 12, 12]);
});

it("ignores non-existing tiles", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(cb);

  pressKey("M");
  await settle();

  expect(exchangeButton()).toBeDisabled();
  click(exchangeButton());
  await settle();
  expect(cb).toBeCalledTimes(0);
});

it("works with multi-letter tiles and shortcut/alias", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(
    cb,
    [1, 13, 19, 6, 21, 17, 10], // A L·L QU E S O I
    StandardCatalanAlphabet,
  );

  pressKey("W"); // alias for L·L
  pressKey("Q");
  await settle();

  expect(exchangeButton()).toBeEnabled();
  click(exchangeButton());
  await settle();
  expect(cb).toHaveBeenCalledWith(new Array<MachineLetter>(13, 19));
});
