import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LabEvent, labCall, onLabEvent, setLabPartsForTest } from "./lab";

// The lab's workers, faked: each records what it is sent and answers each
// request at once, as a test says, the rest with nothing; a test sends each
// worker's events.
type Sent = { id: number; op: string; args?: Record<string, unknown> };
let made: Array<FakeWorker> = [];
let answer: (w: number, m: Sent) => unknown = () => undefined;

class FakeWorker {
  sent: Array<Sent> = [];
  onmessage: ((e: MessageEvent) => void) | null = null;
  index: number;
  constructor() {
    this.index = made.length;
    made.push(this);
  }
  postMessage(m: Sent) {
    this.sent.push(m);
    const result = answer(this.index, m);
    queueMicrotask(() =>
      this.onmessage?.({ data: { id: m.id, result } } as MessageEvent),
    );
  }
  terminate() {}
  event(event: LabEvent) {
    this.onmessage?.({ data: { event } } as MessageEvent);
  }
}

const ops = (w: number) => made[w].sent.map((m) => m.op);
const lastOf = (w: number, op: string) =>
  [...made[w].sent].reverse().find((m) => m.op === op);

let events: Array<LabEvent> = [];
let stopListening = () => {};

const stat = (mean: number, variance = 0) => ({ mean, variance });
const candidate = (play: object, n: number, win: number, more = {}) => ({
  play,
  state: "running",
  n,
  win: stat(win, 0.25),
  equity: stat(win * 10, 4),
  plies: [],
  ...more,
});
const ba = { action: "play", down: false, lane: 7, idx: 7, word: [2, 1] };
const ab = { action: "play", down: false, lane: 7, idx: 7, word: [1, 2] };

beforeEach(() => {
  made = [];
  events = [];
  answer = (_, m) => {
    if (m.op === "sim") {
      const plays = JSON.parse(m.args!.plays as string) as Array<object>;
      return {
        job: 11,
        added: plays.map((play) => ({
          play: { ...play, score: 10 },
          was: "absent",
        })),
      };
    }
    if (m.op === "endgame") return 12;
    if (["add", "remove", "pause", "resume"].includes(m.op)) {
      const plays = JSON.parse(m.args!.plays as string) as Array<object>;
      return plays.map((play) => ({
        play: { ...play, score: 10 },
        was: "running",
      }));
    }
    return undefined;
  };
  vi.stubGlobal("Worker", FakeWorker);
  stopListening = onLabEvent((e) => events.push(e));
});
afterEach(() => {
  stopListening();
  vi.unstubAllGlobals();
});

describe("the lab on one worker", () => {
  it("sends every request to its one worker, and passes its events on", async () => {
    setLabPartsForTest(1);
    const job = await labCall<{ job: number }>({
      op: "sim",
      args: { request: "{}", plays: JSON.stringify([ba]) },
    });
    expect(job.job).toBe(11);
    expect(made).toHaveLength(1);
    // the sim's request as it was, in no parts
    expect(lastOf(0, "sim")?.args?.request).toBe("{}");
    made[0].event({ job: 11, answer: { iterations: 1 }, done: false });
    expect(events).toEqual([
      { job: 11, answer: { iterations: 1 }, done: false },
    ]);
  });
});

describe("a sim in parts", () => {
  const start = async () => {
    setLabPartsForTest(2);
    const made2 = await labCall<{ job: number }>({
      op: "sim",
      args: {
        request: JSON.stringify({ plies: 2 }),
        plays: JSON.stringify([ba, ab]),
      },
    });
    return made2.job;
  };

  it("runs on every worker, the same seed for each, its own part of two", async () => {
    const job = await start();
    expect(made).toHaveLength(2);
    const requests = [0, 1].map((w) =>
      JSON.parse(lastOf(w, "sim")!.args!.request as string),
    );
    expect(requests[0]).toMatchObject({ plies: 2, part: 0, parts: 2 });
    expect(requests[1]).toMatchObject({ plies: 2, part: 1, parts: 2 });
    expect(requests[0].seed).toBe(requests[1].seed);
    expect(Number.isInteger(requests[0].seed)).toBe(true);
    // each part prunes nothing on its own
    await labCall({ op: "run", args: { job, ms: 8, alpha: 0.05 } });
    expect(lastOf(0, "run")?.args).toEqual({ job: 11, ms: 8, prune: false });
    expect(lastOf(1, "run")?.args).toEqual({ job: 11, ms: 8, prune: false });
    // a change goes to every part
    await labCall({ op: "add", args: { job, plays: "[]" } });
    expect(ops(0)).toContain("add");
    expect(ops(1)).toContain("add");
  });

  it("prunes the board's lexicon once, on the first worker, and loads it on each", async () => {
    const bytes = new Uint8Array([1, 2, 3]);
    const plain = answer;
    answer = (w, m) => (m.op === "kbwg_build" ? bytes : plain(w, m));
    const position = {
      lexicon: "CSW24",
      rules: "CrosswordGame",
      board: ["A"],
      max_memory_mb: 128,
    };
    const sim = () =>
      labCall({
        op: "sim",
        args: { request: JSON.stringify(position), plays: "[]" },
      });
    setLabPartsForTest(2);
    await sim();
    expect(ops(0).filter((op) => op === "kbwg_build")).toHaveLength(1);
    expect(ops(1)).not.toContain("kbwg_build");
    // under the cap each worker's job is given
    expect(
      JSON.parse(lastOf(0, "kbwg_build")!.args!.request as string),
    ).toEqual(position);
    const key = lastOf(0, "kbwg_load")!.args!.key as string;
    for (const w of [0, 1]) {
      expect(lastOf(w, "kbwg_load")?.args).toEqual({ key, bytes });
      expect(
        JSON.parse(lastOf(w, "sim")!.args!.request as string),
      ).toMatchObject({ lexicon: key, pruned: true });
    }
    // the same board again: neither built nor loaded again
    await sim();
    expect(ops(0).filter((op) => op === "kbwg_build")).toHaveLength(1);
    expect(ops(1).filter((op) => op === "kbwg_load")).toHaveLength(1);
  });

  it("shares a board's pruned lexicon with every feature on it, and keeps the board before", async () => {
    const plain = answer;
    answer = (w, m) =>
      m.op === "kbwg_build" ? new Uint8Array([1]) : plain(w, m);
    const at = (board: string) => ({ lexicon: "CSW24", rules: "R", board });
    const builds = () => ops(0).filter((op) => op === "kbwg_build").length;
    const asked = (op: string) =>
      JSON.parse(lastOf(0, op)!.args!.request as string);
    setLabPartsForTest(2);
    // an analysis alone builds none
    await labCall({
      op: "analyze",
      args: { request: JSON.stringify(at("A")) },
    });
    expect(builds()).toBe(0);
    expect(asked("analyze").lexicon).toBe("CSW24");
    // a threats count builds it, on the first worker alone, unflagged
    await labCall({
      op: "threats_new",
      args: { request: JSON.stringify(at("A")) },
    });
    expect(builds()).toBe(1);
    expect(made[1]?.sent.some((m) => m.op === "kbwg_load") ?? false).toBe(
      false,
    );
    const keyA = asked("threats_new").lexicon as string;
    expect(keyA).toMatch(/^kbwg:/);
    expect(asked("threats_new").pruned).toBeUndefined();
    // the analysis and the endgame on that board take it, none built again
    await labCall({
      op: "analyze",
      args: { request: JSON.stringify(at("A")) },
    });
    expect(asked("analyze")).toEqual({ ...at("A"), lexicon: keyA });
    await labCall({
      op: "endgame",
      args: { request: JSON.stringify(at("A")) },
    });
    expect(asked("endgame")).toEqual({
      ...at("A"),
      lexicon: keyA,
      pruned: true,
    });
    expect(builds()).toBe(1);
    // the next board's, the inference of the board before on that one's
    await labCall({
      op: "endgame",
      args: { request: JSON.stringify(at("B")) },
    });
    expect(builds()).toBe(2);
    await labCall({ op: "infer", args: { request: JSON.stringify(at("A")) } });
    expect(asked("infer").lexicon).toBe(keyA);
    expect(ops(0)).not.toContain("kbwg_evict");
    // a third board's lets the oldest go
    await labCall({
      op: "endgame",
      args: { request: JSON.stringify(at("C")) },
    });
    expect(lastOf(0, "kbwg_evict")?.args).toEqual({ key: keyA });
    await labCall({ op: "infer", args: { request: JSON.stringify(at("A")) } });
    expect(asked("infer").lexicon).toBe("CSW24");
  });

  it("makes a sim's inference once, on the first worker, for every part", async () => {
    const shared = { bytes: new Uint8Array([7]), count: 3, leaves: [] };
    const plain = answer;
    answer = (w, m) => (m.op === "infer_share" ? shared : plain(w, m));
    setLabPartsForTest(2);
    await labCall({
      op: "sim",
      args: { request: "{}", plays: "[]", infer: '{"board":"A"}' },
    });
    expect(ops(0).filter((op) => op === "infer_share")).toHaveLength(1);
    expect(ops(1)).not.toContain("infer_share");
    for (const w of [0, 1]) {
      expect(lastOf(w, "sim")?.args?.shared).toEqual(shared);
      expect(lastOf(w, "sim")?.args?.infer).toBeUndefined();
    }
  });

  it("leaves each part to infer for itself when none can be shared", async () => {
    setLabPartsForTest(2);
    await labCall({
      op: "sim",
      args: { request: "{}", plays: "[]", infer: '{"board":"A"}' },
    });
    for (const w of [0, 1]) {
      expect(lastOf(w, "sim")?.args?.infer).toBe('{"board":"A"}');
      expect(lastOf(w, "sim")?.args?.shared).toBeUndefined();
    }
  });

  it("goes on with the full lexicon when the board cannot be pruned", async () => {
    setLabPartsForTest(2);
    await labCall({
      op: "sim",
      args: { request: JSON.stringify({ lexicon: "CSW24" }), plays: "[]" },
    });
    expect(ops(1)).not.toContain("kbwg_load");
    const request = JSON.parse(lastOf(1, "sim")!.args!.request as string);
    expect(request.lexicon).toBe("CSW24");
    expect(request.pruned).toBeUndefined();
  });

  it("merges the parts' answers once each has answered", async () => {
    const job = await start();
    made[0].event({
      job: 11,
      answer: {
        iterations: 3,
        candidates: [
          candidate({ ...ba, score: 10 }, 3, 0.5),
          candidate({ ...ab, score: 10 }, 3, 0.2),
        ],
      },
      done: false,
    });
    expect(events).toEqual([]);
    made[1].event({
      job: 11,
      answer: {
        iterations: 5,
        candidates: [
          candidate({ ...ba, score: 10 }, 5, 0.9),
          candidate({ ...ab, score: 10 }, 5, 0.4),
        ],
      },
      done: false,
    });
    expect(events).toHaveLength(1);
    const merged = events[0].answer as {
      iterations: number;
      candidates: Array<{ n: number; win: { mean: number } }>;
    };
    expect(events[0].job).toBe(job);
    expect(merged.iterations).toBe(8);
    expect(merged.candidates[0].n).toBe(8);
    expect(merged.candidates[0].win.mean).toBeCloseTo((3 * 0.5 + 5 * 0.9) / 8);
  });

  it("prunes on the merged numbers: each part read against the leader, the one behind paused everywhere", async () => {
    const behind = {
      win: { n: 8, mean: -0.5, variance: 0.01 },
      equity: { n: 8, mean: -5, variance: 1 },
      objective: { n: 8, mean: -5, variance: 1 },
    };
    const even = {
      win: { n: 8, mean: 0, variance: 0 },
      equity: { n: 8, mean: 0, variance: 0 },
      objective: { n: 8, mean: 0, variance: 0 },
    };
    const base = answer;
    answer = (w, m) =>
      m.op === "peek"
        ? {
            iterations: 8,
            candidates: [
              candidate({ ...ba, score: 10 }, 8, 0.9, { against: even }),
              candidate({ ...ab, score: 10 }, 8, 0.1, { against: behind }),
            ],
          }
        : base(w, m);
    const job = await start();
    await labCall({ op: "run", args: { job, ms: 8, alpha: 0.05 } });
    const step = (w: number, abState = "running") =>
      made[w].event({
        job: 11,
        answer: {
          iterations: 8,
          candidates: [
            candidate({ ...ba, score: 10 }, 8, 0.9),
            candidate({ ...ab, score: 10 }, 8, 0.1, { state: abState }),
          ],
        },
        done: false,
      });
    step(0);
    step(1);
    // 16 iterations merged: a look, against BA as it was given
    await vi.waitFor(() => expect(ops(1)).toContain("pause"));
    expect(lastOf(0, "peek")?.args).toEqual({
      job: 11,
      against: JSON.stringify(ba),
    });
    expect(JSON.parse(lastOf(0, "pause")!.args!.plays as string)).toEqual([ab]);
    expect(JSON.parse(lastOf(1, "pause")!.args!.plays as string)).toEqual([ab]);
    // the parts answer with it paused: the page shows it pruned, and the sim
    // is done with one left, every part stopped
    step(0, "paused");
    step(1, "paused");
    const last = events[events.length - 1];
    expect(
      (last.answer as { candidates: Array<{ state: string }> }).candidates.map(
        (c) => c.state,
      ),
    ).toEqual(["running", "pruned"]);
    expect(last.done).toBe(true);
    expect(ops(0)).toContain("stop");
    expect(ops(1)).toContain("stop");
    // resumed by the user, it is no longer pruned
    await labCall({ op: "resume", args: { job, plays: JSON.stringify([ab]) } });
    step(0, "running");
    step(1, "running");
    const again = events[events.length - 1];
    expect(
      (again.answer as { candidates: Array<{ state: string }> }).candidates.map(
        (c) => c.state,
      ),
    ).toEqual(["running", "running"]);
  });

  it("prunes nothing unpruned", async () => {
    const job = await start();
    await labCall({ op: "run", args: { job, ms: 8, prune: false } });
    for (const w of [0, 1]) {
      made[w].event({
        job: 11,
        answer: {
          iterations: 40,
          candidates: [
            candidate({ ...ba, score: 10 }, 40, 0.9),
            candidate({ ...ab, score: 10 }, 40, 0.1),
          ],
        },
        done: false,
      });
    }
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(ops(0)).not.toContain("peek");
  });

  it("frees the sim's parts when an endgame takes the first worker, and says so", async () => {
    const job = await start();
    const endgame = await labCall<number>({
      op: "endgame",
      args: { request: "{}" },
    });
    expect(endgame).not.toBe(job);
    expect(lastOf(0, "free")?.args).toEqual({ job: 11 });
    expect(lastOf(1, "free")?.args).toEqual({ job: 11 });
    expect(events).toContainEqual({ job, freed: true });
    // the endgame's steps are the page's job's
    made[0].event({ job: 12, answer: { exact: true }, done: true });
    expect(events[events.length - 1]).toEqual({
      job: endgame,
      answer: { exact: true },
      done: true,
    });
    expect(made[1].sent.some((m) => m.op === "endgame")).toBe(false);
  });

  it("stops the sim on a part's error, and says so", async () => {
    const job = await start();
    made[1].event({ job: 11, error: "out of memory" });
    expect(events).toEqual([{ job, error: "out of memory" }]);
    expect(ops(0)).toContain("stop");
  });
});

describe("a pre-endgame in parts", () => {
  const endgamePart = (k: number, value: number, win: number) => ({
    bag: 2,
    steps: 1,
    nodes: 10,
    exact: true,
    play: ba,
    value,
    win,
    spread: 10 + value,
    depth: 127,
    part: [k, 2],
    candidates: [
      {
        play: ba,
        value,
        win,
        exact: true,
        draws: [{ tiles: [k], count: 1, value, win }],
      },
    ],
  });

  it("runs on every worker, each its part of the bag's sets, merged once each has answered", async () => {
    setLabPartsForTest(2);
    const job = await labCall<number>({
      op: "endgame",
      args: { request: JSON.stringify({ model: "info_sets" }) },
    });
    expect(
      [0, 1].map((w) =>
        JSON.parse(lastOf(w, "endgame")!.args!.request as string),
      ),
    ).toEqual([
      { model: "info_sets", part: [0, 2] },
      { model: "info_sets", part: [1, 2] },
    ]);
    await labCall({ op: "run", args: { job, ms: 16 } });
    expect(lastOf(0, "run")?.args).toEqual({ job: 12, ms: 16 });
    expect(lastOf(1, "run")?.args).toEqual({ job: 12, ms: 16 });
    made[0].event({ job: 12, answer: endgamePart(0, 4, 1), done: true });
    expect(events).toEqual([]);
    made[1].event({
      job: 12,
      answer: { ...endgamePart(1, -2, 0), exact: false },
      done: false,
    });
    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({
      job,
      done: false,
      answer: { exact: false, play: ba, value: 1, win: 0.5, spread: 11 },
    });
    made[1].event({ job: 12, answer: endgamePart(1, -2, 0), done: true });
    expect(events[1]).toMatchObject({
      job,
      done: true,
      answer: { exact: true },
    });
    // freed, every part with it
    await labCall({ op: "free", args: { job } });
    expect(lastOf(0, "free")?.args).toEqual({ job: 12 });
    expect(lastOf(1, "free")?.args).toEqual({ job: 12 });
    expect(events[events.length - 1]).toEqual({ job, freed: true });
  });

  it("runs on as few workers as asked, on one with no part", async () => {
    setLabPartsForTest(4);
    const request = JSON.stringify({ model: "info_sets" });
    await labCall<number>({ op: "endgame", args: { request, parts: 2 } });
    expect(made.map((w) => w.sent.some((m) => m.op === "endgame"))).toEqual([
      true,
      true,
    ]);
    expect(JSON.parse(lastOf(1, "endgame")!.args!.request as string)).toEqual({
      model: "info_sets",
      part: [1, 2],
    });
    await labCall<number>({ op: "endgame", args: { request, parts: 1 } });
    expect(lastOf(0, "endgame")!.args).toEqual({ request });
    expect(made[1].sent.filter((m) => m.op === "endgame")).toHaveLength(1);
  });

  it("runs an endgame of one tile or none on the first worker alone", async () => {
    setLabPartsForTest(2);
    await labCall<number>({ op: "endgame", args: { request: "{}" } });
    expect(JSON.parse(lastOf(0, "endgame")!.args!.request as string)).toEqual(
      {},
    );
    expect(made[1]?.sent.some((m) => m.op === "endgame") ?? false).toBe(false);
  });

  it("passes on the answer a part's error stands with, merged", async () => {
    setLabPartsForTest(2);
    const job = await labCall<number>({
      op: "endgame",
      args: { request: JSON.stringify({ model: "info_sets" }) },
    });
    made[0].event({ job: 12, answer: endgamePart(0, 4, 1), done: false });
    made[1].event({
      job: 12,
      error: "out of memory",
      answer: { ...endgamePart(1, -2, 0), exact: false },
    });
    const failed = events.find((e) => e.error) as LabEvent & {
      answer?: object;
    };
    expect(failed).toMatchObject({ job, error: "out of memory" });
    expect(failed.answer).toBeDefined();
  });

  it("stops the others on a part's error, and says so", async () => {
    setLabPartsForTest(2);
    const job = await labCall<number>({
      op: "endgame",
      args: { request: JSON.stringify({ model: "random_draws" }) },
    });
    made[1].event({ job: 12, error: "out of memory" });
    expect(events).toEqual([{ job, error: "out of memory" }]);
    expect(ops(0)).toContain("stop");
  });
});

describe("a worker whose wasm panicked", () => {
  it("is let go, its calls failed, and the next call made on a fresh one", async () => {
    setLabPartsForTest(1);
    const first = labCall({ op: "held" });
    // before it answers, the worker says its wasm panicked, answering another
    // call
    made[0].onmessage?.({
      data: { id: 999, error: "The wasm failed", dead: true },
    } as MessageEvent);
    await expect(first).rejects.toThrow("started again");
    await labCall({ op: "held" });
    expect(made).toHaveLength(2);
    expect(ops(1)).toEqual(["held"]);
  });
});

describe("a worker whose endgame ran out of memory", () => {
  it("has its error heard, then is let go, the next call made on a fresh one", async () => {
    setLabPartsForTest(1);
    await labCall({ op: "held" });
    made[0].onmessage?.({
      data: { event: { job: 3, error: "out of memory" }, spent: true },
    } as MessageEvent);
    expect(events).toEqual([{ job: 3, error: "out of memory" }]);
    await labCall({ op: "held" });
    expect(made).toHaveLength(2);
    expect(ops(1)).toEqual(["held"]);
  });

  it("frees a pre-endgame's other parts, of no use alone", async () => {
    setLabPartsForTest(2);
    await labCall<number>({
      op: "endgame",
      args: { request: JSON.stringify({ model: "random_draws" }), parts: 2 },
    });
    made[1].onmessage?.({
      data: { event: { job: 12, error: "out of memory" }, spent: true },
    } as MessageEvent);
    expect(ops(0)).toContain("free");
  });
});
