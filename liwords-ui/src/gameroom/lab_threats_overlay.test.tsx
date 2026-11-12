import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { publishLabHooks } from "./lab_hooks";
import { LabHooksOverlay } from "./lab_hooks_overlay";
import { LabThreats, publishLabThreats } from "./lab_threats";
import { LabThreatsOverlay } from "./lab_threats_overlay";

const publisher = Symbol("test");

const threats = (more: Partial<LabThreats> = {}): LabThreats => ({
  dim: 3,
  heat: [undefined, { level: 1, label: "40" }, { level: 0.5, label: "20" }],
  ghost: new Map([[2, "S"]]),
  outline: new Set([1]),
  square: null,
  picking: false,
  pick: () => {},
  ...more,
});

afterEach(() => {
  act(() => publishLabThreats(publisher, null));
  act(() => publishLabHooks(publisher, null));
  cleanup();
});

describe("LabThreatsOverlay", () => {
  it("draws nothing until threats are published, or for another board", () => {
    const { container } = render(<LabThreatsOverlay gridDim={3} />);
    expect(container.firstChild).toBeNull();
    act(() => publishLabThreats(publisher, threats({ dim: 15 })));
    expect(container.firstChild).toBeNull();
  });

  it("tints the hot squares, shows the opponent's tiles, outlines the play pointed at", () => {
    act(() => publishLabThreats(publisher, threats()));
    const { container } = render(<LabThreatsOverlay gridDim={3} />);
    const squares = container.querySelectorAll(".lab-threat");
    expect(squares).toHaveLength(9);
    expect(squares[0].textContent).toBe("");
    expect(squares[1].textContent).toBe("40");
    expect(squares[1].className).toContain("lab-outlined");
    // the opponent's tile covers the square's number
    expect(squares[2].textContent).toBe("S");
    expect(squares[2].className).toContain("lab-ghost");
  });

  it("picks the square clicked while picking, and lets it go on a second click", () => {
    const pick = vi.fn();
    act(() => publishLabThreats(publisher, threats({ picking: true, pick })));
    const { container } = render(<LabThreatsOverlay gridDim={3} />);
    expect(container.firstElementChild!.className).toContain("lab-picking");
    fireEvent.click(container.querySelectorAll(".lab-threat")[4]);
    expect(pick).toHaveBeenLastCalledWith(4);
    act(() =>
      publishLabThreats(publisher, threats({ picking: true, pick, square: 4 })),
    );
    fireEvent.click(container.querySelectorAll(".lab-threat")[4]);
    expect(pick).toHaveBeenLastCalledWith(null);
  });

  it("shares the squares with the hooks: the tint alone, and the opponent's tile in place of the hooks", () => {
    act(() => publishLabThreats(publisher, threats()));
    act(() =>
      publishLabHooks(publisher, {
        dim: 3,
        squares: [
          undefined,
          { across: ["A"], down: null },
          { across: ["B"], down: ["C"] },
        ],
      }),
    );
    const { container } = render(
      <div>
        <LabThreatsOverlay gridDim={3} />
        <LabHooksOverlay gridDim={3} />
      </div>,
    );
    const tinted = container.querySelectorAll(".lab-threat");
    // the number gives way to the hooks' letters, the tint stays
    expect(tinted[1].textContent).toBe("");
    expect(tinted[1].querySelector(".lab-heat")).not.toBeNull();
    const hooked = container.querySelectorAll(".lab-hook");
    expect(hooked[1].textContent).toBe("A");
    // the opponent's tile shows; the hooks there do not
    expect(tinted[2].textContent).toBe("S");
    expect(hooked[2].textContent).toBe("");
  });
});
