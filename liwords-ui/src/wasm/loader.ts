export interface Macondo {
  precache: (cacheKey: string, rawBytes: Uint8Array) => void;
  analyze: (jsonBoard: string) => string;
}

const macondoCache = new WeakMap();

export const fetchAndPrecache = async (
  macondo: Macondo,
  cacheKey: string,
  path: string
) => {
  let cachedStuffs = macondoCache.get(macondo);
  if (!cachedStuffs) {
    macondoCache.set(macondo, (cachedStuffs = {}));
  }
  if (!cachedStuffs[cacheKey]) {
    const resp = await fetch(path);
    if (resp.ok) {
      macondo.precache(cacheKey, new Uint8Array(await resp.arrayBuffer()));
      cachedStuffs[cacheKey] = true;
    } else {
      throw new Error(`Unable to cache ${cacheKey}`);
    }
  }
};

// Good enough for now. If need to reload, just refresh the whole page.
const macondoPromise = new Promise<Macondo>((res, rej) => {
  const w = window as any;
  w.resMacondo = res;
  w.rejMacondo = rej;
});

let macondoLoadAttempted = false;

export const getMacondo = async () => {
  if (macondoLoadAttempted) return await macondoPromise;
  macondoLoadAttempted = true;
  try {
    const Go = (window as any).Go;
    // Check if wasm_exec.js is loaded in public/index.html
    if (!Go) throw new Error('Go not loaded');
    const go = new Go();
    const macondoFilename = window.RUNTIME_CONFIGURATION.macondoFilename;
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
      (window as any).rejMacondo(
        new Error('Go did not resolve macondoPromise')
      );
    });
    return await macondoPromise;
  } catch (e) {
    (window as any).rejMacondo(e);
    throw e;
  }
};
