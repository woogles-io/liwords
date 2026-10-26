import { describe, expect, it } from "vitest";
import { ChatEntityObj, ChatEntityType } from "../store/constants";
import { mergeChatHistory } from "./chat_history";

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
