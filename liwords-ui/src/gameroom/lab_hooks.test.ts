import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { hookLine, hookLines, publishLabHooks, useLabHooks } from "./lab_hooks";

describe("hooks", () => {
  it("writes a square's line whole while it fits, else a few and how many more", () => {
    // a square six characters wide
    const fits = (line: string) => line.length <= 6;
    expect(hookLine(["E", "I", "O"], fits)).toBe("EIO");
    expect(hookLine([..."ABDEFH"], fits)).toBe("ABDEFH");
    expect(hookLine([..."ABDEFHJKLMNPTYZ"], fits)).toBe("ABD+12");
    // a tile of several letters, bracketed as the board writes it, is
    // counted once, kept whole, and takes its width
    expect(hookLine(["A", "[LL]", "E", "I", "O", "U"], fits)).toBe("A+5");
    expect(hookLine(["[LL]", "A"], fits)).toBe("[LL]A");
    expect(hookLine(["A", "B", "[LL]", "E", "I", "O"], fits)).toBe("AB+4");
  });

  it("fits a line by what the square measures, and keeps it whole unmeasured", () => {
    // a square four characters wide
    const fits = (line: string) => line.length <= 4;
    expect(hookLine([..."MNOWY"], fits)).toBe("MN+3");
    expect(hookLine([..."MNOW"], fits)).toBe("MNOW");
    expect(hookLine([..."ABDEFHJKLMNPTYZ"])).toBe("ABDEFHJKLMNPTYZ");
  });

  it("lists both ways in a square's tooltip, any where no tile is beside it", () => {
    expect(hookLines({ across: [..."ADS"], down: null })).toEqual([
      "Across: ADS",
      "Down: any",
    ]);
    expect(hookLines({ across: [], down: ["S"] })).toEqual([
      "Across: none",
      "Down: S",
    ]);
  });

  it("shows what an open analyzer publishes, and nothing once none is", () => {
    const { result } = renderHook(() => useLabHooks());
    expect(result.current).toBeNull();
    const a = Symbol("a");
    const b = Symbol("b");
    const hooks = { dim: 15, squares: [] };
    act(() => publishLabHooks(a, hooks));
    act(() => publishLabHooks(b, hooks));
    act(() => publishLabHooks(a, null));
    expect(result.current).toBe(hooks);
    act(() => publishLabHooks(b, null));
    expect(result.current).toBeNull();
  });
});
