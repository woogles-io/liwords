import React from "react";
import { create } from "@bufbuild/protobuf";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { StandingResult } from "../gen/api/proto/ipc/league_pb";
import {
  LeagueRosterPlayerSchema,
  LeagueRosterResponseSchema,
  LeagueRosterSeasonSchema,
} from "../gen/api/proto/league_service/league_service_pb";
import { LeagueRoster } from "./league_roster";

// What each query answers, by its method's name.
const answers = vi.hoisted(() => ({ value: {} as Record<string, unknown> }));

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: (method: { name: string }) => ({
    data: answers.value[method.name],
    isLoading: false,
  }),
}));

// The real one is a dropdown that needs the login, match, and pet contexts.
vi.mock("../shared/usernameWithContext", () => ({
  UsernameWithContext: (props: { username: string }) => (
    <span>{props.username}</span>
  ),
}));

vi.mock("./live_updates", () => ({ setRosterOnScreen: () => {} }));

afterEach(cleanup);

const finished = (
  seasonNumber: number,
  divisionNumber: number,
  rank: number,
  wins: number,
  result = StandingResult.RESULT_STAYED,
) =>
  create(LeagueRosterSeasonSchema, {
    seasonNumber,
    divisionNumber,
    rank,
    wins,
    result,
  });

// Season 26 has its divisions but no games: the server's order is
// alphabetical.
const notStarted = (rank: number) =>
  create(LeagueRosterSeasonSchema, {
    seasonNumber: 26,
    divisionNumber: 1,
    rank,
  });

const player = (
  username: string,
  seasons: Array<ReturnType<typeof finished>>,
) => create(LeagueRosterPlayerSchema, { userId: username, username, seasons });

const renderRoster = (calWins = 2) => {
  answers.value = {
    GetLeagueRoster: create(LeagueRosterResponseSchema, {
      seasonNumbers: [25, 26],
      players: [
        player("andy", [finished(25, 1, 2, 3), notStarted(1)]),
        player("bea", [
          finished(25, 2, 1, 5, StandingResult.RESULT_PROMOTED),
          notStarted(2),
        ]),
        player("cal", [finished(25, 1, 3, calWins), notStarted(3)]),
        player("dee", [finished(25, 1, 1, 4, StandingResult.RESULT_CHAMPION)]),
      ],
    }),
  };
  render(
    <LeagueRoster
      leagueId="league"
      activeSeasonNumber={26}
      activeSeasonId="s26"
      seasonsInProgress={[26]}
      shown
      onJumpToSeason={() => {}}
    />,
  );
};

// A row's Peak, S26, and S25 cells, as text, and whether the Peak's rank is in
// italics.
const cells = (username: string) => {
  const row = screen.getByText(username).closest("tr")!;
  const [, , peak, s26, s25] = within(row).getAllByRole("cell");
  const rank = peak.querySelector<HTMLElement>(".roster-rank");
  return {
    peak: peak.textContent,
    italic: rank?.style.fontStyle === "italic",
    s26: s26.textContent,
    s25: s25.textContent,
  };
};

describe("the roster with a season not started", () => {
  it("gives no rank in its column, the order being alphabetical", () => {
    renderRoster();
    for (const username of ["andy", "bea", "cal"]) {
      expect(cells(username).s26).toBe("D1");
    }
    expect(cells("andy").s25).toBe("D1#2");
  });

  it("puts nobody first in Peak for being first alphabetically", () => {
    renderRoster();
    // Last of three is the worst andy can finish: his D1 #2 stands.
    expect(cells("andy")).toMatchObject({
      peak: "D1#2\u00b7S25",
      italic: false,
    });
  });

  it("counts a new division at its last place, in italics", () => {
    renderRoster();
    expect(cells("bea")).toMatchObject({ peak: "D1#3\u00b7S26", italic: true });
  });

  it("lets an equal finish stand, with no repeat", () => {
    renderRoster();
    expect(cells("cal")).toMatchObject({
      peak: "D1#3\u00b7S25",
      italic: false,
    });
  });
});

describe("a shared finish in the roster", () => {
  it("is marked, in its season and in Peak", () => {
    // cal draws level with andy on wins and spread: both second.
    renderRoster(3);
    for (const username of ["andy", "cal"]) {
      expect(cells(username)).toMatchObject({
        s25: "D1#2=",
        peak: "D1#2=\u00b7S25",
      });
    }
    expect(cells("dee").s25).toBe("D1#1");
  });
});
