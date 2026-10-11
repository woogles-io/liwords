import { describe, expect, it } from "vitest";
import { listsChannel } from "./chat_channels";

// The unread badge counts with this same test, so a channel it rejects can
// no longer light the badge with nothing to show for it.
const ch = (name: string, displayName: string) => ({ name, displayName });
const nobody = new Set<string>();

describe("listsChannel", () => {
  it("shows only direct messages in the lobby", () => {
    expect(
      listsChannel(
        ch("chat.pm.a_b", "pm:a:b"),
        "chat.lobby",
        undefined,
        nobody,
      ),
    ).toBe(true);
    expect(
      listsChannel(
        ch("chat.league.x", "league:A League"),
        "chat.lobby",
        undefined,
        nobody,
      ),
    ).toBe(false);
    expect(
      listsChannel(
        ch("chat.tournament.t", "tournament:T"),
        "chat.lobby",
        undefined,
        nobody,
      ),
    ).toBe(false);
  });

  it("adds the page's own tournament channel on a tournament page", () => {
    const here = "chat.tournament.t";
    expect(
      listsChannel(ch(here, "tournament:T"), "chat.tournament.u", "t", nobody),
    ).toBe(true);
    expect(
      listsChannel(
        ch("chat.tournament.v", "tournament:V"),
        "chat.tournament.u",
        "t",
        nobody,
      ),
    ).toBe(false);
  });

  it("never lists the channel the chat already shows", () => {
    expect(
      listsChannel(
        ch("chat.pm.a_b", "pm:a:b"),
        "chat.pm.a_b",
        undefined,
        nobody,
      ),
    ).toBe(false);
  });

  it("hides channels with an excluded player", () => {
    expect(
      listsChannel(
        ch("chat.pm.a_b", "pm:a:b"),
        "chat.lobby",
        undefined,
        new Set(["b"]),
      ),
    ).toBe(false);
  });
});
