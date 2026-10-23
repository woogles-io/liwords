import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { CrosstableGameSchema } from "../gen/api/proto/league_service/league_service_pb";
import { buildCrosstable, notStarted } from "./crosstable";

const game = (
  o: Partial<Parameters<typeof create<typeof CrosstableGameSchema>>[1]>,
) =>
  create(CrosstableGameSchema, {
    gameId: "g",
    player0UserId: "a",
    player1UserId: "b",
    player0Score: 400,
    player1Score: 350,
    gameEndReason: GameEndReason.STANDARD,
    winnerIndex: 0,
    playerOnTurn: -1,
    ...o,
  });

describe("buildCrosstable", () => {
  it("shows a game from each side", () => {
    const cells = buildCrosstable(["a", "b"], [game({})]);
    expect(cells.get("a|b")).toEqual([
      { gameId: "g", myScore: 400, theirScore: 350, result: "win" },
    ]);
    expect(cells.get("b|a")).toEqual([
      { gameId: "g", myScore: 350, theirScore: 400, result: "loss" },
    ]);
  });

  it("marks draws and whose turn it is", () => {
    const cells = buildCrosstable(
      ["a", "b", "c"],
      [
        game({ gameId: "d", winnerIndex: -1 }),
        game({
          gameId: "l",
          player1UserId: "c",
          gameEndReason: GameEndReason.NONE,
          winnerIndex: -1,
          playerOnTurn: 1,
        }),
      ],
    );
    expect(cells.get("a|b")?.[0].result).toBe("draw");
    expect(cells.get("a|c")?.[0].result).toBe("waiting");
    expect(cells.get("c|a")?.[0].result).toBe("turn");
  });

  it("keeps a repeated pairing and skips players outside the division", () => {
    const cells = buildCrosstable(
      ["a", "b"],
      [
        game({ gameId: "1" }),
        game({ gameId: "2", player0UserId: "b", player1UserId: "a" }),
        game({ gameId: "x", player1UserId: "stranger" }),
      ],
    );
    expect(cells.get("a|b")?.map((c) => c.gameId)).toEqual(["1", "2"]);
    expect(cells.get("a|stranger")).toBeUndefined();
    expect(cells.size).toBe(2);
  });
});

describe("notStarted", () => {
  it("is a live game with no score", () => {
    const cell = { gameId: "g", myScore: 0, theirScore: 0 };
    expect(notStarted({ ...cell, result: "turn" })).toBe(true);
    expect(notStarted({ ...cell, result: "waiting" })).toBe(true);
    expect(notStarted({ ...cell, myScore: 12, result: "waiting" })).toBe(false);
    expect(notStarted({ ...cell, result: "draw" })).toBe(false);
  });
});
