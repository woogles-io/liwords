import { describe, expect, it } from "vitest";
import { annotatedGameRules } from "./variants";

describe("annotatedGameRules", () => {
  it("puts ZOMGWords on the big board with the 200-tile set", () => {
    const r = annotatedGameRules("CSW24", "classic_super");
    expect([
      r.boardLayoutName,
      r.letterDistributionName,
      r.variantName,
    ]).toEqual(["SuperCrosswordGame", "english_super", "classic_super"]);
    expect(
      annotatedGameRules("DISC2", "classic_super").letterDistributionName,
    ).toBe("catalan_super");
  });
  it("keeps the standard board and the lexicon's tiles otherwise", () => {
    const smog = annotatedGameRules("CSW24", "wordsmog");
    expect([smog.boardLayoutName, smog.letterDistributionName]).toEqual([
      "CrosswordGame",
      "english",
    ]);
    expect(annotatedGameRules("RD28", "classic").letterDistributionName).toBe(
      "german",
    );
  });
});
