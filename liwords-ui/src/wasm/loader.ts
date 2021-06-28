import { Unrace } from '../utils/unrace';

// Good enough for now. If need to reload, just refresh the whole page.
class Loadable {
  private whichStep = 0;

  private fetchPromise?: Promise<Response>;

  constructor(readonly cacheKey: string, readonly path: string) {}

  startFetch = async () => {
    if (this.whichStep > 0) return;
    this.whichStep = 1;
    this.fetchPromise = fetch(this.path); // Do not await.
  };

  getArrayBuffer = async () => {
    if (this.whichStep > 1) return;
    this.startFetch(); // In case this is not done yet.
    this.whichStep = 2;
    const resp = await this.fetchPromise!;
    if (resp.ok) {
      return await resp.arrayBuffer();
    }
    throw new Error(`Unable to cache ${this.cacheKey}`);
  };

  disownArrayBuffer = async () => {
    if (this.whichStep > 2) return;
    this.whichStep = 3; // Single-use. Assume only one Wolges worker.
    this.fetchPromise = undefined;
  };

  reset = async () => {
    this.whichStep = 0; // Allow reloading (useful when previous fetch failed).
    this.fetchPromise = undefined;
  };

  getSingleUseArrayBuffer = async () => {
    try {
      const arrayBuffer = await this.getArrayBuffer();
      await this.disownArrayBuffer();
      return arrayBuffer!;
    } catch (e) {
      console.error(`failed to load ${this.cacheKey}`, e);
      await this.reset();
      throw e;
    }
  };
}

const loadablesByLexicon: { [key: string]: Array<Loadable> } = {};

for (const { lexicons, cacheKey, path } of [
  {
    lexicons: [
      'CSW19',
      'CSW19X',
      'NWL20',
      'NWL18',
      'NSWL20',
      'ECWL',
    ].flatMap((name) => [name, `${name}.WordSmog`]),
    cacheKey: 'klv/english',
    path: '/wasm/leaves.klv',
  },
  ...[
    'CSW19',
    'CSW19X',
    'NWL20',
    'NWL18',
    'NSWL20',
    'ECWL',
    'RD28',
    'NSF21',
  ].map((name) => ({
    lexicons: [name],
    cacheKey: `kwg/${name}`,
    path: `/wasm/${name}.kwg`,
  })),
  ...[
    'CSW19',
    'CSW19X',
    'NWL20',
    'NWL18',
    'NSWL20',
    'ECWL',
    'RD28',
    'NSF21',
  ].map((name) => ({
    lexicons: [`${name}.WordSmog`],
    cacheKey: `kwg/${name}.WordSmog`,
    path: `/wasm/${name}.kad`,
  })),
]) {
  const loadable = new Loadable(cacheKey, path);
  for (const lexicon of lexicons) {
    if (!(lexicon in loadablesByLexicon)) {
      loadablesByLexicon[lexicon] = [];
    }
    loadablesByLexicon[lexicon].push(loadable);
  }
}

const unrace = new Unrace();

const wolgesCache = new WeakMap();

export const getWolges = async (lexicon: string) =>
  unrace.run(async () => {
    // Allow these files to start loading.
    const wolgesPromise = import('wolges-wasm');
    const effectiveLoadables = loadablesByLexicon[lexicon] ?? [];
    for (const loadable of effectiveLoadables) {
      loadable.startFetch();
    }

    const wolges = await wolgesPromise;
    let cachedStuffs = wolgesCache.get(wolges);
    if (!cachedStuffs) {
      wolgesCache.set(wolges, (cachedStuffs = {}));
    }

    await Promise.all(
      effectiveLoadables.map(async (loadable) => {
        const cacheKey = loadable.cacheKey;
        if (!cachedStuffs[cacheKey]) {
          const splitAt = cacheKey.indexOf('/');
          if (splitAt < 0) throw new Error(`invalid cache key ${cacheKey}`);
          const type = cacheKey.substring(0, splitAt);
          const name = cacheKey.substring(splitAt + 1);
          if (type === 'klv') {
            await wolges.precache_klv(
              name,
              new Uint8Array(await loadable.getSingleUseArrayBuffer())
            );
          } else if (type === 'kwg') {
            await wolges.precache_kwg(
              name,
              new Uint8Array(await loadable.getSingleUseArrayBuffer())
            );
          } else {
            throw new Error(`invalid cache key ${cacheKey}`);
          }
          cachedStuffs[cacheKey] = true;
        }
      })
    );
    return wolges;
  });
