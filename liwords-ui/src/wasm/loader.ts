import { publish } from '../shared/pubsub';
import { Unrace } from '../utils/unrace';
import MAGPIE from 'magpie-wasm';

export enum MagpieMoveTypes {
  Play = 0,
  Exchange = 1,
  Pass = 2,
}

let pleaseLoadMagpie: (value: unknown) => void;
const canLoadMagpie = new Promise((res, rej) => {
  pleaseLoadMagpie = res;
});

const magpiePromise = (async () => {
  await canLoadMagpie;
  const magpie = await MAGPIE({
    print: (s: string) => {
      publish('magpie.stdout', s);
    },
  });
  console.log('I have awaited and loaded MAGPIE');
  const precacheWrapper = magpie.cwrap('precache_file_data', null, [
    'number',
    'number',
    'number',
  ]);
  const processUCGIWrapper = magpie.cwrap('process_ucgi_command', null, [
    'number',
  ]);
  /**
   * char *score_play(char *cgpstr, int move_type, int row, int col, int vertical,
      uint8_t *tiles, uint8_t *leave, int ntiles, int nleave) 
   */
  const scorePlayWrapper = magpie.cwrap('score_play', 'number', [
    'number',
    'number',
    'number',
    'number',
    'number',
    'array',
    'array',
    'number',
    'number',
  ]);

  magpie.precacheFileData = (filename: string, rawData: Uint8Array) => {
    const buf = magpie._malloc(rawData.length * rawData.BYTES_PER_ELEMENT);
    magpie.HEAPU8.set(rawData, buf);
    const filenameCharArr = magpie.stringToNewUTF8(filename);
    precacheWrapper(filenameCharArr, buf, rawData.length);
    magpie._free(buf);
    magpie._free(filenameCharArr);
  };

  magpie.processUCGICommand = (cmd: string) => {
    const cmdC = magpie.stringToNewUTF8(cmd);
    processUCGIWrapper(cmdC);
    magpie._free(cmdC);
  };

  magpie.scorePlay = (
    cgpstr: string,
    moveType: MagpieMoveTypes,
    row: number,
    col: number,
    vertical: boolean,
    tiles: Uint8Array,
    leave: Uint8Array
  ) => {
    const cgpC = magpie.stringToNewUTF8(cgpstr);
    const ret = scorePlayWrapper(
      cgpC,
      moveType,
      row,
      col,
      vertical ? 1 : 0,
      tiles,
      leave,
      tiles.length,
      leave.length
    );
    const jsStr = magpie.UTF8ToString(ret);
    // Free the ret pointer, which is malloc'ed inside the C score play func
    magpie._free(ret);
    magpie._free(cgpC);

    return jsStr;
  };

  return magpie;
})();

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

  getSingleUseArrayBufferWithAlloc = async () => {};
}

const loadablesByLexicon: { [key: string]: Array<Loadable> } = {};
const magpieOnlyLoadablesByLexicon: { [key: string]: Array<Loadable> } = {};

for (const { lexicons, cacheKey, path } of [
  {
    lexicons: ['CSW19', 'CSW19X', 'NWL20', 'NWL18', 'NSWL20', 'ECWL'].flatMap(
      (name) => [name, `${name}.WordSmog`]
    ),
    cacheKey: 'klv/english',
    path: '/wasm/english.klv2',
  },
  {
    lexicons: ['CSW21'].flatMap((name) => [name, `${name}.WordSmog`]),
    cacheKey: 'klv/CSW21',
    path: '/wasm/CSW21.klv2',
  },
  {
    lexicons: ['FRA20'].flatMap((name) => [name, `${name}.WordSmog`]),
    cacheKey: 'klv/french',
    path: '/wasm/french.klv2',
  },
  {
    lexicons: ['RD28'].flatMap((name) => [name, `${name}.WordSmog`]),
    cacheKey: 'klv/german',
    path: '/wasm/german.klv2',
  },
  {
    lexicons: ['NSF21', 'NSF22'].flatMap((name) => [name, `${name}.WordSmog`]),
    cacheKey: 'klv/norwegian',
    path: '/wasm/norwegian.klv2',
  },
  {
    lexicons: ['DISC2'].flatMap((name) => [name, `${name}.WordSmog`]),
    cacheKey: 'klv/catalan',
    path: '/wasm/catalan.klv2',
  },
  ...[
    'CSW19',
    'CSW19X',
    'CSW21',
    'NWL20',
    'NWL18',
    'NSWL20',
    'ECWL',
    'FRA20',
    'RD28',
    'NSF21',
    'NSF22',
    'DISC2',
  ].map((name) => ({
    lexicons: [name],
    cacheKey: `kwg/${name}`,
    path: `/wasm/${name}.kwg`,
  })),
  ...[
    'CSW19',
    'CSW19X',
    'CSW21',
    'NWL20',
    'NWL18',
    'NSWL20',
    'ECWL',
    'FRA20',
    'RD28',
    'NSF21',
    'NSF22',
    'DISC2',
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

for (const { lexicons, cacheKey, path } of [
  {
    lexicons: [
      'CSW19',
      'CSW19X',
      'CSW21',
      'NWL20',
      'NWL18',
      'NSWL20',
      'ECWL',
    ].flatMap((name) => [name]),
    cacheKey: 'ld/english',
    path: '/wasm/english.csv',
  },
  {
    lexicons: ['FRA20'].flatMap((name) => [name]),
    cacheKey: 'ld/french',
    path: '/wasm/french.csv',
  },
  {
    lexicons: ['RD28'].flatMap((name) => [name]),
    cacheKey: 'ld/german',
    path: '/wasm/german.csv',
  },
  {
    lexicons: ['NSF21', 'NSF22'].flatMap((name) => [name]),
    cacheKey: 'ld/norwegian',
    path: '/wasm/norwegian.csv',
  },
  {
    lexicons: ['DISC2'].flatMap((name) => [name]),
    cacheKey: 'ld/catalan',
    path: '/wasm/catalan.csv',
  },
]) {
  const loadable = new Loadable(cacheKey, path);
  for (const lexicon of lexicons) {
    if (!(lexicon in magpieOnlyLoadablesByLexicon)) {
      magpieOnlyLoadablesByLexicon[lexicon] = [];
    }
    magpieOnlyLoadablesByLexicon[lexicon].push(loadable);
  }
}

// winpct has only been calculated for english letter distribution.
// Just use for all lexica for now.
const loadable = new Loadable(
  'wpct/default_english',
  '/wasm/english-winpct.csv'
);
for (const lexicon in magpieOnlyLoadablesByLexicon) {
  magpieOnlyLoadablesByLexicon[lexicon].push(loadable);
}

const unrace = new Unrace();

const wolgesCache = new WeakMap();

const magpieCache = new WeakMap();

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

export const getMagpie = async (lexicon: string) =>
  unrace.run(async () => {
    // Allow these files to start loading.
    const effectiveLoadables = [
      ...(loadablesByLexicon[lexicon] ?? []),
      ...(magpieOnlyLoadablesByLexicon[lexicon] ?? []),
    ];
    for (const loadable of effectiveLoadables) {
      loadable.startFetch();
    }

    pleaseLoadMagpie(undefined);
    const magpie = await magpiePromise;
    let cachedStuffs = magpieCache.get(magpie);
    if (!cachedStuffs) {
      magpieCache.set(magpie, (cachedStuffs = {}));
    }

    await Promise.all(
      effectiveLoadables.map(async (loadable) => {
        const cacheKey = loadable.cacheKey;
        if (!cachedStuffs[cacheKey]) {
          const splitAt = cacheKey.indexOf('/');
          if (splitAt < 0) throw new Error(`invalid cache key ${cacheKey}`);

          const type = cacheKey.substring(0, splitAt);
          const name = cacheKey.substring(splitAt + 1);

          let magpieCacheKey;
          // magpie internal filenames follow a specific structure. For ease,
          // we will precache using a cache key with the same file structure.
          switch (type) {
            case 'kwg':
              magpieCacheKey = `data/lexica/${name}.kwg`;
              break;
            case 'klv':
              magpieCacheKey = `data/lexica/${name}.klv2`;
              break;
            case 'ld':
              magpieCacheKey = `data/letterdistributions/${name}.csv`;
              break;
            case 'wpct':
              magpieCacheKey = `data/strategy/${name}/winpct.csv`;
              break;
          }

          await magpie.precacheFileData(
            magpieCacheKey,
            new Uint8Array(await loadable.getSingleUseArrayBuffer())
          );
          cachedStuffs[cacheKey] = true;
        }
      })
    );
    return magpie;
  });
