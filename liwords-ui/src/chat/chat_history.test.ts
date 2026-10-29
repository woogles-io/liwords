import { describe, expect, it } from "vitest";
import { ChatEntityObj, ChatEntityType } from "../store/constants";
import { FETCHED_CHATS, mergeChatHistory } from "./chat_history";

const here = "chat.league.x";

// A message of the channel, its id and timestamp from its time.
const said = (t: number, channel = here): ChatEntityObj => ({
  entityType: ChatEntityType.UserChat,
  sender: "someone",
  message: `at ${t}`,
  id: `${t}-0`,
  timestamp: BigInt(t),
  channel,
});

// A notice posted into the chat, never in any history.
const notice = (id: string): ChatEntityObj => ({
  entityType: ChatEntityType.ServerMsg,
  sender: "",
  message: id,
  id,
  channel: "server",
});

const ids = (chat: Array<ChatEntityObj>) => chat.map((e) => e.id);

describe("mergeChatHistory", () => {
  it("adds what was said while away after what is shown, keeping notices", () => {
    const chat = [said(1), said(2), notice("valid")];
    const merged = mergeChatHistory(
      chat,
      [said(1), said(2), said(3), said(4)],
      here,
      150,
    );
    expect(ids(merged)).toEqual(["1-0", "2-0", "valid", "3-0", "4-0"]);
  });

  it("puts a missed message before one the socket brought since", () => {
    const chat = [said(1), notice("valid"), said(5)];
    const merged = mergeChatHistory(
      chat,
      [said(1), said(3), said(5)],
      here,
      150,
    );
    expect(ids(merged)).toEqual(["1-0", "valid", "3-0", "5-0"]);
  });

  it("drops a message deleted meanwhile, but only between the fetched ones", () => {
    // 1 is older than the history reaches and 9 newer than it was fetched;
    // 4 is between them and not in it, so it was deleted.
    const chat = [said(1), said(3), said(4), said(5), said(9)];
    const merged = mergeChatHistory(
      chat,
      [said(3), said(5), said(7)],
      here,
      150,
    );
    expect(ids(merged)).toEqual(["1-0", "3-0", "5-0", "7-0", "9-0"]);
  });

  it("leaves other channels' messages alone", () => {
    const chat = [said(3), said(4, "chat.pm.a_b"), said(5)];
    const merged = mergeChatHistory(chat, [said(3), said(5)], here, 150);
    expect(merged).toBe(chat);
  });

  it("returns the same chat when nothing was missed", () => {
    const chat = [said(1), notice("valid"), said(2)];
    expect(mergeChatHistory(chat, [said(1), said(2)], here, 150)).toBe(chat);
    expect(mergeChatHistory(chat, [], here, 150)).toBe(chat);
  });

  // The newest FETCHED_CHATS messages, said at from, from + 1, ...
  const fullHistory = (from: number) =>
    Array.from({ length: FETCHED_CHATS }, (_, i) => said(from + i));

  it("marks the hole when more was said while away than comes back", () => {
    const chat = [said(1), said(2), notice("valid")];
    const merged = mergeChatHistory(chat, fullHistory(10), here, 150);
    expect(ids(merged).slice(0, 5)).toEqual([
      "1-0",
      "2-0",
      "valid",
      "catch-up-gap-10-0",
      "10-0",
    ]);
    expect(merged).toHaveLength(3 + 1 + FETCHED_CHATS);
    const gap = merged[3];
    expect(gap.entityType).toBe(ChatEntityType.ServerMsg);
    expect(gap.channel).toBe("server");
    expect(gap.message).toBe(
      "Some messages sent while you were away may be missing here.",
    );
    // the next catch-up meets messages the chat has: no second notice
    const again = mergeChatHistory(merged, fullHistory(12), here, 150);
    expect(ids(again).filter((id) => id?.startsWith("catch-up-gap"))).toEqual([
      "catch-up-gap-10-0",
    ]);
  });

  it("marks no hole when the fetched messages reach what the chat had", () => {
    // the oldest fetched one is already shown
    const merged = mergeChatHistory([said(10)], fullHistory(10), here, 150);
    expect(ids(merged)).not.toContain("catch-up-gap-10-0");
    // fewer than the server sends at most: that is all there is
    const short = mergeChatHistory(
      [said(1)],
      fullHistory(10).slice(1),
      here,
      150,
    );
    expect(ids(short).some((id) => id?.startsWith("catch-up-gap"))).toBe(false);
    // nothing of the channel to go after, only another channel and notices
    const fresh = mergeChatHistory(
      [said(1, "chat.pm.a_b"), notice("valid")],
      fullHistory(10),
      here,
      150,
    );
    expect(ids(fresh).some((id) => id?.startsWith("catch-up-gap"))).toBe(false);
  });

  it("keeps the newest entries when there are too many", () => {
    const chat = [said(1), said(2), notice("valid")];
    const merged = mergeChatHistory(
      chat,
      [said(1), said(2), said(3), said(4)],
      here,
      3,
    );
    expect(ids(merged)).toEqual(["valid", "3-0", "4-0"]);
  });
});
