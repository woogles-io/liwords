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
  DeleteDivision,
  SetTourneyGamesOffset,
  SetTourneyMetadata,
  SetDivisionData,
  SetDivisionsData,
  SetDivisionRoundControls,
  SetDivisionPairings,
  DeleteDivisionPairings,
  SetDivisionControls,
  SetDivisionPlayers,
  SetTournamentFinished,
  StartTourneyRound,
  SetReadyForGame,

  /* game actions */
  AddGameEvent,
  RefreshTurns,
  RefreshHistory,
  ClearHistory,
  EndGame,
  ProcessGameMetaEvent,

  /* login state actions */
  SetAuthentication,
  SetConnectedToSocket,
  SetCurrentLagMs,
}
