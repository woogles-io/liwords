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
