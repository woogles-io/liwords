import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { ChatMessageSchema } from "../gen/api/proto/ipc/chat_pb";
import { ChatChannelSummarySchema } from "../gen/api/proto/mod_service/mod_service_pb";
import {
  chatsToCSV,
  formatTime,
  mergeRows,
  summariesToCSV,
} from "./chat_review";

describe("chat review", () => {
  it("writes UTC times from milliseconds", () => {
    expect(formatTime(BigInt(1700000000123))).toBe("2023-11-14T22:13:20.123Z");
    expect(formatTime(BigInt(0))).toBe("");
  });

  it("quotes CSV fields that need it", () => {
    const csv = chatsToCSV([
      create(ChatMessageSchema, {
        timestamp: BigInt(1700000000123),
        channel: "chat.game.abc",
        username: "cesar",
        userId: "u1",
        id: "1700000000123-0",
        message: 'say "hi", then\nbye',
      }),
    ]);
    expect(csv).toBe(
      "time_utc,channel,username,user_id,message_id,message\n" +
        '2023-11-14T22:13:20.123Z,chat.game.abc,cesar,u1,1700000000123-0,"say ""hi"", then\nbye"\n',
    );
  });

  it("lists each sender with a count", () => {
    const csv = summariesToCSV([
      create(ChatChannelSummarySchema, {
        channel: "chat.game.abc",
        messageCount: 3,
        firstTimestamp: BigInt(1700000000123),
        lastTimestamp: BigInt(1700000000123),
        senders: [
          { username: "mina", count: 2 },
          { username: "cesar", count: 1 },
        ],
      }),
    ]);
    expect(csv.split("\n")[1]).toBe(
      "chat.game.abc,,3,2023-11-14T22:13:20.123Z,2023-11-14T22:13:20.123Z,mina:2 cesar:1,,",
    );
  });

  it("skips channels already listed", () => {
    const row = (channel: string) =>
      create(ChatChannelSummarySchema, { channel });
    expect(
      mergeRows(
        [row("chat.game.a"), row("chat.game.b")],
        [row("chat.game.b"), row("chat.game.c")],
      ).map((r) => r.channel),
    ).toEqual(["chat.game.a", "chat.game.b", "chat.game.c"]);
  });

  it("adds the decoded label and the game's times", () => {
    const csv = summariesToCSV([
      create(ChatChannelSummarySchema, {
        channel: "chat.game.abc",
        messageCount: 1,
        context: {
          label: "mina vs cesar",
          gameCreatedAt: BigInt(1700000000123),
          gameUpdatedAt: BigInt(1700000060123),
        },
      }),
    ]);
    expect(csv.split("\n")[1]).toBe(
      "chat.game.abc,mina vs cesar,1,,,,2023-11-14T22:13:20.123Z,2023-11-14T22:14:20.123Z",
    );
  });
});
