import { readFileSync } from "fs";
import { resolve } from "path";
import { drawWhenDogLeftAlone } from "./boot_dog";

// The dog as index.html has it, its own handlers and all: they mark it while
// it is petted, and drawWhenDogLeftAlone reads those marks.
const page = readFileSync(resolve(__dirname, "../../index.html"), "utf8");
const splash = page.match(/<div class="boot-splash">[\s\S]*?<\/div>/)![0];

const putDog = () => {
  document.body.innerHTML = `<div id="root">${splash}</div>`;
  return document.querySelector<HTMLElement>(".boot-splash img")!;
};

const fire = (dog: HTMLElement, type: string) =>
  dog.dispatchEvent(new Event(type));

// The dog's handlers run in jsdom's own window, whose performance clock the
// fake timers do not reach: it is handed theirs, so its stamps keep the same
// time as the app's reads.
const dogWindow = (globalThis as unknown as { jsdom: { window: object } }).jsdom
  .window;
const ownClock = Object.getOwnPropertyDescriptor(dogWindow, "performance")!;

beforeEach(() => {
  vi.useFakeTimers();
  const faked = performance;
  Object.defineProperty(dogWindow, "performance", {
    configurable: true,
    get: () => faked,
  });
});

afterEach(() => {
  Object.defineProperty(dogWindow, "performance", ownClock);
  vi.useRealTimers();
  document.body.innerHTML = "";
});

it("draws at once with no dog", () => {
  const draw = vi.fn();
  drawWhenDogLeftAlone(draw);
  expect(draw).toHaveBeenCalledTimes(1);
});

it("draws at once when the dog was never petted", () => {
  putDog();
  const draw = vi.fn();
  drawWhenDogLeftAlone(draw);
  expect(draw).toHaveBeenCalledTimes(1);
});

it("draws at once when a pointer only passed over the dog", () => {
  const dog = putDog();
  fire(dog, "pointerleave");
  expect(dog.dataset.petAt).toBeUndefined();
  const draw = vi.fn();
  drawWhenDogLeftAlone(draw);
  expect(draw).toHaveBeenCalledTimes(1);
});

it("waits while the dog is held, and a second after it is let go", () => {
  const dog = putDog();
  fire(dog, "pointerdown");
  expect(dog.classList.contains("petted")).toBe(true);
  const draw = vi.fn();
  drawWhenDogLeftAlone(draw);
  vi.advanceTimersByTime(5000);
  expect(draw).not.toHaveBeenCalled();
  fire(dog, "pointerup");
  vi.advanceTimersByTime(999);
  expect(draw).not.toHaveBeenCalled();
  vi.advanceTimersByTime(1);
  expect(draw).toHaveBeenCalledTimes(1);
});

it("counts a press slid off the dog, or cancelled, as let go", () => {
  for (const type of ["pointerleave", "pointercancel"]) {
    const dog = putDog();
    fire(dog, "pointerdown");
    const draw = vi.fn();
    drawWhenDogLeftAlone(draw);
    vi.advanceTimersByTime(2000);
    expect(draw).not.toHaveBeenCalled();
    fire(dog, type);
    vi.advanceTimersByTime(999);
    expect(draw).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(draw).toHaveBeenCalledTimes(1);
  }
});

it("counts the second from the last pet, before the app started", () => {
  const dog = putDog();
  fire(dog, "pointerdown");
  fire(dog, "pointerup");
  vi.advanceTimersByTime(400);
  const draw = vi.fn();
  drawWhenDogLeftAlone(draw);
  vi.advanceTimersByTime(599);
  expect(draw).not.toHaveBeenCalled();
  vi.advanceTimersByTime(1);
  expect(draw).toHaveBeenCalledTimes(1);
});

it("waits again for a pet during the wait", () => {
  const dog = putDog();
  fire(dog, "pointerdown");
  fire(dog, "pointerup");
  const draw = vi.fn();
  drawWhenDogLeftAlone(draw);
  vi.advanceTimersByTime(500);
  fire(dog, "pointerdown");
  vi.advanceTimersByTime(3000);
  expect(draw).not.toHaveBeenCalled();
  fire(dog, "pointerup");
  vi.advanceTimersByTime(999);
  expect(draw).not.toHaveBeenCalled();
  vi.advanceTimersByTime(1);
  expect(draw).toHaveBeenCalledTimes(1);
});

it("counts the page losing focus as letting go", () => {
  const dog = putDog();
  fire(dog, "pointerdown");
  const draw = vi.fn();
  drawWhenDogLeftAlone(draw);
  vi.advanceTimersByTime(2000);
  window.dispatchEvent(new Event("blur"));
  expect(dog.classList.contains("petted")).toBe(false);
  vi.advanceTimersByTime(1000);
  expect(draw).toHaveBeenCalledTimes(1);
});

it("counts the page going hidden as letting go", () => {
  const dog = putDog();
  fire(dog, "pointerdown");
  const draw = vi.fn();
  drawWhenDogLeftAlone(draw);
  const shown = Object.getOwnPropertyDescriptor(document, "visibilityState");
  Object.defineProperty(document, "visibilityState", {
    configurable: true,
    get: () => "hidden",
  });
  try {
    document.dispatchEvent(new Event("visibilitychange"));
  } finally {
    if (shown) Object.defineProperty(document, "visibilityState", shown);
    else delete (document as { visibilityState?: string }).visibilityState;
  }
  vi.advanceTimersByTime(1000);
  expect(draw).toHaveBeenCalledTimes(1);
});

it("waits a second however the system clock is set", () => {
  for (const shift of [-60 * 60 * 1000, 60 * 60 * 1000]) {
    const dog = putDog();
    fire(dog, "pointerdown");
    fire(dog, "pointerup");
    vi.setSystemTime(Date.now() + shift);
    const draw = vi.fn();
    drawWhenDogLeftAlone(draw);
    vi.advanceTimersByTime(999);
    expect(draw).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(draw).toHaveBeenCalledTimes(1);
  }
});

it("draws once, whatever comes after", () => {
  const dog = putDog();
  fire(dog, "pointerdown");
  fire(dog, "pointerup");
  const draw = vi.fn();
  drawWhenDogLeftAlone(draw);
  vi.advanceTimersByTime(1000);
  expect(draw).toHaveBeenCalledTimes(1);
  fire(dog, "pointerdown");
  fire(dog, "pointerup");
  window.dispatchEvent(new Event("blur"));
  vi.advanceTimersByTime(5000);
  expect(draw).toHaveBeenCalledTimes(1);
});
