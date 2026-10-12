import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// A stand-in for the browser's autoplay rule: play() is refused unless sound
// is allowed outright, the call comes during a user gesture, or the element
// already played during one.
let allowSound = false;
let inGesture = false;
// While set, the silent unlock's play() stays pending until the test lets
// it go, so a test can ask for a sound in the middle of the unlock.
let holdSilence = false;
let heldSilence: Array<() => void> = [];
let unlockedElements = new WeakSet<HTMLMediaElement>();
let played: Array<{ src: string; el: HTMLMediaElement }> = [];
const listeners: Array<
  [
    string,
    EventListenerOrEventListenerObject,
    boolean | AddEventListenerOptions,
  ]
> = [];

// boop.ts keeps its state in module scope and adds window listeners when it
// loads, so each test imports a fresh copy (and removes its listeners after).
const loadBoop = async () => {
  vi.resetModules();
  return (await import("./boop")).BoopSounds;
};

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

const click = async () => {
  inGesture = true;
  window.dispatchEvent(new MouseEvent("click"));
  inGesture = false;
  await flush();
};

const plays = (name: string) => played.filter((p) => p.src.includes(name));

const end = async (el: HTMLMediaElement) => {
  el.dispatchEvent(new Event("ended"));
  await flush();
};

beforeEach(() => {
  allowSound = false;
  inGesture = false;
  holdSilence = false;
  heldSilence = [];
  unlockedElements = new WeakSet();
  played = [];
  localStorage.removeItem("enableSilentSite");
  Object.defineProperty(URL, "createObjectURL", {
    configurable: true,
    writable: true,
    value: () => "blob:silence",
  });
  const addEventListener = window.addEventListener;
  vi.spyOn(window, "addEventListener").mockImplementation(
    (type, listener, options) => {
      listeners.push([type, listener, options ?? false]);
      addEventListener.call(window, type, listener, options);
    },
  );
  vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(() => {});
  vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(() => {});
  vi.spyOn(HTMLMediaElement.prototype, "play").mockImplementation(function (
    this: HTMLMediaElement,
  ) {
    if (!allowSound && !inGesture && !unlockedElements.has(this)) {
      return Promise.reject(new DOMException("refused", "NotAllowedError"));
    }
    if (inGesture) unlockedElements.add(this);
    played.push({ src: this.src, el: this });
    if (this.src === "blob:silence" && holdSilence) {
      return new Promise<void>((resolve) => {
        heldSilence.push(resolve);
      });
    }
    return Promise.resolve();
  });
});

afterEach(() => {
  for (const [type, listener, options] of listeners.splice(0)) {
    window.removeEventListener(type, listener, options);
  }
  vi.restoreAllMocks();
});

describe("BoopSounds before the first user gesture", () => {
  it("plays at once where the browser allows sound before a gesture", async () => {
    allowSound = true;
    const boop = await loadBoop();
    boop.playSound("oppMoveSound");
    await flush();
    expect(plays("oppmove")).toHaveLength(1);
  });

  it("plays every refused request after the first click, one after another", async () => {
    const boop = await loadBoop();
    boop.playSound("makeMoveSound");
    boop.playSound("makeMoveSound");
    boop.playSound("makeMoveSound");
    await flush();
    expect(plays("makemove")).toHaveLength(0);

    await click();
    expect(plays("makemove")).toHaveLength(1);
    const el = plays("makemove")[0].el;
    await end(el);
    expect(plays("makemove")).toHaveLength(2);
    await end(el);
    expect(plays("makemove")).toHaveLength(3);
    await end(el);
    expect(plays("makemove")).toHaveLength(3);
    expect(plays("oppmove")).toHaveLength(0);
  });

  it("plays a request made during the unlock right after it", async () => {
    const boop = await loadBoop();
    holdSilence = true;
    await click();
    boop.playSound("eagleScreechSound");
    await flush();
    expect(plays("eagle-screech")).toHaveLength(0);

    for (const release of heldSilence.splice(0)) release();
    await flush();
    expect(plays("eagle-screech")).toHaveLength(1);
  });

  it("plays nothing when sounds are turned off", async () => {
    localStorage.setItem("enableSilentSite", "true");
    const boop = await loadBoop();
    boop.playSound("makeMoveSound");
    await flush();
    await click();
    expect(played).toHaveLength(0);
  });
});
