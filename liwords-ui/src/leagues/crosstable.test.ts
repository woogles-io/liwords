import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { CrosstableGameSchema } from "../gen/api/proto/league_service/league_service_pb";
import {
  buildCrosstable,
  crossClass,
  endMark,
  ifEndedNow,
  onTurnCounts,
  scoreType,
} from "./crosstable";

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

describe("onTurnCounts", () => {
  it("counts the games in progress each player is on turn in", () => {
    const live = { gameEndReason: GameEndReason.NONE, winnerIndex: -1 };
    const cells = buildCrosstable(
      ["a", "b", "c"],
      [
        // finished: on nobody
        game({ gameId: "ab" }),
        // live, a to move, twice over
        game({ ...live, gameId: "ac", player1UserId: "c", playerOnTurn: 0 }),
        game({
          ...live,
          gameId: "ca",
          player0UserId: "c",
          player1UserId: "a",
          playerOnTurn: 1,
        }),
        // live, b to move
        game({
          ...live,
          gameId: "bc",
          player0UserId: "b",
          player1UserId: "c",
          playerOnTurn: 0,
        }),
      ],
    );
    const n = onTurnCounts(["a", "b", "c"], cells);
    expect(n.get("a")).toBe(2);
    expect(n.get("b")).toBe(1);
    expect(n.get("c")).toBe(0);
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

describe("ifEndedNow", () => {
  const live = { gameEndReason: GameEndReason.NONE, winnerIndex: -1 };
  // an official standing: finished games only
  const standing = (
    userId: string,
    o: Partial<{ wins: number; losses: number; draws: number; spread: number }>,
  ) => ({ userId, wins: 0, losses: 0, draws: 0, spread: 0, ...o });

  it("adds a live game for its leader, and a level one as a draw", () => {
    const ids = ["a", "b", "c"];
    const cells = buildCrosstable(ids, [
      // finished: a beat b 400-350, as the standings record
      game({ gameId: "ab" }),
      // live: c leads a 150-120
      game({
        ...live,
        gameId: "ac",
        player1UserId: "c",
        player0Score: 120,
        player1Score: 150,
        playerOnTurn: 0,
      }),
      // live and not yet started: 0-0
      game({
        ...live,
        gameId: "bc",
        player0UserId: "b",
        player1UserId: "c",
        player0Score: 0,
        player1Score: 0,
        playerOnTurn: 0,
      }),
    ]);
    const now = ifEndedNow(
      [
        standing("a", { wins: 1, spread: 50 }),
        standing("b", { losses: 1, spread: -50 }),
        standing("c", {}),
      ],
      cells,
    );
    expect(now.get("c")).toEqual({
      wins: 1,
      losses: 0,
      draws: 1,
      spread: 30,
      rank: 1,
      tied: false,
      final: false,
    });
    expect(now.get("a")).toEqual({
      wins: 1,
      losses: 1,
      draws: 0,
      spread: 20,
      rank: 2,
      tied: false,
      final: false,
    });
    expect(now.get("b")).toEqual({
      wins: 0,
      losses: 1,
      draws: 1,
      spread: -50,
      rank: 3,
      tied: false,
      final: false,
    });
  });

  it("takes finished games from the standings only", () => {
    // b won a forfeit while behind 400-350: the standings have it, so the
    // cross-table's copy is not counted again. a's +30 win is over a player
    // the cross-table no longer lists, and still counts.
    const cells = buildCrosstable(
      ["a", "b"],
      [game({ gameEndReason: GameEndReason.FORCE_FORFEIT, winnerIndex: 1 })],
    );
    const now = ifEndedNow(
      [
        standing("a", { wins: 1, losses: 1, spread: 80 }),
        standing("b", { wins: 1, spread: -50 }),
      ],
      cells,
    );
    expect(now.get("a")).toEqual({
      wins: 1,
      losses: 1,
      draws: 0,
      spread: 80,
      rank: 1,
      tied: false,
      final: true,
    });
    expect(now.get("b")).toEqual({
      wins: 1,
      losses: 0,
      draws: 0,
      spread: -50,
      rank: 2,
      tied: false,
      final: true,
    });
  });

  it("lets players level on points and spread share a rank", () => {
    const ids = ["a", "b", "c"];
    const cells = buildCrosstable(ids, [
      game({ ...live, player0Score: 100, player1Score: 100 }),
    ]);
    const now = ifEndedNow(
      ids.map((id) => standing(id, {})),
      cells,
    );
    expect(now.get("a")).toMatchObject({ rank: 1, tied: true, final: false });
    expect(now.get("b")).toMatchObject({ rank: 1, tied: true, final: false });
    expect(now.get("c")).toMatchObject({ draws: 0, rank: 3, final: true });
  });
});
