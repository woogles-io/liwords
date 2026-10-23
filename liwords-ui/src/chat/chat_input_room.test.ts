import { describe, expect, it } from "vitest";
import { entitiesMaxHeight, roomAboveInput } from "./chat";

const at = (top: number) =>
  ({ getBoundingClientRect: () => ({ top }) }) as unknown as HTMLElement;

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
