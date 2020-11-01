export interface Macondo {
  precache: (cacheKey: string, path: string) => Promise<unknown>;
  analyze: (jsonBoard: string) => Promise<string>;
}

let wrappedWorker: Macondo;

export const getMacondo = async () => {
  if (!wrappedWorker) {
    let pendings: {
      [key: string]: {
        promise: Promise<unknown>;
        res: (a: any) => void;
        rej: (a: any) => void;
      };
    } = {};

    const newPendingId = () => {
      while (true) {
        const d = String(performance.now());
        if (d in pendings) continue;

        let promRes: (a: any) => void;
        let promRej: (a: any) => void;
        const prom = new Promise((res, rej) => {
          promRes = res;
          promRej = rej;
        });

        pendings[d] = {
          promise: prom,
          res: promRes!,
          rej: promRej!,
        };

        return d;
      }
    };

    // First-time load.
    const worker = new Worker('/wasm/macondo.js');

    worker.postMessage([
      'getMacondo',
      window.RUNTIME_CONFIGURATION.macondoFilename,
    ]);

    await new Promise((res, rej) => {
      worker.onmessage = (msg) => {
        if (msg.data[0] === 'response') {
          // ["response", id, true, resp]
          // ["response", id, false] (error)
          const pending = pendings[msg.data[1]];
          if (pending) {
            if (msg.data[2]) {
              pending.res!(msg.data[3]);
            } else {
              pending.rej!(undefined);
            }
          }
        } else if (msg.data[0] === 'getMacondo') {
          // ["getMacondo", true] (ok)
          // ["getMacondo", false] (error)
          msg.data[1] ? res() : rej();
        }
      };
    });

    const sendRequest = async (req: any) => {
      const id = newPendingId();
      worker.postMessage(['request', id, req]);
      try {
        return await pendings[id].promise;
      } finally {
        delete pendings[id];
      }
    };

    wrappedWorker = {
      precache: async (cacheKey: string, path: string) => {
        return await sendRequest(['precache', cacheKey, path]);
      },
      analyze: async (jsonBoard: string) => {
        return (await sendRequest(['analyze', jsonBoard])) as string;
      },
    };
  }

  return wrappedWorker;
};
