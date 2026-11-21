import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ChatEntityObj } from "../store/constants";
import { isMessageID, useChatHistory } from "./use_chat_history";

// UUIDv7-shaped IDs that sort as their numbers do.
const id = (n: number) =>
  `0190a000-0000-7000-8000-${n.toString().padStart(12, "0")}`;
const msg = (n: number, userId = "zed") => ({
  id: id(n),
  username: userId,
  userId,
  channel: "chat.pm.me_zed",
  message: `m${n}`,
  timestamp: BigInt(n),
});

const setup = (pages: Array<object>) => {
  const client = {
    getChatsForChannel: vi.fn(),
  };
  for (const p of pages) {
    client.getChatsForChannel.mockResolvedValueOnce(p);
  }
  const replace = vi.fn<(e: Array<ChatEntityObj>) => void>();
  const insert = vi.fn<(e: Array<ChatEntityObj>, ch: string) => void>();
  const { result } = renderHook(() =>
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    useChatHistory(client as any, "me", replace, insert),
  );
  return { client, replace, insert, result };
};

describe("useChatHistory", () => {
  it("recognizes message IDs", () => {
    expect(isMessageID(id(1))).toBe(true);
    expect(isMessageID("1700000000000-0")).toBe(false);
    expect(isMessageID("random")).toBe(false);
  });

  it("opens around the read position, at the first unread from others", async () => {
    const { client, replace, result } = setup([
      {
        messages: [msg(1), msg(2), msg(3, "me"), msg(4), msg(5)],
        hasOlder: true,
        hasNewer: true,
        lastReadId: id(2),
      },
    ]);
    let w;
    await act(async () => {
      w = await result.current.open("chat.pm.me_zed");
    });
    expect(client.getChatsForChannel).toHaveBeenCalledWith({
      channel: "chat.pm.me_zed",
      aroundLastRead: true,
      limit: 100,
    });
    expect(replace.mock.calls[0][0].map((m) => m.message)).toEqual([
      "m1",
      "m2",
      "m3",
      "m4",
      "m5",
    ]);
    // m3 is the user's own: the line goes before m4.
    expect(w).toMatchObject({
      hasOlder: true,
      hasNewer: true,
      oldest: id(1),
      newest: id(5),
      lastReadAtOpen: id(2),
      dividerBefore: id(4),
    });
  });

  it("has no line on a first visit or with nothing unread", async () => {
    const { result } = setup([
      { messages: [msg(1), msg(2)], lastReadId: "" },
      { messages: [msg(1), msg(2)], lastReadId: id(2) },
    ]);
    await act(async () => {
      expect((await result.current.open("a")).dividerBefore).toBe("");
      expect((await result.current.open("a")).dividerBefore).toBe("");
    });
  });

  it("loads older and newer pages next to what is loaded", async () => {
    const { client, insert, result } = setup([
      { messages: [msg(5), msg(6)], hasOlder: true, hasNewer: true },
      { messages: [msg(3), msg(4)], hasOlder: false },
      { messages: [msg(7)], hasNewer: false },
    ]);
    await act(async () => {
      await result.current.open("c");
    });
    await act(async () => {
      expect(await result.current.loadOlder()).toBe(true);
    });
    expect(client.getChatsForChannel).toHaveBeenLastCalledWith({
      channel: "c",
      beforeId: id(5),
      limit: 100,
    });
    expect(insert.mock.calls[0][1]).toBe("c");
    expect(result.current.win).toMatchObject({
      hasOlder: false,
      oldest: id(3),
    });
    await act(async () => {
      expect(await result.current.loadOlder()).toBe(false); // nothing older
      await result.current.loadNewer();
    });
    expect(client.getChatsForChannel).toHaveBeenLastCalledWith({
      channel: "c",
      afterId: id(6),
      limit: 100,
    });
    expect(result.current.win).toMatchObject({
      hasNewer: false,
      newest: id(7),
    });
  });

  it("jumps to the newest messages, without the line", async () => {
    const { client, replace, result } = setup([
      {
        messages: [msg(1), msg(2)],
        hasNewer: true,
        lastReadId: id(1),
      },
      { messages: [msg(9), msg(10)], hasOlder: true },
    ]);
    await act(async () => {
      await result.current.open("c");
    });
    expect(result.current.win.dividerBefore).toBe(id(2));
    await act(async () => {
      await result.current.jumpToLatest();
    });
    expect(client.getChatsForChannel).toHaveBeenLastCalledWith({
      channel: "c",
      limit: 100,
    });
    expect(replace).toHaveBeenCalledTimes(2);
    expect(result.current.win).toMatchObject({
      hasOlder: true,
      hasNewer: false,
      newest: id(10),
      dividerBefore: "",
    });
  });

  it("moves the newest loaded with live messages, but not over older history", async () => {
    const { result } = setup([
      { messages: [msg(1)], hasNewer: false },
      { messages: [msg(1)], hasNewer: true },
    ]);
    await act(async () => {
      await result.current.open("c");
    });
    act(() => result.current.noteLive(id(2)));
    expect(result.current.win.newest).toBe(id(2));
    await act(async () => {
      await result.current.open("c");
    });
    act(() => result.current.noteLive(id(3)));
    expect(result.current.win.newest).toBe(id(1));
  });
});
