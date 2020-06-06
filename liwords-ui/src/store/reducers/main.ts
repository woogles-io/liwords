import { Action } from '../../actions/actions';
import { LobbyReducer, LobbyState } from './lobby_reducer';

// The main reducer
export function Reducer(state: unknown, action: Action) {
  switch (action.reducer) {
    case 'lobby':
      return LobbyReducer(state as LobbyState, action);
  }
}
