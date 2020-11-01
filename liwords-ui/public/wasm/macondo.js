// This script must be pure JS outside React/webpack.
// So, it is open-source.

(() => {
  const macondoCache = new WeakMap();

  const fetchAndPrecache = async (macondo, cacheKey, path) => {
    let cachedStuffs = macondoCache.get(macondo);
    if (!cachedStuffs) {
      macondoCache.set(macondo, (cachedStuffs = {}));
    }
    if (!cachedStuffs[cacheKey]) {
      const resp = await fetch(path);
      if (resp.ok) {
        await macondo.precache(
          cacheKey,
          new Uint8Array(await resp.arrayBuffer())
        );
        cachedStuffs[cacheKey] = true;
      } else {
        throw new Error(`Unable to cache ${cacheKey}`);
      }
    }
  };

  // Good enough for now. If need to reload, just refresh the whole page.
  const macondoPromise = new Promise((res, rej) => {
    self.resMacondo = res;
    self.rejMacondo = rej;
  });

  let macondoLoadAttempted = false;

  const getMacondo = async (macondoFilename) => {
    if (macondoLoadAttempted) return await macondoPromise;
    macondoLoadAttempted = true;
    try {
      importScripts('wasm_exec.js');
      const Go = self.Go;
      if (!Go) throw new Error('Go not loaded');
      const go = new Go();
      const resp = fetch(`/wasm/${macondoFilename}`);
      let resultPromise;
      if (WebAssembly.instantiateStreaming) {
        // Better browsers.
        resultPromise = WebAssembly.instantiateStreaming(resp, go.importObject);
      } else {
        // Apple browsers.
        resultPromise = WebAssembly.instantiate(
          await (await resp).arrayBuffer(),
          go.importObject
        );
      }
      const result = await resultPromise;
      const instance = go.run(result.instance);
      instance.finally(() => {
        // Good enough for now. Note that we can no longer call the returned functions.
        self.rejMacondo(new Error('Go did not resolve macondoPromise'));
      });
      return await macondoPromise;
    } catch (e) {
      self.rejMacondo(e);
      throw e;
    }
  };

  const doReq = async (req) => {
    if (req[0] === 'analyze') {
      return await (await getMacondo()).analyze(req[1]);
    } else if (req[0] === 'precache') {
      return await fetchAndPrecache(await getMacondo(), req[1], req[2]);
    } else {
      throw new Error('unknown request');
    }
  };

  onmessage = (msg) => {
    if (msg.data[0] === 'request') {
      // ["request", id, req]
      (async () => {
        try {
          const ret = await doReq(msg.data[2]);
          postMessage(['response', msg.data[1], true, ret]);
        } catch (e) {
          postMessage(['response', msg.data[1], false]);
        }
      })();
    } else if (msg.data[0] === 'getMacondo') {
      // ["getMacondo", filename]
      const macondoFilename = msg.data[1];
      (async () => {
        try {
          await getMacondo(macondoFilename);
          postMessage(['getMacondo', true]);
        } catch (e) {
          postMessage(['getMacondo', false]);
        }
      })();
    }
  };
})();
