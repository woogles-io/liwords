import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { Store } from "../store/store";
import { LabPosition } from "./analyzer_lab_panels";
import { InferPanel, Inference, inferenceRows } from "./lab_infer_panel";

// The lab's worker, faked: an inference answers what the test sets.
const lab = vi.hoisted(() => ({
  calls: [] as Array<{ op: string; args?: { request?: string } }>,
  inference: undefined as object | undefined,
}));
vi.mock("../wasm/lab", () => ({
  labCall: async (request: { op: string; args?: { request?: string } }) => {
    lab.calls.push(request);
    if (request.op === "infer") return lab.inference;
    return undefined;
  },
  labParts: () => 1,
  setLabParts: () => {},
  onLabEvent: () => () => {},
}));

const A = 1;
const B = 2;
const E = 5;
const Q = 17;
const S = 19;

const position = (more: Partial<LabPosition> = {}): LabPosition => ({
  loadableKey: "CSW24",
  dim: 15,
  letters: new Array(225).fill(0),
  alphabet: StandardEnglishAlphabet,
  rack: [A, B, E],
  unseen: [A, E, S, S],
  position: {
    rack: [A, B, E],
    board: [],
    lexicon: "CSW24",
    rules: "CrosswordGame",
    scores: [300, 290],
    turn: 0,
    zero_turns: 0,
    pass_turns: 0,
  },
  leave: "CSW24",
  challengeable: false,
  challengeBonus: 5,
  exchanges: [0, 0],
  lastMove: {
    board: [[0]],
    play: {
      action: "play",
      down: false,
      lane: 7,
      idx: 7,
      word: [B, E],
      score: 8,
    },
  },
  names: ["andy", "wolges"],
  key: "k",
  ...more,
});

// E likelier, Q less likely, S unmoved, A none unseen
const inference = (): Inference => {
  const tiles = Array.from({ length: 27 }, () => ({ held: 0, prior: 0 }));
  tiles[E] = { held: 0.772, prior: 0.579 };
  tiles[Q] = { held: 0.05, prior: 0.12 };
  tiles[S] = { held: 0.3, prior: 0.3 };
  return { count: 3457, leaves: [{ tiles: [E, S], weight: 0.0418 }], tiles };
};

const inferRequests = () =>
  lab.calls
    .filter((c) => c.op === "infer")
    .map((c) => JSON.parse(c.args!.request!));

beforeEach(() => {
  lab.calls.length = 0;
  lab.inference = inference();
});

afterEach(cleanup);

describe("inferenceRows", () => {
  it("puts the tiles the inference makes likeliest first, leaving out those none unseen", () => {
    expect(inferenceRows(inference().tiles).map((t) => t.tile)).toEqual([
      E,
      S,
      Q,
    ]);
  });
});

describe("InferPanel", () => {
  it("infers from the opponent's last move, on its own, and shows each tile both ways", async () => {
    render(
      <Store>
        <InferPanel lab={position()} onClose={() => {}} />
      </Store>,
    );
    await screen.findByText(
      "From their last move, 3457 leaves they may have kept; the likeliest: ES 4.2%.",
    );
    expect(inferRequests()).toEqual([
      expect.objectContaining({
        fuzz: 3.105,
        list: 10,
        play: expect.anything(),
      }),
    ]);
    const rows = [
      ...document.querySelectorAll(".lab-infer-tiles tbody tr"),
    ].map((r) => [...r.querySelectorAll("td")].map((d) => d.textContent));
    expect(rows).toEqual([
      ["E", "77.2%", "57.9%", "+19.3"],
      ["S", "30.0%", "30.0%", "+0.0"],
      ["Q", "5.0%", "12.0%", "-7.0"],
    ]);
  });

  it("says so when the inference moves no tile, with no table", async () => {
    const tiles = Array.from({ length: 27 }, () => ({ held: 0.2, prior: 0.2 }));
    lab.inference = { count: 1, leaves: [{ tiles: [], weight: 1 }], tiles };
    render(
      <Store>
        <InferPanel lab={position()} onClose={() => {}} />
      </Store>,
    );
    await screen.findByText(
      "From their last move, 1 leave they may have kept; the likeliest: (none) 100.0%. That moves no tile: each is as likely as by the tiles unseen alone.",
    );
    expect(document.querySelector(".lab-infer-tiles")).toBeNull();
  });

  it("infers again with the fuzz asked", async () => {
    render(
      <Store>
        <InferPanel lab={position()} onClose={() => {}} />
      </Store>,
    );
    await waitFor(() => expect(inferRequests()).toHaveLength(1));
    fireEvent.change(screen.getByLabelText("Fuzz"), { target: { value: "1" } });
    await waitFor(() => expect(inferRequests().at(-1).fuzz).toBe(1));
  });

  it("says when there is nothing to infer from", () => {
    render(
      <Store>
        <InferPanel lab={position({ lastMove: null })} onClose={() => {}} />
      </Store>,
    );
    expect(screen.getByText(/^Nothing to infer:/)).toBeTruthy();
    expect(inferRequests()).toHaveLength(0);
  });
});
