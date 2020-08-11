import Modal from 'antd/lib/modal/Modal';
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
  isRematch: boolean;
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
  let isRematch = false;
  if (req instanceof MatchRequest) {
    console.log('ismatchrequest');
    receivingUser = req.getReceivingUser()!;
    isRematch = req.getIsRematch();
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
    isRematch,
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
    /*      if (matchRequest.isRematch) {
        // If it is a rematch, we want to show a modal.
        // Note that this is a bit of a hack, as there's no way to show
        // a match request outside of the lobby otherwise at the moment.
        // If we want to handle this in a different way, then we need to think
        // more carefully about it.

        Modal.confirm({
          title: 'Match Request',
          icon: <ExclamationCircleOutlined />,
          content: `${mr
            .getUser()
            ?.getDisplayName()} has challenged you to a rematch`,
          onOk() {
            console.log('OK');
          },
          onCancel() {
            console.log('Cancel');
          },
        });
      }
      */

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
