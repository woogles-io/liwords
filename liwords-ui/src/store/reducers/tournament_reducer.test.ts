import { ActionType } from '../../actions/actions';
import {
  DivisionControls,
  DivisionControlsResponse,
  DivisionRoundControls,
  FirstMethod,
  FullTournamentDivisions,
  GameRequest,
  GameRules,
  Pairing,
  PlayersAddedOrRemovedResponse,
  RoundControl,
  TournamentDivisionDataResponse,
  TournamentGame,
  TournamentGameResult,
  TournamentPerson,
  TournamentPersons,
  TournamentRoundStarted,
} from '../../gen/api/proto/realtime/realtime_pb';
import {
  defaultTournamentState,
  TournamentReducer,
} from './tournament_reducer';
import { ftData } from './testdata/tourney_1_divisions';
import { ChallengeRule } from '../../gen/macondo/api/proto/macondo/macondo_pb';

const toArr = (s: string) => {
  const bytes = new Uint8Array(Math.ceil(s.length / 2));
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(s.substr(i * 2, 2), 16);
  }
  return bytes;
};

// This is a fairly complex tourney
const initialTourneyXHRMessage = () => {
  const msg = toArr(ftData);
  return FullTournamentDivisions.deserializeBinary(msg);
};

const tourneyMetadata = () => {
  return {
    name: 'Wolges Incorporated',
    description: 'Welcome to Wolges: population: You',
    directors: ['cesar', 'thedirector'],
    slug: '/tournament/wolges',
    id: 'qzqWHsGVBrAgiuAZp9nJJm',
    type: 'STANDARD',
    divisions: ['CSW', 'NWL'],
  };
};

const startTourneyMessage = () => {
  const msg = new TournamentRoundStarted();
  msg.setTournamentId('qzqWHsGVBrAgiuAZp9nJJm');
  msg.setDivision('CSW');

  return msg;
};

const fullDivisionsState = () => {
  const state = defaultTournamentState;

  const state1 = TournamentReducer(state, {
    actionType: ActionType.SetTourneyMetadata,
    payload: tourneyMetadata(),
  });

  const state2 = TournamentReducer(state1, {
    actionType: ActionType.SetDivisionsData,
    payload: {
      fullDivisions: initialTourneyXHRMessage(),
      loginState: {
        username: 'foo',
        userID: 'foo123',
        loggedIn: true,
        connId: 'conn-123',
        connectedToSocket: true,
      },
    },
  });

  return state2;
};

it('tests initial fulldivisions message', () => {
  const state = defaultTournamentState;
  const newState = TournamentReducer(state, {
    actionType: ActionType.SetDivisionsData,
    payload: {
      fullDivisions: initialTourneyXHRMessage(),
      loginState: {
        username: 'mina',
        userID: 'MoczSz5dksZuKMnxcH6yVT',
        loggedIn: true,
        connId: 'conn-123',
        connectedToSocket: true,
      },
    },
  });

  expect(newState.started).toBe(false);
  expect(newState.divisions['CSW']).toBeTruthy();
  expect(newState.divisions['NWL']).toBeTruthy();
  expect(newState.divisions['TWL']).toBeFalsy();
  expect(newState.competitorState).toEqual({
    isRegistered: true,
    currentRound: -1,
    division: 'CSW',
    status: 'PRETOURNEY',
  });
  expect(newState.initializedFromXHR).toBe(true);
});

it('tests tourneystart', () => {
  const state = fullDivisionsState();

  const finalState = TournamentReducer(state, {
    actionType: ActionType.StartTourneyRound,
    payload: startTourneyMessage(),
  });

  expect(finalState.started).toBe(true);
});

const newDivisionMessage = () => {
  const msg = new TournamentDivisionDataResponse();
  msg.setId('qzqWHsGVBrAgiuAZp9nJJm');
  msg.setDivision('NWL B');
  msg.setCurrentRound(-1);
  return msg;
};

const newPlayersMessage = () => {
  const msg = new PlayersAddedOrRemovedResponse();
  msg.setId('qzqWHsGVBrAgiuAZp9nJJm');
  msg.setDivision('NWL B');

  const tp = new TournamentPersons();
  const personsList = new Array<TournamentPerson>();
  const p1 = new TournamentPerson();
  p1.setId('ViSLeuyqNcSA3GcHJP5rA5:nigel');
  p1.setRating(2344);
  const p2 = new TournamentPerson();
  p2.setId('JkW7MXvVPfj7HdgAwLQzJ4:will');
  p2.setRating(1234);
  personsList.push(p1, p2);
  tp.setPersonsList(personsList);
  msg.setPlayers(tp);

  return msg;
};

const newTournamentControlsMessage = () => {
  const gameReq = new GameRequest();
  const rules = new GameRules();
  rules.setBoardLayoutName('CrosswordGame');
  rules.setLetterDistributionName('English');
  gameReq.setLexicon('NWL20');

  gameReq.setRules(rules);
  gameReq.setInitialTimeSeconds(180);
  gameReq.setChallengeRule(ChallengeRule.DOUBLE);

  const divControls = new DivisionControls();
  divControls.setId('qzqWHsGVBrAgiuAZp9nJJm');
  divControls.setDivision('NWL B');
  divControls.setGameRequest(gameReq);
  divControls.setSuspendedResult(TournamentGameResult.BYE);

  const msg = new DivisionControlsResponse();
  msg.setDivision('NWL B');
  msg.setId('qzqWHsGVBrAgiuAZp9nJJm');
  msg.setDivisionControls(divControls);

  return msg;
};

const newDivisionRoundControlsMessage = () => {
  const msg = new DivisionRoundControls();
  msg.setId('qzqWHsGVBrAgiuAZp9nJJm');
  msg.setDivision('NWL B');

  const rcl = new RoundControl();
  const pairing = new Pairing();

  rcl.setFirstMethod(FirstMethod.AUTOMATIC_FIRST);
  rcl.setGamesPerRound(1);

  pairing.setPlayersList([1, 0]);
  pairing.setRound(0);

  const game = new TournamentGame();
  game.setScoresList([0, 0]);
  game.setResultsList([0, 0]);

  pairing.setGamesList([game]);
  pairing.setOutcomesList([0, 0]);
  pairing.setReadyStatesList(['', '']);

  msg.setRoundControlsList([rcl]);
  msg.setDivisionPairingsList([pairing]);
  return msg;
};

it('adds new divisions and pairings', () => {
  const state = fullDivisionsState();

  const loginState = {
    username: 'cesar',
    userID: 'ncSw3WeNGMzATfwzz7pdkF',
    loggedIn: true,
    connId: 'conn-123',
    connectedToSocket: true,
  };

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
      divisionControls: newTournamentControlsMessage(),
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

  console.log('the final state', finalState);
  expect(finalState.divisions['NWL B'].pairings.length).toBe(1);
  expect(
    finalState.divisions[
      'NWL B'
    ].pairings[0].roundPairings[0].players[0].getId()
  ).toBe('JkW7MXvVPfj7HdgAwLQzJ4:will');
  expect(
    finalState.divisions[
      'NWL B'
    ].pairings[0].roundPairings[0].players[1].getId()
  ).toBe('ViSLeuyqNcSA3GcHJP5rA5:nigel');
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
