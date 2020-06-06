import { Action } from '../../actions/actions';

export type SoughtGame = {
  seeker: string;
  lexicon: string;
  initialTimeSecs: number;
  challengeRule: number;
  // rating: number;
  seekID: string;
};

export function LobbyReducer(state: unknown, action: Action) {
  switch (action.actionType) {
    case 'addSoughtGame': {
      const soughtGames = state as Array<SoughtGame>;
      const soughtGame = action.payload as SoughtGame;
      return [...soughtGames, soughtGame];
    }

    case 'removeGame': {
      const soughtGames = state as Array<SoughtGame>;
      const id = action.payload as string;

      const newArr = soughtGames.filter((sg) => {
        return sg.seekID !== id;
      });
      return newArr;
    }

    case 'addSoughtGames': {
      const soughtGames = action.payload as Array<SoughtGame>;
      return soughtGames;
    }
  }
}
