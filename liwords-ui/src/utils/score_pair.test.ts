import { describe, expect, it } from "vitest";
import { scorePair } from "./score_pair";

describe("scorePair", () => {
  it("joins two scores with a hyphen", () => {
    expect(scorePair(428, 386)).toBe("428-386");
    expect(scorePair(0, 0)).toBe("0-0");
  });
  it("brackets a negative score", () => {
    expect(scorePair(-10, -10)).toBe("(-10)-(-10)");
    expect(scorePair(12, -3)).toBe("12-(-3)");
  });
});
