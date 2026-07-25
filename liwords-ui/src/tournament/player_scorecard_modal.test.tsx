import React from "react";
import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create, fromBinary } from "@bufbuild/protobuf";
import { describe, expect, it, vi } from "vitest";
import {
  FullTournamentDivisionsSchema,
  PlayerStandingSchema,
  RoundStandingsSchema,
  TournamentGameResult,
  TournamentPerson,
  TournamentPersonSchema,
} from "../gen/api/proto/ipc/tournament_pb";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { ActionType } from "../actions/actions";
import {
  defaultTournamentState,
  Division,
  SinglePairing,
  TournamentReducer,
} from "../store/reducers/tournament_reducer";
import { ftData } from "../store/reducers/testdata/tourney_1_divisions";
import { PlayerScorecardModal } from "./player_scorecard_modal";

// The real one is a dropdown needing the login, match and pet contexts. Its
// "View scorecard" entry becomes a button so the wiring can still be clicked.
vi.mock("../shared/usernameWithContext", () => ({
  UsernameWithContext: (props: {
    username: string;
    infoText?: string;
    handleInfoText?: () => void;
  }) => (
    <span>
      {props.username}
      {props.infoText && (
        <button
          onClick={(e) => {
            props.handleInfoText?.();
            // The real menu is portalled outside the modal and unmounts on
            // click, which drops focus onto the body. Reproduce that, since it
            // is what stops Escape reaching the dialog wrapper.
            e.currentTarget.blur();
          }}
        >
          {props.infoText} for {props.username}
        </button>
      )}
    </span>
  ),
}));

const toArr = (s: string) => {
  const bytes = new Uint8Array(Math.ceil(s.length / 2));
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(s.substring(i * 2, i * 2 + 2), 16);
  }
  return bytes;
};

const person = (id: string, suspended = false) =>
  create(TournamentPersonSchema, { id, suspended });

const standing = (
  playerId: string,
  wins: number,
  losses: number,
  spread: number,
  draws = 0,
) => create(PlayerStandingSchema, { playerId, wins, losses, spread, draws });

const ALICE = "aa:alice";
const BOB = "bb:bob";
const CAROL = "cc:carol";
const DAVE = "dd:dave";

type PairingSpec = {
  players: [string, string];
  outcomes: [TournamentGameResult, TournamentGameResult];
  scores?: [number, number];
  gameEndReason?: GameEndReason;
};

// Dave is suspended so an opponent can be marked as removed.
const people: Record<string, TournamentPerson> = {
  [ALICE]: person(ALICE),
  [BOB]: person(BOB),
  [CAROL]: person(CAROL),
  [DAVE]: person(DAVE, true),
};

const playerIndexMap: Record<string, number> = {
  [ALICE]: 0,
  [BOB]: 1,
  [CAROL]: 2,
  [DAVE]: 3,
};

// A round is written as the pairings it holds; each one is stored under both
// players' indexes, the way the reducer stores them.
const buildRound = (specs: PairingSpec[]) => {
  const roundPairings: SinglePairing[] = new Array(4)
    .fill(undefined)
    .map(() => ({
      players: [],
      outcomes: [],
      readyStates: [],
      games: [],
    }));
  for (const spec of specs) {
    const pairing = {
      players: spec.players.map((id) => people[id]),
      outcomes: spec.outcomes,
      readyStates: ["", ""],
      games: [
        {
          scores: spec.scores ?? [0, 0],
          results: spec.outcomes,
          gameEndReason: spec.gameEndReason ?? GameEndReason.NONE,
          id: "",
        },
      ],
    };
    for (const id of spec.players) {
      roundPairings[playerIndexMap[id]] = pairing;
    }
  }
  return { roundPairings };
};

const { WIN, LOSS, DRAW, FORFEIT_WIN, FORFEIT_LOSS, NO_RESULT, VOID } =
  TournamentGameResult;

const madeUpDivision = (): Division =>
  ({
    tournamentID: "t",
    divisionID: "A",
    players: Object.values(people),
    playerIndexMap,
    numRounds: 4,
    currentRound: 3,
    divisionControls: undefined,
    roundControls: [],
    pairings: [
      buildRound([
        { players: [ALICE, BOB], outcomes: [WIN, LOSS], scores: [450, 400] },
        { players: [CAROL, DAVE], outcomes: [WIN, LOSS], scores: [500, 300] },
      ]),
      buildRound([
        {
          players: [CAROL, ALICE],
          outcomes: [WIN, LOSS],
          scores: [420, 380],
          gameEndReason: GameEndReason.RESIGNED,
        },
        { players: [BOB, DAVE], outcomes: [DRAW, DRAW], scores: [400, 400] },
      ]),
      buildRound([
        { players: [ALICE, DAVE], outcomes: [FORFEIT_WIN, FORFEIT_LOSS] },
        { players: [BOB, CAROL], outcomes: [NO_RESULT, NO_RESULT] },
      ]),
      buildRound([
        { players: [ALICE, BOB], outcomes: [VOID, VOID] },
        { players: [CAROL, DAVE], outcomes: [WIN, LOSS], scores: [410, 390] },
      ]),
    ],
    standingsMap: {
      0: create(RoundStandingsSchema, {
        standings: [
          standing(CAROL, 1, 0, 200),
          standing(ALICE, 1, 0, 50),
          standing(BOB, 0, 1, -50),
          standing(DAVE, 0, 1, -200),
        ],
      }),
      // Bob and Dave drew in round 2, so they carry a draw from there on.
      1: create(RoundStandingsSchema, {
        standings: [
          standing(CAROL, 2, 0, 240),
          standing(ALICE, 1, 1, 10),
          standing(BOB, 0, 1, -50, 1),
          standing(DAVE, 0, 1, -200, 1),
        ],
      }),
      2: create(RoundStandingsSchema, {
        standings: [
          standing(CAROL, 2, 0, 240),
          standing(ALICE, 2, 1, 10),
          standing(BOB, 0, 1, -50, 1),
          standing(DAVE, 0, 2, -200, 1),
        ],
      }),
      3: create(RoundStandingsSchema, {
        standings: [
          standing(CAROL, 3, 0, 260),
          standing(ALICE, 2, 1, 10),
          standing(BOB, 0, 1, -50, 1),
          standing(DAVE, 0, 3, -220, 1),
        ],
      }),
    },
  }) as unknown as Division;

const showScorecard = (
  division: Division,
  playerId: string,
  throughRound: number,
  irlMode = true,
  onSelectPlayer: (id: string) => void = () => {},
) =>
  render(
    <PlayerScorecardModal
      division={division}
      playerId={playerId}
      throughRound={throughRound}
      irlMode={irlMode}
      onClose={() => {}}
      onSelectPlayer={onSelectPlayer}
    />,
  );

// A real Escape press lands on whatever holds focus.
const pressEscape = () =>
  fireEvent.keyDown(document.activeElement ?? document.body, {
    key: "Escape",
    keyCode: 27,
  });

// A cell's text without the stand-in scorecard button.
const cellText = (cell: HTMLElement) => {
  const clone = cell.cloneNode(true) as HTMLElement;
  clone.querySelectorAll("button").forEach((button) => button.remove());
  return clone.textContent?.trim();
};

// Every row as [round, opponent, result, score, W, L, spread, rank].
const rows = () =>
  within(screen.getByRole("dialog"))
    .getAllByRole("row")
    .slice(1)
    .map((row) => within(row).getAllByRole("cell").map(cellText));

describe("the player scorecard", () => {
  it("lays out one row per round, through the round it was opened at", () => {
    showScorecard(madeUpDivision(), ALICE, 3);
    expect(rows()).toEqual([
      ["1 1st", "bob", "Win", "450-400 (+50)", "1", "0", "+50", "2"],
      ["2", "carol", "Loss (resigned)", "380-420 (-40)", "1", "1", "+10", "2"],
      ["3", "dave Removed", "Forfeit win", "", "2", "1", "+10", "2"],
      ["4", "bob", "Not playing", "", "2", "1", "+10", "2"],
    ]);
  });

  it("stops at the round it was opened at", () => {
    showScorecard(madeUpDivision(), ALICE, 1);
    expect(rows()).toHaveLength(2);
  });

  it("shows a draw, and says nothing about a game still being played", () => {
    showScorecard(madeUpDivision(), BOB, 2);
    // The record folds a draw in as half a win and half a loss, which is what
    // the standings table and the Games tab behind this both do. League writes
    // the same record as 0-1-1 instead.
    expect(rows()).toEqual([
      ["1", "alice", "Loss", "400-450 (-50)", "0", "1", "-50", "3"],
      [
        "2 1st",
        "dave Removed",
        "Draw",
        "400-400 (+0)",
        "0.5",
        "1.5",
        "-50",
        "3",
      ],
      ["3 1st", "carol", "", "", "0.5", "1.5", "-50", "3"],
    ]);
  });

  it("averages only the rounds that produced a score", () => {
    showScorecard(madeUpDivision(), ALICE, 3);
    // Rounds 1 and 2: 450 and 380 against 400 and 420. The forfeit and the
    // round nobody played stay out of it.
    expect(screen.getByRole("dialog")).toHaveTextContent(
      "Average score 415.00, average opponent score 410.00, over 2 games.",
    );
  });

  it("hands an opponent over so their scorecard can replace this one", async () => {
    const selected: string[] = [];
    showScorecard(madeUpDivision(), ALICE, 1, true, (id) => selected.push(id));
    await userEvent.click(screen.getByText("View scorecard for carol"));
    expect(selected).toEqual([CAROL]);
  });

  // jsdom never runs the motion callback rc-dialog focuses from, so without a
  // focus target of our own nothing inside the modal holds focus here, even on
  // first open. A browser does focus it -- this pins that the wrapper serves
  // both paths.
  it("closes on escape", () => {
    const closed: string[] = [];
    render(
      <PlayerScorecardModal
        division={madeUpDivision()}
        playerId={ALICE}
        throughRound={1}
        irlMode
        onClose={() => closed.push(ALICE)}
        onSelectPlayer={() => {}}
      />,
    );
    pressEscape();
    expect(closed).toEqual([ALICE]);
  });

  it("still closes on escape after following an opponent", async () => {
    // Switching players re-renders this modal rather than reopening it, so
    // rc-dialog never re-runs its focus-on-open, and the click that switched
    // came from a menu outside the modal. Escape is bound to the dialog
    // wrapper, so it only works while focus is still inside.
    const closed: string[] = [];
    const Harness = () => {
      const [playerId, setPlayerId] = React.useState(ALICE);
      return (
        <PlayerScorecardModal
          division={madeUpDivision()}
          playerId={playerId}
          throughRound={1}
          irlMode
          onClose={() => closed.push(playerId)}
          onSelectPlayer={setPlayerId}
        />
      );
    };
    render(<Harness />);

    await userEvent.click(screen.getByText("View scorecard for carol"));
    expect(screen.getByRole("dialog")).toHaveTextContent("carol");

    // Escape has to be fired at whatever holds focus, since rc-dialog listens
    // on the dialog wrapper and a keydown from the body never reaches it.
    // user-event omits the deprecated keyCode that rc-dialog checks, so this
    // goes through fireEvent.
    pressEscape();
    expect(closed).toEqual([CAROL]);
  });

  it("hides the first-move tag outside in-real-life events", () => {
    showScorecard(madeUpDivision(), ALICE, 0, false);
    expect(rows()[0][0]).toBe("1");
  });
});

describe("the player scorecard on real division data", () => {
  // The NWL division of this fixture is three players over eight rounds, and
  // every result in it is a bye.
  const nwl = (): Division => {
    const state = TournamentReducer(defaultTournamentState, {
      actionType: ActionType.SetDivisionsData,
      payload: {
        fullDivisions: fromBinary(FullTournamentDivisionsSchema, toArr(ftData)),
        loginState: {
          username: "josh",
          userID: "dcHy8DxjX3dmsCAaEMR5AH",
          loggedIn: true,
          connId: "conn-123",
          connectedToSocket: true,
        },
      },
    });
    return state.divisions["NWL"];
  };

  it("gives a bye no opponent and no score, but still moves the spread", () => {
    showScorecard(nwl(), "dcHy8DxjX3dmsCAaEMR5AH:josh", 2);
    expect(rows()).toEqual([
      ["1", "", "Bye", "", "1", "0", "+50", "1"],
      // Round 2 leaves josh and lola both on 1-0 with the same spread.
      ["2 1st", "thedirector", "", "", "1", "0", "+50", "=1"],
      ["3", "", "Bye", "", "2", "0", "+100", "1"],
    ]);
  });

  it("ends on the record and rank the standings row it came from shows", () => {
    showScorecard(nwl(), "FFGUXhQd3B8F6EXJUvmn5a:thedirector", 7);
    const last = rows()[7];
    expect(last.slice(4)).toEqual(["2", "0", "+100", "2"]);
  });

  it("leaves the averages off when only byes have landed", () => {
    showScorecard(nwl(), "dcHy8DxjX3dmsCAaEMR5AH:josh", 0);
    expect(screen.queryByText(/Average score/)).toBeNull();
  });
});
