import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ChatEntityType } from "./constants";
import { Store, useChatStoreContext } from "./store";

window.RUNTIME_CONFIGURATION = {};

const serverMsg = (id: string) => ({
  entityType: ChatEntityType.ServerMsg,
  sender: "",
  message: `message ${id}`,
  id,
  channel: "server",
});

describe("the chat store", () => {
  it("counts the times the chat list is replaced", () => {
    const { result } = renderHook(() => useChatStoreContext(), {
      wrapper: Store,
    });
    const before = result.current.chatGeneration;
    act(() => {
      result.current.addChat(serverMsg("a"));
    });
    expect(result.current.chatGeneration).toBe(before);
    act(() => {
      result.current.clearChat();
      result.current.addChats([serverMsg("b")], "chat.game.abc");
    });
    expect(result.current.chatGeneration).toBeGreaterThan(before);
    expect(result.current.chat.map((c) => c.id)).toEqual(["b"]);
    expect(result.current.loadedChatChannel).toBe("chat.game.abc");
  });

  it("does not add an entry whose id is already listed", () => {
    const { result } = renderHook(() => useChatStoreContext(), {
      wrapper: Store,
    });
    act(() => {
      result.current.addChat(serverMsg("a"));
      result.current.addChat(serverMsg("b"));
    });
    act(() => {
      result.current.addChat(serverMsg("a"));
    });
    expect(result.current.chat.map((c) => c.id)).toEqual(["a", "b"]);
  });
});
