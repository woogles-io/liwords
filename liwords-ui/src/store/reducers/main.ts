import { Action } from '../../actions/actions';
import { LobbyReducer, LobbyState } from './lobby_reducer';
import { GameReducer, GameState } from './game_reducer';

// The main reducer
// export function Reducer(state: unknown, action: Action) {
//   switch (action.reducer) {
//     case 'lobby':
//       return LobbyReducer(state as LobbyState, action);
//     case 'game':
//       return GameReducer(state as GameState, action);
//   }
//   throw new Error('Unknown reducer ' + action.reducer);
// }
