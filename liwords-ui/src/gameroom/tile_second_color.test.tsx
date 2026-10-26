import { cleanup, render } from "@testing-library/react";
import { DndProvider } from "react-dnd";
import { TouchBackend } from "react-dnd-touch-backend";
import { afterEach, expect, it } from "vitest";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import Tile from "./tile";

afterEach(cleanup);

// Infuse Second Color, switched the way tile.tsx hears of it from another tab.
const setBicolor = (on: boolean) =>
  window.dispatchEvent(
    new StorageEvent("storage", {
      key: "enableBicolorMode",
      newValue: on ? "true" : "false",
    }),
  );

const tileClass = (props: {
  tentative: boolean;
  lastPlayed: boolean;
  playerOfTile: number;
}) => {
  const { container } = render(
    <DndProvider backend={TouchBackend}>
      <Tile
        letter={1}
        alphabet={StandardEnglishAlphabet}
        value={1}
        grabbable={false}
        {...props}
      />
    </DndProvider>,
  );
  return container.querySelector(".tile")!.className;
};

it("gives player 1's played tiles the second color, not their tentative ones", () => {
  setBicolor(true);
  try {
    const played = tileClass({
      tentative: false,
      lastPlayed: false,
      playerOfTile: 1,
    });
    expect(played).toMatch(/\bsecond-color\b/);
    const placed = tileClass({
      tentative: true,
      lastPlayed: false,
      playerOfTile: 1,
    });
    expect(placed).toMatch(/\btentative\b/);
    expect(placed).not.toMatch(/\bsecond-color\b/);
  } finally {
    setBicolor(false);
  }
});

it("still gives the last played tiles the second color by default", () => {
  expect(
    tileClass({ tentative: false, lastPlayed: true, playerOfTile: 1 }),
  ).toMatch(/\bsecond-color\b/);
  expect(
    tileClass({ tentative: true, lastPlayed: false, playerOfTile: 1 }),
  ).not.toMatch(/\bsecond-color\b/);
});
