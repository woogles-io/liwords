import { TournamentGameResult } from "../../gen/api/proto/ipc/tournament_pb";
import { LoginState } from "../login_state";
import {
  Division,
  TournamentState,
  defaultCompetitorState,
  getPairing,
} from "../reducers/tournament_reducer";

export enum TourneyStatus {
  PRETOURNEY = "PRETOURNEY",
  NOT_CHECKED_IN = "NOT_CHECKED_IN",
  ROUND_BYE = "ROUND_BYE",
  ROUND_OPEN = "ROUND_OPEN",
  ROUND_GAME_FINISHED = "ROUND_GAME_FINISHED",
  ROUND_READY = "ROUND_READY", // waiting for your opponent
  ROUND_OPPONENT_WAITING = "ROUND_OPPONENT_WAITING",
  ROUND_LATE = "ROUND_LATE", // expect this to override opponent waiting
  ROUND_GAME_ACTIVE = "ROUND_GAME_ACTIVE",
  ROUND_FORFEIT_LOSS = "ROUND_FORFEIT_LOSS",
  ROUND_FORFEIT_WIN = "ROUND_FORFEIT_WIN",
  POSTTOURNEY = "POSTTOURNEY",
}

export type CompetitorState = {
  isRegistered: boolean;
  isCheckedIn: boolean;
  division?: string;
  status?: TourneyStatus;
  currentRound: number;
};

// The "Ready" button and pairings should be displayed based on:
//    - the tournament having started
//    - player not having yet started the current round's game
//      (how do we determine that? a combination of the live games
//       currently ongoing and a game result already being in for this game?)
export const tourneyStatus = (
  division: Division,
  loginContext: LoginState,
  tournamentContext: TournamentState,
): TourneyStatus => {
  if (!division) {
    return TourneyStatus.PRETOURNEY; // XXX: maybe a state for not being part of tourney
  }

  const fullPlayerID = `${loginContext.userID}:${loginContext.username}`;

  if (
    tournamentContext.metadata.checkinsOpen &&
    !division.players.find((p) => p.id === fullPlayerID)?.checkedIn &&
    !tournamentContext.started
  ) {
    return TourneyStatus.NOT_CHECKED_IN;
  }

  const pairing = getPairing(division.currentRound, fullPlayerID, division);

  if (!pairing || !pairing.players) {
    return TourneyStatus.PRETOURNEY;
  }

  const playerIdx = pairing.players.map((v) => v.id).indexOf(fullPlayerID);
  if (playerIdx === undefined) {
    return TourneyStatus.PRETOURNEY;
  }
  if (pairing.players[0] === pairing.players[1]) {
    switch (pairing.outcomes[0]) {
      case TournamentGameResult.BYE:
        return TourneyStatus.ROUND_BYE;
      case TournamentGameResult.FORFEIT_LOSS:
        return TourneyStatus.ROUND_FORFEIT_LOSS;
      case TournamentGameResult.FORFEIT_WIN:
        return TourneyStatus.ROUND_FORFEIT_WIN;
    }
    return TourneyStatus.PRETOURNEY;
  }
  if (pairing.games[0] && pairing.games[0].gameEndReason) {
    if (division.currentRound === division.numRounds - 1) {
      return TourneyStatus.POSTTOURNEY;
    }
    // Game already finished
    return TourneyStatus.ROUND_GAME_FINISHED;
  }
  if (
    tournamentContext.activeGames.find((ag) => {
      return (
        ag.players[0].displayName === loginContext.username ||
        ag.players[1].displayName === loginContext.username
      );
    })
  ) {
    return TourneyStatus.ROUND_GAME_ACTIVE;
  }
  if (
    pairing.readyStates[playerIdx] === "" &&
    pairing.readyStates[1 - playerIdx] !== ""
  ) {
    // Our opponent is ready
    return TourneyStatus.ROUND_OPPONENT_WAITING;
  } else if (
    pairing.readyStates[1 - playerIdx] === "" &&
    pairing.readyStates[playerIdx] !== ""
  ) {
    // We're ready
    return TourneyStatus.ROUND_READY;
  }

  if (
    pairing.readyStates[playerIdx] === "" &&
    pairing.readyStates[1 - playerIdx] === ""
  ) {
    return TourneyStatus.ROUND_OPEN;
  }

  // Otherwise just return generic pre-tourney
  return TourneyStatus.PRETOURNEY;
};

/**
 * Selector that computes the competitor state from the tournament state and login state.
 * This ensures that the competitor state is always up-to-date with the underlying data.
 */
export const getCompetitorState = (
  state: TournamentState,
  loginState: LoginState,
): CompetitorState => {
  if (!loginState.userID || !loginState.username) {
    return defaultCompetitorState;
  }

  const fullPlayerID = `${loginState.userID}:${loginState.username}`;

  // Find the division the user is in, if any
  let userDivision: Division | undefined;
  let divisionID: string | undefined;

  for (const [id, division] of Object.entries(state.divisions)) {
    if (fullPlayerID in division.playerIndexMap) {
      userDivision = division;
      divisionID = id;
      break;
    }
  }

  if (!userDivision || !divisionID) {
    return defaultCompetitorState;
  }

  return {
    isRegistered: true,
    isCheckedIn:
      userDivision.players.find((p) => p.id === fullPlayerID)?.checkedIn ||
      false,
    division: divisionID,
    currentRound: userDivision.currentRound,
    status: tourneyStatus(userDivision, loginState, state),
  };
};
