import React from "react";
import { cleanup, render } from "@testing-library/react";

import { ExchangeTiles } from "./exchange_tiles";
import { MachineWord } from "../utils/cwgame/common";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { DndProvider } from "react-dnd";
import { TouchBackend } from "react-dnd-touch-backend";

// Regression tests for a bug where the modal submitted a different set of
// tiles than the one it was drawing as selected. A player exchanged all seven
// of DJKLNVW, watched every tile flip up, and the server recorded -DJLNVW: the
// K stayed on their rack. See game UX8d9KFgRQ.
//
// These deliberately dispatch real DOM events rather than going through
// fireEvent/act, and deliberately do NOT let React's scheduler run between the
// last selection and the submit. That is what a browser looks like when the
// main thread is busy -- a websocket burst, a clock re-render, a GC pause --
// and it was enough to make the modal submit a stale selection.

afterEach(cleanup);

// DJKLNVW
const RACK = [4, 10, 11, 12, 14, 22, 23];
const K_INDEX = 2;

const click = (el: Element) =>
  el.dispatchEvent(
    new MouseEvent("click", { bubbles: true, cancelable: true }),
  );
const pressKey = (key: string) =>
  window.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true }));
const macrotask = () => new Promise((r) => setTimeout(r, 0));
// React 19 flushes a discrete update in a microtask, so this is the boundary a
// browser crosses between an event handler returning and the screen updating.
// It is NOT enough to let the scheduler run passive effects.
const commit = () => Promise.resolve();

// Pin the main thread so no scheduler callback -- and therefore no passive
// effect -- can run between two user actions.
const pinMainThread = (ms: number) => {
  const start = Date.now();
  while (Date.now() - start < ms) {
    /* deliberately blocking */
  }
};

const tiles = () => Array.from(document.querySelectorAll(".rack .tile"));
const selectedTiles = () =>
  tiles()
    .map((t, i) => (t.classList.contains("selected") ? RACK[i] : null))
    .filter((x): x is number => x !== null);
const exchangeButton = () =>
  Array.from(document.querySelectorAll("button")).find(
    (b) => b.textContent === "Exchange",
  )!;

function renderModal(onOk: (t: MachineWord) => void) {
  return render(
    <DndProvider backend={TouchBackend}>
      <ExchangeTiles
        tileColorId={1}
        rack={RACK}
        alphabet={StandardEnglishAlphabet}
        onOk={onOk}
        onCancel={() => {}}
        modalVisible={true}
      />
    </DndProvider>,
  );
}

it("submits every tile it is drawing as selected, even with the main thread pinned", async () => {
  let sent: MachineWord | null = null;
  renderModal((t) => {
    sent = t;
  });
  await new Promise((r) => setTimeout(r, 200));

  // Select six, then -- the way someone double-checking their rack does --
  // spot the one tile that isn't flipped up, click it, and immediately submit.
  for (const i of [0, 1, 3, 4, 5, 6]) {
    click(tiles()[i]);
    await macrotask();
  }
  click(tiles()[K_INDEX]);
  await commit();
  expect(selectedTiles()).toEqual(RACK); // the K is drawn flipped up
  pinMainThread(120);

  click(exchangeButton());
  await macrotask();

  expect(sent).not.toBeNull();
  expect([...sent!].sort((a, b) => a - b)).toEqual(RACK);
});

it("submits the full selection on Enter, even with the main thread pinned", async () => {
  let sent: MachineWord | null = null;
  renderModal((t) => {
    sent = t;
  });
  await new Promise((r) => setTimeout(r, 200));

  pressKey("-"); // select all
  await commit();
  expect(selectedTiles()).toEqual(RACK);
  pinMainThread(120);

  pressKey("Enter");
  await macrotask();

  expect(sent).not.toBeNull();
  expect([...sent!].sort((a, b) => a - b)).toEqual(RACK);
});

it("keeps tiles clicked in the first moments after the modal opens", async () => {
  let sent: MachineWord | null = null;
  renderModal((t) => {
    sent = t;
  });

  // No settling wait: click straight away, inside the window that the modal
  // used to clear on a 100ms timer.
  click(tiles()[K_INDEX]);
  await new Promise((r) => setTimeout(r, 250));

  expect(selectedTiles()).toEqual([RACK[K_INDEX]]);
  click(exchangeButton());
  await macrotask();

  expect(sent).toEqual([RACK[K_INDEX]]);
});

it("does not carry a selection over from a previous visit to the modal", async () => {
  let sent: MachineWord | null = null;
  const view = renderModal((t) => {
    sent = t;
  });
  await new Promise((r) => setTimeout(r, 200));
  click(tiles()[0]);
  await macrotask();

  const rerender = (modalVisible: boolean) =>
    view.rerender(
      <DndProvider backend={TouchBackend}>
        <ExchangeTiles
          tileColorId={1}
          rack={RACK}
          alphabet={StandardEnglishAlphabet}
          onOk={(t) => {
            sent = t;
          }}
          onCancel={() => {}}
          modalVisible={modalVisible}
        />
      </DndProvider>,
    );

  rerender(false);
  await macrotask();
  rerender(true);
  await new Promise((r) => setTimeout(r, 200));

  expect(selectedTiles()).toEqual([]);
  click(tiles()[K_INDEX]);
  await macrotask();
  click(exchangeButton());
  await macrotask();

  expect(sent).toEqual([RACK[K_INDEX]]);
});
