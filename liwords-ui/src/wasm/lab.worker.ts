// The analyzer lab's own wolges-wasm instance, for its long jobs, stepped a
// few milliseconds at a time so that the page never waits on one. It holds one
// job at a time, and only the tables that job needs.
//
// A request {id, op, args} gets {id, result} or {id, error} back, in the order
// sent. While a job runs, each step's answer comes as {event: {job, answer,
// done}}, done once the job has nothing left to do, and a failed step as
// {event: {job, error}}. A job freed, by a request or by the next job made,
// says so as {event: {job, freed: true}}.
//
// The wasm is imported, not loaded statically: a static import would make this
// an async module whose body, the message handler with it, runs only once the
// wasm is instantiated, and a message sent before then would be lost.
const wolgesPromise = import("wolges-wasm");

type Wolges = Awaited<typeof wolgesPromise>;
type LabFile = { cacheKey: string; path: string };
export type LabRequest =
  | { op: "prepare"; args: { files: Array<LabFile> } }
  | { op: "sim"; args: { request: string; plays: string } }
  // the job stepped ms at a time until it is done or stopped
  | { op: "run"; args: { job: number; ms: number } }
  | { op: "stop" }
  | { op: "free"; args: { job: number } };
type Job = { id: number; handle: number };
type Answer = { candidates?: Array<{ state: string }> };

// How a sim prunes its candidates, told to it at every step: wolges's own
// numbers (https://github.com/andy-k/wolges/blob/main/src/move_picker.rs: a
// look every PRUNE_CADENCE iterations, the z of paired_z(1000), and a win
// chance within 1/1000 of 0 or 1 taken as decided), which the wasm in use
// keeps as its own.
const PRUNE_EVERY = 16;
const PRUNE_Z = 3.153563157004781;
const PRUNE_DECIDED = 0.001;
const pruning = {
  prune: true,
  prune_every: PRUNE_EVERY,
  prune_z: PRUNE_Z,
  prune_decided: PRUNE_DECIDED,
};

const post = (message: unknown) =>
  (self as unknown as Worker).postMessage(message);
const errorText = (e: unknown) => (e instanceof Error ? e.message : String(e));
const nameOf = (cacheKey: string) =>
  cacheKey.substring(cacheKey.indexOf("/") + 1);

// The cache keys loaded, as the page's loader names them.
const loaded = new Set<string>();
let job: Job | null = null;
let lastJobId = 0;
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

const freeJob = (wolges: Wolges) => {
  running = false;
  if (!job) return;
  wolges.sim_free(job.handle);
  post({ event: { job: job.id, freed: true } });
  job = null;
};

// Nothing left to do: a sim with fewer than two candidates it has not pruned,
// nothing left to tell apart.
const isDone = (answer: Answer) =>
  (answer.candidates ?? []).filter((c) => c.state !== "pruned").length < 2;

const step = async (j: Job, chain: number, ms: number) => {
  if (!running || job !== j || chain !== chainId) return;
  const wolges = await wolgesPromise;
  try {
    const request = JSON.stringify({ ms, ...pruning });
    const answer: Answer = JSON.parse(wolges.sim_step(j.handle, request));
    const done = isDone(answer);
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
// lexicon's tables at a time, as on the page.
const prepare = async (wolges: Wolges, files: Array<LabFile>) => {
  freeJob(wolges);
  const wanted = new Set(files.map((f) => f.cacheKey));
  for (const cacheKey of [...loaded]) {
    if (wanted.has(cacheKey)) continue;
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
        loaded.add(cacheKey);
      }),
  );
};

const handle = async (req: LabRequest): Promise<unknown> => {
  const wolges = await wolgesPromise;
  switch (req.op) {
    case "prepare":
      await prepare(wolges, req.args.files);
      return undefined;
    case "sim": {
      freeJob(wolges);
      const handle = wolges.sim_new(req.args.request);
      let added: unknown;
      try {
        // each play given, as the sim holds it
        added = JSON.parse(wolges.sim_add(handle, req.args.plays));
      } catch (e) {
        wolges.sim_free(handle);
        throw e;
      }
      job = { id: ++lastJobId, handle };
      return { job: job.id, added };
    }
    case "run": {
      if (!job || job.id !== req.args.job) throw new Error("no such job");
      running = true;
      const j = job;
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
