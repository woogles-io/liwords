import { expect, it } from "vitest";
import {
  StandardGermanAlphabet,
  SuperGermanAlphabet,
  alphabetFromName,
} from "./alphabets";

it("has the super German set: the German letters and scores, 200 tiles", () => {
  const letters = SuperGermanAlphabet.letters;
  expect(letters.map((l) => [l.rune, l.score])).toEqual(
    StandardGermanAlphabet.letters.map((l) => [l.rune, l.score]),
  );
  expect(letters.reduce((n, l) => n + l.count, 0)).toBe(200);
  expect(alphabetFromName("german_super")).toBe(SuperGermanAlphabet);
});
