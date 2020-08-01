import { Action, ActionType } from '../../actions/actions';
import {
  GameMeta,
  SeekRequest,
  RatingMode,
} from '../../gen/api/proto/realtime/realtime_pb';

export type SoughtGame = {
  seeker: string;
  lexicon: string;
  initialTimeSecs: number;
  maxOvertimeMinutes: number;
  challengeRule: number;
  userRating: string;
  rated: boolean;
  seekID: string;
};

type playerMeta = {
  rating: string;
  displayName: string;
};

export type ActiveGame = {
  lexicon: string;
  variant: string;
  initialTimeSecs: number;
  challengeRule: number;
  rated: boolean;
  maxOvertimeMinutes: number;
  gameID: string;
  players: Array<playerMeta>;
};

export type LobbyState = {
  soughtGames: Array<SoughtGame>;
  // + Other things in the lobby here that have state.
  activeGames: Array<ActiveGame>;
};

export const SeekRequestToSoughtGame = (sr: SeekRequest): SoughtGame | null => {
  const gameReq = sr.getGameRequest();
  const user = sr.getUser();
  if (!gameReq || !user) {
    return null;
  }
  return {
    seeker: user.getDisplayName(),
    userRating: user.getRelevantRating(),
    lexicon: gameReq.getLexicon(),
    initialTimeSecs: gameReq.getInitialTimeSeconds(),
    challengeRule: gameReq.getChallengeRule(),
    seekID: gameReq.getRequestId(),
    rated: gameReq.getRatingMode() === RatingMode.RATED,
    maxOvertimeMinutes: gameReq.getMaxOvertimeMinutes(),
  };
};

export const GameMetaToActiveGame = (gm: GameMeta): ActiveGame | null => {
  const users = gm.getUsersList();
  const gameReq = gm.getGameRequest();

  const players = users.map((um) => ({
    rating: um.getRelevantRating(),
    displayName: um.getDisplayName(),
  }));

  if (!gameReq) {
    return null;
  }

  let variant = gameReq.getRules()?.getVariantName();
  if (!variant) {
    variant = gameReq.getRules()?.getBoardLayoutName()!;
  }
  return {
    players,
    lexicon: gameReq.getLexicon(),
    variant,
    initialTimeSecs: gameReq.getInitialTimeSeconds(),
    challengeRule: gameReq.getChallengeRule(),
    rated: gameReq.getRatingMode() === RatingMode.RATED,
    maxOvertimeMinutes: gameReq.getMaxOvertimeMinutes(),
    gameID: gm.getId(),
  };
};

export function LobbyReducer(state: LobbyState, action: Action): LobbyState {
  switch (action.actionType) {
    case ActionType.AddSoughtGame: {
      const { soughtGames } = state;
      const soughtGame = action.payload as SoughtGame;
      return {
        ...state,
        soughtGames: [...soughtGames, soughtGame],
      };
    }

    case ActionType.RemoveSoughtGame: {
      const { soughtGames } = state;
      const id = action.payload as string;

      const newArr = soughtGames.filter((sg) => {
        return sg.seekID !== id;
      });

      return {
        ...state,
        soughtGames: newArr,
      };
    }

    case ActionType.AddSoughtGames: {
      const soughtGames = action.payload as Array<SoughtGame>;
      return {
        ...state,
        soughtGames,
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
  }
  throw new Error(`unhandled action type ${action.actionType}`);
}
