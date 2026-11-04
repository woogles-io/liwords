import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  GameEventSchema,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import {
  SimCandidate,
  passesAtEnd,
  scorelessTurnsAtEnd,
  simOrder,
  unseenBy,
} from "./analyzer_lab";

const events = (...types: Array<GameEvent_Type>) =>
  types.map((type) => create(GameEventSchema, { type, score: 10 }));

const {
  TILE_PLACEMENT_MOVE: PLAY,
  PHONY_TILES_RETURNED: RETURNED,
  PASS,
  CHALLENGE_BONUS: BONUS,
  EXCHANGE,
  UNSUCCESSFUL_CHALLENGE_TURN_LOSS: TURN_LOSS,
  CHALLENGE,
} = GameEvent_Type;

describe("scoreless turns and passes in a row", () => {
  it("counts the scoreless turns since the last play that stood", () => {
    expect(scorelessTurnsAtEnd(events())).toBe(0);
    expect(scorelessTurnsAtEnd(events(PLAY))).toBe(0);
    expect(scorelessTurnsAtEnd(events(PLAY, PASS, EXCHANGE))).toBe(2);
    expect(scorelessTurnsAtEnd(events(PASS, PLAY, PASS))).toBe(1);
    // a play that came off is one scoreless turn, and so is a turn lost
    expect(scorelessTurnsAtEnd(events(PLAY, PASS, PLAY, RETURNED))).toBe(2);
    expect(scorelessTurnsAtEnd(events(PLAY, PLAY, TURN_LOSS))).toBe(1);
    // a challenged play that stood ends the count
    expect(scorelessTurnsAtEnd(events(PASS, PLAY, BONUS))).toBe(0);
  });

  it("counts only passes as passes, over challenges", () => {
    expect(passesAtEnd(events(EXCHANGE, PASS, PASS))).toBe(2);
    expect(passesAtEnd(events(PASS, EXCHANGE))).toBe(0);
    expect(passesAtEnd(events(PASS, CHALLENGE, PASS))).toBe(2);
    expect(passesAtEnd(events(PASS, PLAY, RETURNED))).toBe(0);
  });
});

describe("the tiles unseen", () => {
  it("takes the player's own rack out of the pool", () => {
    // two As, an E, a blank in the pool; an A and the blank on the rack
    expect(unseenBy({ 1: 2, 5: 1, 0: 1 }, [1, 0])).toEqual([1, 5]);
  });
});

const candidate = (name: string, win: number, equity: number) =>
  ({
    play: name,
    state: "running",
    n: 100,
    win: { mean: win, variance: 0 },
    equity: { mean: equity, variance: 0 },
  }) as SimCandidate;

describe("the sim's order", () => {
  it("puts the likelier win first, and of two as likely, the more equity", () => {
    const sorted = [
      candidate("lost, less equity", 0, -40),
      candidate("less win", 0.4, 30),
      candidate("won, less equity", 1, 20),
      candidate("lost, more equity", 0, -10),
      candidate("more win", 0.6, 5),
      candidate("won, more equity", 1, 50),
      candidate("nearly won", 0.9995, 90),
    ].sort(simOrder);
    expect(sorted.map((c) => c.play)).toEqual([
      "won, more equity",
      "won, less equity",
      "nearly won",
      "more win",
      "less win",
      "lost, more equity",
      "lost, less equity",
    ]);
  });
});
