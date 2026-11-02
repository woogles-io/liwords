import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The worker's module, run in the page's scope: its handler is set on self,
// and what it posts is caught. The wasm is a fake, each test saying what its
// calls answer.
const wasm = vi.hoisted(() => ({
  precache_kwg: vi.fn(),
  precache_klv: vi.fn(),
  evict_kwg: vi.fn(() => true),
  evict_klv: vi.fn(() => true),
  sim_new: vi.fn(() => 9),
  sim_add: vi.fn(() =>
    JSON.stringify([
      { play: { action: "exchange", tiles: [] }, was: "absent" },
    ]),
  ),
  sim_remove: vi.fn(),
  sim_step: vi.fn(),
  sim_free: vi.fn(),
}));
vi.mock("wolges-wasm", () => wasm);

const posted: Array<{
  id?: number;
  result?: unknown;
  error?: string;
  event?: {
    job: number;
    answer?: unknown;
    done?: boolean;
    error?: string;
    freed?: boolean;
  };
}> = [];

const send = async (id: number, op: string, args?: object) => {
  (self.onmessage as (e: MessageEvent) => void)({
    data: { id, op, args },
  } as MessageEvent);
  await vi.waitFor(() => expect(posted.some((m) => m.id === id)).toBe(true));
  return posted.find((m) => m.id === id)!;
};

const events = () => posted.filter((m) => m.event).map((m) => m.event!);

// a sim's answer: its candidates, each in the state given
const simAnswer = (...states: Array<string>) =>
  JSON.stringify({ candidates: states.map((state) => ({ state })) });

// how a sim prunes, told with every step: wolges's own numbers
const pruned = {
  prune: true,
  prune_every: 16,
  prune_z: 3.153563157004781,
  prune_decided: 0.001,
};

beforeEach(async () => {
  vi.clearAllMocks();
  posted.length = 0;
  vi.stubGlobal("postMessage", (m: (typeof posted)[number]) => posted.push(m));
  vi.stubGlobal("fetch", (path: string) =>
    Promise.resolve(new Response(`bytes:${path}`, { status: 200 })),
  );
  vi.resetModules();
  await import("./lab.worker");
});

afterEach(async () => {
  // steps a failed test left running stop here, not in the next test, the
  // stop answered while what the worker posts is still caught
  await send(0, "stop");
  vi.unstubAllGlobals();
});

describe("the lab's worker", () => {
  it("loads a job's files, and drops the last job's", async () => {
    await send(1, "prepare", {
      files: [{ cacheKey: "kwg/CSW24", path: "/wasm/2024/CSW24.kwg" }],
    });
    expect(wasm.precache_kwg).toHaveBeenCalledWith(
      "CSW24",
      expect.any(Uint8Array),
    );

    await send(2, "prepare", {
      files: [
        { cacheKey: "kwg/NWL23", path: "/wasm/2024/NWL23.kwg" },
        { cacheKey: "klv/NWL23", path: "/wasm/2024/NWL23.klv2" },
      ],
    });
    expect(wasm.evict_kwg).toHaveBeenCalledWith("CSW24");
    expect(wasm.precache_kwg).toHaveBeenCalledWith(
      "NWL23",
      expect.any(Uint8Array),
    );
    expect(wasm.precache_klv).toHaveBeenCalledWith(
      "NWL23",
      expect.any(Uint8Array),
    );

    // what is loaded already is not loaded again
    await send(3, "prepare", {
      files: [{ cacheKey: "kwg/NWL23", path: "/wasm/2024/NWL23.kwg" }],
    });
    expect(wasm.precache_kwg).toHaveBeenCalledTimes(2);
    expect(wasm.evict_klv).toHaveBeenCalledWith("NWL23");
  });

  it("frees the last job when it makes the next, and says so", async () => {
    const sim = (
      (await send(1, "sim", { request: "{}", plays: "[]" })).result as {
        job: number;
      }
    ).job;
    const next = (
      (await send(2, "sim", { request: "{}", plays: "[]" })).result as {
        job: number;
      }
    ).job;
    expect(wasm.sim_free).toHaveBeenCalledWith(9);
    expect(events()).toEqual([{ job: sim, freed: true }]);
    expect(next).not.toBe(sim);
  });

  it("makes a sim of the plays given, steps it until fewer than two are left unpruned, then frees it", async () => {
    wasm.sim_step
      .mockReturnValueOnce(simAnswer("running", "running", "pruned"))
      .mockReturnValueOnce(simAnswer("running", "pruned", "pruned"));
    const made = await send(1, "sim", { request: "{}", plays: "[]" });
    expect(wasm.sim_new).toHaveBeenCalledWith("{}");
    expect(wasm.sim_add).toHaveBeenCalledWith(9, "[]");
    const { job, added } = made.result as { job: number; added: unknown };
    expect(added).toEqual([
      { play: { action: "exchange", tiles: [] }, was: "absent" },
    ]);

    await send(2, "run", { job, ms: 100 });
    await vi.waitFor(() => expect(events().some((e) => e.done)).toBe(true));
    expect(events().map((e) => e.done)).toEqual([false, true]);
    // each step says how the sim prunes
    expect(wasm.sim_step.mock.calls).toEqual([
      [9, JSON.stringify({ ms: 100, ...pruned })],
      [9, JSON.stringify({ ms: 100, ...pruned })],
    ]);

    await send(3, "free", { job });
    expect(wasm.sim_free).toHaveBeenCalledWith(9);
    expect((await send(4, "run", { job, ms: 100 })).error).toBe("no such job");
  });

  it("frees a sim whose plays the wasm turns down", async () => {
    wasm.sim_add.mockImplementationOnce(() => {
      throw "plays[0]: no such play";
    });
    expect((await send(1, "sim", { request: "{}", plays: "[]" })).error).toBe(
      "plays[0]: no such play",
    );
    expect(wasm.sim_free).toHaveBeenCalledWith(9);
  });

  it("adds plays to a sim and removes them, and reads it as it stands", async () => {
    const job = (
      (await send(1, "sim", { request: "{}", plays: "[]" })).result as {
        job: number;
      }
    ).job;
    const play = { action: "play", down: false, lane: 7, idx: 7, word: [1] };
    wasm.sim_add.mockReturnValueOnce(JSON.stringify([{ play, was: "absent" }]));
    const added = await send(2, "add", { job, plays: JSON.stringify([play]) });
    expect(wasm.sim_add).toHaveBeenLastCalledWith(9, JSON.stringify([play]));
    expect(added.result).toEqual([{ play, was: "absent" }]);

    wasm.sim_remove.mockReturnValueOnce(
      JSON.stringify([{ play, was: "running" }]),
    );
    const removed = await send(3, "remove", {
      job,
      plays: JSON.stringify([play]),
    });
    expect(removed.result).toEqual([{ play, was: "running" }]);

    // a read steps nothing and answers as a step does, saying how the sim
    // prunes as a step does
    wasm.sim_step.mockReturnValueOnce(JSON.stringify({ candidates: [] }));
    await send(4, "read", { job });
    expect(wasm.sim_step).toHaveBeenCalledWith(
      9,
      JSON.stringify({ iterations: 0, ...pruned }),
    );
    expect(events()).toEqual([{ job, answer: { candidates: [] }, done: true }]);

    expect((await send(5, "add", { job: job + 1, plays: "[]" })).error).toBe(
      "no such sim",
    );
  });

  it("stops stepping when told", async () => {
    wasm.sim_step.mockReturnValue(simAnswer("running", "running"));
    const { job } = (await send(1, "sim", { request: "{}", plays: "[]" }))
      .result as { job: number };
    await send(2, "run", { job, ms: 10 });
    await vi.waitFor(() => expect(events().length).toBeGreaterThan(1));
    await send(3, "stop");
    const steps = wasm.sim_step.mock.calls.length;
    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(wasm.sim_step.mock.calls.length).toBe(steps);
  });
});
