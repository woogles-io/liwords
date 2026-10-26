import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";
import { PlayerSeasonGameSchema } from "../gen/api/proto/league_service/league_service_pb";
import { justEndedGames } from "./league_correspondence_games";

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
