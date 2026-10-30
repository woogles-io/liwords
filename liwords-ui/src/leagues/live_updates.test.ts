import { QueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  leagueGameQueryFilters,
  scheduleLeagueRefresh,
  setRosterOnScreen,
} from "./live_updates";

// A connect-query key, as connect-query writes one.
const key = (methodName: string, input: Record<string, unknown>) => [
  "connect-query",
  {
    serviceName: "league_service.LeagueService",
    methodName,
    input,
    cardinality: "finite",
  },
];

const seeded = () => {
  const qc = new QueryClient();
  const keys = {
    standings: key("GetAllDivisionStandings", { seasonId: "s1" }),
    otherSeason: key("GetAllDivisionStandings", { seasonId: "s0" }),
    crosstable: key("GetDivisionCrosstable", { divisionId: "d1" }),
    otherDivision: key("GetDivisionCrosstable", { divisionId: "d2" }),
    banks: key("GetDivisionTimeBankWarnings", {
      divisionId: "d1",
      thresholdHours: 24,
    }),
    aliceGames: key("GetPlayerSeasonGames", {
      userId: "alice",
      seasonId: "s1",
    }),
    carolGames: key("GetPlayerSeasonGames", {
      userId: "carol",
      seasonId: "s1",
    }),
    roster: key("GetLeagueRoster", { leagueId: "smoke" }),
    // the roster's H2H shows whichever player was picked, not only the viewer
    bobH2H: key("GetPlayerLeagueH2H", { userId: "bob", leagueId: "smoke" }),
    carolH2H: key("GetPlayerLeagueH2H", { userId: "carol", leagueId: "smoke" }),
  };
  for (const k of Object.values(keys)) {
    qc.setQueryData(k, {});
  }
  const stale = () =>
    Object.fromEntries(
      Object.entries(keys).map(([name, k]) => [
        name,
        qc.getQueryState(k)?.isInvalidated ?? false,
      ]),
    );
  return { qc, stale };
};

const ev = (ended: boolean) => ({
  seasonId: "s1",
  divisionId: "d1",
  playerIds: ["alice", "bob"],
  ended,
});

describe("leagueGameQueryFilters", () => {
  it("makes a move's division and players' games stale, not the standings", async () => {
    const { qc, stale } = seeded();
    for (const f of leagueGameQueryFilters(ev(false))) {
      await qc.invalidateQueries(f);
    }
    expect(stale()).toEqual({
      standings: false,
      otherSeason: false,
      crosstable: true,
      otherDivision: false,
      banks: true,
      aliceGames: true,
      carolGames: false,
      roster: false,
      bobH2H: false,
      carolH2H: false,
    });
  });

  it("adds the season's standings, the roster and the players' records at the end", async () => {
    const { qc, stale } = seeded();
    for (const f of leagueGameQueryFilters(ev(true), true)) {
      await qc.invalidateQueries(f);
    }
    expect(stale()).toMatchObject({
      standings: true,
      otherSeason: false,
      otherDivision: false,
      carolGames: false,
      roster: true,
      bobH2H: true,
      carolH2H: false,
    });
  });

  it("leaves the roster alone while it is not on screen", async () => {
    const { qc, stale } = seeded();
    for (const f of leagueGameQueryFilters(ev(true), false)) {
      await qc.invalidateQueries(f);
    }
    expect(stale()).toMatchObject({ standings: true, roster: false });
  });

  it("knows whether a roster is on screen", () => {
    const has = () =>
      leagueGameQueryFilters(ev(true)).some((f) =>
        JSON.stringify(f.queryKey).includes("GetLeagueRoster"),
      );
    expect(has()).toBe(false);
    setRosterOnScreen(true);
    expect(has()).toBe(true);
    setRosterOnScreen(false);
    expect(has()).toBe(false);
  });
});

describe("scheduleLeagueRefresh", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("refreshes once, after the wait, however many games report", () => {
    vi.useFakeTimers();
    const qc = new QueryClient();
    const spy = vi.spyOn(qc, "invalidateQueries");
    const filters = leagueGameQueryFilters(ev(true));
    scheduleLeagueRefresh(qc, filters);
    scheduleLeagueRefresh(qc, filters);
    expect(spy).not.toHaveBeenCalled();
    vi.advanceTimersByTime(500);
    expect(spy).toHaveBeenCalledTimes(filters.length);
  });
});
