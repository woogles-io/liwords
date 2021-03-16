import { Action, ActionType } from '../actions/actions';

export type LoginState = {
  username: string;
  userID: string;
  loggedIn: boolean;
  connID: string;
  connectedToSocket: boolean;
  path: string;
  perms: Array<string>;
};

export type AuthInfo = {
  username: string;
  userID: string;
  loggedIn: boolean;
  connID: string;
  perms: Array<string>;
};

export function LoginStateReducer(
  state: LoginState,
  action: Action
): LoginState {
  switch (action.actionType) {
    case ActionType.SetAuthentication: {
      const auth = action.payload as AuthInfo;
      return {
        ...state,
        ...auth,
      };
    }

    case ActionType.SetConnectedToSocket: {
      const connectedToSocket = action.payload as boolean;
      return {
        ...state,
        connectedToSocket,
      };
    }
  }
  throw new Error(`unhandled action type ${action.actionType}`);
}
