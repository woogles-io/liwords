import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { useRef } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import {
  GameEventSchema,
  GameEvent_Direction,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { Store } from "../store/store";
import { LabEvent } from "../wasm/lab";
import { AnalyzerMove } from "./analyzer";
import {
  EndgamePanel,
  LabPosition,
  SimPanel,
  lastMoveOf,
  useLabRoom,
  useLabSim,
} from "./analyzer_lab_panels";

// The lab's worker, faked: a job made is job 1, a sim holds each play as it
// is given, an analysis finds the plays a test sets, and the test sends the
// steps' answers.
const lab = vi.hoisted(() => ({
  calls: [] as Array<{ op: string; args?: object }>,
  listeners: new Set<(e: LabEvent) => void>(),
  top: [] as Array<object>,
  // the workers a sim runs on
  parts: 1,
  // what an inference of the opponent's leave finds
  inferred: {
    count: 120,
    leaves: [
      { tiles: [1, 5], weight: 0.25 },
      { tiles: [], weight: 0.125 },
    ],
  },
}));
vi.mock("../wasm/lab", () => ({
  labCall: async (request: { op: string; args?: { plays?: string } }) => {
    lab.calls.push(request);
    if (request.op === "analyze") return lab.top;
    if (request.op === "endgame") return 1;
    if (request.op === "sim") {
      const args = request.args as { plays: string; infer?: string };
      const plays = JSON.parse(args.plays) as Array<object>;
      return {
        job: 1,
        added: plays.map((play) => ({ play, was: "absent" })),
        ...(args.infer ? { inferred: lab.inferred } : {}),
      };
    }
    if (request.op === "set") {
      const args = request.args as { infer?: string };
      return args.infer ? { inferred: lab.inferred } : {};
    }
    if (["add", "remove", "pause", "resume"].includes(request.op)) {
      const plays = JSON.parse(request.args!.plays!) as Array<object>;
      return plays.map((play) => ({ play, was: "absent" }));
    }
    return undefined;
  },
  setLabParts: (n: number) => {
    lab.parts = n;
  },
  onLabEvent: (listener: (e: LabEvent) => void) => {
    lab.listeners.add(listener);
    return () => {
      lab.listeners.delete(listener);
    };
  },
}));

const emit = (e: LabEvent) =>
  act(() => lab.listeners.forEach((listener) => listener(e)));

// The moves a click on a row puts on the board.
const board = vi.hoisted(() => ({
  placed: [] as Array<{ jsonKey: string }>,
}));
vi.mock("./analyzer", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./analyzer")>();
  return {
    ...actual,
    usePlaceMoveCallback: () => (move: { jsonKey: string }) =>
      board.placed.push(move),
  };
});

// The page's clock, which stands still unless a test moves it.
const clock = vi.hoisted(() => ({ now: 0 }));
vi.mock("./analyzer_lab", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./analyzer_lab")>();
  return { ...actual, labNow: () => clock.now };
});

// The page's own wasm, faked: play_score takes every play, scoring a placed
// one 10, each with a static equity of 4, unless a test says otherwise.
const page = vi.hoisted(() => ({
  scored: [] as Array<object>,
  answer: undefined as object | undefined,
}));
vi.mock("../wasm/loader", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../wasm/loader")>();
  return {
    ...actual,
    getWolges: async () => ({
      play_score: (request: string) => {
        const [play] = JSON.parse(request).plays;
        page.scored.push(play);
        return JSON.stringify([
          page.answer ?? {
            ...play,
            ...(play.action === "play" ? { score: 10 } : {}),
            result: "scored",
            canonical: true,
            valid: true,
            invalid_words: [],
            equity: 4,
          },
        ]);
      },
    }),
  };
});

// The tiles placed on the board, which a test sets.
const tentative = vi.hoisted(() => ({
  placedTiles: new Set<{ row: number; col: number; letter: number }>(),
}));
vi.mock("../store/store", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../store/store")>();
  return {
    ...actual,
    useTentativeTileContext: () => ({
      ...actual.useTentativeTileContext(),
      placedTiles: tentative.placedTiles,
    }),
  };
});

window.RUNTIME_CONFIGURATION = {};

const A = 1;
const B = 2;
const E = 5;
const I = 9;

// An empty board, a full rack of the player to move, and the tiles unseen.
const position = (
  unseen: Array<number>,
  more: Partial<LabPosition> = {},
): LabPosition => ({
  loadableKey: "CSW24",
  dim: 15,
  letters: new Array(225).fill(0),
  alphabet: StandardEnglishAlphabet,
  rack: [A, A, B, E, E, I, I],
  unseen,
  position: {
    rack: [A, A, B, E, E, I, I],
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
  exchanges: [1, 2],
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
  key: JSON.stringify(unseen),
  ...more,
});

// a position with tiles still in the bag
const midgame = () => position([A, B, E, E, I, I, I, I, A, A]);

const across = (row: number, col: number, word: Array<number>) => ({
  action: "play",
  down: false,
  lane: row,
  idx: col,
  word,
  score: 10,
});

const step = async (answer: object) => {
  await waitFor(() => expect(lab.calls.some((c) => c.op === "run")).toBe(true));
  emit({ job: 1, answer, done: false });
};

const bodyRows = () =>
  [...document.querySelectorAll("tbody tr")].map((r) => r.textContent);
const headRow = () => document.querySelector("thead tr")?.textContent;

// The display's frames, 8 ms apart unless a test stamps them: each callback
// requestAnimationFrame takes runs at the next frame, with that frame's stamp,
// as a browser does.
let frameTime = 0;
let frameStamps: Array<number> = [];
let lastFrameId = 0;
let dueAtNextFrame = new Map<number, FrameRequestCallback>();

beforeEach(() => {
  lab.calls.length = 0;
  lab.listeners.clear();
  lab.parts = 1;
  board.placed.length = 0;
  clock.now = 0;
  lab.top = [];
  page.scored.length = 0;
  page.answer = undefined;
  tentative.placedTiles = new Set();
  frameTime = 0;
  frameStamps = [];
  dueAtNextFrame = new Map();
  vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
    if (dueAtNextFrame.size === 0) {
      setTimeout(() => {
        frameTime = frameStamps.shift() ?? frameTime + 8;
        const due = dueAtNextFrame;
        dueAtNextFrame = new Map();
        due.forEach((c) => c(frameTime));
      }, 0);
    }
    dueAtNextFrame.set(++lastFrameId, callback);
    return lastFrameId;
  });
  vi.stubGlobal("cancelAnimationFrame", (id: number) =>
    dueAtNextFrame.delete(id),
  );
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("the endgame panel", () => {
  it("solves an empty bag and lists the line, each ply by whom and from what", async () => {
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, I, I, I, I])}
          onClose={() => {}}
        />
      </Store>,
    );
    await step({
      bag: 0,
      steps: 2,
      exact: false,
      play: across(7, 7, [B, A]),
      value: 12,
      spread: 22,
      depth: 3,
      line: [
        { play: across(7, 7, [B, A]), value: 12 },
        { play: { action: "exchange", tiles: [] }, value: -5 },
      ],
    });
    expect(screen.getByText("Endgame")).toBeInTheDocument();
    expect(
      screen.getByText("Value +12, spread +22, depth 3 after 0 s, solving"),
    ).toBeInTheDocument();
    expect(screen.getByText("andy to move: AABEEII")).toBeInTheDocument();
    expect(screen.getByText("wolges: ABEIIII")).toBeInTheDocument();
    expect(headRow()).toBe("ScoreLeaveValue");
    expect(bodyRows()).toEqual(["andy8HBA10AEEII+12", "wolgesPass0ABEIIII-5"]);
    // the endgame reads the lexicon's word graph and no leaves
    const prepare = lab.calls.find((c) => c.op === "prepare");
    expect(prepare?.args).toEqual({
      files: [
        { cacheKey: "kwg/CSW24", path: expect.stringMatching(/\/CSW24\.kwg$/) },
      ],
    });
    const made = lab.calls.find((c) => c.op === "endgame");
    const request = JSON.parse((made?.args as { request: string }).request);
    expect(request.max_memory_mb).toBe(256);
    expect(request.scores).toEqual([300, 290]);
  });

  it("weighs a tile in the bag by each tile it may be", async () => {
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, E, I, I, I, I])}
          onClose={() => {}}
        />
      </Store>,
    );
    await step({
      bag: 1,
      steps: 3,
      exact: false,
      play: across(7, 7, [B, A]),
      value: 4.875,
      spread: 14.875,
      depth: 2,
      win: 0.75,
      endgames: 40,
      moves: [3, 9],
      tiles: [
        { tile: E, count: 2, value: 6 },
        { tile: I, count: 4, value: -3 },
      ],
    });
    expect(
      screen.getByText("Endgame, one tile in the bag"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Expected value +4.88, wins 75.0%, pass 2, 3 of 9 moves valued, 40 endgames after 0 s, solving",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Unseen: ABEEIIII")).toBeInTheDocument();
    // one tile in the bag is no question of what either player knows
    expect(screen.queryByText("Blind")).not.toBeInTheDocument();
    const made = lab.calls.find((c) => c.op === "endgame");
    const request = JSON.parse((made?.args as { request: string }).request);
    expect(request.model).toBeUndefined();
    expect(request.exchanges).toBeUndefined();
    expect(bodyRows()).toEqual([
      "andy8HBA10AEEII75.0%+4.88",
      // the lead now 10: the game won with either tile
      "8H BA, with the bag holding:",
      "E (2)wins+6",
      "I (4)wins-3",
    ]);
  });

  it("shows no value before its first step, nor any move", async () => {
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, E, I, I, I, I])}
          onClose={() => {}}
        />
      </Store>,
    );
    // a tile in the bag, before any step: no value or win yet
    await step({
      bag: 1,
      steps: 0,
      nodes: 0,
      exact: false,
      play: null,
      spread: 10,
      depth: 0,
      endgames: 0,
      tiles: [],
    });
    expect(screen.getByText("Solving...")).toBeInTheDocument();
    expect(bodyRows()).toEqual([]);
    expect(screen.queryByText(/NaN|undefined/)).not.toBeInTheDocument();
  });

  it("shows how far a step got when it ends before any move has a value", async () => {
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, E, I, I, I, I, I, I, I])}
          onClose={() => {}}
        />
      </Store>,
    );
    // four tiles in the bag, steps that valued no move yet
    await step({
      bag: 4,
      steps: 3,
      nodes: 700,
      exact: false,
      play: null,
      spread: 10,
      depth: 0,
      endgames: 40,
      moves: [0, 31],
      draws: [],
      candidates: [],
    });
    expect(
      screen.getByText(
        "Solving: first pass, 0 of 31 moves valued, 40 endgames after 0 s",
      ),
    ).toBeInTheDocument();
    expect(bodyRows()).toEqual([]);
    expect(screen.queryByText(/NaN|undefined/)).not.toBeInTheDocument();
    // still stepping
    expect(screen.getByRole("button", { name: "Stop" })).toBeInTheDocument();
  });

  it("shows the best of the moves valued so far in a pre-endgame's first pass", async () => {
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, E, I, I, I, I])}
          onClose={() => {}}
        />
      </Store>,
    );
    await step({
      bag: 1,
      steps: 1,
      nodes: 50,
      exact: false,
      play: across(7, 7, [B, A]),
      value: 2,
      spread: 12,
      depth: 0,
      win: 0.5,
      endgames: 6,
      moves: [2, 9],
      tiles: [],
    });
    expect(
      screen.getByText(
        "Expected value +2.00, wins 50.0%, first pass, 2 of 9 moves valued, 6 endgames after 0 s, solving",
      ),
    ).toBeInTheDocument();
    expect(bodyRows()).toEqual(["andy8HBA10AEEII50.0%+2.00"]);
  });

  it("lists every move valued with a tile in the bag, each with its tiles, and shows the tiles of the one picked", async () => {
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, E, I, I, I, I])}
          onClose={() => {}}
        />
      </Store>,
    );
    await step({
      bag: 1,
      steps: 3,
      exact: true,
      play: across(7, 7, [B, A]),
      value: 4,
      spread: 14,
      depth: 127,
      win: 1,
      endgames: 40,
      moves: [9, 9],
      tiles: [
        { tile: E, count: 2, value: 6 },
        { tile: I, count: 4, value: 3 },
      ],
      candidates: [
        {
          play: across(7, 7, [B, A]),
          value: 4,
          exact: true,
          tiles: [
            { tile: E, count: 2, value: 6 },
            { tile: I, count: 4, value: 3 },
          ],
        },
        {
          play: { action: "exchange", tiles: [] },
          value: -9,
          exact: true,
          tiles: [
            { tile: E, count: 2, value: -10 },
            { tile: I, count: 4, value: -9 },
          ],
        },
      ],
    });
    // the lead now 10: the pass wins with I and ties with E
    expect(bodyRows()).toEqual([
      "andy8HBA10AEEII100.0%+4.00",
      "Pass0AABEEII83.3%-9.00",
      "8H BA, with the bag holding:",
      "E (2)wins+6",
      "I (4)wins+3",
    ]);
    fireEvent.click(screen.getByRole("button", { name: "Draws of Pass" }));
    expect(bodyRows()).toEqual([
      "andy8HBA10AEEII100.0%+4.00",
      "Pass0AABEEII83.3%-9.00",
      "Pass, with the bag holding:",
      "E (2)ties-10",
      "I (4)wins-9",
    ]);
    // picking is not placing
    expect(board.placed).toEqual([]);
  });

  it("marks the values not yet searched to the end", async () => {
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, E, I, I, I, I, I])}
          onClose={() => {}}
        />
      </Store>,
    );
    await step({
      bag: 2,
      steps: 1,
      nodes: 90,
      exact: false,
      play: across(7, 7, [B, A]),
      value: 3.5,
      spread: 13.5,
      depth: 1,
      win: 0.6,
      endgames: 12,
      moves: [9, 9],
      draws: [],
      candidates: [
        { play: across(7, 7, [B, A]), win: 0.6, value: 3.5, exact: true },
        {
          play: { action: "exchange", tiles: [] },
          win: 0.25,
          value: -8,
          exact: false,
        },
      ],
    });
    expect(bodyRows()).toEqual([
      "andy8HBA10AEEII60.0%+3.50",
      "Pass0AABEEII25.0%~-8.00",
    ]);
    expect(
      screen.getByText(
        "Expected value +3.50, wins 60.0%, pass 1, 9 of 9 moves valued, 12 endgames after 0 s, solving",
      ),
    ).toBeInTheDocument();
    // the best an estimate too
    await step({
      bag: 2,
      steps: 2,
      nodes: 190,
      exact: false,
      play: across(7, 7, [B, A]),
      value: 3,
      spread: 13,
      depth: 2,
      win: 0.6,
      endgames: 30,
      moves: [9, 9],
      draws: [],
      candidates: [
        { play: across(7, 7, [B, A]), win: 0.6, value: 3, exact: false },
      ],
    });
    expect(
      screen.getByText(
        "Expected value ~+3.00, wins 60.0%, pass 2, 9 of 9 moves valued, 30 endgames after 0 s, solving",
      ),
    ).toBeInTheDocument();
  });

  it("shows each play's own sets of tiles, with the chance to win with each, and with both racks known what it then draws", async () => {
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, E, I, I, I, I, I])}
          onClose={() => {}}
        />
      </Store>,
    );
    const draws = [
      {
        tiles: [E, I],
        count: 10,
        value: 5,
        win: 0.9,
        drawn: [
          { tiles: [E], count: 4, value: 7, win: 1 },
          { tiles: [I], count: 6, value: 3.67, win: 0.83 },
        ],
      },
      { tiles: [I, I], count: 1, value: -12, win: 0, drawn: [] },
    ];
    await step({
      bag: 2,
      steps: 4,
      nodes: 900,
      exact: true,
      play: across(7, 7, [B, A]),
      value: 3.5,
      spread: 13.5,
      depth: 127,
      win: 0.82,
      endgames: 120,
      moves: [9, 9],
      draws,
      candidates: [
        {
          play: across(7, 7, [B, A]),
          win: 0.82,
          value: 3.5,
          exact: true,
          draws,
        },
        {
          play: { action: "exchange", tiles: [] },
          win: 0.25,
          value: -8,
          exact: true,
          draws: [
            { tiles: [E, I], count: 10, value: -7, win: 0.3 },
            { tiles: [I, I], count: 1, value: -18, win: 0 },
          ],
        },
      ],
    });
    expect(bodyRows()).toEqual([
      "andy8HBA10AEEII82.0%+3.50",
      "Pass0AABEEII25.0%-8.00",
      "8H BA, with the bag holding:",
      "EI (10 deals)90.0%+5.00",
      "draws E (4 deals)100.0%+7.00",
      "draws I (6 deals)83.0%+3.67",
      "II (1 deal)0.0%-12.00",
    ]);
    fireEvent.click(screen.getByRole("button", { name: "Draws of Pass" }));
    expect(bodyRows().slice(2)).toEqual([
      // a pass draws nothing
      "Pass, with the bag holding:",
      "EI (10 deals)30.0%-7.00",
      "II (1 deal)0.0%-18.00",
    ]);
  });

  it("solves a pre-endgame of two tiles in the bag, every play valued, blind or with both racks known", async () => {
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, E, I, I, I, I, I])}
          onClose={() => {}}
        />
      </Store>,
    );
    await step({
      bag: 2,
      steps: 4,
      nodes: 900,
      exact: true,
      play: across(7, 7, [B, A]),
      value: 3.5,
      spread: 13.5,
      depth: 127,
      win: 0.6,
      endgames: 120,
      moves: [9, 9],
      draws: [
        { tiles: [E, I], count: 10, value: 5 },
        { tiles: [I, I], count: 1, value: -12 },
      ],
      candidates: [
        { play: across(7, 7, [B, A]), win: 0.6, value: 3.5 },
        { play: { action: "exchange", tiles: [] }, win: 0.25, value: -8 },
      ],
    });
    expect(
      screen.getByText("Pre-endgame, 2 tiles in the bag"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Expected value +3.50, wins 60.0%, searched to the end, 120 endgames after 0 s, exact",
      ),
    ).toBeInTheDocument();
    expect(headRow()).toBe("ScoreLeaveWinValue");
    expect(bodyRows()).toEqual([
      "andy8HBA10AEEII60.0%+3.50",
      "Pass0AABEEII25.0%-8.00",
      // the lead now 10: won with EI in the bag, lost with II
      "8H BA, with the bag holding:",
      "EI (10 deals)wins+5.00",
      "II (1 deal)loses-12.00",
    ]);
    // blind by default, with each seat's exchanges
    const requests = () =>
      lab.calls
        .filter((c) => c.op === "endgame")
        .map((c) => JSON.parse((c.args as { request: string }).request));
    expect(requests()).toHaveLength(1);
    expect(requests()[0].model).toBe("info_sets");
    expect(requests()[0].exchanges).toEqual([1, 2]);
    // estimates first, labeled, then exact
    expect(requests()[0].estimates).toBe(true);
    // both racks known makes another search
    fireEvent.click(screen.getByText("Racks known"));
    await waitFor(() => expect(requests()).toHaveLength(2));
    expect(requests()[1].model).toBe("random_draws");
  });

  it("shows how long it has solved, the time going on between its steps", async () => {
    const endgame = position([A, B, E, I, I, I, I]);
    // made anew each time, so that a rerender renders the panel again
    const panel = () => (
      <Store>
        <EndgamePanel lab={endgame} onClose={() => {}} />
      </Store>
    );
    const { rerender } = render(panel());
    // solving from the page's clock at 0
    await screen.findByRole("button", { name: "Stop" });
    const answer = {
      bag: 0,
      steps: 2,
      exact: false,
      play: across(7, 7, [B, A]),
      value: 12,
      spread: 22,
      depth: 3,
      line: [{ play: across(7, 7, [B, A]), value: 12 }],
    };
    clock.now = 999;
    emit({ job: 1, answer, done: false });
    expect(
      screen.getByText("Value +12, spread +22, depth 3 after 0 s, solving"),
    ).toBeInTheDocument();
    // a step can take more than a second, and the second shows as it passes
    clock.now = 1000;
    expect(
      await screen.findByText(
        "Value +12, spread +22, depth 3 after 1 s, solving",
      ),
    ).toBeInTheDocument();
    // solved, the time stops
    clock.now = 2500;
    emit({ job: 1, answer: { ...answer, depth: 4, exact: true }, done: true });
    clock.now = 9000;
    rerender(panel());
    expect(
      screen.getByText("Value +12, spread +22, depth 4 after 2 s, exact"),
    ).toBeInTheDocument();
  });

  it("solves nothing once the other player has gone out, or without the full rack of the player to move", async () => {
    const { unmount } = render(
      <Store>
        <EndgamePanel lab={position([])} onClose={() => {}} />
      </Store>,
    );
    expect(
      screen.getByText("The other player has gone out: the game is over."),
    ).toBeInTheDocument();
    unmount();
    // the play that went out can still come off
    const { unmount: unmountChallengeable } = render(
      <Store>
        <EndgamePanel
          lab={position([], { challengeable: true })}
          onClose={() => {}}
        />
      </Store>,
    );
    expect(
      screen.getByText(
        "The other player has gone out, but the play can still be challenged.",
      ),
    ).toBeInTheDocument();
    unmountChallengeable();
    render(
      <Store>
        <EndgamePanel
          lab={position([A, B, E, E, I, I, I, I, I], { rack: [A, B] })}
          onClose={() => {}}
        />
      </Store>,
    );
    expect(
      screen.getByText("Needs the full rack of the player to move."),
    ).toBeInTheDocument();
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(lab.calls).toEqual([]);
  });
});

// A move as the analyzer lists it.
const listed = (
  play: object,
  more: Partial<AnalyzerMove> = {},
): AnalyzerMove => ({
  jsonKey: JSON.stringify(play),
  displayMove: JSON.stringify(play),
  coordinates: "",
  leave: [],
  leaveWithGaps: [],
  score: 10,
  equity: 0,
  row: 7,
  col: 7,
  vertical: false,
  tiles: [],
  isExchange: false,
  ...more,
});

// The sim panel as the analyzer has it: the sim kept by the analyzer, the
// panel shown or not.
const Sim = (props: {
  lab: LabPosition;
  moves: Array<AnalyzerMove> | null;
  shown?: boolean;
}) => {
  const shown = props.shown ?? true;
  const sim = useLabSim(props.lab, props.moves, shown);
  return shown ? (
    <SimPanel
      lab={props.lab}
      moves={props.moves}
      sim={sim}
      onClose={() => {}}
    />
  ) : null;
};

const ba = { action: "play", down: false, lane: 7, idx: 7, word: [B, A] };
const ab = { action: "play", down: false, lane: 7, idx: 7, word: [A, B] };
const ai = { action: "play", down: false, lane: 7, idx: 7, word: [A, I] };
const be = { action: "play", down: false, lane: 7, idx: 7, word: [B, E] };

const stat = (mean: number) => ({ mean, variance: 0 });
const candidate = (
  play: object,
  win: number,
  more: { n?: number; equity?: number; bingo?: number; state?: string } = {},
) => ({
  play,
  state: more.state ?? "running",
  n: more.n ?? 32,
  win: stat(win),
  equity: stat(more.equity ?? 0),
  plies: [{ n: more.n ?? 32, score: stat(30), bingo: more.bingo ?? 0 }],
});

const start = async () => {
  act(() => screen.getByRole("button", { name: "Start" }).click());
  await waitFor(() => expect(lab.calls.some((c) => c.op === "run")).toBe(true));
};

// BE placed across from the middle square
const placeBE = () => {
  tentative.placedTiles = new Set([
    { row: 7, col: 7, letter: B },
    { row: 7, col: 8, letter: E },
  ]);
};

// A tap on a row's button, which asks before it acts.
const tap = (button: HTMLElement) => {
  fireEvent.pointerDown(button, { pointerType: "touch" });
  fireEvent.pointerUp(button, { pointerType: "touch" });
  fireEvent.click(button, { detail: 1 });
};

// The question a tap asked, answered with the button named.
const reply = (name: string) =>
  act(() =>
    within(document.querySelector(".ant-popconfirm") as HTMLElement)
      .getByRole("button", { name })
      .click(),
  );

describe("the sim panel", () => {
  it("lists the moves by static equity until the sim starts, then sims them", async () => {
    const down = { action: "play", down: true, lane: 7, idx: 7, word: [B, E] };
    render(
      <Store>
        <Sim
          lab={midgame()}
          moves={[
            listed(ba, { equity: 3 }),
            listed(ab, { chosen: true, valid: true, equity: 8 }),
            listed(down, { chosen: true, valid: false, equity: 20 }),
          ]}
        />
      </Store>,
    );
    // the one played among them, a phony too
    expect(
      screen.getByText("Static equity until the sim starts."),
    ).toBeInTheDocument();
    expect(bodyRows()).toEqual([
      `${JSON.stringify(down)}+20.00*`,
      `${JSON.stringify(ab)}+8.00`,
      `${JSON.stringify(ba)}+3.00`,
    ]);
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(lab.calls).toEqual([]);

    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    const { plays, request } = sim?.args as { plays: string; request: string };
    expect(JSON.parse(plays)).toEqual([ba, ab, down]);
    expect(JSON.parse(request).leave).toBe("CSW24");
    // the plies the lab's own, always sent
    expect(JSON.parse(request).plies).toBe(4);
    // each step as long as a frame
    expect(lab.calls.find((c) => c.op === "run")?.args).toEqual({
      job: 1,
      ms: 8,
      alpha: 0.05,
    });

    // shown by the chance to win
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.4, { equity: 9, bingo: 0.25 }),
          candidate(ab, 0.6, { equity: 3, bingo: 0.125 }),
        ],
      },
      done: false,
    });
    // with the plies its rollouts play, as the sim's answer has them
    expect(
      screen.getByText("32 iterations after 0 s, 2 running; 1 ply"),
    ).toBeInTheDocument();
    expect(headRow()).toBe("WinEquityBingoN");
    expect(bodyRows()).toEqual([
      `${JSON.stringify(ab)}60.0%+3.0012.5%32`,
      `${JSON.stringify(ba)}40.0%+9.0025.0%32`,
    ]);
  });

  it("steps as long as a frame, past frames stamped as the one before", async () => {
    // frames dropped: three stamped alike, then the next
    frameStamps = [100, 100, 100, 116.5];
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    expect(lab.calls.find((c) => c.op === "run")?.args).toEqual({
      job: 1,
      ms: 16.5,
      alpha: 0.05,
    });
  });

  it("sims nothing once the other player has gone out, or with one move", async () => {
    const { unmount } = render(
      <Store>
        <Sim lab={position([])} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    expect(
      screen.getByText("The other player has gone out: the game is over."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start" })).toBeDisabled();
    unmount();
    const { unmount: unmountChallengeable } = render(
      <Store>
        <Sim
          lab={position([], { challengeable: true })}
          moves={[listed(ba), listed(ab)]}
        />
      </Store>,
    );
    expect(
      screen.getByText(
        "The other player has gone out, but the play can still be challenged.",
      ),
    ).toBeInTheDocument();
    unmountChallengeable();
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba)]} />
      </Store>,
    );
    expect(
      screen.getByText("Needs two or more moves to sim."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start" })).toBeDisabled();
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(lab.calls).toEqual([]);
  });

  it("stays stopped once stopped, though a step under way answers after", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    act(() => screen.getByRole("button", { name: "Stop" }).click());
    expect(lab.calls[lab.calls.length - 1]).toEqual({ op: "stop" });
    // the step that was under way when the stop went out
    emit({
      job: 1,
      answer: {
        iterations: 40,
        candidates: [
          candidate(ba, 0.5, { n: 40 }),
          candidate(ab, 0.4, { n: 40 }),
        ],
      },
      done: false,
    });
    expect(
      screen.getByText("40 iterations after 0 s, 2 running, stopped; 1 ply"),
    ).toBeInTheDocument();
    act(() => screen.getByRole("button", { name: "Resume" }).click());
    await waitFor(() =>
      expect(lab.calls[lab.calls.length - 1]).toEqual({
        op: "run",
        args: { job: 1, ms: 8, alpha: 0.05 },
      }),
    );
    expect(screen.getByRole("button", { name: "Stop" })).toBeInTheDocument();
  });

  it("shows how long the sim has run, its runs added up, the time going on between its answers", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    // running from the page's clock at 0
    await screen.findByRole("button", { name: "Stop" });
    const answer = (iterations: number) =>
      emit({
        job: 1,
        answer: {
          iterations,
          candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
        },
        done: false,
      });
    clock.now = 2999;
    answer(32);
    expect(
      screen.getByText(/^32 iterations after 2 s, 2 running/),
    ).toBeInTheDocument();
    // with no answer since, the next second shows as it passes
    clock.now = 3000;
    expect(
      await screen.findByText(/^32 iterations after 3 s, 2 running/),
    ).toBeInTheDocument();
    clock.now = 4500;
    act(() => screen.getByRole("button", { name: "Stop" }).click());
    expect(
      screen.getByText(/^32 iterations after 4 s, 2 running, stopped/),
    ).toBeInTheDocument();
    // the time it stood stopped is not run
    clock.now = 60000;
    act(() => screen.getByRole("button", { name: "Resume" }).click());
    await screen.findByRole("button", { name: "Stop" });
    clock.now = 61600;
    answer(64);
    expect(
      screen.getByText(/^64 iterations after 6 s, 2 running/),
    ).toBeInTheDocument();
    // minutes and hours, each to the second
    clock.now = 60000 + 125000 - 4500;
    answer(96);
    expect(
      screen.getByText(/^96 iterations after 2 min 5 s, 2 running/),
    ).toBeInTheDocument();
    clock.now = 60000 + 3725000 - 4500;
    answer(128);
    expect(
      screen.getByText(/^128 iterations after 1 h 2 min 5 s, 2 running/),
    ).toBeInTheDocument();
  });

  it("keeps the rows' order while the pointer is in them", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    const table = document.querySelector(".lab-panel table")!;
    fireEvent.pointerMove(table, { pointerType: "mouse" });
    // AB now leads, but the rows stay put under the pointer
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.3, { n: 64 }),
          candidate(ab, 0.6, { n: 64 }),
        ],
      },
      done: false,
    });
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ba)}30.0%0.000.0%64`);
    fireEvent.pointerLeave(table);
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ab)}60.0%0.000.0%64`);
  });

  it("holds the rows for no mouse the page says came over them without moving", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    // as the page says of a mouse once what was under it has gone
    fireEvent.pointerOver(document.querySelector(".lab-panel tbody td")!, {
      pointerType: "mouse",
    });
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.3, { n: 64 }),
          candidate(ab, 0.6, { n: 64 }),
        ],
      },
      done: false,
    });
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ab)}60.0%0.000.0%64`);
  });

  it("places the move a press began on, though the rows take the sim's order before its click", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    // a touch holds the rows while it presses, and lifts before its click
    const row = document.querySelector(".lab-panel tbody tr")!;
    fireEvent.pointerDown(row, { pointerType: "touch" });
    fireEvent.pointerUp(row, { pointerType: "touch" });
    // AB now leads, and the rows take that order before the click comes
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.3, { n: 64 }),
          candidate(ab, 0.6, { n: 64 }),
        ],
      },
      done: false,
    });
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ab)}60.0%0.000.0%64`);
    fireEvent.click(document.querySelector(".lab-panel tbody tr")!, {
      detail: 1,
    });
    expect(board.placed.map((m) => m.jsonKey)).toEqual([JSON.stringify(ba)]);
  });

  it("sims the phony listed unchallenged, naming the words the lexicon lacks", async () => {
    const phony = { action: "play", down: true, lane: 7, idx: 7, word: [A, B] };
    render(
      <Store>
        <Sim
          lab={midgame()}
          moves={[
            listed(ba),
            listed(ab),
            listed(phony, {
              displayMove: "SARKIER",
              valid: false,
              invalid_words: ["CONDS"],
            }),
          ]}
        />
      </Store>,
    );
    // as the analyzer shows one: its words starred under it, its equity too
    expect(bodyRows()).toContain("SARKIER(CONDS*)0.00*");
    expect(bodyRows()).toHaveLength(3);
    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    expect(JSON.parse((sim?.args as { plays: string }).plays)).toEqual([
      ba,
      ab,
      phony,
    ]);
  });

  it("prunes at the alpha set, and infers at the fuzz set, listing the leaves asked", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    fireEvent.click(screen.getByText("Infer their leave"));
    const field = (name: string, value: string) =>
      fireEvent.change(screen.getByRole("spinbutton", { name }), {
        target: { value },
      });
    field("Fuzz", "2");
    field("Leaves listed", "5");
    field("Alpha", "0.01");
    act(() => screen.getByRole("button", { name: "Apply" }).click());
    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    expect(JSON.parse((sim?.args as { infer: string }).infer)).toMatchObject({
      fuzz: 2,
      list: 5,
    });
    expect(lab.calls.find((c) => c.op === "run")?.args).toMatchObject({
      alpha: 0.01,
    });
  });

  it("applies no alpha outside 0 to 1", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    fireEvent.change(screen.getByRole("spinbutton", { name: "Alpha" }), {
      target: { value: "1" },
    });
    expect(screen.getByRole("button", { name: "Apply" })).toBeDisabled();
  });

  it("says how many workers the sim runs on, when more than one", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    // one until set; set, the lab runs the next sim on that many
    vi.stubGlobal("navigator", { ...navigator, hardwareConcurrency: 8 });
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    expect(screen.getByRole("spinbutton", { name: "Workers" })).toHaveValue(
      "1",
    );
    fireEvent.change(screen.getByRole("spinbutton", { name: "Workers" }), {
      target: { value: "3" },
    });
    act(() => screen.getByRole("button", { name: "Apply" }).click());
    expect(lab.parts).toBe(3);
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 96,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    expect(
      screen.getByText(
        "96 iterations after 0 s, 2 running; 1 ply; on 3 workers",
      ),
    ).toBeInTheDocument();
  });

  it("is done once fewer than two candidates are left unpruned, and runs on no further", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.9, { n: 64 }),
          candidate(ab, 0.1, { n: 48, state: "pruned" }),
        ],
      },
      done: true,
    });
    expect(
      screen.getByText("64 iterations after 0 s, 1 running, done; 1 ply"),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Resume" })).toBeDisabled();
  });

  it("keeps the moves and the sim while the panel is closed", async () => {
    const at = midgame();
    const moves = [listed(ba), listed(ab)];
    const { rerender } = render(
      <Store>
        <Sim lab={at} moves={moves} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    rerender(
      <Store>
        <Sim lab={at} moves={moves} shown={false} />
      </Store>,
    );
    expect(lab.calls[lab.calls.length - 1]).toEqual({ op: "stop" });
    rerender(
      <Store>
        <Sim lab={at} moves={moves} />
      </Store>,
    );
    expect(
      screen.getByText("32 iterations after 0 s, 2 running, stopped; 1 ply"),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Resume" })).toBeEnabled();
    expect(lab.calls.filter((c) => c.op === "sim")).toHaveLength(1);
    expect(lab.calls.some((c) => c.op === "free")).toBe(false);
  });

  it("lists the moves again, the user's changes kept, once another job takes the worker", async () => {
    placeBE();
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Add" }).click());
    await waitFor(() => expect(bodyRows()).toHaveLength(3));
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.5),
          candidate(ab, 0.4),
          candidate({ ...be, score: 10, equity: 4 }, 0.3),
        ],
      },
      done: false,
    });
    emit({ job: 1, freed: true });
    expect(
      screen.getByText("Static equity until the sim starts."),
    ).toBeInTheDocument();
    expect(bodyRows()).toHaveLength(3);
    expect(screen.getByRole("button", { name: "Start" })).toBeEnabled();
  });

  it("takes a candidate out, before a sim and in one", async () => {
    render(
      <Store>
        <Sim
          lab={midgame()}
          moves={[
            listed(ba, { equity: 3 }),
            listed(ab, { equity: 2 }),
            listed(ai, { equity: 1 }),
          ]}
        />
      </Store>,
    );
    act(() => screen.getAllByRole("button", { name: "Take out" })[1].click());
    expect(bodyRows()).toEqual([
      `${JSON.stringify(ba)}+3.00`,
      `${JSON.stringify(ai)}+1.00`,
    ]);
    expect(lab.calls).toEqual([]);

    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    expect(JSON.parse((sim?.args as { plays: string }).plays)).toEqual([
      ba,
      ai,
    ]);
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ai, 0.3)],
      },
      done: false,
    });
    act(() => screen.getAllByRole("button", { name: "Take out" })[1].click());
    // gone at once, and the sim told
    expect(bodyRows()).toEqual([`${JSON.stringify(ba)}50.0%0.000.0%32`]);
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "read")).toBe(true),
    );
    const removed = lab.calls.find((c) => c.op === "remove");
    expect(JSON.parse((removed?.args as { plays: string }).plays)).toEqual([
      ai,
    ]);
  });

  it("adds the move placed on the board, at once with its static equity, and to a sim", async () => {
    placeBE();
    const { rerender } = render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Add" }).click());
    await waitFor(() => expect(bodyRows()).toHaveLength(3));
    expect(page.scored).toEqual([{ ...be, score: 0 }]);
    expect(bodyRows()).toContain("8HBE+4.00");
    expect(lab.calls).toEqual([]);

    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    expect(JSON.parse((sim?.args as { plays: string }).plays)).toEqual([
      ba,
      ab,
      be,
    ]);
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.5),
          candidate(ab, 0.4),
          candidate({ ...be, score: 10, equity: 4 }, 0.3),
        ],
      },
      done: false,
    });

    // AI placed, added to the sim made: it shows as soon as the sim is read
    tentative.placedTiles = new Set([
      { row: 7, col: 7, letter: A },
      { row: 7, col: 8, letter: I },
    ]);
    rerender(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Add" }).click());
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "read")).toBe(true),
    );
    const add = lab.calls.find((c) => c.op === "add");
    expect(JSON.parse((add?.args as { plays: string }).plays)).toEqual([ai]);
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.5),
          candidate(ab, 0.4),
          candidate({ ...be, score: 10, equity: 4 }, 0.3),
          candidate({ ...ai, score: 10, equity: 4 }, 0, { n: 0 }),
        ],
      },
      done: false,
    });
    expect(bodyRows()).toHaveLength(4);
    expect(bodyRows()[3]).toBe("8HAI+4.000");
  });

  it("adds a phony, naming its words, and turns down a move listed already", async () => {
    placeBE();
    page.answer = {
      ...be,
      score: 10,
      result: "scored",
      canonical: true,
      valid: false,
      invalid_words: [
        [B, A, 26, 17, 16, E],
        [21, B],
      ],
      equity: 50,
    };
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Add" }).click());
    await waitFor(() =>
      expect(bodyRows()).toContain("8HBE(BAZQPE*, UB*)+50.00*"),
    );
    expect(bodyRows()).toHaveLength(2);
    page.answer = undefined;
    act(() => screen.getByRole("button", { name: "Add" }).click());
    expect(
      await screen.findByText("That move is listed already."),
    ).toBeInTheDocument();
    expect(bodyRows()).toHaveLength(2);
  });

  it("adds points below 0, a score counted short, and a phony as it was left standing", async () => {
    placeBE();
    page.answer = {
      ...be,
      score: 10,
      result: "scored",
      canonical: true,
      valid: false,
      invalid_words: [[B, E]],
      equity: 4,
    };
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba)]} />
      </Store>,
    );
    const challenged = () =>
      act(() => screen.getByRole("button", { name: "Challenged" }).click());
    challenged();
    fireEvent.change(screen.getByRole("spinbutton", { name: "Points added" }), {
      target: { value: "-2" },
    });
    act(() =>
      screen.getByRole("button", { name: "Add with the points" }).click(),
    );
    await waitFor(() => expect(bodyRows()).toHaveLength(2));
    expect(bodyRows()).toContain("8HBE -2(BE*)+2.00*");
    challenged();
    act(() =>
      screen.getByRole("button", { name: "Add challenged off" }).click(),
    );
    await waitFor(() => expect(bodyRows()).toHaveLength(3));
    // each marked a phony, with its points or challenged off
    expect(bodyRows()).toContain("8HBE (challenged off)(BE*)+4.00*");
    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    expect(JSON.parse((sim?.args as { plays: string }).plays)).toEqual([
      ba,
      { ...be, bonus: -2 },
      { ...be, challenged_off: true },
    ]);
    // 0 adds nothing
    challenged();
    fireEvent.change(screen.getByRole("spinbutton", { name: "Points added" }), {
      target: { value: "0" },
    });
    expect(
      screen.getByRole("button", { name: "Add with the points" }),
    ).toBeDisabled();
  });

  it("adds the move placed on the board with a challenge's bonus, and challenged off, each a candidate of its own", async () => {
    placeBE();
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba)]} />
      </Store>,
    );
    const challenged = () =>
      act(() => screen.getByRole("button", { name: "Challenged" }).click());
    challenged();
    // the game's own bonus to begin with
    expect(
      screen.getByRole("spinbutton", { name: "Points added" }),
    ).toHaveValue("5");
    act(() =>
      screen.getByRole("button", { name: "Add with the points" }).click(),
    );
    await waitFor(() => expect(bodyRows()).toHaveLength(2));
    expect(bodyRows()).toContain("8HBE +5+9.00");
    challenged();
    act(() =>
      screen.getByRole("button", { name: "Add challenged off" }).click(),
    );
    await waitFor(() => expect(bodyRows()).toHaveLength(3));
    // challenged off: a pass's static equity
    expect(page.scored).toContainEqual({ action: "exchange", tiles: [] });
    expect(bodyRows()).toContain("8HBE (challenged off)+4.00");

    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    const plays = JSON.parse((sim?.args as { plays: string }).plays);
    expect(plays).toEqual([
      ba,
      { ...be, bonus: 5 },
      { ...be, challenged_off: true },
    ]);
    // the sim answers with each as it holds it: the bonus in the score, and
    // the play challenged off flagged, scoring nothing
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.5),
          candidate({ ...be, score: 15 }, 0.4),
          { ...candidate({ ...be, score: 0 }, 0.3), challenged_off: true },
        ],
      },
      done: false,
    });
    expect(bodyRows()).toEqual([
      `${JSON.stringify(ba)}50.0%0.000.0%32`,
      "8HBE +540.0%0.000.0%32",
      "8HBE (challenged off)30.0%0.000.0%32",
    ]);
    // taken out as it was given
    act(() => screen.getAllByRole("button", { name: "Take out" })[1].click());
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "remove")).toBe(true),
    );
    const removed = lab.calls.find((c) => c.op === "remove");
    expect(JSON.parse((removed?.args as { plays: string }).plays)).toEqual([
      { ...be, bonus: 5 },
    ]);
  });

  it("keeps the rows' order while a row's button has the keyboard", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    const first = screen.getAllByRole("button", { name: "Take out" })[0];
    act(() => first.focus());
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.3, { n: 64 }),
          candidate(ab, 0.6, { n: 64 }),
        ],
      },
      done: false,
    });
    // the button keyed still takes out the move it was on
    act(() => first.click());
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "remove")).toBe(true),
    );
    const removed = lab.calls.find((c) => c.op === "remove");
    expect(JSON.parse((removed?.args as { plays: string }).plays)).toEqual([
      ba,
    ]);
    act(() => first.blur());
  });

  it("takes the sim's order again once the pointer that clicked a row's button has left", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab), listed(ai)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.5),
          candidate(ab, 0.4),
          candidate(ai, 0.3),
        ],
      },
      done: false,
    });
    const table = document.querySelector(".lab-panel table")!;
    fireEvent.pointerMove(table, { pointerType: "mouse" });
    const takeOut = screen.getAllByRole("button", { name: "Take out" })[2];
    // the press leaves the keyboard where it was: the button takes no focus
    expect(fireEvent.mouseDown(takeOut)).toBe(false);
    fireEvent.click(takeOut, { detail: 1 });
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "remove")).toBe(true),
    );
    fireEvent.pointerLeave(table);
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.3, { n: 64 }),
          candidate(ab, 0.6, { n: 64 }),
        ],
      },
      done: false,
    });
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ab)}60.0%0.000.0%64`);
  });

  it("lets the rows go once the row whose button had the keyboard is taken out", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab), listed(ai)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.5),
          candidate(ab, 0.4),
          candidate(ai, 0.3),
        ],
      },
      done: false,
    });
    const last = screen.getAllByRole("button", { name: "Take out" })[2];
    act(() => last.focus());
    fireEvent.keyDown(last, { key: "Enter" });
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "remove")).toBe(true),
    );
    const removed = lab.calls.find((c) => c.op === "remove");
    expect(JSON.parse((removed?.args as { plays: string }).plays)).toEqual([
      ai,
    ]);
    // its row gone, and the keyboard with it: AB, leading now, comes first
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.3, { n: 64 }),
          candidate(ab, 0.6, { n: 64 }),
        ],
      },
      done: false,
    });
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ab)}60.0%0.000.0%64`);
  });

  it("takes out the candidate a touch began on, though the rows take the sim's order before its click", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    const first = screen.getAllByRole("button", { name: "Take out" })[0];
    fireEvent.pointerDown(first, { pointerType: "touch" });
    fireEvent.pointerUp(first, { pointerType: "touch" });
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.3, { n: 64 }),
          candidate(ab, 0.6, { n: 64 }),
        ],
      },
      done: false,
    });
    // the button the touch began on holds AB now; the tap asks of BA, the
    // rows back as they were where it landed
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ab)}60.0%0.000.0%64`);
    fireEvent.click(first, { detail: 1 });
    expect(
      await screen.findByText(`Take out ${JSON.stringify(ba)}?`),
    ).toBeInTheDocument();
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ba)}30.0%0.000.0%64`);
    reply("Take out");
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "remove")).toBe(true),
    );
    const removed = lab.calls.find((c) => c.op === "remove");
    expect(JSON.parse((removed?.args as { plays: string }).plays)).toEqual([
      ba,
    ]);
  });

  it("asks before a tap takes out a move, the rows holding while it asks", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    tap(screen.getAllByRole("button", { name: "Take out" })[0]);
    expect(
      await screen.findByText(`Take out ${JSON.stringify(ba)}?`),
    ).toBeInTheDocument();
    // AB now leads, but the rows hold while the question is asked
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.3, { n: 64 }),
          candidate(ab, 0.6, { n: 64 }),
        ],
      },
      done: false,
    });
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ba)}30.0%0.000.0%64`);
    // canceled, nothing is taken out, and the rows take the sim's order
    reply("Cancel");
    expect(lab.calls.some((c) => c.op === "remove")).toBe(false);
    expect(bodyRows()).toHaveLength(2);
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ab)}60.0%0.000.0%64`);
    // a click with the mouse takes out at once
    act(() => screen.getAllByRole("button", { name: "Take out" })[0].click());
    expect(bodyRows()).toEqual([`${JSON.stringify(ba)}30.0%0.000.0%64`]);
  });

  it("lets the rows go once a tap's question goes, though the page says a mouse came over them, and puts no move on the board", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab), listed(ai)]} />
      </Store>,
    );
    await start();
    const answer = (iterations: number, wins: Array<number>) =>
      emit({
        job: 1,
        answer: {
          iterations,
          candidates: [ba, ab, ai].map((play, i) =>
            candidate(play, wins[i], { n: iterations }),
          ),
        },
        done: false,
      });
    answer(32, [0.5, 0.4, 0.3]);
    tap(screen.getAllByRole("button", { name: "Take out" })[0]);
    expect(
      await screen.findByText(`Take out ${JSON.stringify(ba)}?`),
    ).toBeInTheDocument();
    reply("Cancel");
    // the question gone, the page says a mouse came over where the tap was
    fireEvent.pointerOver(document.querySelector(".lab-panel tbody td")!, {
      pointerType: "mouse",
    });
    answer(64, [0.3, 0.6, 0.4]);
    expect(bodyRows()[0]).toBe(`${JSON.stringify(ab)}60.0%0.000.0%64`);
    // an answer to it, either one, is no click on its row
    tap(screen.getAllByRole("button", { name: "Take out" })[2]);
    expect(
      await screen.findByText(`Take out ${JSON.stringify(ba)}?`),
    ).toBeInTheDocument();
    reply("Take out");
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "remove")).toBe(true),
    );
    expect(board.placed).toEqual([]);
  });

  it("changes nothing a sim is made of while it starts", async () => {
    placeBE();
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Start" }).click());
    expect(screen.getByText("Starting...")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add" })).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Exchange or pass" }),
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "Top moves" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Sim settings" })).toBeDisabled();
    expect(screen.queryAllByRole("button", { name: "Take out" })).toEqual([]);
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "run")).toBe(true),
    );
    expect(screen.getByRole("button", { name: "Add" })).toBeEnabled();
    expect(screen.getAllByRole("button", { name: "Take out" })).toHaveLength(2);
  });

  it("adds an exchange of the tiles picked, or a pass", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Exchange or pass" }).click());
    // the rack's B, and the first of its two Es
    act(() => screen.getByRole("button", { name: "B" }).click());
    act(() => screen.getAllByRole("button", { name: "E" })[0].click());
    act(() => screen.getByRole("button", { name: "Exchange" }).click());
    await waitFor(() => expect(bodyRows()).toHaveLength(3));
    expect(page.scored).toEqual([{ action: "exchange", tiles: [B, E] }]);
    expect(bodyRows()).toContain("Exch. BE+4.00");

    act(() => screen.getByRole("button", { name: "Exchange or pass" }).click());
    act(() => screen.getByRole("button", { name: "Pass" }).click());
    await waitFor(() => expect(bodyRows()).toHaveLength(4));
    expect(page.scored[1]).toEqual({ action: "exchange", tiles: [] });
    expect(bodyRows()).toContain("Pass+4.00");
  });

  it("pauses a candidate and resumes one", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.5),
          candidate(ab, 0.4, { state: "pruned" }),
        ],
      },
      done: false,
    });
    act(() => screen.getByRole("button", { name: "Pause candidate" }).click());
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "pause")).toBe(true),
    );
    const paused = lab.calls.find((c) => c.op === "pause");
    expect(JSON.parse((paused?.args as { plays: string }).plays)).toEqual([ba]);
    act(() => screen.getByRole("button", { name: "Resume candidate" }).click());
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "resume")).toBe(true),
    );
    const resumed = lab.calls.find((c) => c.op === "resume");
    expect(JSON.parse((resumed?.args as { plays: string }).plays)).toEqual([
      ab,
    ]);
    await waitFor(() =>
      expect(lab.calls.filter((c) => c.op === "read")).toHaveLength(2),
    );
  });

  it("pauses the candidate a touch began on, as its button showed then", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.5),
          candidate(ab, 0.4, { state: "paused" }),
        ],
      },
      done: false,
    });
    const pause = screen.getByRole("button", { name: "Pause candidate" });
    fireEvent.pointerDown(pause, { pointerType: "touch" });
    fireEvent.pointerUp(pause, { pointerType: "touch" });
    // AB, paused, leads now: the button the touch began on shows its Resume
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.3, { n: 64 }),
          candidate(ab, 0.6, { n: 64, state: "paused" }),
        ],
      },
      done: false,
    });
    expect(pause).toHaveAccessibleName("Resume candidate");
    fireEvent.click(pause, { detail: 1 });
    expect(
      await screen.findByText(`Pause ${JSON.stringify(ba)}?`),
    ).toBeInTheDocument();
    reply("Pause");
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "pause")).toBe(true),
    );
    const paused = lab.calls.find((c) => c.op === "pause");
    expect(JSON.parse((paused?.args as { plays: string }).plays)).toEqual([ba]);
    expect(lab.calls.some((c) => c.op === "resume")).toBe(false);
  });

  // The top moves asked for, as many as count says, or as the box starts.
  const addTop = (count?: string) => {
    act(() => screen.getByRole("button", { name: "Top moves" }).click());
    if (count !== undefined) {
      fireEvent.change(screen.getByRole("spinbutton", { name: "Top moves" }), {
        target: { value: count },
      });
    }
    act(() =>
      screen.getByRole("button", { name: "Add the top moves" }).click(),
    );
  };

  it("adds the position's top moves, as many as asked, at once and to a sim", async () => {
    lab.top = [
      { ...ba, score: 10, equity: 3 },
      { ...be, score: 12, equity: 6 },
      { ...ai, score: 8, equity: 1 },
    ];
    render(
      <Store>
        <Sim
          lab={midgame()}
          moves={[listed(ba, { equity: 3 }), listed(ab, { equity: 2 })]}
        />
      </Store>,
    );
    // the count starts at the 15 the analyzer asks for
    act(() => screen.getByRole("button", { name: "Top moves" }).click());
    expect(screen.getByRole("spinbutton", { name: "Top moves" })).toHaveValue(
      "15",
    );
    act(() => screen.getByRole("button", { name: "Top moves" }).click());
    addTop("3");
    await waitFor(() => expect(bodyRows()).toHaveLength(4));
    const analyzed = lab.calls.find((c) => c.op === "analyze");
    expect(
      JSON.parse((analyzed?.args as { request: string }).request),
    ).toMatchObject({
      count: 3,
      leave: "CSW24",
      scores: [300, 290],
      // the player to move's own exchanges
      exchanges: 1,
    });
    expect(bodyRows()).toEqual([
      "8HBE+6.00",
      `${JSON.stringify(ba)}+3.00`,
      `${JSON.stringify(ab)}+2.00`,
      "8HAI+1.00",
    ]);

    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    expect(JSON.parse((sim?.args as { plays: string }).plays)).toEqual([
      ba,
      ab,
      be,
      ai,
    ]);
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          candidate(ba, 0.5),
          candidate(ab, 0.4),
          candidate(be, 0.3),
          candidate(ai, 0.2),
        ],
      },
      done: false,
    });

    // a sim made rolls out only those it lacks
    const pass = { action: "exchange", tiles: [] };
    lab.top = [...lab.top, { ...pass, equity: -5 }];
    addTop("4");
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "read")).toBe(true),
    );
    const added = lab.calls.find((c) => c.op === "add");
    expect(JSON.parse((added?.args as { plays: string }).plays)).toEqual([
      pass,
    ]);

    // and none, when all are listed
    lab.top = [{ ...ba, score: 10, equity: 3 }];
    addTop();
    expect(
      await screen.findByText("Those moves are listed already."),
    ).toBeInTheDocument();
  });

  // Set in the settings, applied: the sim made goes, and the next one takes
  // them.
  const setSettings = (plies: string, known: string) => {
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    const field = screen.getByRole("spinbutton", { name: "Plies" });
    // the wasm says what count it takes
    expect(field).not.toHaveAttribute("aria-valuemax");
    fireEvent.change(field, {
      target: { value: plies },
    });
    fireEvent.change(screen.getByPlaceholderText("tiles, ? a blank"), {
      target: { value: known },
    });
    act(() => screen.getByRole("button", { name: "Apply" }).click());
  };

  it("changes the plies and the opponent's tiles of the sim made in place, its rollouts kept", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    // plies never set are liwords's own 4, sent with the sim, as whether the
    // opponent passes
    const first = lab.calls.find((c) => c.op === "sim");
    const made = JSON.parse((first?.args as { request: string }).request);
    expect(made.plies).toBe(4);
    expect(made.opponents_pass).toBe(false);
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    setSettings("2", "ie");
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "set")).toBe(true),
    );
    const set = lab.calls.find((c) => c.op === "set");
    expect(set?.args).toEqual({
      job: 1,
      request: JSON.stringify({ plies: 2, known: [I, E] }),
    });
    expect(lab.calls.some((c) => c.op === "free")).toBe(false);
    // the sim's answer has each candidate's two plies
    const twoPlies = (c: ReturnType<typeof candidate>) => ({
      ...c,
      plies: [c.plies[0], c.plies[0]],
    });
    emit({
      job: 1,
      answer: {
        iterations: 48,
        candidates: [
          twoPlies(candidate(ba, 0.5)),
          twoPlies(candidate(ab, 0.4)),
        ],
      },
      done: false,
    });
    expect(
      await screen.findByText(
        "48 iterations after 0 s, 2 running; 2 plies; the opponent holds IE",
      ),
    ).toBeInTheDocument();
  });

  it("changes nothing of the sim made with tiles the opponent cannot hold", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    setSettings("4", "zz");
    expect(
      await screen.findByText(
        "Not applied: ZZ are not all among the tiles unseen.",
      ),
    ).toBeInTheDocument();
    expect(lab.calls.some((c) => c.op === "set")).toBe(false);
  });

  it("rolls out to the game's end, and with the opponent passing, which makes a new sim", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    fireEvent.click(screen.getByText("To the game's end"));
    expect(screen.getByRole("spinbutton", { name: "Plies" })).toBeDisabled();
    fireEvent.click(screen.getByText("The opponent passes"));
    act(() => screen.getByRole("button", { name: "Apply" }).click());
    expect(
      screen.getByText("Static equity until the sim starts."),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "free")).toBe(true),
    );
    expect(lab.calls.some((c) => c.op === "set")).toBe(false);
    lab.calls.length = 0;
    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    const request = JSON.parse((sim?.args as { request: string }).request);
    expect(request.plies).toBeNull();
    expect(request.opponents_pass).toBe(true);
    emit({
      job: 1,
      answer: {
        iterations: 16,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    expect(
      screen.getByText(
        "16 iterations after 0 s, 2 running; to the game's end; the opponent passes",
      ),
    ).toBeInTheDocument();
  });

  it("starts the plies at 4, and applies no settings with them left empty", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    const field = screen.getByRole("spinbutton", { name: "Plies" });
    expect(field).toHaveValue("4");
    fireEvent.change(field, { target: { value: "" } });
    const apply = screen.getByRole("button", { name: "Apply" });
    expect(apply).toBeDisabled();
    expect(
      document.querySelector(".ant-input-number-status-error"),
    ).toContainElement(field);
    // nor by Enter in the tiles' field
    fireEvent.keyDown(screen.getByPlaceholderText("tiles, ? a blank"), {
      key: "Enter",
      code: "Enter",
      keyCode: 13,
    });
    // closed so, the next sim takes the plies it had
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    expect(JSON.parse((sim?.args as { request: string }).request).plies).toBe(
      4,
    );
  });

  it("starts no sim with tiles the opponent cannot hold", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    setSettings("4", "zz");
    act(() => screen.getByRole("button", { name: "Start" }).click());
    expect(
      await screen.findByText(
        "Not started: ZZ are not all among the tiles unseen.",
      ),
    ).toBeInTheDocument();
    expect(lab.calls.some((c) => c.op === "sim")).toBe(false);
  });

  it("shows a sim's details, each candidate and then each ply", async () => {
    render(
      <Store>
        <Sim
          lab={midgame()}
          moves={[listed(ba, { equity: 3 }), listed(ab, { equity: 2 })]}
        />
      </Store>,
    );
    expect(screen.getByRole("button", { name: "Details" })).toBeDisabled();
    await start();
    // four plies, the ply numbered i scoring 30 + i, 4 either way, a bingo a
    // quarter of the time
    const plies = Array.from({ length: 4 }, (_, i) => ({
      n: 32,
      score: { mean: 30 + i, variance: 16 },
      bingo: 0.25,
    }));
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          { ...candidate(ba, 0.5, { equity: 9 }), plies },
          { ...candidate(ab, 0.4, { equity: 3 }), plies },
        ],
      },
      done: false,
    });
    act(() => screen.getByRole("button", { name: "Details" }).click());
    expect(
      await screen.findByText("32 iterations of AABEEII"),
    ).toBeInTheDocument();
    const details = document.querySelector(".lab-details")!;
    const tables = [...details.querySelectorAll("table")];
    const rowsOf = (table: Element) =>
      [...table.querySelectorAll("tbody tr")].map((r) => r.textContent);
    expect(
      [...details.querySelectorAll("h4")].map((h) => h.textContent),
    ).toEqual([
      "Opponent's next turn",
      "Your next turn",
      "Opponent's turn 2",
      "Your turn 2",
    ]);
    expect(rowsOf(tables[0])).toEqual([
      `${JSON.stringify(ba)}50.0%0.0%+9.000.010+3.0032`,
      `${JSON.stringify(ab)}40.0%0.0%+3.000.010+2.0032`,
    ]);
    expect(rowsOf(tables[4])[0]).toBe(`${JSON.stringify(ba)}33.04.025.0%32`);
  });

  it("shows each equity's two parts, and each candidate less a move compared with", async () => {
    render(
      <Store>
        <Sim
          lab={midgame()}
          moves={[listed(ba, { equity: 3 }), listed(ab, { equity: 2 })]}
        />
      </Store>,
    );
    await start();
    const parts = (spread: number, leaves: number) => ({
      spread: { mean: spread, variance: 4 },
      leaves: { mean: leaves, variance: 1 },
      objective: { mean: spread + leaves, variance: 5 },
    });
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          { ...candidate(ba, 0.5, { equity: 9 }), ...parts(7, 2) },
          { ...candidate(ab, 0.4, { equity: 3 }), ...parts(4, -1) },
        ],
      },
      done: false,
    });
    act(() => screen.getByRole("button", { name: "Details" }).click());
    const details = document.querySelector(".lab-details")!;
    const rows = () =>
      [
        ...details.querySelectorAll("table")[0].querySelectorAll("tbody tr"),
      ].map((r) => r.textContent);
    expect(rows()).toEqual([
      `${JSON.stringify(ba)}50.0%0.0%+9.000.0+7.00+2.0010+3.0032`,
      `${JSON.stringify(ab)}40.0%0.0%+3.000.0+4.00-1.0010+2.0032`,
    ]);

    // compared with BA: the sim is told, and read at once
    const select = within(details as HTMLElement).getByRole("combobox", {
      name: "Compare with",
    });
    fireEvent.mouseDown(select);
    act(() =>
      (
        [...document.querySelectorAll(".ant-select-item-option")].find(
          (o) => o.textContent === JSON.stringify(ba),
        ) as HTMLElement
      ).click(),
    );
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "read")).toBe(true),
    );
    expect(lab.calls.find((c) => c.op === "against")?.args).toEqual({
      job: 1,
      play: JSON.stringify(ba),
    });
    const less = (n: number, win: number, equity: number) => ({
      win: { n, mean: win, variance: 0.04 * n },
      equity: { n, mean: equity, variance: 25 * n },
      objective: { n, mean: equity, variance: 25 * n },
    });
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [
          {
            ...candidate(ba, 0.5, { equity: 9 }),
            ...parts(7, 2),
            against: less(32, 0, 0),
          },
          {
            ...candidate(ab, 0.4, { equity: 3 }),
            ...parts(4, -1),
            against: less(25, -0.1, -6),
          },
        ],
      },
      done: false,
    });
    expect(rows()[1]).toBe(
      `${JSON.stringify(ab)}40.0%0.0%+3.000.0+4.00-1.0010+2.0032-10.0%-6.005.00`,
    );
    act(() => screen.getByRole("radio", { name: "By move" }).click());
    expect(
      [
        ...details
          .querySelectorAll(".lab-details-block")[1]
          .querySelectorAll("li"),
      ].map((li) => li.textContent),
    ).toContain(
      `Less ${JSON.stringify(ba)}, over 25: win -10.0% (SE 20.0%), equity -6.00 (SE 5.00)`,
    );
    // and with none
    fireEvent.mouseDown(select);
    act(() =>
      (
        [...document.querySelectorAll(".ant-select-item-option")].find(
          (o) => o.textContent === "None",
        ) as HTMLElement
      ).click(),
    );
    await waitFor(() =>
      expect(lab.calls.filter((c) => c.op === "against")).toHaveLength(2),
    );
    expect(lab.calls.filter((c) => c.op === "against")[1].args).toEqual({
      job: 1,
      play: null,
    });
  });

  it("compares with no move once the move compared with is taken out", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    act(() => screen.getByRole("button", { name: "Details" }).click());
    const details = document.querySelector(".lab-details") as HTMLElement;
    fireEvent.mouseDown(
      within(details).getByRole("combobox", { name: "Compare with" }),
    );
    act(() =>
      (
        [...document.querySelectorAll(".ant-select-item-option")].find(
          (o) => o.textContent === JSON.stringify(ab),
        ) as HTMLElement
      ).click(),
    );
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "against")).toBe(true),
    );
    // AB, second by the chance to win, taken out
    act(() => screen.getAllByRole("button", { name: "Take out" })[1].click());
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "remove")).toBe(true),
    );
    const ops = lab.calls.map((c) => c.op);
    const cleared = ops.lastIndexOf("against");
    expect(lab.calls[cleared].args).toEqual({ job: 1, play: null });
    expect(cleared).toBeLessThan(ops.indexOf("remove"));
  });

  it("infers the opponent's leave from their last move, in place of the tiles known", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    fireEvent.change(screen.getByPlaceholderText("tiles, ? a blank"), {
      target: { value: "ie" },
    });
    fireEvent.click(screen.getByText("Infer their leave"));
    expect(screen.getByPlaceholderText("tiles, ? a blank")).toBeDisabled();
    act(() => screen.getByRole("button", { name: "Apply" }).click());
    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    const { request, infer } = sim?.args as { request: string; infer: string };
    expect(JSON.parse(request).known).toEqual([]);
    expect(JSON.parse(infer)).toEqual({
      rack: [A, A, B, E, E, I, I],
      board: [[0]],
      lexicon: "CSW24",
      leave: "CSW24",
      rules: "CrosswordGame",
      play: midgame().lastMove!.play,
      fuzz: 3.105,
      list: 0,
    });
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    expect(
      screen.getByText(
        "32 iterations after 0 s, 2 running; 1 ply; their leave inferred, 120 leaves",
      ),
    ).toBeInTheDocument();
    act(() => screen.getByRole("button", { name: "Details" }).click());
    expect(
      screen.getByText(
        "Their leave, inferred from 120 leaves, the likeliest: AE 25.0%, (none) 12.5%",
      ),
    ).toBeInTheDocument();
  });

  it("infers nothing with no play or exchange of the opponent's last, and changes the sim made in place", async () => {
    const { unmount } = render(
      <Store>
        <Sim
          lab={position([A, B, E, E, I, I, I, I, A, A], { lastMove: null })}
          moves={[listed(ba), listed(ab)]}
        />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    expect(
      screen.getByRole("checkbox", { name: /Infer their leave/ }),
    ).toBeDisabled();
    expect(
      screen.getByText("Their last turn is no play or exchange to infer from."),
    ).toBeInTheDocument();
    unmount();
    // a sim made, then told to infer: sim_set, its rollouts kept
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 32,
        candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
      },
      done: false,
    });
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    fireEvent.click(screen.getByText("Infer their leave"));
    act(() => screen.getByRole("button", { name: "Apply" }).click());
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "set")).toBe(true),
    );
    const set = lab.calls.find((c) => c.op === "set");
    expect((set?.args as { infer?: string }).infer).toBeDefined();
    expect(lab.calls.some((c) => c.op === "free")).toBe(false);
    expect(
      await screen.findByText(
        "32 iterations after 0 s, 2 running; 1 ply; their leave inferred, 120 leaves",
      ),
    ).toBeInTheDocument();
  });

  it("shows a sim's details by move, a block a candidate, a row's when clicked", async () => {
    const scroll = vi.fn();
    Element.prototype.scrollIntoView = scroll;
    try {
      render(
        <Store>
          <Sim
            lab={midgame()}
            moves={[listed(ba, { equity: 3 }), listed(ab, { equity: 2 })]}
          />
        </Store>,
      );
      await start();
      const plies = Array.from({ length: 2 }, (_, i) => ({
        n: 32,
        score: { mean: 30 + i, variance: 16 },
        bingo: 0.25,
      }));
      emit({
        job: 1,
        answer: {
          iterations: 32,
          candidates: [
            { ...candidate(ba, 0.5, { equity: 9 }), plies },
            { ...candidate(ab, 0.4, { equity: 3 }), plies },
          ],
        },
        done: false,
      });
      act(() => screen.getByRole("button", { name: "Details" }).click());
      expect(
        await screen.findByText("32 iterations of AABEEII"),
      ).toBeInTheDocument();
      const details = document.querySelector(".lab-details")!;
      const blocks = () => [...details.querySelectorAll(".lab-details-block")];
      expect(blocks()).toHaveLength(0);

      act(() => screen.getByRole("radio", { name: "By move" }).click());
      expect(blocks().map((b) => b.querySelector("h4")?.textContent)).toEqual([
        JSON.stringify(ba),
        JSON.stringify(ab),
      ]);
      expect(
        [...blocks()[0].querySelectorAll("li")].map((li) => li.textContent),
      ).toEqual([
        "Win: 50.0% (SD 0.0%)",
        "Equity: +9.00 (SD 0.0)",
        "Score: 10, static +3.00",
        "N: 32",
        "Opponent's next turn: 30.0 (SD 4.0), bingo 25.0%",
        "Your next turn: 31.0 (SD 4.0), bingo 25.0%",
      ]);
      expect(scroll).not.toHaveBeenCalled();

      act(() => screen.getByRole("radio", { name: "By ply" }).click());
      expect(blocks()).toHaveLength(0);
      const second = details
        .querySelectorAll("table")[0]
        .querySelectorAll("tbody tr")[1] as HTMLElement;
      act(() => second.click());
      expect(blocks()).toHaveLength(2);
      expect(scroll).toHaveBeenCalledTimes(1);
      expect(scroll.mock.contexts[0]).toBe(blocks()[1]);
    } finally {
      delete (Element.prototype as { scrollIntoView?: unknown }).scrollIntoView;
    }
  });

  it("saves a sim's numbers to a file", async () => {
    let saved: Blob | undefined;
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: (blob: Blob) => {
        saved = blob;
        return "blob:sim";
      },
      revokeObjectURL: () => {},
    });
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, "click")
      .mockImplementation(() => {});
    try {
      render(
        <Store>
          <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
        </Store>,
      );
      await start();
      emit({
        job: 1,
        answer: {
          iterations: 32,
          candidates: [candidate(ba, 0.5), candidate(ab, 0.4)],
        },
        done: false,
      });
      act(() => screen.getByRole("button", { name: "Details" }).click());
      act(() => screen.getByRole("button", { name: "Save" }).click());
      expect(click).toHaveBeenCalledTimes(1);
      expect((click.mock.contexts[0] as HTMLAnchorElement).download).toBe(
        "sim-AABEEII.json",
      );
      const json = JSON.parse(await saved!.text());
      // the plies as the sim's numbers have them, one here
      expect(json).toMatchObject({
        rack: "AABEEII",
        lexicon: "CSW24",
        iterations: 32,
        plies: 1,
        known: "",
      });
      expect(json.candidates.map((c: { move: string }) => c.move)).toEqual([
        JSON.stringify(ba),
        JSON.stringify(ab),
      ]);
      expect(json.candidates[0]).toMatchObject({ n: 32, win: { mean: 0.5 } });
    } finally {
      click.mockRestore();
    }
  });

  // The unpruned box in the settings, set so and applied.
  const setUnpruned = (unpruned: boolean) => {
    act(() => screen.getByRole("button", { name: "Sim settings" }).click());
    const box = screen.getByRole("checkbox", { name: /Unpruned/ });
    if ((box as HTMLInputElement).checked !== unpruned) {
      act(() => box.click());
    }
    act(() => screen.getByRole("button", { name: "Apply" }).click());
  };

  it("sims even one move unpruned, each candidate rolled out until stopped", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba)]} />
      </Store>,
    );
    expect(
      screen.getByText("Needs two or more moves to sim."),
    ).toBeInTheDocument();
    setUnpruned(true);
    expect(
      screen.getByText("Static equity until the sim starts."),
    ).toBeInTheDocument();
    await start();
    expect(lab.calls.find((c) => c.op === "run")?.args).toEqual({
      job: 1,
      ms: 8,
      prune: false,
    });
    emit({
      job: 1,
      answer: { iterations: 32, candidates: [candidate(ba, 0.5)] },
      done: false,
    });
    expect(
      screen.getByText("32 iterations after 0 s, 1 running; 1 ply; unpruned"),
    ).toBeInTheDocument();
  });

  it("applies unpruned, or not, to the sim made, keeping its rollouts", async () => {
    render(
      <Store>
        <Sim lab={midgame()} moves={[listed(ba), listed(ab)]} />
      </Store>,
    );
    await start();
    emit({
      job: 1,
      answer: {
        iterations: 64,
        candidates: [
          candidate(ba, 0.9, { n: 64 }),
          candidate(ab, 0.1, { n: 48, state: "pruned" }),
        ],
      },
      done: true,
    });
    expect(screen.getByRole("button", { name: "Resume" })).toBeDisabled();
    setUnpruned(true);
    // the sim made stays, with more to do now
    expect(
      screen.getByText(
        "64 iterations after 0 s, 1 running, stopped; 1 ply; unpruned",
      ),
    ).toBeInTheDocument();
    expect(lab.calls.some((c) => c.op === "free")).toBe(false);
    act(() => screen.getByRole("button", { name: "Resume" }).click());
    await waitFor(() =>
      expect(lab.calls[lab.calls.length - 1]).toEqual({
        op: "run",
        args: { job: 1, ms: 8, prune: false },
      }),
    );
    // running, it takes the change from its next step on
    setUnpruned(false);
    await waitFor(() =>
      expect(lab.calls[lab.calls.length - 1]).toEqual({
        op: "run",
        args: { job: 1, ms: 8, alpha: 0.05 },
      }),
    );
  });
});

describe("a panel's card", () => {
  // the window as tall as given, and the card's body at the top and of the
  // height given, with 1 px of the card's border below it
  const layOut = (innerHeight: number, top: number, height: number) => {
    vi.stubGlobal("innerHeight", innerHeight);
    return vi
      .spyOn(HTMLElement.prototype, "getBoundingClientRect")
      .mockImplementation(function (this: HTMLElement) {
        const bottom = this.classList.contains("ant-card-body")
          ? top + height
          : top + height + 1;
        return { top, bottom, height } as DOMRect;
      });
  };
  const Panel = () => {
    const panel = useRef<HTMLDivElement>(null);
    useLabRoom(panel);
    return <div ref={panel} />;
  };
  const inCard = (card: string) => (
    <div className={card}>
      <div className="ant-card-body">
        <Panel />
      </div>
    </div>
  );
  const body = () => document.querySelector<HTMLElement>(".ant-card-body")!;
  let rects: { mockRestore: () => void } | undefined;
  afterEach(() => rects?.mockRestore());

  it("keeps its own height, with room down to a margin above the window's bottom where that is nearer than half the window", () => {
    rects = layOut(760, 499, 190);
    render(inCard("analyzer-card"));
    expect(body()).toHaveClass("lab-room");
    expect(body().style.getPropertyValue("--lab-own")).toBe("190px");
    expect(body().style.getPropertyValue("--lab-room")).toBe("236px");
  });

  it("has its whole room, past half the window, on a page laid out as on a phone", () => {
    vi.stubGlobal("innerHeight", 844);
    rects = vi
      .spyOn(HTMLElement.prototype, "getBoundingClientRect")
      .mockImplementation(function (this: HTMLElement) {
        if (this.classList.contains("sticky-player-card-container")) {
          return { top: 0, bottom: 166, height: 166 } as DOMRect;
        }
        const bottom = this.classList.contains("ant-card-body") ? 2007 : 2008;
        return { top: 1767, bottom, height: 240 } as DOMRect;
      });
    const shown = vi
      .spyOn(HTMLElement.prototype, "getClientRects")
      .mockImplementation(() => [{}] as unknown as DOMRectList);
    render(
      <div>
        <div className="sticky-player-card-container" />
        {inCard("analyzer-card")}
      </div>,
    );
    expect(body().style.getPropertyValue("--lab-room")).toBe("629px");
    shown.mockRestore();
  });

  it("has room for half the window's height at most", () => {
    rects = layOut(1200, 499, 190);
    render(inCard("analyzer-card"));
    expect(body().style.getPropertyValue("--lab-room")).toBe("600px");
  });

  it("has room for its own height at least", () => {
    rects = layOut(600, 499, 190);
    render(inCard("analyzer-card"));
    expect(body().style.getPropertyValue("--lab-room")).toBe("190px");
  });

  it("measures again as the window is resized, and gives its room back once closed", () => {
    rects = layOut(760, 499, 190);
    const { unmount } = render(inCard("analyzer-card"));
    expect(body().style.getPropertyValue("--lab-room")).toBe("236px");
    vi.stubGlobal("innerHeight", 1200);
    window.dispatchEvent(new Event("resize"));
    expect(body().style.getPropertyValue("--lab-room")).toBe("600px");
    const held = body();
    unmount();
    expect(held).not.toHaveClass("lab-room");
    expect(held.style.getPropertyValue("--lab-own")).toBe("");
    expect(held.style.getPropertyValue("--lab-room")).toBe("");
  });

  it("leaves the score card, which holds the analyzer on narrower windows, as it is", () => {
    rects = layOut(768, 146, 215);
    render(inCard("score-card"));
    expect(body()).not.toHaveClass("lab-room");
    expect(body().style.getPropertyValue("--lab-room")).toBe("");
  });
});

describe("the opponent's last move, to infer their leave from", () => {
  // CAB across row 8 from H, its A through a tile already there and its B a
  // blank; the board's letters as liwords keeps them
  const letters = () => {
    const l = new Array(225).fill(0);
    l[7 * 15 + 7] = 3;
    l[7 * 15 + 8] = A;
    l[7 * 15 + 9] = B | 0x80;
    return l;
  };
  const event = (more: object) =>
    create(GameEventSchema, { playerIndex: 1, score: 9, ...more });

  it("takes a play's tiles off the board, the tiles already there left as 0 in its word", () => {
    const last = lastMoveOf(
      [
        event({
          type: GameEvent_Type.TILE_PLACEMENT_MOVE,
          row: 7,
          column: 7,
          direction: GameEvent_Direction.HORIZONTAL,
          playedTiles: "C.b",
        }),
      ],
      0,
      letters(),
      15,
    );
    expect(last?.play).toEqual({
      action: "play",
      down: false,
      lane: 7,
      idx: 7,
      word: [3, 0, -B],
      score: 9,
    });
    expect(last?.board[7].slice(7, 10)).toEqual([0, A, 0]);
    expect(last?.board).toHaveLength(15);
  });

  it("writes a play down by its column, and an exchange by how many tiles it took", () => {
    const down = lastMoveOf(
      [
        event({
          type: GameEvent_Type.TILE_PLACEMENT_MOVE,
          row: 3,
          column: 7,
          direction: GameEvent_Direction.VERTICAL,
          playedTiles: "AB",
        }),
      ],
      0,
      new Array(225).fill(1),
      15,
    );
    expect(down?.play).toMatchObject({ down: true, lane: 7, idx: 3 });
    expect(down?.board[3][7]).toBe(0);
    expect(down?.board[4][7]).toBe(0);
    expect(down?.board[5][7]).toBe(1);
    const exchange = lastMoveOf(
      [event({ type: GameEvent_Type.EXCHANGE, exchanged: "AEI" })],
      0,
      letters(),
      15,
    );
    expect(exchange?.play).toEqual({ action: "exchange", tiles: [0, 0, 0] });
    // hidden, the count from the rack
    const hidden = lastMoveOf(
      [event({ type: GameEvent_Type.EXCHANGE, numTilesFromRack: 2 })],
      0,
      letters(),
      15,
    );
    expect(hidden?.play).toEqual({ action: "exchange", tiles: [0, 0] });
  });

  it("is none after a pass, a play challenged off, or the player to move's own turn", () => {
    const at = (type: GameEvent_Type, playerIndex = 1) =>
      lastMoveOf([event({ type, playerIndex })], 0, letters(), 15);
    expect(at(GameEvent_Type.PASS)).toBeNull();
    expect(at(GameEvent_Type.PHONY_TILES_RETURNED)).toBeNull();
    expect(at(GameEvent_Type.TILE_PLACEMENT_MOVE, 0)).toBeNull();
    expect(lastMoveOf([], 0, letters(), 15)).toBeNull();
  });
});
