import type { LabRequest } from "./lab.worker";
import {
  PartAnswer,
  behindLeader,
  leaderOf,
  mergeAnswers,
  stitchedZ,
} from "./lab_merge";

// The page's side of the analyzer lab's workers (lab.worker.ts), made on first
// use and kept for the tab. labCall sends a request and resolves with its
// answer; onLabEvent hears each step of the job that is running, and when a job
// is freed.
//
// A sim runs on every worker at once, where the device has the cores and the
// memory for more than one: each worker holds its own wasm, lexicon and leave,
// and rolls out its part of the one sim (the same seed, part i of n), so that
// the parts together are the sim one worker would run. Their answers are
// merged as they come, and with more than one part the page prunes on the
// merged numbers, as the wasm prunes one sim's: no part prunes on its own,
// which would leave the parts with other candidates. Every other job, an
// endgame, an analysis, runs on the first worker.

export type LabEvent = {
  job: number;
  answer?: unknown;
  done?: boolean;
  error?: string;
  freed?: boolean;
};

// A request a worker has not answered yet: to whom, what, and when sent.
type Pending = {
  resolve: (result: unknown) => void;
  reject: (e: Error) => void;
  worker: number;
  request: LabRequest;
};

type WorkerEvent = LabEvent;

// How many parts a sim runs in, a worker each, each holding its own wasm and
// tables: one until the page sets more.
let parts = 1;
export const labParts = () => parts;

// The parts the sims made from now on run in.
export const setLabParts = (n: number) => {
  parts = Math.max(1, Math.floor(n));
};

// For tests: the parts a sim runs in from now on, the workers made anew.
export const setLabPartsForTest = (n: number) => {
  parts = n;
  workers.forEach((w) => w?.terminate());
  workers = [];
  current.clear();
  busy.clear();
  pending.clear();
  sims.clear();
  jobs.clear();
};

let workers: Array<Worker | undefined> = [];
// The job each worker last made, as the worker numbers it.
const current = new Map<number, number>();
let lastId = 0;
const pending = new Map<number, Pending>();
const listeners = new Set<(e: LabEvent) => void>();

// The page's jobs, by the page's own number for each: a job is one worker's
// (by its number there), or a sim's parts, one on each worker.
let lastJob = 0;
const jobs = new Map<number, Array<{ worker: number; job: number }>>();

// How a sim prunes on its merged numbers: as the worker prunes one part's
// (lab.worker.ts), at the alpha its run was given.
const PRUNE_EVERY = 16;
const PRUNE_DECIDED = 0.001;

// A sim in parts as the page keeps it: each part's last answer, whether it
// prunes, the merged iterations it last looked at, the candidates it pruned
// (by the play each was given as), the play each candidate was given as (by
// how it answers), whether a look is under way, and whether it is running.
type Sim = {
  answers: Array<PartAnswer | undefined>;
  prune: boolean;
  alpha: number;
  looked: number;
  pruned: Set<string>;
  given: Map<string, unknown>;
  looking: boolean;
  running: boolean;
  freed: boolean;
};
const sims = new Map<number, Sim>();

const emit = (e: LabEvent) => listeners.forEach((listener) => listener(e));

const getWorker = (w: number) => {
  let worker = workers[w];
  if (!worker) {
    worker = new Worker(new URL("./lab.worker.ts", import.meta.url));
    worker.onmessage = (e: MessageEvent) => {
      const { id, result, error, event } = e.data;
      if (event) {
        if (event.done || event.error || event.freed) busy.delete(w);
        heard(w, event);
        return;
      }
      const p = pending.get(id);
      if (!p) return;
      pending.delete(id);
      if (error !== undefined) {
        p.reject(new Error(error));
        return;
      }
      // a stop answered: the worker runs no steps
      if (p.request.op === "stop") busy.delete(w);
      // a job made is the one the worker now holds
      if (p.request.op === "endgame") current.set(w, result as number);
      if (p.request.op === "sim") {
        current.set(w, (result as { job: number }).job);
      }
      p.resolve(result);
    };
    workers[w] = worker;
  }
  return worker;
};

// The workers running a job's steps: from a run until the run is done, fails,
// or a stop is answered.
const busy = new Set<number>();

// A job is freed by sending the worker a free, unless the worker is still
// running its steps: a step may be inside a wasm call that does not give the
// worker back, so the worker is ended at once (the job was to go anyway) and
// a new one takes its place. Nothing waits on a timer: how long to let a job
// run is the user's to decide.
const send = (id: number, p: Pending) => {
  if (p.request.op === "free" && busy.has(p.worker)) {
    replaceWorker(p.worker);
    p.resolve(undefined);
    return;
  }
  if (p.request.op === "run") busy.add(p.worker);
  pending.set(id, p);
  getWorker(p.worker).postMessage({ id, ...p.request });
};

const call = <T>(worker: number, request: LabRequest): Promise<T> =>
  new Promise<T>((resolve, reject) => {
    send(++lastId, {
      resolve: resolve as (result: unknown) => void,
      reject,
      worker,
      request,
    });
  });

// A worker ended, and another made in its place, which holds no tables and no
// job: the job it held is freed, saying so; the requests waiting on it that
// make something new go to the new worker; a stop or a free is done; any
// other, for a job that is gone, fails.
const replaceWorker = (w: number) => {
  workers[w]?.terminate();
  workers[w] = undefined;
  busy.delete(w);
  const gone = current.get(w);
  current.delete(w);
  const waiting = [...pending].filter(([, p]) => p.worker === w);
  waiting.forEach(([id]) => pending.delete(id));
  if (labParts() === 1) {
    if (gone !== undefined) emit({ job: gone, freed: true });
  } else {
    for (const [page, held] of [...jobs]) {
      if (held.some((h) => h.worker === w)) {
        if (sims.has(page)) freeParts(page, w);
        else jobs.delete(page);
        emit({ job: page, freed: true });
      }
    }
  }
  for (const [id, p] of waiting) {
    const { op } = p.request;
    if (
      op === "prepare" ||
      op === "analyze" ||
      op === "sim" ||
      op === "endgame"
    ) {
      send(id, p);
    } else if (op === "stop" || op === "free") {
      p.resolve(undefined);
    } else {
      p.reject(new Error("the job is gone"));
    }
  }
};

// The page's job a worker's job is part of.
const pageJob = (worker: number, job: number) => {
  for (const [page, held] of jobs) {
    if (held.some((h) => h.worker === worker && h.job === job)) return page;
  }
  return undefined;
};

// A candidate as it answers, and as it was given: a play with a bonus
// answers with the bonus in its score.
const answerKey = (c: { play: unknown; challenged_off?: boolean }) =>
  JSON.stringify([c.play, !!c.challenged_off]);
const givenKey = (play: unknown) => JSON.stringify(play);

const heard = (worker: number, e: WorkerEvent) => {
  // one worker: its jobs are the page's
  if (labParts() === 1) {
    emit(e);
    return;
  }
  const page = pageJob(worker, e.job);
  if (page === undefined) return;
  const sim = sims.get(page);
  if (!sim) {
    if (e.freed) jobs.delete(page);
    emit({ ...e, job: page });
    return;
  }
  const part = jobs.get(page)!.findIndex((h) => h.worker === worker);
  if (e.freed) {
    // one part gone, the sim with it
    if (!sim.freed) {
      sim.freed = true;
      freeParts(page, worker);
      emit({ job: page, freed: true });
    }
    return;
  }
  if (e.error) {
    sim.running = false;
    stopParts(page);
    emit({ job: page, error: e.error });
    return;
  }
  if (e.answer) sim.answers[part] = e.answer as PartAnswer;
  report(page, sim);
};

// The sim's merged answer, the candidates it pruned so shown, sent; and a
// look at who is behind, every PRUNE_EVERY iterations, while it prunes.
const report = (page: number, sim: Sim) => {
  if (sim.answers.some((a) => !a)) return;
  const merged = mergeAnswers(sim.answers as Array<PartAnswer>);
  if (!merged) return;
  const answer = {
    ...merged,
    candidates: merged.candidates.map((c) => {
      const given = sim.given.get(answerKey(c));
      return given !== undefined && sim.pruned.has(givenKey(given))
        ? { ...c, state: "pruned" }
        : c;
    }),
  };
  const states = answer.candidates.map((c) => c.state);
  const done =
    !states.includes("running") ||
    (sim.prune && states.filter((s) => s !== "pruned").length < 2);
  if (done && sim.running) {
    sim.running = false;
    stopParts(page);
  }
  emit({ job: page, answer, done });
  const look = Math.floor(answer.iterations / PRUNE_EVERY);
  if (sim.prune && sim.running && !sim.looking && look > sim.looked) {
    sim.looked = look;
    lookBehind(page, sim, answer);
  }
};

// The candidates behind the leader on the merged numbers, paused in every
// part and shown as pruned: each part read against the leader's play (no
// iterations run), the reads merged, and each compared with the leader.
const lookBehind = async (page: number, sim: Sim, answer: PartAnswer) => {
  const running = (i: number) => answer.candidates[i].state === "running";
  const lead = leaderOf(answer.candidates, running, PRUNE_DECIDED);
  if (lead < 0) return;
  const leadGiven = sim.given.get(answerKey(answer.candidates[lead]));
  if (leadGiven === undefined) return;
  sim.looking = true;
  try {
    const reads = await Promise.all(
      jobs.get(page)!.map((h) =>
        call<PartAnswer>(h.worker, {
          op: "peek",
          args: { job: h.job, against: JSON.stringify(leadGiven) },
        }),
      ),
    );
    const merged = mergeAnswers(reads);
    if (!merged || !sims.has(page)) return;
    const z = stitchedZ(merged.iterations, sim.alpha);
    const behind = behindLeader(
      merged.candidates,
      lead,
      (i) => merged.candidates[i].state === "running",
      z,
      PRUNE_DECIDED,
    ).flatMap((i) => {
      const given = sim.given.get(answerKey(merged.candidates[i]));
      return given === undefined ? [] : [given];
    });
    if (!behind.length) return;
    behind.forEach((g) => sim.pruned.add(givenKey(g)));
    const plays = JSON.stringify(behind);
    await Promise.all(
      jobs
        .get(page)!
        .map((h) =>
          call(h.worker, { op: "pause", args: { job: h.job, plays } }),
        ),
    );
    // the pruned shown at once, though the parts are not stepping
    jobs
      .get(page)!
      .forEach((h) => call(h.worker, { op: "read", args: { job: h.job } }));
  } catch (e) {
    emit({ job: page, error: e instanceof Error ? e.message : String(e) });
  } finally {
    sim.looking = false;
  }
};

const stopParts = (page: number) => {
  for (const h of jobs.get(page) ?? []) {
    call(h.worker, { op: "stop" }).catch(() => {});
  }
};

// The sim's parts freed but the one already gone.
const freeParts = (page: number, gone?: number) => {
  for (const h of jobs.get(page) ?? []) {
    if (h.worker !== gone) {
      call(h.worker, { op: "free", args: { job: h.job } }).catch(() => {});
    }
  }
  jobs.delete(page);
  sims.delete(page);
};

// A job's parts, each with the request given its own job's number there.
const partsOf = (job: number) => {
  const held = jobs.get(job);
  if (!held) throw new Error("no such job");
  return held;
};

const each = <T>(
  job: number,
  request: (job: number) => LabRequest,
): Promise<Array<T>> =>
  Promise.all(partsOf(job).map((h) => call<T>(h.worker, request(h.job))));

// The sims made before this job, freed: a worker holds one job at a time,
// so a job made on one worker frees the sim's part there, and the sim goes.
const freeSims = () => {
  for (const page of [...sims.keys()]) {
    freeParts(page);
    emit({ job: page, freed: true });
  }
};

const rand53 = () =>
  Math.floor(Math.random() * 2 ** 21) * 2 ** 32 +
  Math.floor(Math.random() * 2 ** 32);

const multi = async (request: LabRequest): Promise<unknown> => {
  const n = labParts();
  switch (request.op) {
    case "prepare":
      // each part holds its own tables
      await Promise.all(Array.from({ length: n }, (_, w) => call(w, request)));
      return undefined;
    case "analyze":
      return call(0, request);
    case "endgame": {
      freeSims();
      const job = await call<number>(0, request);
      const page = ++lastJob;
      jobs.set(page, [{ worker: 0, job }]);
      return page;
    }
    case "sim": {
      freeSims();
      const seed = rand53();
      const base = JSON.parse(request.args.request);
      const made = await Promise.all(
        Array.from({ length: n }, (_, w) =>
          call<{
            job: number;
            added: Array<{ play: unknown; challenged_off?: boolean }>;
            inferred?: unknown;
          }>(w, {
            op: "sim",
            args: {
              ...request.args,
              request: JSON.stringify({ ...base, seed, part: w, parts: n }),
            },
          }),
        ),
      );
      const page = ++lastJob;
      jobs.set(
        page,
        made.map((m, w) => ({ worker: w, job: m.job })),
      );
      const given = new Map<string, unknown>();
      const plays = JSON.parse(request.args.plays) as Array<unknown>;
      made[0].added.forEach((a, i) => given.set(answerKey(a), plays[i]));
      sims.set(page, {
        answers: new Array(n).fill(undefined),
        prune: true,
        alpha: 0,
        looked: 0,
        pruned: new Set(),
        given,
        looking: false,
        running: false,
        freed: false,
      });
      return { ...made[0], job: page };
    }
    case "add":
    case "remove":
    case "pause":
    case "resume": {
      const { job, plays } = request.args;
      const sim = sims.get(job);
      if (!sim) throw new Error("no such sim");
      const results = await each<
        Array<{ play: unknown; challenged_off?: boolean; was: string }>
      >(job, (j) => ({ ...request, args: { job: j, plays } }));
      const given = JSON.parse(plays) as Array<unknown>;
      results[0].forEach((r, i) => {
        if (request.op === "add") sim.given.set(answerKey(r), given[i]);
        if (request.op === "remove") sim.given.delete(answerKey(r));
        // a candidate the page pruned, resumed or gone, is no longer pruned
        if (request.op !== "add" && request.op !== "pause") {
          sim.pruned.delete(givenKey(given[i]));
        }
      });
      return results[0];
    }
    case "set": {
      const results = await each<unknown>(request.args.job, (j) => ({
        ...request,
        args: { ...request.args, job: j },
      }));
      return results[0];
    }
    case "against":
      await each(request.args.job, (j) => ({
        ...request,
        args: { ...request.args, job: j },
      }));
      return undefined;
    case "read":
      await each(request.args.job, (j) => ({
        op: "read",
        args: { job: j },
      }));
      return undefined;
    case "peek": {
      // each part's answer against the play given, merged
      const { against } = request.args;
      const reads = await each<PartAnswer>(request.args.job, (j) => ({
        op: "peek",
        args: { job: j, against },
      }));
      return mergeAnswers(reads);
    }
    case "run": {
      const { job } = request.args;
      const sim = sims.get(job);
      if (sim) {
        // the page prunes the parts, none of them on its own
        sim.prune = request.args.prune !== false;
        if (sim.prune && request.args.alpha === undefined) {
          throw new Error("a pruning run needs an alpha");
        }
        sim.alpha = request.args.alpha ?? 0;
        sim.running = true;
        await each(job, (j) => ({
          op: "run",
          args: { job: j, ms: request.args.ms, prune: false },
        }));
        return undefined;
      }
      await each(job, (j) => ({
        ...request,
        args: { ...request.args, job: j },
      }));
      return undefined;
    }
    case "stop":
      sims.forEach((sim) => {
        sim.running = false;
      });
      await Promise.all(
        workers.flatMap((worker, w) => (worker ? [call(w, request)] : [])),
      );
      return undefined;
    case "free": {
      const { job } = request.args;
      if (sims.has(job)) {
        freeParts(job);
        emit({ job, freed: true });
        return undefined;
      }
      await each(job, (j) => ({ op: "free", args: { job: j } }));
      jobs.delete(job);
      return undefined;
    }
  }
};

export const labCall = <T = undefined>(request: LabRequest): Promise<T> =>
  labParts() > 1
    ? (multi(request) as Promise<T>)
    : (call<T>(0, request) as Promise<T>);

export const onLabEvent = (listener: (e: LabEvent) => void) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};
