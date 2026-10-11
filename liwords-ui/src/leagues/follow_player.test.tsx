import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FollowBar } from "./follow_bar";
import {
  followedDivision,
  followNeighbors,
  followPlayer,
} from "./follow_player";

// Played S3 in division 10 and S5 in division 9, sat out S4, and has
// registered for S6 without a division yet.
const andy = followPlayer({
  userId: "andy-id",
  username: "andy",
  seasons: [
    { seasonNumber: 5, divisionNumber: 9 },
    { seasonNumber: 6, divisionNumber: 0 },
    { seasonNumber: 3, divisionNumber: 10 },
  ],
});

describe("followNeighbors", () => {
  it("skips the seasons the player sat out", () => {
    expect(followNeighbors(andy, 5)).toEqual({
      prev: { seasonNumber: 3, divisionNumber: 10 },
      next: { seasonNumber: 6, divisionNumber: 0 },
    });
  });

  it("steps back to the latest earlier season, from a season the player sat out too", () => {
    expect(followNeighbors(andy, 6).prev).toEqual({
      seasonNumber: 5,
      divisionNumber: 9,
    });
    expect(followNeighbors(andy, 4)).toEqual({
      prev: { seasonNumber: 3, divisionNumber: 10 },
      next: { seasonNumber: 5, divisionNumber: 9 },
    });
  });

  it("has nothing before the first season or after the last", () => {
    expect(followNeighbors(andy, 3).prev).toBeUndefined();
    expect(followNeighbors(andy, 6).next).toBeUndefined();
  });
});

describe("followedDivision", () => {
  it("is the player's division in a season they played", () => {
    expect(followedDivision(andy, 3)).toBe(10);
    expect(followedDivision(andy, 5)).toBe(9);
  });

  it("is nothing for a season they sat out or were not placed in", () => {
    expect(followedDivision(andy, 4)).toBeUndefined();
    expect(followedDivision(andy, 6)).toBeUndefined();
  });
});

describe("FollowBar", () => {
  it("steps to the neighbors, labeled with season and division", () => {
    const onGo = vi.fn();
    const onStop = vi.fn();
    render(
      <FollowBar
        username="andy"
        {...followNeighbors(andy, 5)}
        onGo={onGo}
        onStop={onStop}
      />,
    );
    expect(screen.getByText("Following andy")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /S3 D10/ }));
    expect(onGo).toHaveBeenCalledWith({ seasonNumber: 3, divisionNumber: 10 });
    fireEvent.click(screen.getByRole("button", { name: /S6/ }));
    expect(onGo).toHaveBeenLastCalledWith({
      seasonNumber: 6,
      divisionNumber: 0,
    });
    fireEvent.click(screen.getByRole("button", { name: "Stop" }));
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("disables the side with nowhere to go", () => {
    render(
      <FollowBar
        username="andy"
        {...followNeighbors(andy, 3)}
        onGo={() => {}}
        onStop={() => {}}
      />,
    );
    const [prev] = screen.getAllByRole("button");
    expect(prev).toBeDisabled();
  });
});
