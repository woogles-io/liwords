// The analyzer lab's own wolges-wasm instance, for its long jobs: a sim or an
// endgame, stepped a few milliseconds at a time so that the page never waits
// on one. It holds one job at a time, and only the tables that job needs.
//
// A request {id, op, args} gets {id, result} or {id, error} back, in the order
// sent. While a job runs, each step's answer comes as {event: {job, answer,
// done}}, done once the job has nothing left to do, and a failed step as
// {event: {job, error}}. A job freed, by a request or by the next job made,
// says so as {event: {job, freed: true}}.

import { OUT_OF_MEMORY } from "./lab_errors";
import { importWolges } from "./wolges";

// The wasm is imported, not loaded statically: a static import would make this
// an async module whose body, the message handler with it, runs only once the
// wasm is instantiated, and a message sent before then would be lost.
const wolgesPromise = importWolges();

type Wolges = Awaited<typeof wolgesPromise>;
type LabFile = { cacheKey: string; path: string };
export type LabRequest =
  // the files a job needs loaded, answered with each one's size in bytes
  | { op: "prepare"; args: { files: Array<LabFile> } }
  // the position's plays, best first, as many as the request's count asks
  | { op: "analyze"; args: { request: string } }
  // what the opponent may play next, by score alone: every play listed, then
  // the racks weighed a step at a time, each step answering as the whole
  // count stands; with an inference's request, from racks starting with the
  // leave inferred, as a sim's does. Each count is held, apart from any job,
  // till it is let go; a step of ms 0 weighs nothing more, answering as the
  // count stands, with the top count plays, through the square if given.
  | { op: "threats_new"; args: { request: string; infer?: string } }
  | {
      op: "threats_step";
      args: {
        id: number;
        ms: number;
        count?: number;
        square?: [number, number];
      };
    }
  | { op: "threats_free"; args: { id: number } }
  // an endgame; a pre-endgame on as many workers as parts says, if fewer than
  // the lab's (lab.ts), which a worker reads no further
  | { op: "endgame"; args: { request: string; parts?: number } }
  // a sim of the plays given; with an inference's request, the opponent's
  // leave inferred, as infer takes one, and dealt in each iteration
  | { op: "sim"; args: { request: string; plays: string; infer?: string } }
  // plays joining a sim, or leaving it; each answers as the sim holds it
  | { op: "add"; args: { job: number; plays: string } }
  | { op: "remove"; args: { job: number; plays: string } }
  // candidates held out of a sim's rollouts, keeping theirs, or let back in
  | { op: "pause"; args: { job: number; plays: string } }
  | { op: "resume"; args: { job: number; plays: string } }
  // a sim's plies or the opponent's tiles changed in place, its rollouts kept,
  // answering as a step does
  | { op: "set"; args: { job: number; request: string; infer?: string } }
  // the play each of a sim's candidates is compared with in its answers, on
  // the iterations both ran, or none
  | { op: "against"; args: { job: number; play: string | null } }
  // the job's answer as it stands, sent as a step's is, without stepping
  | { op: "read"; args: { job: number } }
  // an empty bag's endgame, every first move valued on the table its steps
  // kept, in at most ms, without stepping; sent as a step's answer is
  | { op: "values"; args: { job: number; ms: number } }
  // a sim's answer as it stands, compared with the play given in place of the
  // one it is compared with, answered and not sent
  | { op: "peek"; args: { job: number; against: string } }
  // the job stepped ms at a time until it is done or stopped; a sim told
  // prune false prunes none of its candidates
  | {
      op: "run";
      args: { job: number; ms: number; prune?: boolean; alpha?: number };
    }
  // what the opponent likely kept, from their last move: how many leaves,
  // the likeliest, and by tile the chance their rack holds one, as inferred
  // and with nothing inferred
  | { op: "infer"; args: { request: string } }
  // the bytes the wasm holds now, as an endgame's memory cap counts them
  | { op: "held" }
  | { op: "stop" }
  | { op: "free"; args: { job: number } };
type Job = {
  id: number;
  kind: "sim" | "endgame";
  handle: number;
  // false when a sim's last run said to prune none of its candidates, and
  // the alpha it said to prune at otherwise
  prune?: boolean;
  alpha?: number;
  // a sim's iterations, as its last answer had them
  iterations?: number;
  // an endgame's nodes searched, as its last step's answer had them
  nodes?: number;
  // a pre-endgame (one asked with a model of the bag), whose answers count no
  // nodes, so say nothing of its room
  modeled?: boolean;
  // the play a sim's candidates are compared with, as the sim takes one
  against?: unknown;
};
type Answer = {
  exact?: boolean;
  // true when the step stopped short of exact on the memory cap, in wasms
  // that say so
  out_of_memory?: boolean;
  nodes?: number;
  iterations?: number;
  candidates?: Array<{ state: string }>;
};

// How a sim prunes its candidates, told to it at every step: wolges's own
// look every PRUNE_CADENCE iterations and win chance within 1/1000 of 0 or 1
// taken as decided (https://github.com/andy-k/wolges/blob/main/src/move_picker.rs);
// and, as a sim here has no iteration budget, the stitched boundary's z at
// its iterations so far (stitched_z), at the alpha its run was given: the
// chance it ever prunes its best candidate behind another, however long it
// runs. A run that prunes and gives no alpha fails.
const PRUNE_EVERY = 16;
const PRUNE_DECIDED = 0.001;
const pruning = (wolges: Wolges, j: Job) => {
  if (j.prune !== false && j.alpha === undefined) {
    throw new Error("a pruning run needs an alpha");
  }
  return {
    ...(j.prune !== false
      ? {
          prune: true,
          prune_every: PRUNE_EVERY,
          prune_z: wolges.stitched_z(Math.max(1, j.iterations ?? 0), j.alpha!),
          prune_decided: PRUNE_DECIDED,
        }
      : { prune: false }),
    ...(j.against !== undefined ? { against: j.against } : {}),
  };
};

const post = (message: unknown) =>
  (self as unknown as Worker).postMessage(message);
const errorText = (e: unknown) => (e instanceof Error ? e.message : String(e));
const nameOf = (cacheKey: string) =>
  cacheKey.substring(cacheKey.indexOf("/") + 1);

// The cache keys loaded, as the page's loader names them, each with its
// size in bytes.
const loaded = new Map<string, number>();
let job: Job | null = null;
let lastJobId = 0;
// The threats counts held, each id's wasm handle.
const threats = new Map<number, number>();
const freeThreats = (wolges: Wolges) => {
  threats.forEach((handle) => wolges.threats_free(handle));
  threats.clear();
};
let running = false;
// Each run starts a new chain of steps; an older chain sees it and ends.
let chainId = 0;

// Each step is let through by a message the worker sends itself, so that the
// requests that came in meanwhile get in first. A timeout would do the same,
// but browsers make a chain of timeouts wait 4 ms or more each.
const stepPort = new MessageChannel();
const nextSteps: Array<() => void> = [];
stepPort.port1.onmessage = () => nextSteps.shift()?.();
const later = (next: () => void) => {
  nextSteps.push(next);
  stepPort.port2.postMessage(null);
};

// A sim's request, with the opponent's leave inferred when asked: the
// inference made, its handle given in place of the known tiles, and let go
// once the sim has its leaves, which the sim keeps. An inference of no leaves
// (as when the bag is empty) leaves the request as it was. The inference's
// count and likeliest leaves come back with the call's own result.
const inferring = <T>(
  wolges: Wolges,
  request: string,
  infer: string | undefined,
  call: (request: string) => T,
): { result: T; inferred?: { count: number; leaves: unknown } } => {
  if (infer === undefined) return { result: call(request) };
  const { handle, count, leaves } = JSON.parse(wolges.infer(infer)) as {
    handle: number;
    count: number;
    leaves: unknown;
  };
  try {
    const result = call(
      count > 0
        ? JSON.stringify({
            ...JSON.parse(request),
            known: undefined,
            inferred: handle,
          })
        : request,
    );
    return { result, inferred: { count, leaves } };
  } finally {
    wolges.infer_free(handle);
  }
};

const freeJob = (wolges: Wolges) => {
  running = false;
  if (!job) return;
  if (job.kind === "sim") wolges.sim_free(job.handle);
  else wolges.endgame_free(job.handle);
  post({ event: { job: job.id, freed: true } });
  job = null;
};

// Nothing left to do: an exact endgame; or a sim with no candidate running,
// or, unless it prunes none, with fewer than two candidates it has not pruned
// (running or paused), nothing left to tell apart.
const isDone = (j: Job, answer: Answer) => {
  if (j.kind === "endgame") return !!answer.exact;
  const states = (answer.candidates ?? []).map((c) => c.state);
  return (
    !states.includes("running") ||
    (j.prune !== false && states.filter((s) => s !== "pruned").length < 2)
  );
};

const step = async (j: Job, chain: number, ms: number) => {
  if (!running || job !== j || chain !== chainId) return;
  const wolges = await wolgesPromise;
  try {
    // each step, a sim's or an endgame's, goes on where the last one stopped,
    // so every step runs the same frame
    const request = JSON.stringify(
      j.kind === "sim" ? { ms, ...pruning(wolges, j) } : { ms },
    );
    const answer: Answer = JSON.parse(
      j.kind === "sim"
        ? wolges.sim_step(j.handle, request)
        : wolges.endgame_step(j.handle, request),
    );
    j.iterations = answer.iterations;
    // A step of an endgame not yet exact that searched no node stopped on the
    // memory cap, which counts every byte the instance holds, the lexicon too:
    // no step after it can search either. A pre-endgame's answers count no
    // nodes, searching or not, so tell nothing of it; a wasm that says when a
    // step stopped on the cap says so for either.
    if (
      j.kind === "endgame" &&
      (answer.out_of_memory ||
        (!j.modeled &&
          !answer.exact &&
          answer.nodes !== undefined &&
          answer.nodes === j.nodes))
    ) {
      running = false;
      post({ event: { job: j.id, error: OUT_OF_MEMORY } });
      return;
    }
    j.nodes = answer.nodes;
    const done = isDone(j, answer);
    if (done) running = false;
    post({ event: { job: j.id, answer, done } });
    if (done) return;
  } catch (e) {
    running = false;
    post({ event: { job: j.id, error: errorText(e) } });
    return;
  }
  later(() => step(j, chain, ms));
};

// Load the files a job needs, after dropping the ones it does not: one
// lexicon's tables at a time, as on the page. The job held goes first when
// any are dropped, and stays when none are.
const prepare = async (wolges: Wolges, files: Array<LabFile>) => {
  const wanted = new Set(files.map((f) => f.cacheKey));
  const unwanted = [...loaded.keys()].filter(
    (cacheKey) => !wanted.has(cacheKey),
  );
  if (unwanted.length) {
    freeJob(wolges);
    freeThreats(wolges);
  }
  for (const cacheKey of unwanted) {
    if (cacheKey.startsWith("klv/")) wolges.evict_klv(nameOf(cacheKey));
    else wolges.evict_kwg(nameOf(cacheKey));
    loaded.delete(cacheKey);
  }
  await Promise.all(
    files
      .filter((f) => !loaded.has(f.cacheKey))
      .map(async ({ cacheKey, path }) => {
        const resp = await fetch(path);
        if (!resp.ok) throw new Error(`Unable to cache ${cacheKey}`);
        const bytes = new Uint8Array(await resp.arrayBuffer());
        if (cacheKey.startsWith("klv/")) {
          wolges.precache_klv(nameOf(cacheKey), bytes);
        } else {
          wolges.precache_kwg(nameOf(cacheKey), bytes);
        }
        loaded.set(cacheKey, bytes.length);
      }),
  );
  return Object.fromEntries(
    files.map(({ cacheKey }) => [cacheKey, loaded.get(cacheKey)!]),
  );
};

const handle = async (req: LabRequest): Promise<unknown> => {
  const wolges = await wolgesPromise;
  switch (req.op) {
    case "prepare":
      return prepare(wolges, req.args.files);
    case "held":
      return wolges.memory_held();
    case "infer": {
      const { handle, ...inferred } = JSON.parse(
        wolges.infer(req.args.request),
      ) as { handle: number };
      wolges.infer_free(handle);
      return inferred;
    }
    case "analyze":
      return JSON.parse(wolges.analyze(req.args.request));
    case "threats_new": {
      const { result: handle, inferred } = inferring(
        wolges,
        req.args.request,
        req.args.infer,
        wolges.threats_new,
      );
      const id = ++lastJobId;
      threats.set(id, handle);
      return { id, ...(inferred ? { inferred } : {}) };
    }
    case "threats_step": {
      const { id, ...step } = req.args;
      const handle = threats.get(id);
      if (handle === undefined) throw new Error("no such threats count");
      return JSON.parse(wolges.threats_step(handle, JSON.stringify(step)));
    }
    case "threats_free": {
      const handle = threats.get(req.args.id);
      if (handle !== undefined) wolges.threats_free(handle);
      threats.delete(req.args.id);
      return undefined;
    }
    case "endgame":
      freeJob(wolges);
      job = {
        id: ++lastJobId,
        kind: "endgame",
        handle: wolges.endgame_new(req.args.request),
        // a new endgame has searched none
        nodes: 0,
        modeled: JSON.parse(req.args.request).model !== undefined,
      };
      return job.id;
    case "sim": {
      freeJob(wolges);
      const { result: handle, inferred } = inferring(
        wolges,
        req.args.request,
        req.args.infer,
        wolges.sim_new,
      );
      let added: unknown;
      try {
        // each play given, as the sim holds it
        added = JSON.parse(wolges.sim_add(handle, req.args.plays));
      } catch (e) {
        wolges.sim_free(handle);
        throw e;
      }
      job = { id: ++lastJobId, kind: "sim", handle };
      return { job: job.id, added, ...(inferred ? { inferred } : {}) };
    }
    case "add":
    case "remove":
    case "pause":
    case "resume": {
      if (!job || job.id !== req.args.job || job.kind !== "sim") {
        throw new Error("no such sim");
      }
      const change = {
        add: wolges.sim_add,
        remove: wolges.sim_remove,
        pause: wolges.sim_pause,
        resume: wolges.sim_resume,
      }[req.op];
      return JSON.parse(change(job.handle, req.args.plays));
    }
    case "set": {
      if (!job || job.id !== req.args.job || job.kind !== "sim") {
        throw new Error("no such sim");
      }
      const { handle } = job;
      const { result, inferred } = inferring(
        wolges,
        req.args.request,
        req.args.infer,
        (request) => wolges.sim_set(handle, request),
      );
      const answer: Answer = JSON.parse(result);
      post({ event: { job: job.id, answer, done: isDone(job, answer) } });
      return inferred ? { inferred } : {};
    }
    case "against": {
      if (!job || job.id !== req.args.job || job.kind !== "sim") {
        throw new Error("no such sim");
      }
      job.against =
        req.args.play === null ? undefined : JSON.parse(req.args.play);
      return undefined;
    }
    case "peek": {
      if (!job || job.id !== req.args.job || job.kind !== "sim") {
        throw new Error("no such sim");
      }
      return JSON.parse(
        wolges.sim_step(
          job.handle,
          JSON.stringify({
            iterations: 0,
            prune: false,
            against: JSON.parse(req.args.against),
          }),
        ),
      );
    }
    case "read": {
      if (!job || job.id !== req.args.job) throw new Error("no such job");
      const answer: Answer = JSON.parse(
        job.kind === "sim"
          ? wolges.sim_step(
              job.handle,
              JSON.stringify({
                // a read runs no iteration, so prunes nothing
                iterations: 0,
                prune: false,
                ...(job.against !== undefined ? { against: job.against } : {}),
              }),
            )
          : wolges.endgame_step(job.handle, JSON.stringify({ ms: 0 })),
      );
      post({ event: { job: job.id, answer, done: isDone(job, answer) } });
      return undefined;
    }
    case "values": {
      if (!job || job.id !== req.args.job || job.kind !== "endgame") {
        throw new Error("no such endgame");
      }
      const answer: Answer = JSON.parse(
        wolges.endgame_step(
          job.handle,
          JSON.stringify({ ms: 0, first_moves_ms: req.args.ms }),
        ),
      );
      post({ event: { job: job.id, answer, done: isDone(job, answer) } });
      return undefined;
    }
    case "run": {
      if (!job || job.id !== req.args.job) throw new Error("no such job");
      running = true;
      const j = job;
      j.prune = req.args.prune;
      j.alpha = req.args.alpha;
      const chain = ++chainId;
      const { ms } = req.args;
      later(() => step(j, chain, ms));
      return undefined;
    }
    case "stop":
      running = false;
      return undefined;
    case "free":
      if (job && job.id === req.args.job) freeJob(wolges);
      return undefined;
  }
};

// One request at a time, so that a job is never made before its tables are
// loaded. Steps run between requests, outside this queue.
let queue: Promise<void> = Promise.resolve();
self.onmessage = (e: MessageEvent) => {
  const { id, ...req } = e.data;
  queue = queue.then(() =>
    handle(req as LabRequest).then(
      (result) => post({ id, result }),
      (err) => post({ id, error: errorText(err) }),
    ),
  );
};
