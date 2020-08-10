import { Action, ActionType } from '../../actions/actions';
import {
  GameMeta,
  SeekRequest,
  RatingMode,
  MatchRequest,
  MatchUser,
} from '../../gen/api/proto/realtime/realtime_pb';

export type SoughtGame = {
  seeker: string;
  lexicon: string;
  initialTimeSecs: number;
  incrementSecs: number;
  maxOvertimeMinutes: number;
  challengeRule: number;
  userRating: string;
  rated: boolean;
  seekID: string;
  // Only for direct match requests:
  receiver: MatchUser;
};

type playerMeta = {
  rating: string;
  displayName: string;
};

export type ActiveGame = {
  lexicon: string;
  variant: string;
  initialTimeSecs: number;
  incrementSecs: number;
  challengeRule: number;
  rated: boolean;
  maxOvertimeMinutes: number;
  gameID: string;
  players: Array<playerMeta>;
};

export type LobbyState = {
  soughtGames: Array<SoughtGame>;
  matchRequests: Array<SoughtGame>;
  // + Other things in the lobby here that have state.
  activeGames: Array<ActiveGame>;
};

export const SeekRequestToSoughtGame = (
  req: SeekRequest | MatchRequest
): SoughtGame | null => {
  const gameReq = req.getGameRequest();
  const user = req.getUser();
  if (!gameReq || !user) {
    return null;
  }

  let receivingUser = new MatchUser();
  if (req instanceof MatchRequest) {
    console.log('ismatchrequest');
    receivingUser = req.getReceivingUser()!;
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
    receiver: receivingUser,
    incrementSecs: gameReq.getIncrementSeconds(),
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
    incrementSecs: gameReq.getIncrementSeconds(),
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
      // Look for match requests too.
      const { soughtGames, matchRequests } = state;
      const id = action.payload as string;

      const newSought = soughtGames.filter((sg) => {
        return sg.seekID !== id;
      });
      const newMatch = matchRequests.filter((mr) => {
        return mr.seekID !== id;
      });

      return {
        ...state,
        soughtGames: newSought,
        matchRequests: newMatch,
      };
    }

    case ActionType.AddSoughtGames: {
      const soughtGames = action.payload as Array<SoughtGame>;
      console.log('soughtGames', soughtGames);
      soughtGames.sort((a, b) => {
        return a.userRating < b.userRating ? -1 : 1;
      });
      return {
        ...state,
        soughtGames,
      };
    }

    case ActionType.AddMatchRequest: {
      const { matchRequests } = state;
      const matchRequest = action.payload as SoughtGame;
      // it's a match request; put new ones on top.
      return {
        ...state,
        matchRequests: [matchRequest, ...matchRequests],
      };
    }

    case ActionType.AddMatchRequests: {
      const matchRequests = action.payload as Array<SoughtGame>;
      // These are match requests.
      console.log('matchRequests', matchRequests);
      matchRequests.sort((a, b) => {
        return a.userRating < b.userRating ? -1 : 1;
      });
      return {
        ...state,
        matchRequests,
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
