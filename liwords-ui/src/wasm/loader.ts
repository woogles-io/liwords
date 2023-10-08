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
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
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
      // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
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
    lexicons: ['CSW19', 'CSW19X', 'NWL20', 'NWL18', 'NSWL20'].flatMap(
      (name) => [name, `${name}.WordSmog`]
    ),
    cacheKey: 'klv/english',
    path: '/wasm/2023/english.klv2',
  },
  {
    lexicons: ['CSW19', 'CSW19X', 'NWL20', 'NWL18', 'NSWL20'].flatMap(
      (name) => [`${name}.Super`, `${name}.WordSmog.Super`]
    ),
    cacheKey: 'klv/super-english',
    path: '/wasm/2023/super-english.klv2',
  },
  {
    lexicons: ['ECWL'].flatMap((name) => [name, `${name}.WordSmog`]),
    cacheKey: 'klv/CEL',
    path: '/wasm/2023/CEL.klv2',
  },
  {
    lexicons: ['ECWL'].flatMap((name) => [
      `${name}.Super`,
      `${name}.WordSmog.Super`,
    ]),
    cacheKey: 'klv/super-CEL',
    path: '/wasm/2023/super-CEL.klv2',
  },
  {
    lexicons: ['CSW21'].flatMap((name) => [name, `${name}.WordSmog`]),
    cacheKey: 'klv/CSW21',
    path: '/wasm/2023/CSW21.klv2',
  },
  {
    lexicons: ['CSW21'].flatMap((name) => [
      `${name}.Super`,
      `${name}.WordSmog.Super`,
    ]),
    cacheKey: 'klv/super-CSW21',
    path: '/wasm/2023/super-CSW21.klv2',
  },
  {
    lexicons: ['FRA20'].flatMap((name) => [
      name,
      `${name}.WordSmog`,
      `${name}.Super`,
      `${name}.WordSmog.Super`,
    ]),
    cacheKey: 'klv/french',
    path: '/wasm/2023/french.klv2',
  },
  {
    lexicons: ['RD28'].flatMap((name) => [
      name,
      `${name}.WordSmog`,
      `${name}.Super`,
      `${name}.WordSmog.Super`,
    ]),
    cacheKey: 'klv/german',
    path: '/wasm/2023/german.klv2',
  },
  {
    lexicons: ['NSF21', 'NSF22', 'NSF23'].flatMap((name) => [
      name,
      `${name}.WordSmog`,
      `${name}.Super`,
      `${name}.WordSmog.Super`,
    ]),
    cacheKey: 'klv/norwegian',
    path: '/wasm/2023/norwegian.klv2',
  },
  {
    lexicons: ['DISC2'].flatMap((name) => [name, `${name}.WordSmog`]),
    cacheKey: 'klv/catalan',
    path: '/wasm/2023/catalan.klv2',
  },
  {
    lexicons: ['DISC2'].flatMap((name) => [
      `${name}.Super`,
      `${name}.WordSmog.Super`,
    ]),
    cacheKey: 'klv/super-catalan',
    path: '/wasm/2023/super-catalan.klv2',
  },
  ...[
    'CSW19',
    'CSW19X',
    'NWL18',
    'NSWL20',
    'ECWL',
    'FRA20',
    'NSF21',
    'NSF22',
    'NSF23',
    'DISC2',
  ].map((name) => ({
    lexicons: [name, `${name}.Super`],
    cacheKey: `kwg/${name}`,
    path: `/wasm/${name}.kwg`,
  })),
  ...['CSW21', 'NWL20', 'RD28'].map((name) => ({
    lexicons: [name, `${name}.Super`],
    cacheKey: `kwg/${name}`,
    path: `/wasm/2023/${name}.kwg`,
  })),
  ...[
    'CSW19',
    'CSW19X',
    'NWL18',
    'NSWL20',
    'ECWL',
    'FRA20',
    'NSF21',
    'NSF22',
    'NSF23',
    'DISC2',
  ].map((name) => ({
    lexicons: [`${name}.WordSmog`, `${name}.WordSmog.Super`],
    cacheKey: `kwg/${name}.WordSmog`,
    path: `/wasm/${name}.kad`,
  })),
  ...['CSW21', 'NWL20', 'RD28'].map((name) => ({
    lexicons: [`${name}.WordSmog`, `${name}.WordSmog.Super`],
    cacheKey: `kwg/${name}.WordSmog`,
    path: `/wasm/2023/${name}.kad`,
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
