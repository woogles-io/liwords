export type Action = {
  actionType: ActionType;
  payload: unknown;
  reducer: string;
};

export enum ActionType {
  /* lobby actions */
  AddSoughtGame,
  AddSoughtGames,
  RemoveSoughtGame,
  /* game actions */
  AddGameEvent,
  RefreshHistory,
}
