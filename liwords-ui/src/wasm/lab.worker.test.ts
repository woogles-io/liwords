import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The worker's module, run in the page's scope: its handler is set on self,
// and what it posts is caught. The wasm is a fake, each test saying what its
// calls answer.
const wasm = vi.hoisted(() => ({
  // the web build's instantiation, nothing to fetch here
  default: async () => {},
  precache_kwg: vi.fn(),
  precache_klv: vi.fn(),
  evict_kwg: vi.fn(() => true),
  evict_klv: vi.fn(() => true),
  analyze: vi.fn(),
  endgame_new: vi.fn(() => 7),
  endgame_step: vi.fn(),
  endgame_free: vi.fn(),
  sim_new: vi.fn((_request: string) => 9),
  sim_add: vi.fn(() =>
    JSON.stringify([
      { play: { action: "exchange", tiles: [] }, was: "absent" },
    ]),
  ),
  sim_remove: vi.fn(),
  sim_pause: vi.fn(() => "[]"),
  sim_resume: vi.fn(() => "[]"),
  sim_step: vi.fn(),
  sim_free: vi.fn(),
  sim_set: vi.fn(),
  infer: vi.fn(() =>
    JSON.stringify({
      handle: 4,
      count: 3,
      leaves: [{ tiles: [1], weight: 1 }],
    }),
  ),
  infer_free: vi.fn(),
  stitched_z: vi.fn((iterations: number, alpha: number) => iterations + alpha),
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

// how a sim prunes, told with every step: wolges's own numbers, the z the
// stitched boundary's at the sim's iterations so far (1 before any), at an
// alpha of 0.05
const pruned = {
  prune: true,
  prune_every: 16,
  prune_z: 1.05,
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

    // what is loaded already is not loaded again, its size still told
    const again = await send(3, "prepare", {
      files: [{ cacheKey: "kwg/NWL23", path: "/wasm/2024/NWL23.kwg" }],
    });
    expect(again.result).toEqual({
      "kwg/NWL23": "bytes:/wasm/2024/NWL23.kwg".length,
    });
    expect(wasm.precache_kwg).toHaveBeenCalledTimes(2);
    expect(wasm.evict_klv).toHaveBeenCalledWith("NWL23");
  });

  it("steps an endgame until it is exact, then frees it", async () => {
    wasm.endgame_step
      .mockReturnValueOnce(JSON.stringify({ exact: false, depth: 3 }))
      .mockReturnValueOnce(JSON.stringify({ exact: true, depth: 5 }));
    const made = await send(1, "endgame", { request: "{}" });
    expect(wasm.endgame_new).toHaveBeenCalledWith("{}");
    const job = made.result as number;

    await send(2, "run", { job, ms: 50, alpha: 0.05 });
    await vi.waitFor(() => expect(events().some((e) => e.done)).toBe(true));
    expect(events()).toEqual([
      { job, answer: { exact: false, depth: 3 }, done: false },
      { job, answer: { exact: true, depth: 5 }, done: true },
    ]);
    expect(wasm.endgame_step).toHaveBeenCalledWith(
      7,
      JSON.stringify({ ms: 50 }),
    );
    expect(wasm.endgame_step).toHaveBeenCalledTimes(2);

    // a sim's own calls are no endgame's
    expect((await send(3, "add", { job, plays: "[]" })).error).toBe(
      "no such sim",
    );

    await send(4, "free", { job });
    expect(wasm.endgame_free).toHaveBeenCalledWith(7);
    expect(events()[events().length - 1]).toEqual({ job, freed: true });
    expect((await send(5, "run", { job, ms: 50, alpha: 0.05 })).error).toBe(
      "no such job",
    );
  });

  it("stops an endgame whose step searched no node, out of memory", async () => {
    wasm.endgame_step
      .mockReturnValueOnce(JSON.stringify({ exact: false, nodes: 40 }))
      .mockReturnValueOnce(JSON.stringify({ exact: false, nodes: 40 }));
    const job = (await send(1, "endgame", { request: "{}" })).result as number;
    await send(2, "run", { job, ms: 16 });
    await vi.waitFor(() => expect(events().some((e) => e.error)).toBe(true));
    expect(events()).toEqual([
      { job, answer: { exact: false, nodes: 40 }, done: false },
      {
        job,
        error:
          "Out of memory: the solve has no room to search. Use fewer workers.",
      },
    ]);
    expect(wasm.endgame_step).toHaveBeenCalledTimes(2);
  });

  it("stops an endgame whose first step searched no node", async () => {
    wasm.endgame_step.mockReturnValueOnce(
      JSON.stringify({ exact: false, nodes: 0, play: null }),
    );
    const job = (await send(1, "endgame", { request: "{}" })).result as number;
    await send(2, "run", { job, ms: 16 });
    await vi.waitFor(() => expect(events().some((e) => e.error)).toBe(true));
    expect(wasm.endgame_step).toHaveBeenCalledTimes(1);
  });

  it("values an empty bag's every first move without stepping, only for an endgame", async () => {
    const answer = {
      exact: true,
      depth: 5,
      candidates: [{ play: { action: "exchange", tiles: [] }, value: 3 }],
    };
    wasm.endgame_step.mockReturnValueOnce(JSON.stringify(answer));
    const job = (await send(1, "endgame", { request: "{}" })).result as number;
    await send(2, "values", { job, ms: 1500 });
    expect(wasm.endgame_step).toHaveBeenCalledWith(
      7,
      JSON.stringify({ ms: 0, first_moves_ms: 1500 }),
    );
    expect(events()).toEqual([{ job, answer, done: true }]);
    const sim = (await send(3, "sim", { request: "{}", plays: "[]" }))
      .result as { job: number };
    expect((await send(4, "values", { job: sim.job, ms: 1 })).error).toBe(
      "no such endgame",
    );
  });

  it("steps an endgame the same frame every step, whatever its bag and depth", async () => {
    const stepsMs = () =>
      wasm.endgame_step.mock.calls.map(([, r]) => JSON.parse(r as string).ms);
    const answer = (bag: number, depth: number, exact = false) =>
      JSON.stringify({ bag, depth, exact });
    wasm.endgame_step
      .mockReturnValueOnce(answer(0, 1))
      .mockReturnValueOnce(answer(0, 1))
      .mockReturnValueOnce(answer(0, 1))
      .mockReturnValueOnce(answer(0, 2))
      .mockReturnValueOnce(answer(0, 2, true));
    const job = (await send(1, "endgame", { request: "{}" })).result as number;
    await send(2, "run", { job, ms: 8, alpha: 0.05 });
    await vi.waitFor(() => expect(events().some((e) => e.done)).toBe(true));
    expect(stepsMs()).toEqual([8, 8, 8, 8, 8]);

    // tiles in the bag: the same
    wasm.endgame_step.mockReset();
    wasm.endgame_step
      .mockReturnValueOnce(answer(2, 0))
      .mockReturnValueOnce(answer(2, 0))
      .mockReturnValueOnce(answer(2, 127, true));
    const last = (await send(3, "endgame", { request: "{}" })).result as number;
    await send(4, "run", { job: last, ms: 8, alpha: 0.05 });
    await vi.waitFor(() =>
      expect(events().filter((e) => e.done)).toHaveLength(2),
    );
    expect(stepsMs()).toEqual([8, 8, 8]);
  });

  it("prunes at the stitched boundary's z for the iterations the sim has run", async () => {
    wasm.sim_step
      .mockReturnValueOnce(
        JSON.stringify({
          iterations: 40,
          candidates: [{ state: "running" }, { state: "running" }],
        }),
      )
      .mockReturnValueOnce(simAnswer("running", "pruned"));
    const made = await send(1, "sim", { request: "{}", plays: "[]" });
    const { job } = made.result as { job: number };
    await send(2, "run", { job, ms: 10, alpha: 0.05 });
    await vi.waitFor(() => expect(events().some((e) => e.done)).toBe(true));
    expect(wasm.stitched_z.mock.calls).toEqual([
      [1, 0.05],
      [40, 0.05],
    ]);
    expect(JSON.parse(wasm.sim_step.mock.calls[1][1] as string).prune_z).toBe(
      40.05,
    );
  });

  it("changes a sim's plies or the opponent's tiles in place, answering as a step does", async () => {
    wasm.sim_set.mockReturnValueOnce(simAnswer("running", "running"));
    const made = await send(1, "sim", { request: "{}", plays: "[]" });
    const { job } = made.result as { job: number };
    const request = JSON.stringify({ plies: null });
    await send(2, "set", { job, request });
    expect(wasm.sim_set).toHaveBeenCalledWith(9, request);
    expect(events()).toEqual([
      {
        job,
        answer: { candidates: [{ state: "running" }, { state: "running" }] },
        done: false,
      },
    ]);
    // an endgame has no plies
    const endgame = (await send(3, "endgame", { request: "{}" }))
      .result as number;
    expect((await send(4, "set", { job: endgame, request })).error).toBe(
      "no such sim",
    );
  });

  it("compares a sim's candidates with a play on every step and read, until told none", async () => {
    wasm.sim_step.mockReturnValue(simAnswer("running", "running"));
    const made = await send(1, "sim", { request: "{}", plays: "[]" });
    const { job } = made.result as { job: number };
    const play = { action: "exchange", tiles: [] };
    await send(2, "against", { job, play: JSON.stringify(play) });
    await send(3, "read", { job });
    const requests = () =>
      wasm.sim_step.mock.calls.map(([, r]) => JSON.parse(r as string));
    expect(requests()[0].against).toEqual(play);
    await send(4, "run", { job, ms: 10, alpha: 0.05 });
    await vi.waitFor(() => expect(requests().length).toBeGreaterThan(1));
    await send(5, "stop");
    expect(requests()[1].against).toEqual(play);
    await send(6, "against", { job, play: null });
    await send(7, "read", { job });
    expect(requests()[requests().length - 1].against).toBeUndefined();
    wasm.sim_step.mockReset();
  });

  it("infers the opponent's leave for a sim, giving the sim its handle and letting it go", async () => {
    const made = await send(1, "sim", {
      request: JSON.stringify({ plies: 2, known: [] }),
      plays: "[]",
      infer: "the inference",
    });
    expect(wasm.infer).toHaveBeenCalledWith("the inference");
    expect(JSON.parse(wasm.sim_new.mock.calls[0][0] as string)).toEqual({
      plies: 2,
      inferred: 4,
    });
    expect(wasm.infer_free).toHaveBeenCalledWith(4);
    expect(made.result).toMatchObject({
      inferred: { count: 3, leaves: [{ tiles: [1], weight: 1 }] },
    });
    // for a sim made, through sim_set
    wasm.sim_set.mockReturnValueOnce(simAnswer("running", "running"));
    const { job } = made.result as { job: number };
    const set = await send(2, "set", {
      job,
      request: JSON.stringify({ plies: 2, known: [] }),
      infer: "again",
    });
    expect(JSON.parse(wasm.sim_set.mock.calls[0][1] as string)).toEqual({
      plies: 2,
      inferred: 4,
    });
    expect(wasm.infer_free).toHaveBeenCalledTimes(2);
    expect(set.result).toMatchObject({ inferred: { count: 3 } });
  });

  it("gives a sim no inference of no leaves, letting it go all the same", async () => {
    wasm.infer.mockReturnValueOnce(
      JSON.stringify({ handle: 5, count: 0, leaves: [] }),
    );
    const request = JSON.stringify({ plies: 2, known: [] });
    await send(1, "sim", { request, plays: "[]", infer: "{}" });
    expect(wasm.sim_new).toHaveBeenCalledWith(request);
    expect(wasm.infer_free).toHaveBeenCalledWith(5);
  });

  it("answers a sim's numbers against a play without stepping or sending them", async () => {
    wasm.sim_step.mockReturnValueOnce(simAnswer("running", "running"));
    const made = await send(1, "sim", { request: "{}", plays: "[]" });
    const { job } = made.result as { job: number };
    const play = { action: "exchange", tiles: [] };
    const peeked = await send(2, "peek", {
      job,
      against: JSON.stringify(play),
    });
    expect(JSON.parse(wasm.sim_step.mock.calls[0][1] as string)).toEqual({
      iterations: 0,
      prune: false,
      against: play,
    });
    expect(peeked.result).toEqual({
      candidates: [{ state: "running" }, { state: "running" }],
    });
    expect(events()).toEqual([]);
  });

  it("frees the last job when it makes the next, and says so", async () => {
    const sim = (
      (await send(1, "sim", { request: "{}", plays: "[]" })).result as {
        job: number;
      }
    ).job;
    const endgame = (await send(2, "endgame", { request: "{}" }))
      .result as number;
    expect(wasm.sim_free).toHaveBeenCalledWith(9);
    expect(events()).toEqual([{ job: sim, freed: true }]);
    expect(endgame).not.toBe(sim);
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

    await send(2, "run", { job, ms: 100, alpha: 0.05 });
    await vi.waitFor(() => expect(events().some((e) => e.done)).toBe(true));
    expect(events().map((e) => e.done)).toEqual([false, true]);
    // each step says how the sim prunes
    expect(wasm.sim_step.mock.calls).toEqual([
      [9, JSON.stringify({ ms: 100, ...pruned })],
      [9, JSON.stringify({ ms: 100, ...pruned })],
    ]);

    await send(3, "free", { job });
    expect(wasm.sim_free).toHaveBeenCalledWith(9);
    expect((await send(4, "run", { job, ms: 100, alpha: 0.05 })).error).toBe(
      "no such job",
    );
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
      JSON.stringify({ iterations: 0, prune: false }),
    );
    expect(events()).toEqual([{ job, answer: { candidates: [] }, done: true }]);

    expect((await send(5, "add", { job: job + 1, plays: "[]" })).error).toBe(
      "no such sim",
    );
  });

  it("pauses a sim's candidates and resumes them", async () => {
    const { job } = (await send(1, "sim", { request: "{}", plays: "[]" }))
      .result as { job: number };
    const plays = JSON.stringify([{ action: "exchange", tiles: [] }]);
    await send(2, "pause", { job, plays });
    expect(wasm.sim_pause).toHaveBeenCalledWith(9, plays);
    await send(3, "resume", { job, plays });
    expect(wasm.sim_resume).toHaveBeenCalledWith(9, plays);
  });

  it("rolls one candidate out while the others are paused, and is done once none runs", async () => {
    wasm.sim_step
      .mockReturnValueOnce(simAnswer("running", "paused", "pruned"))
      .mockReturnValueOnce(simAnswer("paused", "paused", "pruned"));
    const { job } = (await send(1, "sim", { request: "{}", plays: "[]" }))
      .result as { job: number };
    await send(2, "run", { job, ms: 10, alpha: 0.05 });
    await vi.waitFor(() => expect(events().some((e) => e.done)).toBe(true));
    expect(events().map((e) => e.done)).toEqual([false, true]);
  });

  it("prunes none when told, rolling out while any candidate runs", async () => {
    wasm.sim_step
      .mockReturnValueOnce(simAnswer("running", "pruned", "pruned"))
      .mockReturnValueOnce(simAnswer("paused", "pruned", "pruned"));
    const { job } = (await send(1, "sim", { request: "{}", plays: "[]" }))
      .result as { job: number };
    await send(2, "run", { job, ms: 10, prune: false });
    await vi.waitFor(() => expect(events().some((e) => e.done)).toBe(true));
    expect(events().map((e) => e.done)).toEqual([false, true]);
    expect(wasm.sim_step).toHaveBeenCalledWith(
      9,
      JSON.stringify({ ms: 10, prune: false }),
    );
    // and a read of it says so too
    wasm.sim_step.mockReturnValueOnce(JSON.stringify({ candidates: [] }));
    await send(3, "read", { job });
    expect(wasm.sim_step).toHaveBeenLastCalledWith(
      9,
      JSON.stringify({ iterations: 0, prune: false }),
    );
  });

  it("analyzes a position, keeping the job held while its tables stay", async () => {
    const files = [
      { cacheKey: "kwg/CSW24", path: "/wasm/2024/CSW24.kwg" },
      { cacheKey: "klv/CSW24", path: "/wasm/2024/CSW24.klv2" },
    ];
    await send(1, "prepare", { files });
    const { job } = (await send(2, "sim", { request: "{}", plays: "[]" }))
      .result as { job: number };
    await send(3, "prepare", { files });
    expect(wasm.sim_free).not.toHaveBeenCalled();

    const plays = [{ action: "exchange", tiles: [], equity: 1 }];
    wasm.analyze.mockReturnValueOnce(JSON.stringify(plays));
    const top = await send(4, "analyze", { request: '{"count":1}' });
    expect(wasm.analyze).toHaveBeenCalledWith('{"count":1}');
    expect(top.result).toEqual(plays);

    // tables dropped take the job with them
    await send(5, "prepare", { files: files.slice(0, 1) });
    expect(wasm.sim_free).toHaveBeenCalledWith(9);
    expect(events()).toEqual([{ job, freed: true }]);
  });

  it("stops stepping when told", async () => {
    wasm.sim_step.mockReturnValue(simAnswer("running", "running"));
    const { job } = (await send(1, "sim", { request: "{}", plays: "[]" }))
      .result as { job: number };
    await send(2, "run", { job, ms: 10, alpha: 0.05 });
    await vi.waitFor(() => expect(events().length).toBeGreaterThan(1));
    await send(3, "stop");
    const steps = wasm.sim_step.mock.calls.length;
    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(wasm.sim_step.mock.calls.length).toBe(steps);
  });
});
