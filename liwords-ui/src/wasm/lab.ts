import type { LabRequest } from "./lab.worker";

// The page's side of the analyzer lab's worker (lab.worker.ts), made on first
// use and kept for the tab. labCall sends a request and resolves with its
// answer; onLabEvent hears each step of the job that is running, and when a job
// is freed.

export type LabEvent = {
  job: number;
  answer?: unknown;
  done?: boolean;
  error?: string;
  freed?: boolean;
};

type Pending = {
  resolve: (result: unknown) => void;
  reject: (e: Error) => void;
};

let worker: Worker | undefined;
let lastId = 0;
const pending = new Map<number, Pending>();
const listeners = new Set<(e: LabEvent) => void>();

const getWorker = () => {
  if (!worker) {
    worker = new Worker(new URL("./lab.worker.ts", import.meta.url));
    worker.onmessage = (e: MessageEvent) => {
      const { id, result, error, event } = e.data;
      if (event) {
        listeners.forEach((listener) => listener(event));
        return;
      }
      const p = pending.get(id);
      if (!p) return;
      pending.delete(id);
      if (error !== undefined) p.reject(new Error(error));
      else p.resolve(result);
    };
  }
  return worker;
};

export const labCall = <T = undefined>(request: LabRequest): Promise<T> =>
  new Promise<T>((resolve, reject) => {
    const id = ++lastId;
    pending.set(id, { resolve: resolve as (result: unknown) => void, reject });
    getWorker().postMessage({ id, ...request });
  });

export const onLabEvent = (listener: (e: LabEvent) => void) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};
