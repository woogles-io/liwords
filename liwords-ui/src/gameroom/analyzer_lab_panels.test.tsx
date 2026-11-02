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
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { Store } from "../store/store";
import { LabEvent } from "../wasm/lab";
import { AnalyzerMove } from "./analyzer";
import {
  LabPosition,
  SimPanel,
  useLabRoom,
  useLabSim,
} from "./analyzer_lab_panels";

// The lab's worker, faked: a job made is job 1, a sim holds each play as it
// is given, and the test sends the steps' answers.
const lab = vi.hoisted(() => ({
  calls: [] as Array<{ op: string; args?: object }>,
  listeners: new Set<(e: LabEvent) => void>(),
}));
vi.mock("../wasm/lab", () => ({
  labCall: async (request: { op: string; args?: { plays?: string } }) => {
    lab.calls.push(request);
    if (request.op === "sim") {
      const plays = JSON.parse(request.args!.plays!) as Array<object>;
      return { job: 1, added: plays.map((play) => ({ play, was: "absent" })) };
    }
    if (["add", "remove"].includes(request.op)) {
      const plays = JSON.parse(request.args!.plays!) as Array<object>;
      return plays.map((play) => ({ play, was: "absent" }));
    }
    return undefined;
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
  key: JSON.stringify(unseen),
  ...more,
});

// a position with tiles still in the bag
const midgame = () => position([A, B, E, E, I, I, I, I, A, A]);

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
  board.placed.length = 0;
  clock.now = 0;
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
    // the one played among them, but no phony
    expect(
      screen.getByText("Static equity until the sim starts."),
    ).toBeInTheDocument();
    expect(bodyRows()).toEqual([
      `${JSON.stringify(ab)}+8.00`,
      `${JSON.stringify(ba)}+3.00`,
    ]);
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(lab.calls).toEqual([]);

    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    const { plays, request } = sim?.args as { plays: string; request: string };
    expect(JSON.parse(plays)).toEqual([ba, ab]);
    expect(JSON.parse(request).leave).toBe("CSW24");
    // the plies the lab's own, always sent
    expect(JSON.parse(request).plies).toBe(4);
    // each step as long as a frame
    expect(lab.calls.find((c) => c.op === "run")?.args).toEqual({
      job: 1,
      ms: 8,
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
    expect(
      screen.getByText("32 iterations after 0 s, 2 running"),
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
      screen.getByText("40 iterations after 0 s, 2 running, stopped"),
    ).toBeInTheDocument();
    act(() => screen.getByRole("button", { name: "Resume" }).click());
    await waitFor(() =>
      expect(lab.calls[lab.calls.length - 1]).toEqual({
        op: "run",
        args: { job: 1, ms: 8 },
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

  it("leaves out the phony listed, naming the words the lexicon lacks", async () => {
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
    expect(
      screen.getByText("SARKIER is not simmed: CONDS is not in CSW24."),
    ).toBeInTheDocument();
    expect(bodyRows()).toHaveLength(2);
    await start();
    const sim = lab.calls.find((c) => c.op === "sim");
    expect(JSON.parse((sim?.args as { plays: string }).plays)).toEqual([
      ba,
      ab,
    ]);
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
      screen.getByText("64 iterations after 0 s, 1 running, done"),
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
      screen.getByText("32 iterations after 0 s, 2 running, stopped"),
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

  it("turns down a phony, naming its words, and a move listed already", async () => {
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
        <Sim lab={midgame()} moves={[listed(ba), listed(be)]} />
      </Store>,
    );
    act(() => screen.getByRole("button", { name: "Add" }).click());
    expect(
      await screen.findByText("Not added: BAZQPE and UB are not in CSW24."),
    ).toBeInTheDocument();
    page.answer = undefined;
    act(() => screen.getByRole("button", { name: "Add" }).click());
    expect(
      await screen.findByText("That move is listed already."),
    ).toBeInTheDocument();
    expect(bodyRows()).toHaveLength(2);
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
    expect(screen.queryAllByRole("button", { name: "Take out" })).toEqual([]);
    await waitFor(() =>
      expect(lab.calls.some((c) => c.op === "run")).toBe(true),
    );
    expect(screen.getByRole("button", { name: "Add" })).toBeEnabled();
    expect(screen.getAllByRole("button", { name: "Take out" })).toHaveLength(2);
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
