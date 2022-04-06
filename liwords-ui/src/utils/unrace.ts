// Remove race conditions by only allowing one thing to run.

export class Unrace {
  queue: Array<{ turn: Promise<void>; ready: () => void }> = [];

  run: <TArgs, TResult>(
    func: (...args: TArgs[]) => TResult,
    ...args: TArgs[]
  ) => Promise<TResult> = async (func, ...args) => {
    let ready: () => void;
    const turn = new Promise<void>((res) => {
      ready = res;
    });
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    this.queue.push({ turn, ready: ready! });
    if (this.queue.length === 1) {
      // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
      ready!();
    } else {
      await turn;
    }
    try {
      return await func(...args);
    } finally {
      this.queue.shift(); // O(n^2) ok, assume small n.
      if (this.queue.length > 0) {
        this.queue[0].ready();
      }
    }
  };
}
