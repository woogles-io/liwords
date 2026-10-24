import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  GameEvent_Type,
  GameEventSchema,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { ClockRules, moveTimes } from "./move_times";

const evt = (type: GameEvent_Type, playerIndex: number, millis: number) =>
  create(GameEventSchema, { type, playerIndex, millisRemaining: millis });

const live: ClockRules = {
  timeStarted: 1_000_000,
  initialMs: 60_000,
  incrementMs: 0,
  resetToIncrement: false,
  maxOvertimeMs: 0,
};

describe("moveTimes", () => {
  it("adds up each player's thinking time from the start", () => {
    // p0 thinks 10s, p1 thinks 5s, p0 thinks 7s.
    const t = moveTimes(
      [
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, 50_000),
        evt(GameEvent_Type.EXCHANGE, 1, 55_000),
        evt(GameEvent_Type.PASS, 0, 43_000),
      ],
      live,
    );
    expect(t).toEqual([1_010_000, 1_015_000, 1_022_000]);
  });

  it("adds the increment after a move", () => {
    // 2s increment: p0 had 60s, used 10s, has 52s for the next turn.
    const t = moveTimes(
      [
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, 50_000),
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 1, 57_000),
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, 40_000),
      ],
      { ...live, incrementMs: 2_000 },
    );
    expect(t).toEqual([1_010_000, 1_013_000, 1_025_000]);
  });

  it("times a challenge on the challenger's clock without ending the turn", () => {
    // p0 plays at +10s; p1 challenges 4s later and the phony comes off,
    // then p1 plays 6s after that.
    const t = moveTimes(
      [
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, 50_000),
        evt(GameEvent_Type.PHONY_TILES_RETURNED, 0, 56_000),
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 1, 50_000),
      ],
      live,
    );
    expect(t).toEqual([1_010_000, 1_014_000, 1_020_000]);
  });

  it("ends the challenger's turn when a double challenge fails", () => {
    const t = moveTimes(
      [
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, 50_000),
        evt(GameEvent_Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS, 1, 57_000),
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, 45_000),
      ],
      live,
    );
    expect(t).toEqual([1_010_000, 1_013_000, 1_018_000]);
  });

  it("resets a correspondence clock each turn, overruns included", () => {
    const hour = 3_600_000;
    const t = moveTimes(
      [
        // each player's first turn is on the initial time.
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, 22 * hour),
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 1, 11 * hour),
        // 13h used of a 10h turn: the clock went 3h negative.
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, -3 * hour),
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 1, 9 * hour),
      ],
      {
        timeStarted: 0,
        initialMs: 24 * hour,
        incrementMs: 10 * hour,
        resetToIncrement: true,
        maxOvertimeMs: 0,
      },
    );
    expect(t).toEqual([2 * hour, 15 * hour, 28 * hour, 29 * hour]);
  });

  it("caps overtime as the server does", () => {
    // p0 runs 30s over a 1-minute overtime cap... capped at -60s.
    const t = moveTimes(
      [
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, -90_000),
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 1, 50_000),
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, -70_000),
      ],
      { ...live, maxOvertimeMs: 60_000 },
    );
    expect(t).toEqual([1_150_000, 1_160_000, 1_170_000]);
  });

  it("gives end-of-game rack events no time", () => {
    const t = moveTimes(
      [
        evt(GameEvent_Type.TILE_PLACEMENT_MOVE, 0, 50_000),
        evt(GameEvent_Type.END_RACK_PTS, 0, 50_000),
      ],
      live,
    );
    expect(t).toEqual([1_010_000, undefined]);
  });
});
