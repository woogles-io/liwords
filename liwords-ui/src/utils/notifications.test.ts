import { afterEach, describe, expect, it, vi } from "vitest";
import { showTurnNotification } from "./notifications";

const shown: Array<{ title: string; tag?: string }> = [];

const stubNotification = () => {
  function FakeNotification(
    this: { onclick: unknown; close: () => void },
    title: string,
    opts: { tag?: string },
  ) {
    shown.push({ title, tag: opts.tag });
    this.onclick = null;
    this.close = () => {};
  }
  FakeNotification.permission = "granted";
  vi.stubGlobal("Notification", FakeNotification);
  localStorage.setItem("turnNotificationsEnabled", "true");
  vi.spyOn(document, "hasFocus").mockReturnValue(false);
};

afterEach(() => {
  shown.length = 0;
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  localStorage.clear();
});

describe("showTurnNotification", () => {
  it("tags a game's notification as its pushed turn is tagged", () => {
    stubNotification();
    showTurnNotification({ opponentName: "cesar", gameId: "abc" });
    expect(shown).toEqual([
      { title: "Your turn vs cesar", tag: "woogles-turn-abc" },
    ]);
  });

  it("keeps the plain tag without a game", () => {
    stubNotification();
    showTurnNotification({ opponentName: "cesar" });
    expect(shown[0].tag).toBe("woogles-turn");
  });
});
