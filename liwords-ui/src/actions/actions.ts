export type Action = {
  actionType: ActionType;
  payload: unknown;
};

export enum ActionType {
  /* lobby actions */
  AddSoughtGame,
  AddSoughtGames,
  RemoveSoughtGame,

  AddMatchRequest,
  AddMatchRequests,

  AddActiveGame,
  AddActiveGames,
  RemoveActiveGame,

  /* game actions */
  AddGameEvent,
  RefreshTurns,
  RefreshHistory,
  ClearHistory,
  EndGame,
  SetAllRacks,

  /* login state actions */
  SetAuthentication,
  SetConnectedToSocket,
  SetCurrentLagMs,
}
