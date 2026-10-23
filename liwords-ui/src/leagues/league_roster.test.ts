import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  LeagueRosterPlayerSchema,
  LeagueRosterSeasonSchema,
} from "../gen/api/proto/league_service/league_service_pb";
import { peakRepeats, totalTimeouts } from "./league_roster";

const season = (seasonNumber: number, divisionNumber: number, rank: number) =>
  create(LeagueRosterSeasonSchema, { seasonNumber, divisionNumber, rank });

describe("peakRepeats", () => {
  const player = create(LeagueRosterPlayerSchema, {
    userId: "u",
    seasons: [
      season(7, 1, 1),
      season(2, 1, 1),
      season(3, 1, 2),
      season(5, 1, 1),
      season(4, 2, 1),
    ],
  });

  it("lists every season with the peak finish, in order", () => {
    const peak = { season: player.seasons[1], displayRank: 1 };
    expect(peakRepeats(player, peak, new Map())).toEqual([2, 5, 7]);
  });

  it("goes by the shown (tie-aware) rank", () => {
    // Season 3 was a shared first place.
    const peak = { season: player.seasons[1], displayRank: 1 };
    expect(peakRepeats(player, peak, new Map([["3:u", 1]]))).toEqual([
      2, 3, 5, 7,
    ]);
  });

  it("counts an unplaced peak once", () => {
    const loner = create(LeagueRosterPlayerSchema, {
      userId: "v",
      seasons: [season(1, 0, 0), season(2, 0, 0)],
    });
    expect(
      peakRepeats(
        loner,
        { season: loner.seasons[0], displayRank: 0 },
        new Map(),
      ),
    ).toEqual([1]);
  });
});

describe("totalTimeouts", () => {
  it("adds up every season", () => {
    const player = create(LeagueRosterPlayerSchema, {
      seasons: [
        create(LeagueRosterSeasonSchema, { seasonNumber: 1, timeouts: 2 }),
        create(LeagueRosterSeasonSchema, { seasonNumber: 2 }),
        create(LeagueRosterSeasonSchema, { seasonNumber: 3, timeouts: 1 }),
      ],
    });
    expect(totalTimeouts(player)).toBe(3);
  });
});
