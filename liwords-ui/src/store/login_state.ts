import { Action, ActionType } from "../actions/actions";

export type LoginState = {
  username: string;
  userID: string;
  loggedIn: boolean;
  connID: string;
  connectedToSocket: boolean;
  path: string;
  // permissions holds the user's effective granular permission codes
  // (e.g. "can_create_broadcasts") fetched via GetSelfPermissions RPC.
  // Gate all UI on these codes rather than on roles or JWT short codes.
  permissions: Array<string>;
};

export type AuthInfo = {
  username: string;
  userID: string;
  loggedIn: boolean;
  connID: string;
};

export function LoginStateReducer(
  state: LoginState,
  action: Action,
): LoginState {
  switch (action.actionType) {
    case ActionType.SetAuthentication: {
      const auth = action.payload as AuthInfo;
      return {
        ...state,
        ...auth,
      };
    }

    case ActionType.SetPermissions: {
      const permissions = (action.payload as Array<string>) ?? [];
      return {
        ...state,
        permissions,
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
