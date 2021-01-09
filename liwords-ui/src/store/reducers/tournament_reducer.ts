import { Action, ActionType } from '../../actions/actions';
import {
  GameEndReasonMap,
  TournamentGameResultMap,
} from '../../gen/api/proto/realtime/realtime_pb';

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
  game_end_reason: gameEndReason;
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
  currentRound: number;
  // Add Standings here
};

export type TournamentState = {
  metadata: TournamentMetadata;
  // standings, pairings, etc. more stuff here to come.
  started: boolean;
  // Note: currentRound is zero-indexed
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
  }
  throw new Error(`unhandled action type ${action.actionType}`);
}
