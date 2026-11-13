import { afterEach, describe, expect, it } from "vitest";
import { applyHandoverStorage } from "./handover";

afterEach(() => localStorage.clear());

describe("applyHandoverStorage", () => {
  it("sets each key to its value", () => {
    applyHandoverStorage({ enableAnalyzerLab: "true", n: 3 });
    expect(localStorage.getItem("enableAnalyzerLab")).toBe("true");
    expect(localStorage.getItem("n")).toBe("3");
  });

  it("removes a key given as null", () => {
    localStorage.setItem("enableAnalyzerLabb", "true");
    applyHandoverStorage({ enableAnalyzerLabb: null });
    expect(localStorage.getItem("enableAnalyzerLabb")).toBeNull();
    expect(localStorage.length).toBe(0);
  });
});
