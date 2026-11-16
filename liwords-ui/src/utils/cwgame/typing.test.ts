import { expect, it } from "vitest";
import {
  StandardEnglishAlphabet,
  StandardGermanAlphabet,
  StandardNorwegianAlphabet,
} from "../../constants/alphabets";
import { isAltTileKey, isTileKey } from "./typing";

it("knows a game's tiles typed in either case, and nothing else", () => {
  expect(isTileKey("å", StandardNorwegianAlphabet)).toBe(true);
  expect(isTileKey("Ø", StandardNorwegianAlphabet)).toBe(true);
  expect(isTileKey("å", StandardEnglishAlphabet)).toBe(false);
  expect(isTileKey("ü", StandardGermanAlphabet)).toBe(true);
  expect(isTileKey("a", StandardEnglishAlphabet)).toBe(true);
  for (const key of ["?", "3", "Dead", "Enter", "", "ß"]) {
    expect(isTileKey(key, StandardEnglishAlphabet)).toBe(false);
  }
});

it("takes only a letter beyond a-z as Alt/Opt's own tile", () => {
  expect(isAltTileKey("å", StandardNorwegianAlphabet)).toBe(true);
  // Alt+a where Alt makes no character: still a shortcut's key
  expect(isAltTileKey("a", StandardNorwegianAlphabet)).toBe(false);
  expect(isAltTileKey("A", StandardEnglishAlphabet)).toBe(false);
});
