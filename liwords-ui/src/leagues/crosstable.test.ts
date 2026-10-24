import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { CrosstableGameSchema } from "../gen/api/proto/league_service/league_service_pb";
import { buildCrosstable, crossClass, endMark, scoreType } from "./crosstable";

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
    const std = GameEndReason.STANDARD;
    expect(cells.get("a|b")).toEqual([
      {
        gameId: "g",
        opponentId: "b",
        endReason: std,
        myScore: 400,
        theirScore: 350,
        result: "win",
      },
    ]);
    expect(cells.get("b|a")).toEqual([
      {
        gameId: "g",
        opponentId: "a",
        endReason: std,
        myScore: 350,
        theirScore: 400,
        result: "loss",
      },
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

describe("endMark", () => {
  it("marks games that were not played out", () => {
    expect(endMark(GameEndReason.TIME)).toBe("T");
    expect(endMark(GameEndReason.RESIGNED)).toBe("R");
    expect(endMark(GameEndReason.FORCE_FORFEIT)).toBe("F");
    expect(endMark(GameEndReason.TRIPLE_CHALLENGE)).toBe("C");
    expect(endMark(GameEndReason.STANDARD)).toBe("");
  });
});

describe("scoreType", () => {
  const cell = {
    gameId: "g",
    opponentId: "b",
    endReason: GameEndReason.NONE,
  };
  it("colors a live game by who is ahead", () => {
    expect(
      scoreType({ ...cell, result: "turn", myScore: 300, theirScore: 250 }),
    ).toBe("success");
    expect(
      scoreType({ ...cell, result: "waiting", myScore: 200, theirScore: 250 }),
    ).toBe("danger");
    expect(
      scoreType({ ...cell, result: "waiting", myScore: 0, theirScore: 0 }),
    ).toBe("secondary");
  });
  it("colors a finished game by its result", () => {
    // won on time while behind
    expect(
      scoreType({
        ...cell,
        endReason: GameEndReason.TIME,
        result: "win",
        myScore: 300,
        theirScore: 350,
      }),
    ).toBe("success");
    expect(
      scoreType({ ...cell, result: "draw", myScore: 400, theirScore: 400 }),
    ).toBe("secondary");
  });
});

describe("crossClass", () => {
  const hover = { row: "a", col: "b" };
  it("marks the hovered game and its mirror", () => {
    expect(crossClass("a", "b", hover)).toBe("xt-game");
    expect(crossClass("b", "a", hover)).toBe("xt-game");
  });
  it("shades both players' rows and columns, and nothing else", () => {
    expect(crossClass("a", "c", hover)).toBe("xt-cross");
    expect(crossClass("c", "b", hover)).toBe("xt-cross");
    expect(crossClass("b", "c", hover)).toBe("xt-cross");
    expect(crossClass("c", "a", hover)).toBe("xt-cross");
    expect(crossClass("c", "d", hover)).toBe("");
    expect(crossClass("a", "b", null)).toBe("");
  });
});
