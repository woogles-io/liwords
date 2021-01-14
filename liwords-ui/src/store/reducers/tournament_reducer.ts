import { Action, ActionType } from '../../actions/actions';
import {
  FullTournamentDivisions,
  GameEndReasonMap,
  MessageType,
  PlayerRoundInfo,
  ReadyForTournamentGame,
  RoundStandings,
  TournamentDivisionDataResponse,
  TournamentGameEndedEvent,
  TournamentGameResult,
  TournamentGameResultMap,
  TournamentRoundStarted,
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
  players: Array<string>;
  outcomes: Array<tournamentGameResult>;
  readyStates: Array<string>;
  games: Array<TournamentGame>;
};

export type Division = {
  tournamentID: string;
  divisionID: string;
  players: Array<string>;
  // Add TournamentControls here.
  roundInfo: Array<string>; // a 1-d array, implementing the backend 2-d array of pairings
  playerIndexMap: { [playerID: string]: number };
  pairingMap: { [roundUserKey: string]: SinglePairing };
  numRounds: number;
  // Note: currentRound is zero-indexed
  currentRound: number;
  // Add Standings here
  standingsMap: { [round: number]: RoundStandings.AsObject };
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
};

export const defaultTournamentState = {
  metadata: {
    name: '',
    description: '',
    directors: new Array<string>(),
    slug: '',
    id: '',
    type: 'STANDARD' as tourneytypes,
    divisions: new Array<string>(),
  },
  started: false,
  divisions: {},
  competitorState: {
    isRegistered: false,
    currentRound: 0,
  },
  activeGames: new Array<ActiveGame>(),
  finishedTourneyGames: new Array<RecentGame>(),
  gamesPageSize: 0,
  gamesOffset: 0,
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

export type CompetitorState = {
  isRegistered: boolean;
  division?: string;
  status?: TourneyStatus;
  currentRound: number;
};

export const defaultCompetitorState = {
  isRegistered: false,
  currentRound: 0,
};

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

const divisionDataResponseToObj = (
  dd: TournamentDivisionDataResponse
): Division => {
  const ret = {
    tournamentID: dd.getId(),
    divisionID: dd.getDivisionId(),
    players: dd.getPlayersList(),
    currentRound: dd.getCurrentRound(),
    numRounds: dd.getControls()?.toObject().roundControlsList.length || 0,
    roundInfo: dd.getDivisionList(),
    pairingMap: {},
    playerIndexMap: {},
    standingsMap: {},
  };

  const pairingMap: { [key: string]: SinglePairing } = {};
  const playerIndexMap: { [playerID: string]: number } = {};
  const standingsMap: { [roundId: number]: RoundStandings.AsObject } = {};
  dd.getPairingMapMap().forEach((value: PlayerRoundInfo, key: string) => {
    pairingMap[key] = {
      players: value.getPlayersList(),
      outcomes: value.getOutcomesList(),
      readyStates: value.getReadyStatesList(),
      games: value.getGamesList().map((g) => ({
        scores: g.getScoresList(),
        gameEndReason: g.getGameEndReason(),
        id: g.getId(),
        results: g.getResultsList(),
      })),
    };
  });
  dd.getPlayerIndexMapMap().forEach((value: number, key: string) => {
    playerIndexMap[key] = value;
  });
  dd.getStandingsMap().forEach((value: RoundStandings, key: number) => {
    standingsMap[key] = value.toObject();
  });
  ret.pairingMap = pairingMap;
  ret.playerIndexMap = playerIndexMap;
  ret.standingsMap = standingsMap;
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

const getRoundInfo = (
  round: number,
  fullPlayerID: string,
  division: Division
): SinglePairing => {
  const idx = division.playerIndexMap[fullPlayerID];
  const key = division.roundInfo[idx + round * division.players.length];
  return division.pairingMap[key];
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
  if (!started) {
    return TourneyStatus.PRETOURNEY;
  }
  if (!division) {
    return TourneyStatus.PRETOURNEY; // XXX: maybe a state for not being part of tourney
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
  const fullPlayerID = `${loginContext.userID}:${loginContext.username}`;
  if (!division) {
    // This really shouldn't happen, but it's a check to make sure we don't crash.
    return TourneyStatus.PRETOURNEY;
  }
  const roundInfo = getRoundInfo(currentRound, fullPlayerID, division);

  if (!roundInfo) {
    return TourneyStatus.PRETOURNEY;
  }
  const playerIdx = roundInfo.players.indexOf(fullPlayerID);
  if (playerIdx === undefined) {
    return TourneyStatus.PRETOURNEY;
  }

  if (roundInfo.players[0] === roundInfo.players[1]) {
    switch (roundInfo.outcomes[0]) {
      case TournamentGameResult.BYE:
        return TourneyStatus.ROUND_BYE;
      case TournamentGameResult.FORFEIT_LOSS:
        return TourneyStatus.ROUND_FORFEIT_LOSS;
      case TournamentGameResult.FORFEIT_WIN:
        return TourneyStatus.ROUND_FORFEIT_WIN;
    }
    return TourneyStatus.PRETOURNEY;
  }

  if (
    roundInfo.readyStates[playerIdx] === '' &&
    roundInfo.readyStates[1 - playerIdx] !== ''
  ) {
    // Our opponent is ready
    return TourneyStatus.ROUND_OPPONENT_WAITING;
  } else if (
    roundInfo.readyStates[1 - playerIdx] === '' &&
    roundInfo.readyStates[playerIdx] !== ''
  ) {
    // We're ready
    return TourneyStatus.ROUND_READY;
  }

  if (roundInfo.games[0] && roundInfo.games[0].gameEndReason) {
    // Game already finished
    return TourneyStatus.ROUND_GAME_FINISHED;
  }
  if (
    roundInfo.readyStates[playerIdx] === '' &&
    roundInfo.readyStates[1 - playerIdx] === ''
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

    // This message gets set often, every time anything happens (a game ends,
    // users get edited, new pairings get set, etc.)
    case ActionType.SetDivisionData: {
      // Convert the protobuf object to a nicer JS representation:
      const dd = action.payload as {
        divisionMessage: TournamentDivisionDataResponse;
        loginState: LoginState;
      };
      const divData = divisionDataResponseToObj(dd.divisionMessage);
      const fullLoggedInID = `${dd.loginState.userID}:${dd.loginState.username}`;
      let registeredDivision: Division | undefined;
      if (divData.players.includes(fullLoggedInID)) {
        registeredDivision = divData;
      }
      let competitorState: CompetitorState = defaultCompetitorState;
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
      return {
        ...state,
        competitorState,
        divisions: {
          ...state.divisions,
          [dd.divisionMessage.getDivisionId()]: divData,
        },
      };
    }

    case ActionType.SetDivisionsData: {
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
          if (divisions[key].players.includes(fullLoggedInID)) {
            registeredDivision = divisions[key];
          }
        }
      );
      // However we should check to see if we've already played the game,
      // or are playing the game.

      let competitorState: CompetitorState = defaultCompetitorState;
      if (registeredDivision) {
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

      return {
        ...state,
        started: true,
        divisions: {
          ...state.divisions,
          division: {
            ...state.divisions[division],
            currentRound: m.getRound(),
          },
        },
        competitorState: {
          ...state.competitorState,
          currentRound:
            // Don't touch the current round if the division doesn't match
            state.competitorState.division === division
              ? m.getRound()
              : state.competitorState.currentRound,

          status:
            // only change to ROUND_OPEN if we are in this tournament.
            // There might be a potential race condition here where our
            // opponent clicks Ready immediately when they receive this
            // message. However, that's ok, because the backend will
            // still take us to the game once we click "I'm ready".
            state.competitorState.division === division
              ? TourneyStatus.ROUND_OPEN
              : state.competitorState.status,
        },
      };
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
