import "@testing-library/jest-dom/vitest";
import ResizeObserver from "resize-observer-polyfill";

// https://github.com/jsdom/jsdom/issues/3368#issuecomment-1396749033

global.ResizeObserver = ResizeObserver;

// antd's responsive Grid/Table reads window.matchMedia, which jsdom does not
// implement. Stub it (no query ever matches) so those components can render.
if (!window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList;
}

// Node >= 26 exposes a global `localStorage` that is `undefined` unless the
// process was started with --localstorage-file. That global shadows the one
// jsdom installs, so every module doing `localStorage.getItem(...)` at import
// time throws. Put jsdom's back when that happens.
if (typeof globalThis.localStorage === "undefined") {
  const store = new Map<string, string>();
  const shim: Storage = {
    getItem: (k) => (store.has(k) ? store.get(k)! : null),
    setItem: (k, v) => void store.set(k, String(v)),
    removeItem: (k) => void store.delete(k),
    clear: () => store.clear(),
    key: (i) => Array.from(store.keys())[i] ?? null,
    get length() {
      return store.size;
    },
  };
  Object.defineProperty(globalThis, "localStorage", {
    value: shim,
    configurable: true,
    writable: true,
  });
}
