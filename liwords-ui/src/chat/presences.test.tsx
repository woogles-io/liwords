import React from "react";
import { act, cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PresenceEntity } from "../store/constants";
import {
  ARRIVAL_HIGHLIGHT_MS,
  Presences,
  sortByUsername,
  updateFirstSeen,
} from "./presences";

vi.mock("../shared/usernameWithContext", () => ({
  UsernameWithContext: (props: { username: string }) => (
    <span>{props.username}</span>
  ),
}));

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

const p = (username: string, anon = false): PresenceEntity => ({
  uuid: `u-${username}`,
  username,
  channel: "chat.tournament.t",
  anon,
  deleting: false,
});

describe("presence helpers", () => {
  it("sorts by username, ignoring case", () => {
    expect(
      sortByUsername([p("zed"), p("Bea"), p("amy")]).map((x) => x.username),
    ).toEqual(["amy", "Bea", "zed"]);
  });

  it("dates arrivals, not those already there, and forgets leavers", () => {
    let seen = updateFirstSeen(new Map(), ["a", "b"], 1000, true);
    expect([...seen]).toEqual([
      ["a", 0],
      ["b", 0],
    ]);
    seen = updateFirstSeen(seen, ["a", "b", "c"], 2000, false);
    expect(seen.get("c")).toBe(2000);
    seen = updateFirstSeen(seen, ["a", "c"], 3000, false);
    expect(seen.has("b")).toBe(false);
    seen = updateFirstSeen(seen, ["a", "b", "c"], 4000, false);
    expect(seen.get("b")).toBe(4000);
  });
});

it("lists names alphabetically and highlights a newcomer for a while", () => {
  vi.useFakeTimers();
  const channel = "chat.tournament.t";
  const { container, rerender } = render(
    <Presences players={[p("zed"), p("amy")]} channel={channel} />,
  );
  expect(container.textContent).toBe("amy, zed");
  expect(container.querySelectorAll(".recent-arrival").length).toBe(0);

  rerender(
    <Presences players={[p("zed"), p("amy"), p("Bea")]} channel={channel} />,
  );
  expect(container.textContent).toBe("amy, Bea, zed");
  const recent = container.querySelectorAll(".recent-arrival");
  expect(recent.length).toBe(1);
  expect(recent[0].textContent).toBe("Bea");

  act(() => {
    vi.advanceTimersByTime(ARRIVAL_HIGHLIGHT_MS + 1);
  });
  expect(container.querySelectorAll(".recent-arrival").length).toBe(0);
});
