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

describe("rack drag: lift and come back down", () => {
  it("inserts, not swaps: B up, over F, down gives ACDEFBG", () => {
    // Drag B (slot 1) up off the rack: every tile returns home.
    const lifted = drive(1, [{ x: 160, y: 400 }]);
    expect(lifted).toEqual({ from: 1, mode: "lifted", target: null });
    expect(previewSlots(7, lifted)).toEqual([0, 1, 2, 3, 4, 5, 6]);

    // Sideways to above F (slot 5), then back down onto it.
    const above = nextRackDragState(lifted, { x: 320, y: 400 }, geo);
    expect(above.target).toBeNull();
    const s = nextRackDragState(above, onRack(320), geo);
    expect(s).toEqual({ from: 1, mode: "insert", target: 5 });
    // C, D, E and F slide left; B lands in F's old slot.
    expect(previewSlots(7, s)).toEqual([0, 5, 1, 2, 3, 4, 6]);
    expect(applyRackDrag(rack("ABCDEFG"), s)).toEqual(rack("ACDEFBG"));
  });

  it("gives the same result as dragging straight along the rack", () => {
    const straight = drive(1, [onRack(320)]);
    const viaLift = drive(1, [
      { x: 160, y: 400 },
      { x: 320, y: 400 },
      onRack(320),
    ]);
    expect(viaLift).toEqual(straight);
  });

  it("inserts leftward too (IHEASES: right E down on H gives IEHEASS)", () => {
    const s = drive(5, [{ x: 300, y: 400 }, onRack(165)]);
    expect(applyRackDrag(rack("IHEASES"), s)).toEqual(rack("IEHEASS"));
  });

  it("keeps sliding tiles as it moves along after coming back down", () => {
    const s = drive(5, [{ x: 300, y: 400 }, onRack(165), onRack(240)]);
    expect(s).toEqual({ from: 5, mode: "insert", target: 3 });
    expect(applyRackDrag(rack("IHEASES"), s)).toEqual(rack("IHEEASS"));
  });

  it("is a no-op when returning to the origin slot", () => {
    const s = drive(5, [{ x: 300, y: 600 }, onRack(318)]);
    expect(s.target).toBe(5);
    expect(applyRackDrag(rack("IHEASES"), s)).toEqual(rack("IHEASES"));
  });

  it("clamps to the ends when coming down beside the rack", () => {
    expect(drive(2, [{ x: 200, y: 400 }, onRack(40)]).target).toBe(0);
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

    const viaLift = drive(0, [{ x: 0, y: 0 }, onRack(200)]);
    expect(applyRackDrag(r, viaLift)).toEqual(rack("B_AD_FG"));
  });

  it("always previews a permutation", () => {
    for (let from = 0; from < 7; from++) {
      for (let target = 0; target < 7; target++) {
        const slots = previewSlots(7, { from, mode: "insert", target });
        expect([...slots].sort()).toEqual([0, 1, 2, 3, 4, 5, 6]);
      }
    }
  });
});

describe("rack drag: short racks", () => {
  // Three known tiles, centred in the rack (annotated games).
  const geo3: RackGeometry = { ...geo, slotCenters: [200, 240, 280] };
  const drive3 = (from: number, points: { x: number; y: number }[]) =>
    points.reduce<RackDragState>(
      (s, p) => nextRackDragState(s, p, geo3),
      initialRackDragState(from),
    );

  it("slides within a three-tile rack and clamps at its ends", () => {
    const s = drive3(2, [onRack(100)]);
    expect(s.target).toBe(0);
    expect(applyRackDrag(rack("ABC"), s)).toEqual(rack("CAB"));
    expect(drive3(0, [onRack(900)]).target).toBe(2);
  });

  it("inserts after a lift within a three-tile rack", () => {
    const s = drive3(0, [{ x: 200, y: 400 }, onRack(280)]);
    expect(applyRackDrag(rack("ABC"), s)).toEqual(rack("BCA"));
  });

  it("is a no-op for a single tile", () => {
    const geo1: RackGeometry = { ...geo, slotCenters: [240] };
    const s = [onRack(100), onRack(400)].reduce<RackDragState>(
      (st, p) => nextRackDragState(st, p, geo1),
      initialRackDragState(0),
    );
    expect(applyRackDrag(rack("A"), s)).toEqual(rack("A"));
  });

  it("leaves the rack alone if it shrank during the drag", () => {
    const s = drive(6, [onRack(120)]);
    expect(applyRackDrag(rack("ABC"), s)).toEqual(rack("ABC"));
  });
});
