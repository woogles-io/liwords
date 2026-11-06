import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  PlayerStandingSchema,
  RoundControlSchema,
  RoundStandingsSchema,
  TournamentGameResult,
  TournamentPersonSchema,
} from "../gen/api/proto/ipc/tournament_pb";
import { Division, SinglePairing } from "../store/reducers/tournament_reducer";
import { tableNumbers } from "./tables";

// A player's pairing in a round that only has to have its results in, or not.
const entered = (id: string, result: TournamentGameResult): SinglePairing => ({
  players: [id, id].map((x) => create(TournamentPersonSchema, { id: x })),
  outcomes: [result, result],
  readyStates: [],
  games: [],
});

// Five players, seeded a to e. standings gives, by round, the order after it;
// the rounds through complete have every result in.
const SEEDS = ["a", "b", "c", "d", "e"];
const division = (
  standings: Record<number, Array<string>>,
  lags: Array<number> = [0, 0, 0],
  complete = Object.keys(standings).length - 1,
): Division => ({
  tournamentID: "t",
  divisionID: "d",
  players: SEEDS.map((p) => create(TournamentPersonSchema, { id: p })),
  standingsMap: Object.fromEntries(
    Object.entries(standings).map(([round, order]) => [
      round,
      create(RoundStandingsSchema, {
        standings: order.map((playerId) =>
          create(PlayerStandingSchema, { playerId }),
        ),
      }),
    ]),
  ),
  pairings: lags.map((_, round) => ({
    roundPairings: SEEDS.map((p) =>
      entered(
        p,
        round <= complete
          ? TournamentGameResult.BYE
          : TournamentGameResult.NO_RESULT,
      ),
    ),
  })),
  divisionControls: undefined,
  roundControls: lags.map((pairingLag) =>
    create(RoundControlSchema, { pairingLag }),
  ),
  currentRound: 2,
  playerIndexMap: Object.fromEntries(SEEDS.map((p, idx) => [p, idx])),
  numRounds: lags.length,
});

const pair = (p: string, q: string): SinglePairing => ({
  players: [p, q].map((id) => create(TournamentPersonSchema, { id })),
  outcomes:
    p === q
      ? [TournamentGameResult.BYE, TournamentGameResult.BYE]
      : [TournamentGameResult.NO_RESULT, TournamentGameResult.NO_RESULT],
  readyStates: [],
  games: [],
});

// Each pairing as "p-q", by its table, then those with none.
const seating = (
  d: Division,
  round: number,
  pairings: Array<SinglePairing>,
) => {
  const tables = tableNumbers(d, round, pairings)!;
  const name = (p: SinglePairing) => p.players.map((x) => x.id).join("-");
  return pairings
    .filter((p) => tables.has(p))
    .sort((p, q) => tables.get(p)! - tables.get(q)!)
    .map((p) => `${tables.get(p)}:${name(p)}`)
    .concat(pairings.filter((p) => !tables.has(p)).map(name));
};

describe("table numbers", () => {
  it("seats the first round by seed", () => {
    expect(
      seating(division({}), 0, [
        pair("b", "d"),
        pair("e", "e"),
        pair("c", "a"),
      ]),
    ).toEqual(["1:c-a", "2:b-d", "e-e"]);
  });

  it("seats a later round by the standings it was paired from", () => {
    // After round 1: d, e, a, c, b. Table 1 is the leader's game; e, second,
    // sits there with d, so table 2 is the game of a, third.
    const d = division({ 0: ["d", "e", "a", "c", "b"] });
    expect(
      seating(d, 1, [pair("c", "a"), pair("e", "d"), pair("b", "b")]),
    ).toEqual(["1:e-d", "2:c-a", "b-b"]);
  });

  it("keeps the round's own results out of it", () => {
    // Round 2's standings, half entered, put b on top: the tables stay.
    const d = division({
      0: ["d", "e", "a", "c", "b"],
      1: ["b", "c", "a", "d", "e"],
    });
    expect(
      seating(d, 1, [pair("c", "a"), pair("e", "d"), pair("b", "b")]),
    ).toEqual(["1:e-d", "2:c-a", "b-b"]);
  });

  it("seats a lagged round by the standings two rounds back", () => {
    const d = division(
      { 0: ["d", "a", "e", "c", "b"], 1: ["b", "c", "a", "d", "e"] },
      [0, 0, 1],
    );
    expect(
      seating(d, 2, [pair("b", "a"), pair("c", "d"), pair("e", "e")]),
    ).toEqual(["1:c-d", "2:b-a", "e-e"]);
  });

  it("places anyone the standings leave out after everyone in them", () => {
    // e joined after round 1.
    const d = division({ 0: ["d", "c", "b", "a"] });
    expect(
      seating(d, 1, [pair("e", "a"), pair("b", "c"), pair("d", "d")]),
    ).toEqual(["1:b-c", "2:e-a", "d-d"]);
  });

  it("numbers nothing until the standings it was paired from are final", () => {
    // Round 1 still has results to enter; round 2 is already paired.
    const d = division({ 0: ["d", "e", "a", "c", "b"] }, [0, 0, 0], -1);
    expect(
      tableNumbers(d, 1, [pair("c", "a"), pair("e", "d"), pair("b", "b")]),
    ).toBeUndefined();
    // The first round goes by seed, which is final from the start.
    expect(
      seating(d, 0, [pair("b", "d"), pair("e", "e"), pair("c", "a")]),
    ).toEqual(["1:c-a", "2:b-d", "e-e"]);
  });
});
