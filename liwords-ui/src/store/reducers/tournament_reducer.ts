import * as jspb from 'google-protobuf';

import { Action, ActionType } from '../../actions/actions';
import {
  FullTournamentDivisions,
  GameEndReasonMap,
  MessageType,
  ReadyForTournamentGame,
  RoundStandings,
  TournamentDivisionDataResponse,
  DivisionPairingsResponse,
  PlayersAddedOrRemovedResponse,
  DivisionControlsResponse,
  DivisionRoundControls,
  // TournamentFinishedResponse,
  DivisionControls,
  RoundControl,
  Pairing,
  TournamentPerson,
  TournamentGameEndedEvent,
  TournamentGameResult,
  TournamentGameResultMap,
  TournamentRoundStarted,
  TournamentDivisionDeletedResponse,
} from '../../gen/api/proto/realtime/realtime_pb';
import { RecentGame } from '../../tournament/recent_game';
import { encodeToSocketFmt } from '../../utils/protobuf';
import { LoginState } from '../login_state';
import { ActiveGame } from './lobby_reducer';

type tourneytypes = 'STANDARD' | 'CLUB' | 'CHILD' | 'LEGACY';
type valueof<T> = T[keyof T];

type tournamentGameResult = valueof<TournamentGameResultMap>;
type gameEndReason = valueof<GameEndReasonMap>;

export type TournamentMetadata = {
  name: string;
  description: string;
  directors: Array<string>;
  slug: string;
  id: string;
  type: tourneytypes;
  divisions: Array<string>;
};

type TournamentGame = {
  scores: Array<number>;
  results: Array<tournamentGameResult>;
  gameEndReason: gameEndReason;
  id?: string;
};

export type SinglePairing = {
  players: Array<TournamentPerson>;
  outcomes: Array<tournamentGameResult>;
  readyStates: Array<string>;
  games: Array<TournamentGame>;
};

type RoundPairings = {
  roundPairings: Array<SinglePairing>;
};

export type Division = {
  tournamentID: string;
  divisionID: string;
  players: Array<TournamentPerson>;
  standingsMap: jspb.Map<number, RoundStandings>;
  pairings: Array<RoundPairings>;
  divisionControls: DivisionControls | undefined;
  roundControls: Array<RoundControl>;
  // currentRound is zero-indexed
  currentRound: number;
  playerIndexMap: { [playerID: string]: number };
  numRounds: number;
  // checkedInPlayers: Set<string>;
};

export type CompetitorState = {
  isRegistered: boolean;
  division?: string;
  status?: TourneyStatus;
  currentRound: number;
};

export const defaultCompetitorState = {
  isRegistered: false,
  currentRound: -1,
};

export type TournamentState = {
  metadata: TournamentMetadata;
  // standings, pairings, etc. more stuff here to come.
  started: boolean;
  divisions: { [name: string]: Division };
  competitorState: CompetitorState;

  // activeGames in this tournament.
  activeGames: Array<ActiveGame>;

  finishedTourneyGames: Array<RecentGame>;
  gamesPageSize: number;
  gamesOffset: number;
  finished: boolean;
};

export const defaultTournamentState = {
  metadata: {
    name: '',
    description: '',
    directors: new Array<string>(),
    slug: '',
    id: '',
    type: 'LEGACY' as tourneytypes,
    divisions: new Array<string>(),
  },
  started: false,
  divisions: {},
  competitorState: defaultCompetitorState,
  activeGames: new Array<ActiveGame>(),
  finishedTourneyGames: new Array<RecentGame>(),
  gamesPageSize: 20,
  gamesOffset: 0,
  finished: false,
};

export enum TourneyStatus {
  PRETOURNEY = 'PRETOURNEY',
  ROUND_BYE = 'ROUND_BYE',
  ROUND_OPEN = 'ROUND_OPEN',
  ROUND_GAME_FINISHED = 'ROUND_GAME_FINISHED',
  ROUND_READY = 'ROUND_READY', // waiting for your opponent
  ROUND_OPPONENT_WAITING = 'ROUND_OPPONENT_WAITING',
  ROUND_LATE = 'ROUND_LATE', // expect this to override opponent waiting
  ROUND_GAME_ACTIVE = 'ROUND_GAME_ACTIVE',
  ROUND_FORFEIT_LOSS = 'ROUND_FORFEIT_LOSS',
  ROUND_FORFEIT_WIN = 'ROUND_FORFEIT_WIN',
  POSTTOURNEY = 'POSTTOURNEY',
}

export const readyForTournamentGame = (
  sendSocketMsg: (msg: Uint8Array) => void,
  tournamentID: string,
  competitorState: CompetitorState
) => {
  const evt = new ReadyForTournamentGame();
  const division = competitorState.division;
  if (!division) {
    return;
  }
  const round = competitorState.currentRound;
  evt.setDivision(division);
  evt.setTournamentId(tournamentID);
  evt.setRound(round);
  sendSocketMsg(
    encodeToSocketFmt(
      MessageType.READY_FOR_TOURNAMENT_GAME,
      evt.serializeBinary()
    )
  );
};

const reducePairings = (
  players: Array<TournamentPerson>,
  existingPairings: Array<RoundPairings>,
  newPairings: Pairing[]
): Array<RoundPairings> => {
  const updatedPairings = existingPairings;

  newPairings.forEach((value: Pairing) => {
    const newSinglePairing = {
      players: value.getPlayersList().map((v) => players[v]),
      outcomes: value.getOutcomesList(),
      readyStates: value.getReadyStatesList(),
      games: value.getGamesList().map((g) => ({
        scores: g.getScoresList(),
        gameEndReason: g.getGameEndReason(),
        id: g.getId(),
        results: g.getResultsList(),
      })),
    } as SinglePairing;
    updatedPairings[value.getRound()].roundPairings[
      value.getPlayersList()[0]
    ] = newSinglePairing;
    updatedPairings[value.getRound()].roundPairings[
      value.getPlayersList()[1]
    ] = newSinglePairing;
  });
  return updatedPairings;
};

// Create a deep copy.
const copyPairings = (existingPairings: Array<RoundPairings>) => {
  const pairingsCopy = new Array<RoundPairings>();

  existingPairings.forEach((value: RoundPairings) => {
    const roundPairings = new Array<SinglePairing>();

    value.roundPairings.forEach((value: SinglePairing) => {
      const players = new Array<TournamentPerson>();
      value.players.forEach((person) => {
        players.push(person.cloneMessage());
      });
      const newSinglePairing = {
        players,
        outcomes: [...value.outcomes],
        readyStates: [...value.readyStates],
        games: value.games.map((g) => ({
          scores: [...g.scores],
          gameEndReason: g.gameEndReason,
          id: g.id,
          results: [...g.results],
        })),
      } as SinglePairing;

      roundPairings.push(newSinglePairing);
    });
    pairingsCopy.push({ roundPairings });
  });
  return pairingsCopy;
};

const reduceStandings = (
  existingStandings: jspb.Map<number, RoundStandings>,
  newStandings: jspb.Map<number, RoundStandings>
): jspb.Map<number, RoundStandings> => {
  const updatedStandings = existingStandings;

  newStandings.forEach((value: RoundStandings, key: number) => {
    updatedStandings.set(key, value);
  });
  return updatedStandings;
};

const divisionDataResponseToObj = (
  dd: TournamentDivisionDataResponse
): Division => {
  const ret = {
    tournamentID: dd.getId(),
    divisionID: dd.getDivision(),
    players: Array<TournamentPerson>(),
    pairings: Array<RoundPairings>(),
    divisionControls: dd.getControls(), // game request, etc
    roundControls: Array<RoundControl>(),
    currentRound: dd.getCurrentRound(),
    playerIndexMap: {},
    numRounds: 0,
    standingsMap: dd.getStandingsMap(),
    //     checkedInPlayers: new Set<string>(),
  };

  // Reduce Standings

  // const standingsMap: { [roundId: number]: RoundStandings } = {};

  // dd.getStandingsMap().forEach((value: RoundStandings, key: number) => {
  //   standingsMap[key] = value;
  // });

  /**
   *     if (value.getCheckedIn()) {
      checkedInPlayers.add(dd.getPlayersList()[index]);
    }
   */

  // Reduce playerIndexMap and players

  const playerIndexMap: { [playerID: string]: number } = {};
  const newPlayers = Array<TournamentPerson>();
  dd.getPlayers()
    ?.getPersonsList()
    .forEach((value: TournamentPerson, index: number) => {
      playerIndexMap[value.getId()] = index;
      newPlayers.push(value);
    });

  ret.playerIndexMap = playerIndexMap;
  ret.players = newPlayers;

  // Reduce pairings

  const newPairings = new Array<RoundPairings>();

  dd.getRoundControlsList().forEach(() => {
    const newRoundPairings = new Array<SinglePairing>();
    dd.getPlayers()
      ?.getPersonsList()
      .forEach(() => {
        newRoundPairings.push({} as SinglePairing);
      });
    newPairings.push({ roundPairings: newRoundPairings });
  });
  ret.numRounds = dd.getRoundControlsList().length;

  dd.getPairingMapMap().forEach((value: Pairing, key: string) => {
    const newPairing = {
      players: value.getPlayersList().map((v) => newPlayers[v]),
      outcomes: value.getOutcomesList(),
      readyStates: value.getReadyStatesList(),
      games: value.getGamesList().map((g) => ({
        scores: g.getScoresList(),
        gameEndReason: g.getGameEndReason(),
        id: g.getId(),
        results: g.getResultsList(),
      })),
    } as SinglePairing;
    newPairings[value.getRound()].roundPairings[
      playerIndexMap[newPairing.players[0].getId()]
    ] = newPairing;
    newPairings[value.getRound()].roundPairings[
      playerIndexMap[newPairing.players[1].getId()]
    ] = newPairing;
  });

  ret.pairings = newPairings;

  return ret;
};

const toResultStr = (r: 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7) => {
  return {
    0: 'NO_RESULT',
    1: 'WIN',
    2: 'LOSS',
    3: 'DRAW',
    4: 'BYE',
    5: 'FORFEIT_WIN',
    6: 'FORFEIT_LOSS',
    7: 'ELIMINATED',
  }[r];
};

const toEndReason = (r: 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8) => {
  return {
    0: 'NONE',
    1: 'TIME',
    2: 'STANDARD',
    3: 'CONSECUTIVE_ZEROES',
    4: 'RESIGNED',
    5: 'ABORTED',
    6: 'TRIPLE_CHALLENGE',
    7: 'CANCELLED',
    8: 'FORCE_FORFEIT',
  }[r];
};

export const TourneyGameEndedEvtToRecentGame = (
  evt: TournamentGameEndedEvent
): RecentGame => {
  const evtPlayers = evt.getPlayersList();

  const players = evtPlayers.map((ep) => ({
    username: ep.getUsername(),
    score: ep.getScore(),
    result: toResultStr(ep.getResult()),
  }));

  return {
    players,
    end_reason: toEndReason(evt.getEndReason()),
    game_id: evt.getGameId(),
    time: evt.getTime(),
    round: evt.getRound(),
  };
};

const getPairing = (
  round: number,
  fullPlayerID: string,
  division: Division
): SinglePairing | undefined => {
  if (
    !(
      division.pairings &&
      division.pairings[round] &&
      division.pairings[round].roundPairings
    )
  ) {
    return undefined;
  }
  return division.pairings[round].roundPairings[
    division.playerIndexMap[fullPlayerID]
  ];
};

// The "Ready" button and pairings should be displayed based on:
//    - the tournament having started
//    - player not having yet started the current round's game
//      (how do we determine that? a combination of the live games
//       currently ongoing and a game result already being in for this game?)
const tourneyStatus = (
  started: boolean,
  division: Division,
  activeGames: Array<ActiveGame>,
  currentRound: number,
  loginContext: LoginState
): TourneyStatus => {
  if (!division) {
    return TourneyStatus.PRETOURNEY; // XXX: maybe a state for not being part of tourney
  }

  const fullPlayerID = `${loginContext.userID}:${loginContext.username}`;
  const pairing = getPairing(currentRound, fullPlayerID, division);

  if (!pairing || !pairing.players) {
    return TourneyStatus.PRETOURNEY;
  }

  const playerIdx = pairing.players.map((v) => v.getId()).indexOf(fullPlayerID);
  if (playerIdx === undefined) {
    return TourneyStatus.PRETOURNEY;
  }
  if (pairing.players[0] === pairing.players[1]) {
    switch (pairing.outcomes[0]) {
      case TournamentGameResult.BYE:
        return TourneyStatus.ROUND_BYE;
      case TournamentGameResult.FORFEIT_LOSS:
        return TourneyStatus.ROUND_FORFEIT_LOSS;
      case TournamentGameResult.FORFEIT_WIN:
        return TourneyStatus.ROUND_FORFEIT_WIN;
    }
    return TourneyStatus.PRETOURNEY;
  }
  if (pairing.games[0] && pairing.games[0].gameEndReason) {
    // Game already finished
    return TourneyStatus.ROUND_GAME_FINISHED;
  }
  if (
    activeGames.find((ag) => {
      return (
        ag.players[0].displayName === loginContext.username ||
        ag.players[1].displayName === loginContext.username
      );
    })
  ) {
    return TourneyStatus.ROUND_GAME_ACTIVE;
  }
  if (
    pairing.readyStates[playerIdx] === '' &&
    pairing.readyStates[1 - playerIdx] !== ''
  ) {
    // Our opponent is ready
    return TourneyStatus.ROUND_OPPONENT_WAITING;
  } else if (
    pairing.readyStates[1 - playerIdx] === '' &&
    pairing.readyStates[playerIdx] !== ''
  ) {
    // We're ready
    return TourneyStatus.ROUND_READY;
  }

  if (
    pairing.readyStates[playerIdx] === '' &&
    pairing.readyStates[1 - playerIdx] === ''
  ) {
    return TourneyStatus.ROUND_OPEN;
  }

  // Otherwise just return generic pre-tourney
  return TourneyStatus.PRETOURNEY;
};

export function TournamentReducer(
  state: TournamentState,
  action: Action
): TournamentState {
  switch (action.actionType) {
    case ActionType.SetTourneyMetadata:
      const metadata = action.payload as TournamentMetadata;
      return {
        ...state,
        metadata,
      };

    case ActionType.SetDivisionRoundControls: {
      const drc = action.payload as {
        roundControls: DivisionRoundControls;
        loginState: LoginState;
      };
      const division = drc.roundControls.getDivision();

      return Object.assign({}, state, {
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            roundControls: drc.roundControls,
          }),
        }),
      });
    }

    case ActionType.SetDivisionControls: {
      const dc = action.payload as {
        divisionControls: DivisionControlsResponse;
        loginState: LoginState;
      };
      const division = dc.divisionControls.getDivision();

      return Object.assign({}, state, {
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            divisionControls: dc.divisionControls,
          }),
        }),
      });
    }

    case ActionType.SetDivisionPairings: {
      const dp = action.payload as {
        dpr: DivisionPairingsResponse;
        loginState: LoginState;
      };
      const division = dp.dpr.getDivision();
      const newPairings = reducePairings(
        state.divisions[division].players,
        state.divisions[division].pairings,
        dp.dpr.getDivisionPairingsList()
      );

      const newStandings = reduceStandings(
        state.divisions[division].standingsMap,
        dp.dpr.getDivisionStandingsMap()
      );

      return Object.assign({}, state, {
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            pairings: newPairings,
            standings: newStandings,
          }),
        }),
      });
    }

    case ActionType.SetDivisionPlayers: {
      const dp = action.payload as {
        parr: PlayersAddedOrRemovedResponse;
        loginState: LoginState;
      };

      const division = dp.parr.getDivision();
      const respPlayers = dp.parr.getPlayers()?.getPersonsList();
      const newPlayerIndexMap: { [playerID: string]: number } = {};
      const newPlayers = Array<TournamentPerson>();

      respPlayers?.forEach((value: TournamentPerson, index: number) => {
        newPlayerIndexMap[value.getId()] = index;
        newPlayers.push(value);
      });

      const expandedPairings = copyPairings(state.divisions[division].pairings);

      if (
        state.started &&
        respPlayers &&
        respPlayers?.length > newPlayers.length
      ) {
        // Players have been added and the tournament has already started
        // This means we must expand the current pairings
        const numberOfAddedPlayers = respPlayers?.length - newPlayers.length;

        expandedPairings.forEach((value: RoundPairings) => {
          for (let i = numberOfAddedPlayers; i >= 0; i--) {
            value.roundPairings.push({} as SinglePairing);
          }
        });
      }

      const newPairings = reducePairings(
        state.divisions[division].players,
        expandedPairings,
        dp.parr.getDivisionPairingsList()
      );
      const newStandings = reduceStandings(
        state.divisions[division].standingsMap,
        dp.parr.getDivisionStandingsMap()
      );

      const fullLoggedInID = `${dp.loginState.userID}:${dp.loginState.username}`;

      let registeredDivision: Division | undefined;
      if (fullLoggedInID in newPlayerIndexMap) {
        registeredDivision = state.divisions[division];
      }
      console.log(
        'registered division',
        registeredDivision,
        fullLoggedInID,
        newPlayerIndexMap
      );
      let competitorState: CompetitorState = state.competitorState;
      console.log('old competitor state is', JSON.stringify(competitorState));

      if (registeredDivision) {
        competitorState = {
          isRegistered: true,
          division: registeredDivision.divisionID,
          currentRound: registeredDivision.currentRound,
          status: tourneyStatus(
            state.started,
            registeredDivision,
            state.activeGames,
            registeredDivision.currentRound,
            dp.loginState
          ),
        };
      } else {
        competitorState = {
          ...competitorState,
          isRegistered: false,
        };
      }
      console.log('competitor state is', JSON.stringify(competitorState));

      return Object.assign({}, state, {
        competitorState,
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            pairings: newPairings,
            standings: newStandings,
            playerIndexMap: newPlayerIndexMap,
            players: newPlayers,
          }),
        }),
      });
    }

    case ActionType.SetTournamentFinished: {
      return {
        ...state,
        finished: true,
      };
    }

    case ActionType.SetDivisionData: {
      // Convert the protobuf object to a nicer JS representation:
      const dd = action.payload as {
        divisionMessage: TournamentDivisionDataResponse;
        loginState: LoginState;
      };
      const divData = divisionDataResponseToObj(dd.divisionMessage);
      const fullLoggedInID = `${dd.loginState.userID}:${dd.loginState.username}`;
      let registeredDivision: Division | undefined;
      if (fullLoggedInID in divData.playerIndexMap) {
        registeredDivision = divData;
      }
      let competitorState: CompetitorState = state.competitorState;
      if (registeredDivision) {
        competitorState = {
          isRegistered: true,
          division: registeredDivision.divisionID,
          currentRound: registeredDivision.currentRound,
          status: tourneyStatus(
            state.started,
            registeredDivision,
            state.activeGames,
            registeredDivision.currentRound,
            dd.loginState
          ),
        };
      }
      return Object.assign({}, state, {
        competitorState,
        divisions: Object.assign({}, state.divisions, {
          [dd.divisionMessage.getDivision()]: divData,
        }),
      });
    }

    case ActionType.DeleteDivision: {
      const dd = action.payload as TournamentDivisionDeletedResponse;
      // Only empty divisions can be deleted, so no need to worry about competitor state

      const deleted = dd.getDivision();

      const { [deleted]: _, ...divisions } = state.divisions;

      return Object.assign({}, state, {
        divisions: Object.assign({}, divisions),
      });
    }

    case ActionType.SetDivisionsData: {
      const dd = action.payload as {
        fullDivisions: FullTournamentDivisions;
        loginState: LoginState;
      };

      console.log('fdivs', dd.fullDivisions.toObject());
      const divisions: { [name: string]: Division } = {};
      const divisionsMap = dd.fullDivisions.getDivisionsMap();
      const fullLoggedInID = `${dd.loginState.userID}:${dd.loginState.username}`;
      let registeredDivision: Division | undefined;
      divisionsMap.forEach(
        (value: TournamentDivisionDataResponse, key: string) => {
          divisions[key] = divisionDataResponseToObj(value);
          if (fullLoggedInID in divisions[key].playerIndexMap) {
            registeredDivision = divisions[key];
          }
        }
      );

      console.log('divisions', divisions);
      let competitorState: CompetitorState = state.competitorState;
      if (registeredDivision) {
        console.log('registereddiv', registeredDivision);
        competitorState = {
          isRegistered: true,
          division: registeredDivision.divisionID,
          currentRound: registeredDivision.currentRound,
          status: tourneyStatus(
            dd.fullDivisions.getStarted(),
            registeredDivision,
            state.activeGames,
            registeredDivision.currentRound,
            dd.loginState
          ),
        };
      }

      return {
        ...state,
        started: dd.fullDivisions.getStarted(),
        divisions,
        competitorState,
      };
    }

    case ActionType.StartTourneyRound: {
      const m = action.payload as TournamentRoundStarted;
      // Make sure the tournament ID matches. (Why wouldn't it, though?)
      if (state.metadata.id !== m.getTournamentId()) {
        return state;
      }
      const division = m.getDivision();
      // Mark the round for the passed-in division to be the passed-in round.
      return Object.assign({}, state, {
        started: true,
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            currentRound: m.getRound(),
          }),
        }),
        competitorState: Object.assign({}, state.competitorState, {
          currentRound:
            state.competitorState.division === division
              ? m.getRound()
              : state.competitorState.currentRound,
          status:
            state.competitorState.division === division
              ? TourneyStatus.ROUND_OPEN
              : state.competitorState.status,
        }),
      });
    }

    case ActionType.SetTourneyStatus: {
      const m = action.payload as TourneyStatus;
      return {
        ...state,
        competitorState: {
          ...state.competitorState,
          status: m,
        },
      };
    }

    case ActionType.AddActiveGames: {
      const activeGames = action.payload as Array<ActiveGame>;
      return {
        ...state,
        activeGames,
      };
    }

    case ActionType.AddActiveGame: {
      const { activeGames } = state;
      const activeGame = action.payload as ActiveGame;
      return {
        ...state,
        activeGames: [...activeGames, activeGame],
      };
    }

    case ActionType.RemoveActiveGame: {
      const { activeGames } = state;
      const id = action.payload as string;

      const newArr = activeGames.filter((ag) => {
        return ag.gameID !== id;
      });

      return {
        ...state,
        activeGames: newArr,
      };
    }

    case ActionType.AddTourneyGameResult: {
      const { finishedTourneyGames, gamesOffset, gamesPageSize } = state;
      const evt = action.payload as TournamentGameEndedEvent;
      const game = TourneyGameEndedEvtToRecentGame(evt);
      // If a tourney game comes in while we're looking at another page,
      // do nothing.
      if (gamesOffset > 0) {
        return state;
      }

      // Bring newest game to the top.
      const newGames = [game, ...finishedTourneyGames];
      if (newGames.length > gamesPageSize) {
        newGames.length = gamesPageSize;
      }

      return {
        ...state,
        finishedTourneyGames: newGames,
      };
    }

    case ActionType.AddTourneyGameResults: {
      const finishedTourneyGames = action.payload as Array<RecentGame>;
      return {
        ...state,
        finishedTourneyGames,
      };
    }

    case ActionType.SetTourneyGamesOffset: {
      const offset = action.payload as number;
      return {
        ...state,
        gamesOffset: offset,
      };
    }
  }
  throw new Error(`unhandled action type ${action.actionType}`);
}
