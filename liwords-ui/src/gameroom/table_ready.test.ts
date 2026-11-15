import { describe, expect, it } from "vitest";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { readyToSend } from "./table";

describe("readyToSend", () => {
  it("is sent by a player while the game goes on", () => {
    expect(readyToSend(GameEndReason.NONE, false, false)).toBe(true);
  });

  it("is not sent once the game is over, though gameInfo still says it goes on", () => {
    expect(readyToSend(GameEndReason.NONE, true, false)).toBe(false);
  });

  it("is not sent for a game that ended, or by an observer", () => {
    expect(readyToSend(GameEndReason.STANDARD, false, false)).toBe(false);
    expect(readyToSend(GameEndReason.NONE, false, true)).toBe(false);
  });
});
