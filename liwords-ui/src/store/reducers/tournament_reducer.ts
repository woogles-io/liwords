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
  DivisionPairingsDeletedResponse,
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
import {
  TournamentMetadata,
  TType,
} from '../../gen/api/proto/tournament_service/tournament_service_pb';
import { RecentGame } from '../../tournament/recent_game';
import { encodeToSocketFmt } from '../../utils/protobuf';
import { LoginState } from '../login_state';
import { ActiveGame } from './lobby_reducer';

type valueof<T> = T[keyof T];

type tournamentGameResult = valueof<TournamentGameResultMap>;
type gameEndReason = valueof<GameEndReasonMap>;

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
  pairingCount?: number;
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
  directors: Array<string>;
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
  initializedFromXHR: boolean;
};

const defaultMetadata = new TournamentMetadata();
defaultMetadata.setType(TType.LEGACY);

export const defaultTournamentState = {
  metadata: defaultMetadata,
  directors: new Array<string>(),
  started: false,
  divisions: {},
  competitorState: defaultCompetitorState,
  activeGames: new Array<ActiveGame>(),
  finishedTourneyGames: new Array<RecentGame>(),
  gamesPageSize: 20,
  gamesOffset: 0,
  finished: false,
  initializedFromXHR: false,
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

const findOpponentIdx = (
  player: number,
  playerIndexMap: { [playerID: string]: number },
  pairings: Array<RoundPairings>,
  round: number
): number => {
  if (!pairings[round].roundPairings[player].players) {
    return -1;
  }

  let opponent =
    playerIndexMap[pairings[round].roundPairings[player].players[0].getId()];
  if (opponent === player) {
    opponent =
      playerIndexMap[pairings[round].roundPairings[player].players[1].getId()];
  }
  return opponent;
};

const deletePairings = (
  players: Array<TournamentPerson>,
  playerIndexMap: { [playerID: string]: number },
  existingPairings: Array<RoundPairings>,
  round: number
): Array<RoundPairings> => {
  const updatedPairings = [...existingPairings];

  for (let i = 0; i < updatedPairings[round].roundPairings.length; i++) {
    updatedPairings[round].roundPairings[i] = {} as SinglePairing;
  }
  return updatedPairings;
};

const reducePairings = (
  players: Array<TournamentPerson>,
  playerIndexMap: { [playerID: string]: number },
  existingPairings: Array<RoundPairings>,
  newPairings: Pairing[]
): Array<RoundPairings> => {
  const updatedPairings = [...existingPairings];

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
    };

    // check if any of these players are already paired.

    for (let pidx = 0; pidx <= 1; pidx++) {
      const opp = findOpponentIdx(
        value.getPlayersList()[pidx],
        playerIndexMap,
        updatedPairings,
        value.getRound()
      );
      if (opp !== -1) {
        updatedPairings[value.getRound()].roundPairings[
          opp
        ] = {} as SinglePairing;
      }
    }

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
      if (!value.players || !value.players.length) {
        roundPairings.push({} as SinglePairing);
      } else {
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
      }
    });
    pairingsCopy.push({ roundPairings });
  });
  return pairingsCopy;
};

const reduceStandings = (
  existingStandings: jspb.Map<number, RoundStandings>,
  newStandings: jspb.Map<number, RoundStandings>
): jspb.Map<number, RoundStandings> => {
  const updatedStandings = new jspb.Map<number, RoundStandings>([]);

  existingStandings.forEach((value: RoundStandings, key: number) => {
    updatedStandings.set(key, value);
  });

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
    roundControls: dd.getRoundControlsList(),
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
  ret.numRounds = dd.getRoundControlsList().length;

  for (let i = 0; i < ret.numRounds; i++) {
    const newRoundPairings = new Array<SinglePairing>();
    dd.getPlayers()
      ?.getPersonsList()
      .forEach(() => {
        newRoundPairings.push({} as SinglePairing);
      });
    newPairings.push({ roundPairings: newRoundPairings });
  }

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
  division: Division,
  activeGames: Array<ActiveGame>,
  loginContext: LoginState
): TourneyStatus => {
  if (!division) {
    return TourneyStatus.PRETOURNEY; // XXX: maybe a state for not being part of tourney
  }

  const fullPlayerID = `${loginContext.userID}:${loginContext.username}`;
  const pairing = getPairing(division.currentRound, fullPlayerID, division);

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
    if (division.currentRound === division.numRounds - 1) {
      return TourneyStatus.POSTTOURNEY;
    }
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
  if (!state.initializedFromXHR) {
    // Throw away messages if we haven't received the XHR back yet.
    // Yes, this can result in potential race conditions.
    // We should buffer messages received prior to the XHR, apply them
    // post-XHR receipt, and make all reducers idempotent.
    if (
      ![
        ActionType.SetDivisionsData,
        ActionType.SetTourneyMetadata,
        ActionType.AddActiveGames,
        ActionType.AddActiveGame,
        // These are legacy events for CLUB/LEGACY tournament types

        ActionType.RemoveActiveGame,
        ActionType.AddTourneyGameResult,
        ActionType.AddTourneyGameResults,
        ActionType.SetTourneyGamesOffset,
      ].includes(action.actionType)
    ) {
      return state;
    }
  }

  switch (action.actionType) {
    case ActionType.SetTourneyMetadata:
      const m = action.payload as {
        directors: Array<string>;
        metadata: TournamentMetadata;
      };
      console.log('gonna set metadata', m);
      return {
        ...state,
        directors: m.directors,
        metadata: m.metadata,
      };

    case ActionType.SetDivisionRoundControls: {
      const drc = action.payload as {
        roundControls: DivisionRoundControls;
        loginState: LoginState;
      };
      const division = drc.roundControls.getDivision();
      // copy old stuff
      let newNumRounds = state.divisions[division].numRounds;
      let newRoundControls = state.divisions[division].roundControls;
      let newPairings = copyPairings(state.divisions[division].pairings);
      let newStandings = reduceStandings(
        state.divisions[division].standingsMap,
        new jspb.Map<number, RoundStandings>([])
      );

      if (!state.started) {
        // This can only be a full set of round controls
        newPairings = new Array<RoundPairings>();
        newRoundControls = drc.roundControls.getRoundControlsList();
        newNumRounds = newRoundControls.length;
        for (let i = 0; i < newNumRounds; i++) {
          // reset all pairings
          const newRoundPairings = new Array<SinglePairing>();
          state.divisions[division].players.forEach(() => {
            newRoundPairings.push({} as SinglePairing);
          });
          newPairings.push({ roundPairings: newRoundPairings });
        }
        newPairings = reducePairings(
          state.divisions[division].players,
          state.divisions[division].playerIndexMap,
          newPairings,
          drc.roundControls.getDivisionPairingsList()
        );
        newStandings = drc.roundControls.getDivisionStandingsMap();
      } else {
        // This can only be an individual round control in the future.
        newRoundControls = new Array<RoundControl>();
        state.divisions[division].roundControls.forEach((rc: RoundControl) => {
          newRoundControls.push(rc.cloneMessage());
        });
        drc.roundControls.getRoundControlsList().forEach((rc: RoundControl) => {
          newRoundControls[rc.getRound()] = rc;
        });
      }

      return Object.assign({}, state, {
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            roundControls: newRoundControls,
            standingsMap: newStandings,
            pairings: newPairings,
            numRounds: newNumRounds,
          }),
        }),
      });
    }

    case ActionType.SetDivisionControls: {
      const dc = action.payload as {
        divisionControlsResponse: DivisionControlsResponse;
        loginState: LoginState;
      };
      const division = dc.divisionControlsResponse.getDivision();

      return Object.assign({}, state, {
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            divisionControls: dc.divisionControlsResponse.getDivisionControls(),
          }),
        }),
      });
    }

    case ActionType.DeleteDivisionPairings: {
      const dp = action.payload as {
        dpdr: DivisionPairingsDeletedResponse;
        loginState: LoginState;
      };
      const division = dp.dpdr.getDivision();
      const newPairings = deletePairings(
        state.divisions[division].players,
        state.divisions[division].playerIndexMap,
        state.divisions[division].pairings,
        dp.dpdr.getRound()
      );
      const newState = Object.assign({}, state, {
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            pairings: newPairings,
          }),
        }),
      });
      return newState;
    }

    case ActionType.SetDivisionPairings: {
      const dp = action.payload as {
        dpr: DivisionPairingsResponse;
        loginState: LoginState;
      };
      const division = dp.dpr.getDivision();
      const newPairings = reducePairings(
        state.divisions[division].players,
        state.divisions[division].playerIndexMap,
        state.divisions[division].pairings,
        dp.dpr.getDivisionPairingsList()
      );

      const newStandings = reduceStandings(
        state.divisions[division].standingsMap,
        dp.dpr.getDivisionStandingsMap()
      );

      const fullLoggedInID = `${dp.loginState.userID}:${dp.loginState.username}`;
      const userIndex =
        state.divisions[division].playerIndexMap[fullLoggedInID];
      let newStatus = state.competitorState.status;
      if (userIndex != null) {
        dp.dpr.getDivisionPairingsList().forEach((pairing: Pairing) => {
          if (pairing.getRound() === state.divisions[division].currentRound) {
            const pairingPlayers = pairing.getPlayersList();
            if (
              pairingPlayers &&
              (pairingPlayers[0] === userIndex ||
                pairingPlayers[1] === userIndex)
            ) {
              let playerIndex = 0;
              if (pairingPlayers[1] === userIndex) {
                playerIndex = 1;
              }
              const outcome = pairing.getOutcomesList()[playerIndex];
              if (outcome !== TournamentGameResult.NO_RESULT) {
                newStatus = TourneyStatus.ROUND_GAME_FINISHED;
              }
            }
          }
        });
      }

      const finishedGamesMap: { [id: string]: boolean } = {};
      dp.dpr.getDivisionPairingsList().forEach((p) => {
        p.getGamesList().forEach((tg) => {
          const gameID = tg.getId();
          if (tg.getGameEndReason()) {
            finishedGamesMap[gameID] = true;
          }
        });
      });

      const newActiveGames = state.activeGames.filter(
        (ag) => !finishedGamesMap[ag.gameID]
      );

      const newState = Object.assign({}, state, {
        competitorState: Object.assign({}, state.competitorState, {
          status: newStatus,
        }),
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            pairings: newPairings,
            standingsMap: newStandings,
          }),
        }),
        activeGames: newActiveGames,
      });
      return newState;
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

      let expandedPairings = copyPairings(state.divisions[division].pairings);
      let newStandings = reduceStandings(
        state.divisions[division].standingsMap,
        new jspb.Map<number, RoundStandings>([])
      );

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

      if (!state.started) {
        expandedPairings = Array<RoundPairings>();
        for (let i = 0; i < state.divisions[division].numRounds; i++) {
          const newRoundPairings = new Array<SinglePairing>();
          newPlayers.forEach(() => {
            newRoundPairings.push({} as SinglePairing);
          });
          expandedPairings.push({ roundPairings: newRoundPairings });
        }
        newStandings = new jspb.Map<number, RoundStandings>([]);
      }

      const newPairings = reducePairings(
        newPlayers,
        newPlayerIndexMap,
        expandedPairings,
        dp.parr.getDivisionPairingsList()
      );
      newStandings = reduceStandings(
        newStandings,
        dp.parr.getDivisionStandingsMap()
      );

      const fullLoggedInID = `${dp.loginState.userID}:${dp.loginState.username}`;
      const myPreviousDivision = state.competitorState.division;
      console.log('divisions are', state.divisions);
      let myRegisteredDivision: Division | undefined;
      if (fullLoggedInID in newPlayerIndexMap) {
        myRegisteredDivision = state.divisions[division];
      }
      console.log(
        'registered division',
        myRegisteredDivision,
        fullLoggedInID,
        newPlayerIndexMap
      );
      let competitorState: CompetitorState = state.competitorState;

      if (myRegisteredDivision) {
        competitorState = {
          isRegistered: true,
          division: myRegisteredDivision.divisionID,
          currentRound: myRegisteredDivision.currentRound,
          status: tourneyStatus(
            myRegisteredDivision,
            state.activeGames,
            dp.loginState
          ),
        };
      } else {
        competitorState = {
          ...competitorState,
          isRegistered:
            // we're only still registered if we were already registered,
            // and the division we were registered in is not the division that came in
            // (otherwise, it would have listed us as a player)
            myPreviousDivision !== undefined && myPreviousDivision !== division,
        };
      }
      const newState = Object.assign({}, state, {
        competitorState,
        divisions: Object.assign({}, state.divisions, {
          [division]: Object.assign({}, state.divisions[division], {
            pairings: newPairings,
            standingsMap: newStandings,
            playerIndexMap: newPlayerIndexMap,
            players: newPlayers,
          }),
        }),
      });
      return newState;
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
            registeredDivision,
            state.activeGames,
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
      // Handles XHR request for GetDivisions
      const dd = action.payload as {
        fullDivisions: FullTournamentDivisions;
        loginState: LoginState;
      };

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

      let competitorState: CompetitorState = state.competitorState;
      if (registeredDivision) {
        console.log(
          'registereddiv',
          registeredDivision,
          'stateactivegames',
          state.activeGames
        );
        competitorState = {
          isRegistered: true,
          division: registeredDivision.divisionID,
          currentRound: registeredDivision.currentRound,
          status: tourneyStatus(
            registeredDivision,
            state.activeGames,
            dd.loginState
          ),
        };
      }

      return {
        ...state,
        started: dd.fullDivisions.getStarted(),
        divisions,
        competitorState,
        initializedFromXHR: true,
      };
    }

    case ActionType.StartTourneyRound: {
      const m = action.payload as {
        trs: TournamentRoundStarted;
        loginState: LoginState;
      };
      // Make sure the tournament ID matches. (Why wouldn't it, though?)
      if (state.metadata.getId() !== m.trs.getTournamentId()) {
        return state;
      }
      const division = m.trs.getDivision();
      // Mark the round for the passed-in division to be the passed-in round.

      const newDivisions = Object.assign({}, state.divisions, {
        [division]: Object.assign({}, state.divisions[division], {
          currentRound: m.trs.getRound(),
        }),
      });

      const newStatus =
        state.competitorState.division === division
          ? tourneyStatus(
              newDivisions[division],
              state.activeGames,
              m.loginState
            )
          : state.competitorState.status;

      return Object.assign({}, state, {
        started: true,
        divisions: newDivisions,
        competitorState: Object.assign({}, state.competitorState, {
          currentRound:
            state.competitorState.division === division
              ? m.trs.getRound()
              : state.competitorState.currentRound,
          status: newStatus,
        }),
      });
    }

    case ActionType.SetReadyForGame: {
      const m = action.payload as {
        ready: ReadyForTournamentGame;
        loginState: LoginState;
      };

      const registeredDivision = state.competitorState.division;
      if (!registeredDivision) {
        // this should not happen, we should not get a ready state if we
        // are not in some division
        return state;
      }
      const division = state.divisions[registeredDivision];
      const fullPlayerID = `${m.loginState.userID}:${m.loginState.username}`;
      if (m.ready.getRound() !== division.currentRound) {
        // this should not happen, the ready state should always be for the
        // current round.
        console.error('ready state current round does not match');
        return state;
      }
      if (m.ready.getDivision() !== registeredDivision) {
        // this should not happen, the ready state should always be for the
        // current division.
        console.error('ready state current division does not match');
        return state;
      }
      const pairing = getPairing(m.ready.getRound(), fullPlayerID, division);
      if (!pairing) {
        return state;
      }
      const newPairing = {
        ...pairing,
        readyStates: [...pairing.readyStates],
      };
      // find out where _we_ are
      let usLoc;
      const involvedIDs = newPairing.players.map((x) => x.getId());
      if (newPairing.players[0].getId() === fullPlayerID) {
        usLoc = 0;
      } else if (newPairing.players[1].getId() === fullPlayerID) {
        usLoc = 1;
      } else {
        console.error('unexpected usLoc', newPairing);
        return state;
      }
      let toModify;
      if (m.ready.getPlayerId() === fullPlayerID) {
        toModify = usLoc;
      } else {
        // it's the opponent
        toModify = 1 - usLoc;
      }

      newPairing.readyStates[toModify] = m.ready.getUnready() ? '' : 'ready';

      const updatedPairings = copyPairings(division.pairings);
      updatedPairings[m.ready.getRound()].roundPairings[
        division.playerIndexMap[involvedIDs[0]]
      ] = newPairing;
      updatedPairings[m.ready.getRound()].roundPairings[
        division.playerIndexMap[involvedIDs[1]]
      ] = newPairing;

      const newRegisteredDiv = Object.assign(
        {},
        state.divisions[registeredDivision],
        {
          pairings: updatedPairings,
        }
      );

      const newCompetitorState = {
        ...state.competitorState,
        status: tourneyStatus(
          newRegisteredDiv,
          state.activeGames,
          m.loginState
        ),
      };

      return Object.assign({}, state, {
        divisions: Object.assign({}, state.divisions, {
          [registeredDivision]: newRegisteredDiv,
        }),
        competitorState: newCompetitorState,
      });
    }

    // For the following two actions, it is important to recalculate
    // the competitorState if it exists; this is because
    // competitorState.status depends on state.activeGames.
    case ActionType.AddActiveGames: {
      const g = action.payload as {
        activeGames: Array<ActiveGame>;
        loginState: LoginState;
      };
      const registeredDivision = state.competitorState.division;
      let newCompetitorState = state.competitorState;
      if (registeredDivision) {
        newCompetitorState = {
          ...state.competitorState,
          status: tourneyStatus(
            state.divisions[registeredDivision],
            g.activeGames,
            g.loginState
          ),
        };
      }

      return {
        ...state,
        activeGames: g.activeGames,
        competitorState: newCompetitorState,
      };
    }

    case ActionType.AddActiveGame: {
      const { activeGames } = state;
      const g = action.payload as {
        activeGame: ActiveGame;
        loginState: LoginState;
      };
      const registeredDivision = state.competitorState.division;
      let newCompetitorState = state.competitorState;
      if (registeredDivision) {
        newCompetitorState = {
          ...state.competitorState,
          status: tourneyStatus(
            state.divisions[registeredDivision],
            [...state.activeGames, g.activeGame],
            g.loginState
          ),
        };
      }

      return {
        ...state,
        activeGames: [...activeGames, g.activeGame],
        competitorState: newCompetitorState,
      };
    }

    case ActionType.RemoveActiveGame: {
      // LEGACY event. When games end in regular tournaments, we just get
      // a divisions pairing message.
      const { activeGames } = state;
      const g = action.payload as string;

      const newArr = activeGames.filter((ag) => {
        return ag.gameID !== g;
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
