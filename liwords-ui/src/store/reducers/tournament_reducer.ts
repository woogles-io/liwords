import { Action, ActionType } from '../../actions/actions';
import {
  FullTournamentDivisions,
  GameEndReasonMap,
  MessageType,
  PlayerRoundInfo,
  ReadyForTournamentGame,
  TournamentDivisionDataResponse,
  TournamentGameResultMap,
  TournamentRoundStarted,
} from '../../gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from '../../utils/protobuf';

type tourneytypes = 'STANDARD' | 'CLUB' | 'CLUB_SESSION';
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
};

type SinglePairing = {
  players: Array<string>;
  outcomes: Array<tournamentGameResult>;
  readyStates: Array<string>;
  games: Array<TournamentGame>;
};

type Division = {
  tournamentID: string;
  divisionID: string;
  players: Array<string>;
  // Add TournamentControls here.
  roundInfo: { [roundUserKey: string]: SinglePairing };
  // Note: currentRound is zero-indexed
  currentRound: number;
  // Add Standings here
};

export type TournamentState = {
  metadata: TournamentMetadata;
  // standings, pairings, etc. more stuff here to come.
  started: boolean;
  divisions: { [name: string]: Division };
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
  ROUND_FORFEIT = 'ROUND_FORFEIT',
  POSTTOURNEY = 'POSTTOURNEY',
}

export type CompetitorState = {
  isRegistered: boolean;
  division?: string;
  status?: TourneyStatus;
  currentRound: number; // Should be the 1 based user facing round
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
  // Note: the competitorState round is 1-indexed (for display purposes),
  // so we subtract 1 here for the socket msg.
  const round = competitorState.currentRound - 1;
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
    roundInfo: {},
  };

  const roundInfo: { [key: string]: SinglePairing } = {};
  const divmap = dd.getDivisionMap();
  divmap.forEach((value: PlayerRoundInfo, key: string) => {
    roundInfo[key] = {
      players: value.getPlayersList(),
      outcomes: value.getOutcomesList(),
      readyStates: value.getReadyStatesList(),
      games: value.getGamesList().map((g) => ({
        scores: g.getScoresList(),
        gameEndReason: g.getGameEndReason(),
        results: g.getResultsList(),
      })),
    };
  });
  ret.roundInfo = roundInfo;
  return ret;
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

    case ActionType.SetDivisionData: {
      // Convert the protobuf object to a nicer JS representation:
      const dd = action.payload as TournamentDivisionDataResponse;
      const divData = divisionDataResponseToObj(dd);
      return {
        ...state,
        divisions: {
          ...state.divisions,
          [dd.getDivisionId()]: divData,
        },
      };
    }

    case ActionType.SetDivisionsData: {
      const dd = action.payload as FullTournamentDivisions;
      const divisions: { [name: string]: Division } = {};

      dd.getDivisionsMap().forEach(
        (value: TournamentDivisionDataResponse, key: string) => {
          divisions[key] = divisionDataResponseToObj(value);
        }
      );

      return {
        ...state,
        divisions,
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
      // The "Ready" button and pairings should be displayed based on:
      //    - the tournament having started
      //    - player not having yet started the current round's game
      //      (how do we determine that? a combination of the live games
      //       currently ongoing and a game result already being in for this game?)
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
      };
    }
  }
  throw new Error(`unhandled action type ${action.actionType}`);
}
