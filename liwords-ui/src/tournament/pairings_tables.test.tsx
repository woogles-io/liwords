import React from "react";
import { create } from "@bufbuild/protobuf";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  PlayerStandingSchema,
  RoundControlSchema,
  RoundStandingsSchema,
  TournamentGameResult,
  TournamentPersonSchema,
} from "../gen/api/proto/ipc/tournament_pb";
import { Division, SinglePairing } from "../store/reducers/tournament_reducer";
import { Pairings } from "./pairings";

const divisions = vi.hoisted(() => ({ value: {} as Record<string, Division> }));

vi.mock("../store/store", () => ({
  useTournamentStoreContext: () => ({
    tournamentContext: { divisions: divisions.value, activeGames: [] },
  }),
}));

vi.mock("../hooks/use_tournament_competitor_state", () => ({
  useTournamentCompetitorState: () => ({ currentRound: 1, activeRound: 1 }),
}));

vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));

afterEach(cleanup);

// Five players, ids "uuid:name", seeded alan to ken. Round 2 (index 1) is
// open, paired from round 1's standings: edsger, barbara, alan, donald, ken.
const person = (name: string) =>
  create(TournamentPersonSchema, { id: `${name}-id:${name}` });
const SEEDS = ["alan", "barbara", "donald", "edsger", "ken"];

// A pairing; played, its result is in (the first player won).
const pair = (p: string, q: string, played = false): SinglePairing => ({
  players: [person(p), person(q)],
  outcomes:
    p === q
      ? [TournamentGameResult.BYE, TournamentGameResult.BYE]
      : played
        ? [TournamentGameResult.WIN, TournamentGameResult.LOSS]
        : [TournamentGameResult.NO_RESULT, TournamentGameResult.NO_RESULT],
  readyStates: [],
  games: [],
});

// A round's pairings as the store keeps them, one for each player by seed.
const round = (...pairings: Array<SinglePairing>) => ({
  roundPairings: SEEDS.map(
    (name) =>
      pairings.find((p) => p.players.some((x) => x.id === person(name).id))!,
  ),
});

const division = (): Division => ({
  tournamentID: "t",
  divisionID: "d",
  players: SEEDS.map(person),
  standingsMap: {
    0: create(RoundStandingsSchema, {
      standings: ["edsger", "barbara", "alan", "donald", "ken"].map((name) =>
        create(PlayerStandingSchema, { playerId: person(name).id }),
      ),
    }),
  },
  pairings: [
    round(
      pair("alan", "donald", true),
      pair("edsger", "barbara", true),
      pair("ken", "ken"),
    ),
    round(
      pair("alan", "ken"),
      pair("donald", "barbara"),
      pair("edsger", "edsger"),
    ),
  ],
  divisionControls: undefined,
  roundControls: [0, 1].map(() => create(RoundControlSchema, {})),
  currentRound: 1,
  playerIndexMap: Object.fromEntries(
    SEEDS.map((name, idx) => [person(name).id, idx]),
  ),
  numRounds: 2,
});

const renderRound = (showTables: boolean) => {
  divisions.value = { d: division() };
  render(
    <Pairings
      selectedDivision="d"
      selectedRound={1}
      sendReady={() => true}
      isDirector={false}
      showTables={showTables}
      tentative={false}
    />,
  );
};

const header = () =>
  screen
    .getAllByRole("columnheader")
    .map((cell) => cell.textContent?.trim())
    .filter(Boolean);

// Every row's first two cells, the players' names run together.
const rows = () =>
  screen
    .getAllByRole("row")
    .slice(1)
    .map((row) =>
      within(row)
        .getAllByRole("cell")
        .slice(0, 2)
        .map((cell) => cell.textContent?.replace(/\s+/g, " ").trim()),
    );

describe("the pairings' table numbers", () => {
  it("numbers games over the board in table order, the bye after them", () => {
    renderRound(true);
    expect(header()).toEqual(["Table", "Players", "W/L", "Score"]);
    // barbara, placed second, has table 1 with donald; alan, third, then
    // table 2. edsger, the leader, has the bye.
    expect(rows()).toEqual([
      ["1", "donald barbara"],
      ["2", "alan ken"],
      ["", "edsger Bye"],
    ]);
  });

  it("leaves online pairings as they were", () => {
    renderRound(false);
    expect(header()).toEqual(["Players", "W/L", "Score"]);
    expect(rows().map(([players]) => players)).toEqual([
      "alan ken",
      "donald barbara",
      "edsger Bye",
    ]);
  });
});
