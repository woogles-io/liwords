import { Action, ActionType } from '../../actions/actions';

export type SoughtGame = {
  seeker: string;
  lexicon: string;
  initialTimeSecs: number;
  challengeRule: number;
  // rating: number;
  rated: boolean;
  seekID: string;
};

export type LobbyState = {
  soughtGames: Array<SoughtGame>;
  // + Other things in the lobby here that have state.
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
  }
  throw new Error(`unhandled action type ${action.actionType}`);
}
