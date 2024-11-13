import { Unrace } from '../utils/unrace';

// Good enough for now. If need to reload, just refresh the whole page.
class Loadable {
  private whichStep = 0;

  private fetchPromise?: Promise<Response>;

  constructor(
    readonly cacheKey: string,
    readonly path: string
  ) {}

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

const loadablesByKey: { [key: string]: Array<Loadable> } = {};
{
  const filenames = [
    'CSW19.kad',
    'CSW19.klv2',
    'CSW19.kwg',
    'CSW19X.kad',
    'CSW19X.klv2',
    'CSW19X.kwg',
    'CSW21.kad',
    'CSW21.klv2',
    'CSW21.kwg',
    'DISC2.kad',
    'DISC2.klv2',
    'DISC2.kwg',
    'ECWL.kad',
    'ECWL.klv2',
    'ECWL.kwg',
    'FILE2017.kad',
    'FILE2017.klv2',
    'FILE2017.kwg',
    'FRA20.kad',
    'FRA20.klv2',
    'FRA20.kwg',
    'FRA24.kad',
    'FRA24.klv2',
    'FRA24.kwg',
    'NSF21.kad',
    'NSF21.klv2',
    'NSF21.kwg',
    'NSF22.kad',
    'NSF22.klv2',
    'NSF22.kwg',
    'NSF23.kad',
    'NSF23.klv2',
    'NSF23.kwg',
    'NSWL20.kad',
    'NSWL20.klv2',
    'NSWL20.kwg',
    'NWL18.kad',
    'NWL18.klv2',
    'NWL18.kwg',
    'NWL20.kad',
    'NWL20.klv2',
    'NWL20.kwg',
    'NWL23.kad',
    'NWL23.klv2',
    'NWL23.kwg',
    'OSPS49.kad',
    'OSPS49.klv2',
    'OSPS49.kwg',
    'RD28.kad',
    'RD28.klv2',
    'RD28.kwg',
    'super-CSW19.klv2',
    'super-CSW19X.klv2',
    'super-CSW21.klv2',
    'super-DISC2.klv2',
    'super-ECWL.klv2',
    'super-NSWL20.klv2',
    'super-NWL18.klv2',
    'super-NWL20.klv2',
    'super-NWL23.klv2',
  ];

  // convention-over-configuration.
  const lexicons: { [key: string]: true } = {};
  const loadables: { [key: string]: Loadable } = {};
  const unsupportedFilenames = [];
  for (const filename of filenames) {
    const m = filename.match(/^(super-)?(\w+)(\.klv2|\.kwg|\.kad)$/);
    if (m) {
      const lexicon = m[2];
      const baseFilename = m[1] ? `super-${lexicon}` : lexicon;
      const extension = m[3];
      const cacheKey =
        extension === '.kwg'
          ? `kwg/${baseFilename}`
          : extension === '.kad'
            ? `kwg/${baseFilename}.WordSmog`
            : extension === '.klv2'
              ? `klv/${baseFilename}`
              : null;
      if (cacheKey) {
        lexicons[lexicon] = true;
        loadables[filename] = new Loadable(cacheKey, `/wasm/2024/${filename}`);
        continue;
      }
    }
    unsupportedFilenames.push(filename);
  }

  for (const lexicon in lexicons) {
    loadablesByKey[`${lexicon}`] = [
      loadables[`${lexicon}.kwg`],
      loadables[`${lexicon}.klv2`],
    ];
    loadablesByKey[`${lexicon}.WordSmog`] = [
      loadables[`${lexicon}.kad`],
      loadables[`${lexicon}.klv2`],
    ];
    loadablesByKey[`super-${lexicon}`] = [
      loadables[`super-${lexicon}.kwg`] ?? loadables[`${lexicon}.kwg`],
      loadables[`super-${lexicon}.klv2`] ?? loadables[`${lexicon}.klv2`],
    ];
    loadablesByKey[`super-${lexicon}.WordSmog`] = [
      loadables[`super-${lexicon}.kad`] ?? loadables[`${lexicon}.kad`],
      loadables[`super-${lexicon}.klv2`] ?? loadables[`${lexicon}.klv2`],
    ];
  }
  const missingFiles = [];
  for (const k in loadablesByKey) {
    if (loadablesByKey[k].some((v) => !v)) {
      missingFiles.push(k);
    }
  }

  const errors = [];
  if (unsupportedFilenames.length > 0) {
    errors.push(
      `unsupported filenames: ${unsupportedFilenames.sort().join(', ')}`
    );
  }
  if (missingFiles.length > 0) {
    errors.push(`missing files: ${missingFiles.sort().join(', ')}`);
  }
  if (errors.length > 0) {
    throw new Error(errors.join('; '));
  }
}

const unrace = new Unrace();

const wolgesCache = new WeakMap();

export const getLexiconKey = (loadableKey: string) =>
  loadablesByKey[loadableKey]?.[0]?.cacheKey?.replace(/^kwg\//, '');

export const getLeaveKey = (loadableKey: string) =>
  loadablesByKey[loadableKey]?.[1]?.cacheKey?.replace(/^klv\//, '');

export const getWolges = async (loadableKey: string) =>
  unrace.run(async () => {
    // Allow these files to start loading.
    const wolgesPromise = import('wolges-wasm');
    const effectiveLoadables = loadablesByKey[loadableKey] ?? [];
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
