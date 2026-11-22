import { describe, expect, it } from "vitest";
import {
  listsChannel,
  nextChannelPage,
  withFirstPage,
  withOlderPage,
} from "./chat_channels";
import { create } from "@bufbuild/protobuf";
import {
  ActiveChatChannels,
  ActiveChatChannels_ChannelSchema,
  ActiveChatChannelsSchema,
} from "../gen/api/proto/user_service/user_service_pb";

// The unread badge counts with this same test, so a channel it rejects can
// no longer light the badge with nothing to show for it.
const ch = (name: string, displayName: string) => ({ name, displayName });
const nobody = new Set<string>();

describe("listsChannel", () => {
  it("shows private chats only", () => {
    expect(
      listsChannel(ch("chat.pm.a_b", "pm:a:b"), "chat.lobby", nobody),
    ).toBe(true);
    for (const room of ["chat.league.x", "chat.tournament.t", "chat.game.g"]) {
      expect(listsChannel(ch(room, "x"), "chat.tournament.t2", nobody)).toBe(
        false,
      );
    }
  });

  it("never lists the channel the chat already shows", () => {
    expect(
      listsChannel(ch("chat.pm.a_b", "pm:a:b"), "chat.pm.a_b", nobody),
    ).toBe(false);
  });

  it("hides channels with an excluded player", () => {
    expect(
      listsChannel(ch("chat.pm.a_b", "pm:a:b"), "chat.lobby", new Set(["b"])),
    ).toBe(false);
  });
});

// A chat listed, named after its newest message's ID, which is its cursor.
const listed = (n: number, name = `chat.pm.${n}`) =>
  create(ActiveChatChannels_ChannelSchema, {
    name,
    displayName: "pm:a:b",
    lastUpdate: BigInt(n),
    lastMessageId: `id-${n.toString().padStart(3, "0")}`,
  });
const page = (ns: Array<number>, hasMore: boolean) =>
  create(ActiveChatChannelsSchema, {
    channels: ns.map((n) => listed(n)),
    hasMore,
  });
const names = (p: ActiveChatChannels) => p.channels.map((c) => c.name);

describe("the chat list's pages", () => {
  it("adds older pages after what is listed", () => {
    let cur = page([9, 8], true);
    expect(nextChannelPage(cur)).toBe("id-008");
    cur = withOlderPage(cur, page([7, 6], false));
    expect(names(cur)).toEqual([
      "chat.pm.9",
      "chat.pm.8",
      "chat.pm.7",
      "chat.pm.6",
    ]);
    expect(nextChannelPage(cur)).toBe("");
  });

  it("keeps the older pages loaded when the first is fetched again", () => {
    const cur = withOlderPage(page([9, 8], true), page([7, 6], false));
    // 6 has a new message: it moves to the top, and 9 to the second page.
    const fresh = withFirstPage(cur, page([10, 8], true));
    expect(names(fresh)).toEqual([
      "chat.pm.10",
      "chat.pm.8",
      "chat.pm.7",
      "chat.pm.6",
    ]);
    expect(fresh.hasMore).toBe(false);
  });

  it("lists a chat that moved up only once", () => {
    const cur = withOlderPage(page([9, 8], true), page([7, 6], true));
    const moved = page([11, 9], true);
    moved.channels[0] = listed(11, "chat.pm.7");
    expect(names(withFirstPage(cur, moved))).toEqual([
      "chat.pm.7",
      "chat.pm.9",
      "chat.pm.8",
      "chat.pm.6",
    ]);
  });

  it("keeps what was loaded past a fresh first page", () => {
    const fresh = withFirstPage(page([9, 8], true), page([10, 9], true));
    // 8 is still the chat after 9.
    expect(names(fresh)).toEqual(["chat.pm.10", "chat.pm.9", "chat.pm.8"]);
    expect(fresh.hasMore).toBe(true);
    expect(names(withFirstPage(undefined, page([1], false)))).toEqual([
      "chat.pm.1",
    ]);
  });
});
