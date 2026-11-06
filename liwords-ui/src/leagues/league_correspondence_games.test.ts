import { create, fromJsonString } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";
import { PlayerSeasonGameSchema } from "../gen/api/proto/league_service/league_service_pb";
import { justEndedGames, mistakesText } from "./league_correspondence_games";

const H = 60 * 60 * 1000;
const now = Date.UTC(2026, 9, 25, 12);
const game = (gameId: string, result: string, hoursAgo: number) =>
  create(PlayerSeasonGameSchema, {
    gameId,
    result,
    gameDate: timestampFromDate(new Date(now - hoursAgo * H)),
  });

describe("justEndedGames", () => {
  it("keeps the last day's finished games, newest first", () => {
    const games = [
      game("old", "win", 30),
      game("loss2h", "loss", 2),
      game("live", "turn", 1),
      game("draw20m", "draw", 1 / 3),
      game("waiting", "in_progress", 0.5),
      game("win23h", "win", 23),
    ];
    expect(justEndedGames(games, now).map((g) => g.gameId)).toEqual([
      "draw20m",
      "loss2h",
      "win23h",
    ]);
  });
});

describe("mistakesText", () => {
  const analyzed = (mine?: number, theirs?: number) =>
    create(PlayerSeasonGameSchema, {
      result: "win",
      mistakeIndex: mine,
      opponentMistakeIndex: theirs,
    });

  it("gives the player's first, as the score does", () => {
    expect(mistakesText(analyzed(3.2, 5.14))).toBe("MI 3.2-5.1");
  });

  it("shows a perfect game as 0, not as unanalyzed", () => {
    expect(mistakesText(analyzed(0, 1.5))).toBe("MI 0.0-1.5");
    // Both perfect, as the server's JSON writes it (a set 0 is kept, an
    // unanalyzed game leaves both out).
    const json = (s: string) =>
      mistakesText(fromJsonString(PlayerSeasonGameSchema, s));
    expect(json('{"mistakeIndex":0,"opponentMistakeIndex":0}')).toBe(
      "MI 0.0-0.0",
    );
    expect(json('{"gameId":"unanalyzed"}')).toBe("MI -");
  });

  it("shows a dash until the game has been analyzed", () => {
    expect(mistakesText(analyzed())).toBe("MI -");
    // One side alone, which the server never sends: an analysis gives both.
    expect(mistakesText(analyzed(2.5))).toBe("MI -");
    expect(mistakesText(analyzed(undefined, 2.5))).toBe("MI -");
  });
});
