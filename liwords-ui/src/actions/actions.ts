export type Action = {
  actionType: ActionType;
  payload: unknown;
  reducer: string;
};

export enum ActionType {
  AddSoughtGame,
  AddSoughtGames,
  RemoveSoughtGame,
}
