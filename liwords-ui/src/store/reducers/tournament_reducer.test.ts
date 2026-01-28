import { ActionType } from "../../actions/actions";
import {
  defaultTournamentState,
  TournamentReducer,
} from "./tournament_reducer";
import { ftData } from "./testdata/tourney_1_divisions";
import { ChallengeRule } from "../../gen/api/proto/vendored/macondo/macondo_pb";
import {
  TournamentMetadataSchema,
  TType,
} from "../../gen/api/proto/tournament_service/tournament_service_pb";
import {
  TournamentGameResult,
  FirstMethod,
  FullTournamentDivisionsSchema,
  TournamentRoundStartedSchema,
  TournamentDivisionDataResponseSchema,
  PlayersAddedOrRemovedResponseSchema,
  TournamentPersonsSchema,
  TournamentPersonSchema,
  DivisionControlsSchema,
  DivisionControlsResponseSchema,
  DivisionRoundControlsSchema,
  RoundControlSchema,
  TournamentGameSchema,
  PairingSchema,
} from "../../gen/api/proto/ipc/tournament_pb";
import { create, fromBinary } from "@bufbuild/protobuf";
import {
  GameRequestSchema,
  GameRulesSchema,
} from "../../gen/api/proto/ipc/omgwords_pb";
import { getCompetitorState } from "../selectors/tournament_selectors";

const toArr = (s: string) => {
  const bytes = new Uint8Array(Math.ceil(s.length / 2));
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(s.substring(i * 2, i * 2 + 2), 16);
  }
  return bytes;
};

// This is a fairly complex tourney
const initialTourneyXHRMessage = () => {
  const msg = toArr(ftData);
  return fromBinary(FullTournamentDivisionsSchema, msg);
};

const tourneyMetadataPayload = () => {
  const metadata = create(TournamentMetadataSchema, {
    name: "Wolges Incorporated",
    description: "Welcome to Wolges: population: You",
    slug: "/tournament/wolges",
    id: "qzqWHsGVBrAgiuAZp9nJJm",
    type: TType.STANDARD,
  });

  return {
    directors: ["cesar", "thedirector"],
    metadata,
  };
};

const startTourneyMessage = () => {
  const msg = create(TournamentRoundStartedSchema, {
    tournamentId: "qzqWHsGVBrAgiuAZp9nJJm",
    division: "CSW",
  });

  return msg;
};

const cesarLoginState = () => {
  return {
    username: "cesar",
    userID: "ncSw3WeNGMzATfwzz7pdkF",
    loggedIn: true,
    connId: "conn-123",
    connectedToSocket: true,
  };
};

const fullDivisionsState = () => {
  const state = defaultTournamentState;

  const state1 = TournamentReducer(state, {
    actionType: ActionType.SetTourneyMetadata,
    payload: tourneyMetadataPayload(),
  });

  const state2 = TournamentReducer(state1, {
    actionType: ActionType.SetDivisionsData,
    payload: {
      fullDivisions: initialTourneyXHRMessage(),
      loginState: {
        username: "foo",
        userID: "foo123",
        loggedIn: true,
        connId: "conn-123",
        connectedToSocket: true,
      },
    },
  });

  return state2;
};

it("tests initial fulldivisions message", () => {
  const state = defaultTournamentState;
  const loginState = {
    username: "mina",
    userID: "MoczSz5dksZuKMnxcH6yVT",
    loggedIn: true,
    connID: "conn-123",
    path: "/foo",
    perms: [],
    connectedToSocket: true,
  };
  const newState = TournamentReducer(state, {
    actionType: ActionType.SetDivisionsData,
    payload: {
      fullDivisions: initialTourneyXHRMessage(),
      loginState,
    },
  });
  const competitorState = getCompetitorState(newState, loginState);

  expect(newState.started).toBe(false);
  expect(newState.divisions["CSW"]).toBeTruthy();
  expect(newState.divisions["NWL"]).toBeTruthy();
  expect(newState.divisions["TWL"]).toBeFalsy();
  expect(competitorState).toEqual({
    isRegistered: true,
    isCheckedIn: false,
    currentRound: -1,
    division: "CSW",
    status: "PRETOURNEY",
  });
  expect(newState.initializedFromXHR).toBe(true);
});

it("tests tourneystart", () => {
  const state = fullDivisionsState();

  const finalState = TournamentReducer(state, {
    actionType: ActionType.StartTourneyRound,
    payload: {
      trs: startTourneyMessage(),
      loginState: cesarLoginState(),
    },
  });

  expect(finalState.started).toBe(true);
});

const newDivisionMessage = () => {
  const msg = create(TournamentDivisionDataResponseSchema, {
    id: "qzqWHsGVBrAgiuAZp9nJJm",
    division: "NWL B",
    currentRound: -1,
  });
  return msg;
};

const newPlayersMessage = () => {
  const msg = create(PlayersAddedOrRemovedResponseSchema, {
    id: "qzqWHsGVBrAgiuAZp9nJJm",
    division: "NWL B",
    players: create(TournamentPersonsSchema, {
      persons: [
        create(TournamentPersonSchema, {
          id: "ViSLeuyqNcSA3GcHJP5rA5:nigel",
          rating: 2344,
        }),
        create(TournamentPersonSchema, {
          id: "JkW7MXvVPfj7HdgAwLQzJ4:will",
          rating: 1234,
        }),
      ],
    }),
  });

  return msg;
};

const newTournamentControlsMessage = () => {
  const rules = create(GameRulesSchema, {
    boardLayoutName: "CrosswordGame",
    letterDistributionName: "English",
  });
  const gameReq = create(GameRequestSchema, {
    lexicon: "NWL20",
    rules: rules,
    initialTimeSeconds: 180,
    challengeRule: ChallengeRule.DOUBLE,
  });

  const divControls = create(DivisionControlsSchema, {
    id: "qzqWHsGVBrAgiuAZp9nJJm",
    division: "NWL B",
    gameRequest: gameReq,
    suspendedResult: TournamentGameResult.BYE,
  });

  const msg = create(DivisionControlsResponseSchema, {
    division: "NWL B",
    id: "qzqWHsGVBrAgiuAZp9nJJm",
    divisionControls: divControls,
  });

  return msg;
};

const newDivisionRoundControlsMessage = () => {
  const msg = create(DivisionRoundControlsSchema, {
    id: "qzqWHsGVBrAgiuAZp9nJJm",
    division: "NWL B",
  });
  const rcl = create(RoundControlSchema, {
    firstMethod: FirstMethod.AUTOMATIC_FIRST,
    gamesPerRound: 1,
  });

  const game = create(TournamentGameSchema, {
    scores: [0, 0],
    results: [0, 0],
  });
  const pairing = create(PairingSchema, {
    players: [1, 0],
    round: 0,
    games: [game],
    outcomes: [0, 0],
    readyStates: ["", ""],
  });

  msg.roundControls = [rcl];
  msg.divisionPairings = [pairing];
  return msg;
};

it("adds new divisions and pairings", () => {
  const state = fullDivisionsState();

  const loginState = cesarLoginState();

  // Add a new division, add two players, add random pairings.

  const state1 = TournamentReducer(state, {
    actionType: ActionType.SetDivisionData,
    payload: {
      divisionMessage: newDivisionMessage(),
      loginState,
    },
  });

  const state2 = TournamentReducer(state1, {
    actionType: ActionType.SetDivisionPlayers,
    payload: {
      parr: newPlayersMessage(),
      loginState,
    },
  });

  const state3 = TournamentReducer(state2, {
    actionType: ActionType.SetDivisionControls,
    payload: {
      divisionControlsResponse: newTournamentControlsMessage(),
      loginState,
    },
  });

  const finalState = TournamentReducer(state3, {
    actionType: ActionType.SetDivisionRoundControls,
    payload: {
      roundControls: newDivisionRoundControlsMessage(),
      loginState,
    },
  });

  console.log("the final state", finalState);
  expect(finalState.divisions["NWL B"].pairings.length).toBe(1);
  expect(
    finalState.divisions["NWL B"].pairings[0].roundPairings[0].players[0].id,
  ).toBe("JkW7MXvVPfj7HdgAwLQzJ4:will");
  expect(
    finalState.divisions["NWL B"].pairings[0].roundPairings[0].players[1].id,
  ).toBe("ViSLeuyqNcSA3GcHJP5rA5:nigel");
});

// it('tests my pairings', () => {
//   const state = defaultTournamentState;

//   const state1 = TournamentReducer(state, {
//     actionType: ActionType.SetTourneyMetadata,
//     payload: tourneyMetadata(),
//   });

//   const state2 = TournamentReducer(state1, {
//     actionType: ActionType.SetDivisionsData,
//     payload: {
//       fullDivisions: initialTourneyXHRMessage(),
//       loginState: {
//         username: 'cesar',
//         userID: 'ncSw3WeNGMzATfwzz7pdkF',
//         loggedIn: true,
//         connId: 'conn-123',
//         connectedToSocket: true,
//       },
//     },
//   });

//   const state3 = TournamentReducer(state2, {
//     actionType: ActionType.StartTourneyRound,
//     payload: startTourneyMessage(),
//   });

//   expect(state3.)
// });
