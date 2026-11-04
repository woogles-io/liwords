import { afterEach, describe, expect, it, vi } from "vitest";
import { refitOnPageChanges, roomClass, roomInWindow } from "./window_room";

// each of the page's affixes as given: pinned, kept in place, or not pinned
type Affixes = Partial<
  Record<"rack-affix" | "examiner-controls", "fixed" | "static" | "unpinned">
>;

// the window as tall as given; a card's body at the top given, 190 px tall,
// with 1 px of the card's border below it; and, where affixes are given, the
// page's sticky player cards (0 to 166) and those affixes: the rack (126 to
// 224) and the examiner's controls (210 to 288)
const layOut = (innerHeight: number, top: number, affixes?: Affixes) => {
  vi.stubGlobal("innerHeight", innerHeight);
  const card = document.createElement("div");
  const body = document.createElement("div");
  card.append(body);
  if (affixes) {
    const sticky = document.createElement("div");
    sticky.className = "sticky-player-card-container";
    document.body.append(sticky);
    for (const [name, state] of Object.entries(affixes)) {
      const affix = document.createElement("div");
      affix.className = name;
      const inner = document.createElement("div");
      if (state !== "unpinned") {
        inner.className = "ant-affix";
        inner.style.position = state;
      }
      affix.append(inner);
      document.body.append(affix);
    }
  }
  document.body.append(card);
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(
    function (this: HTMLElement) {
      if (this.classList.contains("sticky-player-card-container")) {
        return { top: 0, bottom: 166, height: 166 } as DOMRect;
      }
      if (this.parentElement?.classList.contains("rack-affix")) {
        return { top: 126, bottom: 224, height: 98 } as DOMRect;
      }
      if (this.parentElement?.classList.contains("examiner-controls")) {
        return { top: 210, bottom: 288, height: 78 } as DOMRect;
      }
      const bottom = this === body ? top + 190 : top + 191;
      return { top, bottom, height: bottom - top } as DOMRect;
    },
  );
  vi.spyOn(HTMLElement.prototype, "getClientRects").mockImplementation(
    () => [{}] as unknown as DOMRectList,
  );
  return { body, card };
};

// the page's mutation records, delivered
const settle = () => new Promise((resolve) => setTimeout(resolve));

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

describe("roomInWindow", () => {
  it("has the room down to a margin above the window's bottom from where the body starts", () => {
    const { body, card } = layOut(760, 499);
    expect(roomInWindow(body, card)).toEqual({
      own: 190,
      phone: false,
      room: 236,
    });
  });

  it("measures the body's own height without the classes that size it, then puts them back", () => {
    vi.stubGlobal("innerHeight", 760);
    const card = document.createElement("div");
    const body = document.createElement("div");
    card.append(body);
    document.body.append(card);
    const sized = roomClass("sized-room");
    body.classList.add(sized, "kept");
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(
      function (this: HTMLElement) {
        const height = body.classList.contains(sized) ? 530 : 190;
        const bottom = this === body ? 499 + height : 499 + height + 1;
        return { top: 499, bottom, height: bottom - 499 } as DOMRect;
      },
    );
    expect(roomInWindow(body, card)).toEqual({
      own: 190,
      phone: false,
      room: 236,
    });
    expect(body.classList.contains(sized)).toBe(true);
    expect(body.classList.contains("kept")).toBe(true);
  });

  it("measures with the page at its top, however far it is scrolled", () => {
    vi.stubGlobal("scrollY", 100);
    const { body, card } = layOut(760, 399);
    expect(roomInWindow(body, card).room).toBe(236);
  });

  it("has none, on a page laid out for a wider window, where the body starts below the window", () => {
    const { body, card } = layOut(844, 1767);
    expect(roomInWindow(body, card)).toEqual({
      own: 190,
      phone: false,
      room: -948,
    });
  });

  it("has the window below what the page pins, on a page laid out as on a phone", () => {
    const { body, card } = layOut(844, 1767, {
      "rack-affix": "fixed",
      "examiner-controls": "fixed",
    });
    expect(roomInWindow(body, card).room).toBe(507);
  });

  it("leaves out an affix the page keeps in place, or one not pinned", () => {
    const { body, card } = layOut(844, 1767, {
      "rack-affix": "fixed",
      "examiner-controls": "static",
    });
    expect(roomInWindow(body, card).room).toBe(571);
    document.body.replaceChildren();
    const again = layOut(844, 1767, { "examiner-controls": "unpinned" });
    expect(roomInWindow(again.body, again.card).room).toBe(629);
  });
});

describe("refitOnPageChanges", () => {
  it("fits again as the window is resized, until stopped", () => {
    const fit = vi.fn();
    const stop = refitOnPageChanges(fit);
    window.dispatchEvent(new Event("resize"));
    expect(fit).toHaveBeenCalledTimes(1);
    stop();
    window.dispatchEvent(new Event("resize"));
    expect(fit).toHaveBeenCalledTimes(1);
  });

  it("fits again as an affix pins, even one the page shows anew, and only then", async () => {
    const fit = vi.fn();
    const stop = refitOnPageChanges(fit);
    const other = document.createElement("div");
    document.body.append(other);
    other.className = "elsewhere";
    await settle();
    expect(fit).not.toHaveBeenCalled();
    const controls = document.createElement("div");
    controls.className = "examiner-controls";
    const pinned = document.createElement("div");
    controls.append(pinned);
    document.body.append(controls);
    pinned.className = "ant-affix";
    await settle();
    expect(fit).toHaveBeenCalledTimes(1);
    stop();
    pinned.className = "";
    await settle();
    expect(fit).toHaveBeenCalledTimes(1);
  });
});
