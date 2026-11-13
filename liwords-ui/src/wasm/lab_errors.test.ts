import { describe, expect, it } from "vitest";
import { namedTiles } from "./lab_errors";

describe("namedTiles", () => {
  it("puts each tile's letter for its number", () => {
    const rune = (t: number) => String.fromCharCode(64 + t);
    expect(namedTiles("too many tile 17 (bag contains only 1)", rune)).toBe(
      "too many Q (bag contains only 1)",
    );
    expect(namedTiles("tiles 3 left", rune)).toBe("tiles 3 left");
  });
});
