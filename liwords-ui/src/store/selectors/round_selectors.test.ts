import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  PairingMethod,
  RoundControlSchema,
  TournamentGameResult,
  TournamentPersonSchema,
} from "../../gen/api/proto/ipc/tournament_pb";
import {
  Division,
  SinglePairing,
  TournamentState,
  defaultTournamentState,
} from "../reducers/tournament_reducer";
import {
  hasLaggedRounds,
  isCalledToRound,
  isRoundComplete,
  openRounds,
  pairingBasisRound,
  playerActiveRound,
  roundsCompleteThrough,
} from "./round_selectors";
import { TourneyStatus, getCompetitorState } from "./tournament_selectors";

// Four players, three rounds. Round 1 pairs alice v bob and carol v dave,
// round 2 alice v carol and bob v dave, round 3 alice v dave and bob v carol.
const ids = ["a1:alice", "b2:bob", "c3:carol", "d4:dave"];
const schedule = [
  [
    [0, 1],
    [2, 3],
  ],
  [
    [0, 2],
    [1, 3],
  ],
  [
    [0, 3],
    [1, 2],
  ],
];

// results[round] lists the games (by index into that round's schedule) that
// have a result.
const makeDivision = (
  currentRound: number,
  lags: number[],
  results: number[][],
): Division => {
  const players = ids.map((id) => create(TournamentPersonSchema, { id }));
  const playerIndexMap: { [id: string]: number } = {};
  ids.forEach((id, idx) => (playerIndexMap[id] = idx));
  const pairings = schedule.map((games, round) => {
    const roundPairings = new Array<SinglePairing>(ids.length);
    games.forEach(([p1, p2], gameIdx) => {
      const done = results[round]?.includes(gameIdx);
      const pairing: SinglePairing = {
        players: [players[p1], players[p2]],
        outcomes: done
          ? [TournamentGameResult.WIN, TournamentGameResult.LOSS]
          : [TournamentGameResult.NO_RESULT, TournamentGameResult.NO_RESULT],
        readyStates: ["", ""],
        games: [
          {
            scores: done ? [400, 300] : [0, 0],
            results: [],
            gameEndReason: done ? 1 : 0,
            id: `game-${round}-${gameIdx}`,
          },
        ],
      };
      roundPairings[p1] = pairing;
      roundPairings[p2] = pairing;
    });
    return { roundPairings };
  });
  return {
    tournamentID: "t",
    divisionID: "D",
    players,
    standingsMap: {},
    pairings,
    divisionControls: undefined,
    roundControls: lags.map((pairingLag, round) =>
      create(RoundControlSchema, {
        round,
        pairingMethod: PairingMethod.SWISS,
        pairingLag,
      }),
    ),
    currentRound,
    playerIndexMap,
    numRounds: schedule.length,
  };
};

const stateWith = (division: Division): TournamentState => ({
  ...defaultTournamentState,
  started: true,
  divisions: { D: division },
  initializedFromXHR: true,
});

const loginAs = (fullID: string) => {
  const [userID, username] = fullID.split(":");
  return {
    username,
    userID,
    loggedIn: true,
    connID: "conn",
    connectedToSocket: true,
    path: "/",
    permissions: [],
  };
};

describe("rounds with no lag", () => {
  it("only ever has the current round open", () => {
    // Round 2 is open while one round 1 game is still unfinished, which the
    // server never allows without lag, but the selectors must not care.
    const div = makeDivision(1, [0, 0, 0], [[0], []]);
    expect(hasLaggedRounds(div)).toBe(false);
    expect(openRounds(div)).toEqual([1]);
    ids.forEach((id) => expect(playerActiveRound(div, id)).toBe(1));
  });

  it("uses the previous round as the basis", () => {
    const div = makeDivision(1, [0, 0, 0], []);
    expect(pairingBasisRound(div, 2)).toBe(1);
    // A lag other than the round's own, as when it is about to change.
    expect(pairingBasisRound(div, 2, 1)).toBe(0);
  });

  it("has nothing open before the tournament starts", () => {
    const div = makeDivision(-1, [0, 0, 0], []);
    expect(openRounds(div)).toEqual([]);
    expect(playerActiveRound(div, ids[0])).toBe(-1);
  });
});

describe("a lagged round", () => {
  it("runs alongside an unfinished previous round", () => {
    const div = makeDivision(1, [0, 1, 0], [[0], []]);
    expect(hasLaggedRounds(div)).toBe(true);
    expect(pairingBasisRound(div, 1)).toBe(-1);
    expect(isRoundComplete(div, 0)).toBe(false);
    expect(openRounds(div)).toEqual([0, 1]);
  });

  it("closes the previous round once it is complete", () => {
    const div = makeDivision(1, [0, 1, 0], [[0, 1], [0]]);
    expect(isRoundComplete(div, 0)).toBe(true);
    expect(roundsCompleteThrough(div, 0)).toBe(true);
    expect(roundsCompleteThrough(div, 1)).toBe(false);
    expect(openRounds(div)).toEqual([1]);
  });

  it("keeps each player on their earliest unfinished game", () => {
    // alice and bob are done with round 1; carol and dave are not.
    const div = makeDivision(1, [0, 1, 0], [[0], []]);
    expect(playerActiveRound(div, "a1:alice")).toBe(1);
    expect(playerActiveRound(div, "b2:bob")).toBe(1);
    expect(playerActiveRound(div, "c3:carol")).toBe(0);
    expect(playerActiveRound(div, "d4:dave")).toBe(0);
  });

  it("falls back to the current round once a player is done", () => {
    const div = makeDivision(1, [0, 1, 0], [[0], [0]]);
    expect(playerActiveRound(div, "a1:alice")).toBe(1);
  });
});

describe("calling players to a round that just opened", () => {
  it("calls everyone who is not in a game without lag", () => {
    const div = makeDivision(0, [0, 0, 0], [[0, 1]]);
    expect(isCalledToRound(div, 1, "a1:alice", false)).toBe(true);
    expect(isCalledToRound(div, 1, "a1:alice", true)).toBe(false);
  });

  it("only calls players who finished the round before a lagged round", () => {
    // alice and bob are done with round 1; carol and dave are not.
    const div = makeDivision(0, [0, 1, 0], [[0]]);
    expect(isCalledToRound(div, 1, "a1:alice", false)).toBe(true);
    expect(isCalledToRound(div, 1, "c3:carol", false)).toBe(false);
  });
});

describe("the competitor state in a lagged round", () => {
  it("is still on the previous round for a player who is playing it", () => {
    const div = makeDivision(1, [0, 1, 0], [[0], []]);
    const cs = getCompetitorState(stateWith(div), loginAs("c3:carol"));
    expect(cs.currentRound).toBe(1);
    expect(cs.activeRound).toBe(0);
    expect(cs.status).toBe(TourneyStatus.ROUND_OPEN);
  });

  it("moves on for a player who finished the previous round", () => {
    const div = makeDivision(1, [0, 1, 0], [[0], []]);
    const cs = getCompetitorState(stateWith(div), loginAs("a1:alice"));
    expect(cs.activeRound).toBe(1);
    expect(cs.status).toBe(TourneyStatus.ROUND_OPEN);
  });

  it("only counts an active game in the player's active round", () => {
    const div = makeDivision(1, [0, 1, 0], [[0], []]);
    const state = stateWith(div);
    const gameIn = (round: number) => ({
      ...state,
      activeGames: [
        {
          players: [
            { displayName: "carol", rating: "", uuid: "c3" },
            { displayName: "dave", rating: "", uuid: "d4" },
          ],
          tournamentRound: round,
        } as unknown as TournamentState["activeGames"][number],
      ],
    });
    expect(getCompetitorState(gameIn(0), loginAs("c3:carol")).status).toBe(
      TourneyStatus.ROUND_GAME_ACTIVE,
    );
    expect(getCompetitorState(gameIn(1), loginAs("c3:carol")).status).toBe(
      TourneyStatus.ROUND_OPEN,
    );
  });
});
