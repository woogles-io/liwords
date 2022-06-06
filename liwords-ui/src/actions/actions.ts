export type Action = {
  actionType: ActionType;
  payload: unknown;
};

export enum ActionType {
  /* lobby actions */
  AddSoughtGame,
  AddSoughtGames,
  RemoveSoughtGame,

  AddActiveGame,
  AddActiveGames,
  RemoveActiveGame,

  setLobbyFilterByLexicon,

  // XXX: should move somewhere else?
  UpdateProfile,

  /* tourney actions */
  AddTourneyGameResult,
  AddTourneyGameResults,
  DeleteDivision,
  SetTourneyGamesOffset,
  SetTourneyMetadata,
  SetTourneyReducedMetadata,
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
  SetupStaticPosition,

  /* login state actions */
  SetAuthentication,
  SetConnectedToSocket,
  SetCurrentLagMs,
}
