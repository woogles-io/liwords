import { describe, expect, it } from "vitest";
import {
  entitiesMaxHeight,
  inputMaxHeight,
  layoutListCap,
  MIN_CHAT_LIST_PX,
  roomAboveInput,
  roomToGive,
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

describe("roomToGive", () => {
  // a chat whose message list starts at listTop and whose input starts at
  // inputTop, the list shown or not
  const chat = (listTop: number, inputTop: number, listShown = true) => {
    const el = document.createElement("div");
    const place = (child: HTMLElement, top: number, shown: boolean) => {
      child.getBoundingClientRect = () => ({ top, bottom: top }) as DOMRect;
      child.getClientRects = () =>
        (shown ? [{}] : []) as unknown as DOMRectList;
      el.appendChild(child);
    };
    const list = document.createElement("div");
    list.className = "entities game";
    place(list, listTop, listShown);
    const input = document.createElement("textarea");
    input.name = "chat-input";
    place(input, inputTop, listShown);
    return el;
  };

  it("gives down to where the list keeps its minimum above the input", () => {
    expect(roomToGive(chat(283, 795))).toBe(795 - 283 - MIN_CHAT_LIST_PX);
  });

  it("gives nothing while collapsed", () => {
    const collapsed = chat(283, 795);
    collapsed.classList.add("collapsed");
    expect(roomToGive(collapsed)).toBe(0);
  });

  it("gives what it last could while its messages are not shown", () => {
    const tabbed = chat(283, 795);
    expect(roomToGive(tabbed)).toBe(795 - 283 - MIN_CHAT_LIST_PX);
    tabbed.querySelectorAll<HTMLElement>("div, textarea").forEach((el) => {
      el.getClientRects = () => [] as unknown as DOMRectList;
    });
    expect(roomToGive(tabbed)).toBe(795 - 283 - MIN_CHAT_LIST_PX);
  });

  it("gives what the layout's cap alone allows before its messages were ever shown", () => {
    const hidden = chat(283, 795, false);
    Object.defineProperty(hidden, "clientHeight", { value: 353 });
    // 353 - 106 - 100 = 147
    expect(roomToGive(hidden)).toBe(147 - MIN_CHAT_LIST_PX);
  });

  it("is never negative", () => {
    expect(roomToGive(chat(283, 300))).toBe(0);
  });

  it("keeps the list its minimum under the layout's cap too", () => {
    const capped = chat(283, 795);
    Object.defineProperty(capped, "clientHeight", { value: 353 });
    const context = document.createElement("div");
    context.id = "chat-context";
    Object.defineProperty(context, "clientHeight", { value: 67 });
    capped.appendChild(context);
    // a cap of 353 - 106 - 100 = 147, under the room of 512 above the input
    expect(roomToGive(capped)).toBe(147 - MIN_CHAT_LIST_PX);
  });
});

describe("layoutListCap", () => {
  it("leaves the context panel at least 106 px, and 100 px more", () => {
    expect(layoutListCap(353, 67)).toBe(147);
    expect(layoutListCap(353, 120)).toBe(133);
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
