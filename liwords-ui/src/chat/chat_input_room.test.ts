import { describe, expect, it } from "vitest";
import {
  entitiesMaxHeight,
  inputMaxHeight,
  MIN_CHAT_LIST_PX,
  roomAboveInput,
  rowsThatFit,
} from "./chat";

const at = (top: number, bottom = top) =>
  ({
    getBoundingClientRect: () => ({ top, bottom }),
  }) as unknown as HTMLElement;

describe("roomAboveInput", () => {
  it("is the distance from the list's top to the input's top", () => {
    expect(roomAboveInput(at(256), at(795))).toBe(539);
  });

  it("is never negative", () => {
    expect(roomAboveInput(at(300), at(200))).toBe(0);
  });
});

describe("entitiesMaxHeight", () => {
  it("takes the smaller of the layout's cap and the room", () => {
    expect(entitiesMaxHeight(132, 100)).toBe(100);
    expect(entitiesMaxHeight(132, 400)).toBe(132);
  });

  it("uses whichever is known", () => {
    expect(entitiesMaxHeight(undefined, 539)).toBe(539);
    expect(entitiesMaxHeight(132, undefined)).toBe(132);
    expect(entitiesMaxHeight(undefined, undefined)).toBeUndefined();
  });
});

describe("inputMaxHeight", () => {
  it("leaves the list its minimum above the input", () => {
    // list starts at 283, input ends at 434
    expect(inputMaxHeight(at(283), at(292, 434))).toBe(
      434 - 283 - MIN_CHAT_LIST_PX,
    );
  });

  it("is never negative", () => {
    expect(inputMaxHeight(at(400), at(410, 420))).toBe(0);
  });
});

describe("rowsThatFit", () => {
  const ta = () => {
    const el = document.createElement("textarea");
    el.style.lineHeight = "22px";
    el.style.padding = "4px 11px";
    el.style.border = "1px solid black";
    document.body.appendChild(el);
    return el;
  };

  it("counts whole rows inside the padding and border", () => {
    // 10px of frame, then 22px a row
    expect(rowsThatFit(ta(), 10 + 3 * 22 + 5)).toBe(3);
  });

  it("stays between one and six", () => {
    expect(rowsThatFit(ta(), 5)).toBe(1);
    expect(rowsThatFit(ta(), 1000)).toBe(6);
  });
});
