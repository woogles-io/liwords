import { readFileSync } from "fs";
import { resolve } from "path";
import { describe, expect, it, vi } from "vitest";

// Runs public/sw.js (this repository's own file) against a fake service
// worker scope and returns its push handler and the notifications it shows.
const loadWorker = () => {
  const handlers: Record<string, (event: unknown) => void> = {};
  const showNotification = vi.fn(async () => undefined);
  const self = {
    addEventListener: (type: string, fn: (event: unknown) => void) => {
      handlers[type] = fn;
    },
    skipWaiting: () => undefined,
    registration: { showNotification },
  };
  const clients = { claim: async () => {} };
  const source = readFileSync(resolve(__dirname, "../../public/sw.js"), "utf8");
  new Function("self", "clients", source)(self, clients);
  const push = async (data: object) => {
    let done: Promise<unknown> = Promise.resolve();
    handlers.push({
      data: { json: () => data },
      waitUntil: (p: Promise<unknown>) => {
        done = p;
      },
    });
    await done;
  };
  return { push, showNotification };
};

describe("sw.js push", () => {
  it("shows the turn with the deadline in the device's own time", async () => {
    const w = loadWorker();
    const deadline = Date.UTC(2026, 10, 14, 6, 30);
    await w.push({
      gameId: "abc",
      title: "Your turn vs cesar4",
      body: "It's your turn to play",
      deadline,
    });
    const moveBy = new Intl.DateTimeFormat(undefined, {
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
      timeZoneName: "short",
    }).format(new Date(deadline));
    expect(w.showNotification).toHaveBeenCalledWith("Your turn vs cesar4", {
      body: `It's your turn to play - move by ${moveBy}`,
      icon: "/logo192.png",
      badge: "/favicon.ico",
      tag: "woogles-turn-abc",
      data: { gameId: "abc" },
    });
  });

  it("shows the turn without a deadline as it is", async () => {
    const w = loadWorker();
    await w.push({ gameId: "abc", title: "t", body: "b" });
    expect(w.showNotification).toHaveBeenCalledWith(
      "t",
      expect.objectContaining({ body: "b" }),
    );
  });

  it("shows nothing for a push that is not a turn", async () => {
    const w = loadWorker();
    await w.push({ nothing: true });
    expect(w.showNotification).not.toHaveBeenCalled();
  });
});
