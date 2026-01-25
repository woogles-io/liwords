import "@testing-library/jest-dom/vitest";
import ResizeObserver from "resize-observer-polyfill";

// https://github.com/jsdom/jsdom/issues/3368#issuecomment-1396749033

global.ResizeObserver = ResizeObserver;
