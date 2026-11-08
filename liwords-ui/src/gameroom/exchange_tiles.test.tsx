import React from "react";
import { cleanup, render } from "@testing-library/react";

import { ExchangeTiles } from "./exchange_tiles";
import { setPreferredSortOrder } from "../store/constants";
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
  Array.from(document.querySelectorAll("button")).find((b) =>
    b.textContent?.startsWith("Exchange"),
  )!;
// the tiles a row holds, by their places in the rack, in the order drawn
const row = (name: "keep" | "exchange") =>
  Array.from(
    document.querySelectorAll(`[data-row="${name}"] [data-rack-index]`),
  ).map((el) => Number(el.getAttribute("data-rack-index")));
const tileAt = (idx: number) =>
  document.querySelector(`[data-rack-index="${idx}"] .tile`)!;
const footerLabel = () =>
  document.querySelector(".ant-modal-footer .label")?.textContent;

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
  expect(footerLabel()).toBeUndefined();

  pressKey("B");
  pressKey("E");
  await settle();

  expect(row("keep")).toEqual([0, 3, 4, 5, 6]);
  expect(row("exchange")).toEqual([1, 2]);
  expect(exchangeButton().textContent).toBe("Exchange BE");
  expect(footerLabel()).toBe("Keep AHLLL");
  expect(exchangeButton()).toBeEnabled();
  click(exchangeButton());
  await settle();
  expect(cb).toHaveBeenCalledWith(new Array<MachineLetter>(2, 5));
});

it("moves a tile to the other row on a click, each row in the rack's order", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(cb);
  expect(row("keep")).toEqual([0, 1, 2, 3, 4, 5, 6]);
  expect(row("exchange")).toEqual([]);

  click(tileAt(5));
  click(tileAt(0));
  await settle();
  expect(row("keep")).toEqual([1, 2, 3, 4, 6]);
  expect(row("exchange")).toEqual([0, 5]);

  click(tileAt(5)); // back among those kept, in its place
  await settle();
  expect(row("keep")).toEqual([1, 2, 3, 4, 5, 6]);
  expect(row("exchange")).toEqual([0]);
  expect(exchangeButton().textContent).toBe("Exchange A");
  expect(footerLabel()).toBe("Keep BEHLLL");
});

it("keeps each tile at its place in the rack, in either row", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(cb);
  // each row's places, left to right: the rack place of its tile, or null
  const places = (name: "keep" | "exchange") =>
    Array.from(
      document.querySelector(`[data-row="${name}"] .rack`)!.children,
    ).map((el) => el.getAttribute("data-rack-index"));

  click(tileAt(1));
  click(tileAt(5));
  await settle();
  expect(places("keep")).toEqual(["0", null, "2", "3", "4", null, "6"]);
  expect(places("exchange")).toEqual([null, "1", null, null, null, "5", null]);
  // every tile, in either row, shows the hand of a thing clicked
  expect(
    [0, 1, 2, 3, 4, 5, 6].map(
      (idx) => (tileAt(idx) as HTMLElement).style.cursor,
    ),
  ).toEqual(Array(7).fill("pointer"));
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
  expect(row("exchange")).toEqual([4, 5, 6]);

  pressKey("L"); // all three are selected, so this clears them
  await settle();
  expect(row("exchange")).toEqual([]);
  expect(exchangeButton()).toBeDisabled();
});

it("selects and clears the whole rack with -", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(cb);

  pressKey("-");
  await settle();
  expect(row("exchange")).toEqual([0, 1, 2, 3, 4, 5, 6]);
  expect(row("keep")).toEqual([]);
  expect(exchangeButton().textContent).toBe("Exchange ABEHLLL");
  expect(footerLabel()).toBe("Keep none");

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

  // a tile of more than one letter is named in brackets
  expect(exchangeButton().textContent).toMatch(/^Exchange \[L.L\]\[QU\]$/);
  expect(exchangeButton()).toBeEnabled();
  click(exchangeButton());
  await settle();
  expect(cb).toHaveBeenCalledWith(new Array<MachineLetter>(13, 19));
});

it("names the tiles in the user's tile order, not the rack's", async () => {
  const cb = vi.fn();
  await renderExchangeTiles(cb, [12, 1, 5, 2, 0, 8, 12]); // LAEB?HL
  click(tileAt(0));
  click(tileAt(3));
  click(tileAt(4));
  await settle();
  // no order set: by letter, the blank last
  expect(exchangeButton().textContent).toBe("Exchange BL?");
  expect(footerLabel()).toBe("Keep AEHL");
  try {
    setPreferredSortOrder("EAL");
    click(tileAt(1)); // A goes, and comes back, to show the new order
    click(tileAt(1));
    await settle();
    expect(exchangeButton().textContent).toBe("Exchange LB?");
    expect(footerLabel()).toBe("Keep EALH");
  } finally {
    setPreferredSortOrder("");
  }
  // what is sent stays in the rack's order
  click(exchangeButton());
  await settle();
  expect(cb).toHaveBeenCalledWith([12, 2, 0]);
});

// Each tile's top: the row to exchange at 0, the row kept at 100.
const stubTops = () => {
  const rect = HTMLElement.prototype.getBoundingClientRect;
  HTMLElement.prototype.getBoundingClientRect = function () {
    const top = this.closest('[data-row="exchange"]') ? 0 : 100;
    return new DOMRect(0, top, 40, 40);
  };
  return () => {
    HTMLElement.prototype.getBoundingClientRect = rect;
  };
};

it("slides a tile clicked from its old row to its new one", async () => {
  const animate = vi.fn();
  HTMLElement.prototype.animate = animate;
  const unstub = stubTops();
  try {
    await renderExchangeTiles(vi.fn());
    expect(animate).not.toHaveBeenCalled(); // nothing slides as it opens
    click(tileAt(2));
    await settle();
    expect(animate).toHaveBeenCalledTimes(1);
    expect(animate.mock.contexts[0]).toBe(
      document.querySelector('[data-rack-index="2"]'),
    );
    expect(animate.mock.calls[0][0]).toEqual([
      { transform: "translateY(100px)" },
      { transform: "none" },
    ]);
  } finally {
    unstub();
    delete (HTMLElement.prototype as { animate?: unknown }).animate;
  }
});

it("slides nothing where the user asks for less motion", async () => {
  const animate = vi.fn();
  HTMLElement.prototype.animate = animate;
  const unstub = stubTops();
  vi.stubGlobal("matchMedia", (query: string) => ({
    matches: query === "(prefers-reduced-motion: reduce)",
  }));
  try {
    await renderExchangeTiles(vi.fn());
    click(tileAt(2));
    pressKey("-");
    await settle();
    expect(row("exchange")).toEqual([0, 1, 3, 4, 5, 6]);
    expect(animate).not.toHaveBeenCalled();
  } finally {
    vi.unstubAllGlobals();
    unstub();
    delete (HTMLElement.prototype as { animate?: unknown }).animate;
  }
});
