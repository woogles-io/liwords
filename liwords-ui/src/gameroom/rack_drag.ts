// Pure logic for rearranging tiles within the rack while dragging.
//
// Two gestures:
// - insert: drag sideways along the rack; tiles between the origin and the
//   pointer slide one slot toward the origin (like a sortable list).
// - swap: drag off the rack vertically ("lift"), then come back down onto it;
//   the tile under the pointer trades places with the dragged tile.
//
// Nothing here touches the DOM, so the rack can render a preview from
// previewSlots() and commit applyRackDrag() once, when the drag ends.

export type RackDragMode = "insert" | "lifted" | "swap";

export type RackDragState = {
  from: number;
  mode: RackDragMode;
  target: number | null;
};

export type RackGeometry = {
  top: number;
  bottom: number;
  // Horizontal centres of each slot, measured before any preview transform.
  slotCenters: number[];
  slotWidth: number;
  tileHeight: number;
};

export type Point = { x: number; y: number };

// How far (in tile heights) the pointer may stray above or below the rack
// before the tile counts as lifted off it.
const LIFT_TOLERANCE = 0.5;

export const initialRackDragState = (from: number): RackDragState => ({
  from,
  mode: "insert",
  target: from,
});

const nearestSlot = (x: number, geo: RackGeometry): number => {
  let best = 0;
  let bestDist = Infinity;
  geo.slotCenters.forEach((c, i) => {
    const d = Math.abs(x - c);
    if (d < bestDist) {
      bestDist = d;
      best = i;
    }
  });
  return best;
};

export const nextRackDragState = (
  prev: RackDragState,
  pointer: Point,
  geo: RackGeometry,
): RackDragState => {
  const tolerance = LIFT_TOLERANCE * geo.tileHeight;
  const inBand =
    pointer.y >= geo.top - tolerance && pointer.y <= geo.bottom + tolerance;

  if (!inBand) {
    return { from: prev.from, mode: "lifted", target: null };
  }

  if (prev.mode === "insert") {
    return {
      from: prev.from,
      mode: "insert",
      target: nearestSlot(pointer.x, geo),
    };
  }

  // Once lifted, coming back onto the rack swaps rather than inserts.
  const n = geo.slotCenters.length;
  const half = geo.slotWidth / 2;
  const overRack =
    n > 0 &&
    pointer.x >= geo.slotCenters[0] - half &&
    pointer.x <= geo.slotCenters[n - 1] + half;
  return {
    from: prev.from,
    mode: "swap",
    target: overRack ? nearestSlot(pointer.x, geo) : null,
  };
};

// Returns the resulting order as a list of original indices: element k is the
// original index of the tile that ends up in slot k.
const resultingOrder = (len: number, state: RackDragState): number[] => {
  const order = Array.from({ length: len }, (_, i) => i);
  const { from, target, mode } = state;
  if (target === null || target === from || from < 0 || from >= len) {
    return order;
  }
  if (target < 0 || target >= len) {
    return order;
  }
  if (mode === "swap") {
    order[from] = target;
    order[target] = from;
  } else if (mode === "insert") {
    order.splice(from, 1);
    order.splice(target, 0, from);
  }
  return order;
};

// For each original index, the slot it is displayed in.
export const previewSlots = (len: number, state: RackDragState): number[] => {
  const order = resultingOrder(len, state);
  const slots = new Array<number>(len);
  order.forEach((orig, slot) => {
    slots[orig] = slot;
  });
  return slots;
};

export const applyRackDrag = <T>(rack: T[], state: RackDragState): T[] =>
  resultingOrder(rack.length, state).map((i) => rack[i]);

export const sameRackDragState = (
  a: RackDragState | null,
  b: RackDragState | null,
): boolean =>
  a === b ||
  (a !== null &&
    b !== null &&
    a.from === b.from &&
    a.mode === b.mode &&
    a.target === b.target);
