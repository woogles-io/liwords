import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  type LeagueRosterPlayer,
  LeagueRosterPlayerSchema,
  LeagueRosterSeasonSchema,
} from "../gen/api/proto/league_service/league_service_pb";
import {
  peakOf,
  peakRepeats,
  seasonFloors,
  totalTimeouts,
} from "./league_roster";

const season = (seasonNumber: number, divisionNumber: number, rank: number) =>
  create(LeagueRosterSeasonSchema, { seasonNumber, divisionNumber, rank });

// Season 26 is the one the league has not completed.
const IN_PROGRESS = new Set([26]);

// A season in progress, with the games won in it so far.
const live = (
  seasonNumber: number,
  divisionNumber: number,
  rank: number,
  wins = 0,
) =>
  create(LeagueRosterSeasonSchema, {
    seasonNumber,
    divisionNumber,
    rank,
    wins,
  });

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
    expect(peakRepeats(player, peak, new Map(), new Set())).toEqual([2, 5, 7]);
  });

  it("goes by the shown (tie-aware) rank", () => {
    // Season 3 was a shared first place.
    const peak = { season: player.seasons[1], displayRank: 1 };
    expect(peakRepeats(player, peak, new Map([["3:u", 1]]), new Set())).toEqual(
      [2, 3, 5, 7],
    );
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
        new Set(),
      ),
    ).toEqual([1]);
  });

  it("never counts a season in progress as a repeat", () => {
    const now = create(LeagueRosterPlayerSchema, {
      userId: "w",
      seasons: [season(10, 1, 15), live(26, 1, 15)],
    });
    const peak = { season: now.seasons[0], displayRank: 15 };
    expect(peakRepeats(now, peak, new Map(), IN_PROGRESS)).toEqual([10]);
  });
});

// The rest of a season-in-progress division, ranked alphabetically before
// anybody has played (rank 2 on, the player under test being 1).
const others = (
  count: number,
  seasonNumber: number,
  divisionNumber: number,
  played = false,
) =>
  Array.from({ length: count }, (_, i) =>
    create(LeagueRosterPlayerSchema, {
      userId: `other${i}`,
      seasons: [live(seasonNumber, divisionNumber, i + 2, played ? 1 : 0)],
    }),
  );

const peakAt = (
  me: LeagueRosterPlayer,
  rest: Array<LeagueRosterPlayer>,
  worstRanks = new Map<string, number>(),
) => {
  const floors = seasonFloors([me, ...rest], IN_PROGRESS, 26, worstRanks);
  const peak = peakOf(me, new Map(), new Set(), floors, IN_PROGRESS);
  return (
    peak && [
      peak.season.seasonNumber,
      peak.displayRank,
      peak.provisional,
      peak.repeats,
    ]
  );
};

describe("a season in progress in Peak", () => {
  it("never pre-wins on the alphabetical order before the first game", () => {
    const andy = create(LeagueRosterPlayerSchema, {
      userId: "andy",
      seasons: [season(24, 1, 4), live(26, 1, 1)],
    });
    expect(peakAt(andy, others(14, 26, 1))).toEqual([24, 4, false, [24]]);
  });

  it("counts at last place when that beats every finish", () => {
    const promoted = create(LeagueRosterPlayerSchema, {
      userId: "up",
      seasons: [season(25, 2, 1), live(26, 1, 1)],
    });
    expect(peakAt(promoted, others(14, 26, 1))).toEqual([26, 15, true, [26]]);
  });

  it("lets an equal finish stand, once", () => {
    const again = create(LeagueRosterPlayerSchema, {
      userId: "again",
      seasons: [season(10, 1, 15), live(26, 1, 1)],
    });
    expect(peakAt(again, others(14, 26, 1))).toEqual([10, 15, false, [10]]);
  });

  it("takes the server's worst place once games are played", () => {
    const rising = create(LeagueRosterPlayerSchema, {
      userId: "rising",
      seasons: [season(20, 1, 8), live(26, 1, 1, 5)],
    });
    expect(
      peakAt(rising, others(14, 26, 1, true), new Map([["rising", 6]])),
    ).toEqual([26, 6, true, [26]]);
  });

  it("leaves a season in progress out with no worst place from the server", () => {
    const rising = create(LeagueRosterPlayerSchema, {
      userId: "rising",
      seasons: [season(20, 1, 8), live(26, 1, 1, 5)],
    });
    expect(peakAt(rising, others(14, 26, 1, true))).toEqual([
      20,
      8,
      false,
      [20],
    ]);
  });

  it("goes by the season's status, not by an outcome missing", () => {
    // Season 20 is completed though its standing has no outcome: a finish,
    // which beats season 26's floor.
    const old = create(LeagueRosterPlayerSchema, {
      userId: "old",
      seasons: [season(20, 1, 3), live(26, 1, 1)],
    });
    expect(peakAt(old, others(14, 26, 1))).toEqual([20, 3, false, [20]]);
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
