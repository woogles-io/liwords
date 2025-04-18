import React from "react";
import { cleanup, fireEvent, render } from "@testing-library/react";
import { MachineLetter } from "../utils/cwgame/common";
import { BlankSelector } from "./blank_selector";
import {
  Alphabet,
  StandardCatalanAlphabet,
  StandardEnglishAlphabet,
  StandardNorwegianAlphabet,
} from "../constants/alphabets";

function renderSelectTiles(
  alphabet: Alphabet,
  callback: (letter: MachineLetter) => void,
) {
  return render(
    <BlankSelector
      tileColorId={1}
      handleSelection={callback}
      alphabet={alphabet}
    />,
  );
}

afterEach(cleanup);

it("selects a weird tile", async () => {
  const cb = vi.fn();

  const { findByText } = renderSelectTiles(StandardCatalanAlphabet, cb);

  const LLtile = await findByText("L·L");
  fireEvent.click(LLtile);
  expect(cb).toHaveBeenCalledWith(13);
});

it("selects last tile", async () => {
  const cb = vi.fn();

  const { findByText } = renderSelectTiles(StandardCatalanAlphabet, cb);

  const ztile = await findByText("Z");
  fireEvent.click(ztile);
  expect(cb).toHaveBeenCalledWith(26);
});

it("selects norwegian tile", async () => {
  const cb = vi.fn();

  const { findByText } = renderSelectTiles(StandardNorwegianAlphabet, cb);

  const Øtile = await findByText("Ø");
  fireEvent.click(Øtile);
  expect(cb).toHaveBeenCalledWith(30);
});

it("selects first english tile", async () => {
  const cb = vi.fn();

  const { findByText } = renderSelectTiles(StandardEnglishAlphabet, cb);

  const atile = await findByText("A");
  fireEvent.click(atile);
  expect(cb).toHaveBeenCalledWith(1);
});
