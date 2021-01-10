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

  /* tourney actions */
  AddTourneyGameResult,
  AddTourneyGameResults,
  SetTourneyGamesOffset,

  SetTourneyMetadata,
  SetDivisionData,
  SetDivisionsData,
  StartTourneyRound,

  /* game actions */
  AddGameEvent,
  RefreshTurns,
  RefreshHistory,
  ClearHistory,
  EndGame,

  /* login state actions */
  SetAuthentication,
  SetConnectedToSocket,
  SetCurrentLagMs,
}
