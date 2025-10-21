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

  AddCorrespondenceGame,
  AddCorrespondenceGames,
  RemoveCorrespondenceGame,
  UpdateCorrespondenceGame,
  SetCorrespondenceSeeks,

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
  SetTourneyPlayerCheckin,
  SetMonitoringData,
  UpdateMonitoringStream,

  /* game actions */
  AddGameEvent, // will be obsolete when we move to OMGWordsEVent
  RefreshHistory, // will be obsolete when we move to GameDocument fully
  ClearHistory,
  EndGame,
  ProcessGameMetaEvent,
  SetupStaticPosition,
  InitFromDocument,
  AddOMGWordsEvent,

  /* game comment actions */
  ReloadComments,
  AddComment,
  DeleteComment,
  EditComment,

  /* login state actions */
  SetAuthentication,
  SetConnectedToSocket,
  SetCurrentLagMs,
}
