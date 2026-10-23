import { describe, expect, it } from "vitest";
import {
  entitiesMaxHeight,
  inputMaxHeight,
  MIN_CHAT_LIST_PX,
  roomAboveInput,
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
