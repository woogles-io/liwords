import { describe, expect, it } from "vitest";
import { chatInputGrowth } from "./chat";

const textarea = (height: number) => {
  const ta = document.createElement("textarea");
  ta.style.lineHeight = "22px";
  ta.style.padding = "4px 11px";
  ta.style.border = "1px solid black";
  Object.defineProperty(ta, "offsetHeight", { value: height });
  document.body.appendChild(ta);
  return ta;
};

describe("chatInputGrowth", () => {
  it("is zero for one row", () => {
    // 22 line + 4 + 4 padding + 1 + 1 border
    expect(chatInputGrowth(textarea(32))).toBe(0);
  });

  it("is the extra rows' height once it grows", () => {
    expect(chatInputGrowth(textarea(32 + 5 * 22))).toBe(110);
  });
});
