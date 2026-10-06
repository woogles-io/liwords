import { describe, expect, it } from "vitest";
import {
  RackDragState,
  RackGeometry,
  applyRackDrag,
  initialRackDragState,
  nextRackDragState,
  previewSlots,
} from "./rack_drag";

// Seven 40px slots starting at x=100; rack spans y 500..540.
const geo: RackGeometry = {
  top: 500,
  bottom: 540,
  slotCenters: [120, 160, 200, 240, 280, 320, 360],
  slotWidth: 40,
  tileHeight: 40,
};
const onRack = (x: number) => ({ x, y: 520 });

const drive = (from: number, points: { x: number; y: number }[]) =>
  points.reduce<RackDragState>(
    (s, p) => nextRackDragState(s, p, geo),
    initialRackDragState(from),
  );

const rack = (s: string) => s.split("");

describe("rack drag: insert", () => {
  it("does nothing until the pointer reaches the neighbouring slot", () => {
    const s = drive(3, [onRack(235), onRack(222)]);
    expect(s).toEqual({ from: 3, mode: "insert", target: 3 });
    expect(applyRackDrag(rack("ABCDEFG"), s)).toEqual(rack("ABCDEFG"));
  });

  it("pushes the left neighbour toward the origin when dragging left", () => {
    const s = drive(3, [onRack(230), onRack(210)]);
    expect(s.target).toBe(2);
    expect(previewSlots(7, s)).toEqual([0, 1, 3, 2, 4, 5, 6]);
    expect(applyRackDrag(rack("ABCDEFG"), s)).toEqual(rack("ABDCEFG"));
  });

  it("is symmetric when dragging right", () => {
    const s = drive(3, [onRack(265)]);
    expect(s.target).toBe(4);
    expect(applyRackDrag(rack("ABCDEFG"), s)).toEqual(rack("ABCEDFG"));
  });

  it("slides every tile in between on a long drag", () => {
    const s = drive(5, [onRack(110)]);
    expect(s.target).toBe(0);
    expect(previewSlots(7, s)).toEqual([1, 2, 3, 4, 5, 0, 6]);
    expect(applyRackDrag(rack("ABCDEFG"), s)).toEqual(rack("FABCDEG"));
  });

  it("clamps past the ends of the rack", () => {
    expect(drive(0, [onRack(900)]).target).toBe(6);
    expect(drive(6, [onRack(-50)]).target).toBe(0);
  });

  it("tolerates a little vertical wobble", () => {
    const s = drive(3, [{ x: 200, y: 485 }]);
    expect(s).toEqual({ from: 3, mode: "insert", target: 2 });
  });
});

describe("rack drag: lift and swap", () => {
  it("swaps with the tile it comes back down on (IHEASES example)", () => {
    // Drag the rightmost E (slot 5) upward.
    const lifted = drive(5, [onRack(320), { x: 300, y: 400 }]);
    expect(lifted).toEqual({ from: 5, mode: "lifted", target: null });
    expect(previewSlots(7, lifted)).toEqual([0, 1, 2, 3, 4, 5, 6]);

    // Come back down on the H.
    const s = nextRackDragState(lifted, onRack(165), geo);
    expect(s).toEqual({ from: 5, mode: "swap", target: 1 });
    expect(previewSlots(7, s)).toEqual([0, 5, 2, 3, 4, 1, 6]);
    expect(applyRackDrag(rack("IHEASES"), s)).toEqual(rack("IEEASHS"));
  });

  it("keeps swapping (not inserting) while sliding along after re-entry", () => {
    const s = drive(5, [{ x: 300, y: 400 }, onRack(165), onRack(240)]);
    expect(s).toEqual({ from: 5, mode: "swap", target: 3 });
    expect(applyRackDrag(rack("IHEASES"), s)).toEqual(rack("IHEESAS"));
  });

  it("is a no-op when returning to the origin slot", () => {
    const s = drive(5, [{ x: 300, y: 600 }, onRack(318)]);
    expect(s.target).toBe(5);
    expect(applyRackDrag(rack("IHEASES"), s)).toEqual(rack("IHEASES"));
  });

  it("has no target beside the rack after lifting", () => {
    const s = drive(2, [{ x: 200, y: 400 }, onRack(40)]);
    expect(s).toEqual({ from: 2, mode: "swap", target: null });
  });

  it("re-lifts when leaving again", () => {
    const s = drive(2, [{ x: 200, y: 400 }, onRack(160), { x: 160, y: 700 }]);
    expect(s).toEqual({ from: 2, mode: "lifted", target: null });
  });
});

describe("rack drag: gaps", () => {
  const GAP = "_";
  it("preserves length and gaps as ordinary slots", () => {
    const r = rack("AB_D_FG");
    const ins = drive(6, [onRack(160)]);
    const out = applyRackDrag(r, ins);
    expect(out).toEqual(rack("AGB_D_F"));
    expect(out.filter((x) => x === GAP)).toHaveLength(2);

    const sw = drive(0, [{ x: 0, y: 0 }, onRack(200)]);
    expect(applyRackDrag(r, sw)).toEqual(rack("_BAD_FG"));
  });

  it("always previews a permutation", () => {
    for (let from = 0; from < 7; from++) {
      for (let target = 0; target < 7; target++) {
        for (const mode of ["insert", "swap"] as const) {
          const slots = previewSlots(7, { from, mode, target });
          expect([...slots].sort()).toEqual([0, 1, 2, 3, 4, 5, 6]);
        }
      }
    }
  });
});
