import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  LabEvent,
  labCall,
  onLabEvent,
  partsFor,
  setLabPartsForTest,
} from "./lab";

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
    await labCall({ op: "run", args: { job, ms: 8 } });
    expect(lastOf(0, "run")?.args).toEqual({ job: 11, ms: 8, prune: false });
    expect(lastOf(1, "run")?.args).toEqual({ job: 11, ms: 8, prune: false });
    // a change goes to every part
    await labCall({ op: "add", args: { job, plays: "[]" } });
    expect(ops(0)).toContain("add");
    expect(ops(1)).toContain("add");
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
    await labCall({ op: "run", args: { job, ms: 8 } });
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

describe("how many workers a sim runs on", () => {
  it("runs on many only where the device says it has the memory", () => {
    expect(partsFor(8, 8)).toBe(4);
    expect(partsFor(3, 16)).toBe(2);
    expect(partsFor(8, 4)).toBe(1);
    expect(partsFor(8, 2)).toBe(1);
    // a browser that says nothing of its memory
    expect(partsFor(8, undefined)).toBe(2);
    expect(partsFor(1, undefined)).toBe(1);
    expect(partsFor(undefined, undefined)).toBe(1);
  });
});
