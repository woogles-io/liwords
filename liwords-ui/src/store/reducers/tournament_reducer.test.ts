import { ActionType } from '../../actions/actions';
import {
  FullTournamentDivisions,
  TournamentRoundStarted,
} from '../../gen/api/proto/realtime/realtime_pb';
import {
  defaultTournamentState,
  TournamentReducer,
} from './tournament_reducer';
import { ftData } from './testdata/tourney_1_divisions';

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

it('adds new divisions and pairings', () => {
  const state = fullDivisionsState();

  // Add a new division, add two players, add random pairings.

  // const finalState = TournamentReducer(state, {
  //   actionType:
  // })
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
