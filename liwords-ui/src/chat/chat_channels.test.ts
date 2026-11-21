import { describe, expect, it } from "vitest";
import { listsChannel } from "./chat_channels";

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
