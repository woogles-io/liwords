import React from "react";
import { useTournamentStoreContext } from "../store/store";
import { getCompetitorState } from "../store/selectors/tournament_selectors";
import { TourneyStatus } from "../store/reducers/tournament_reducer";
import { LoginState } from "../store/login_state";

type Props = {
  userID?: string;
  username?: string;
  loggedIn: boolean;
};

/**
 * This is an example component that demonstrates how to use the selector pattern
 * for accessing competitorState instead of directly accessing it from the state.
 */
export const TournamentStatusDisplay: React.FC<Props> = (props) => {
  // Get the tournament context
  const { tournamentContext } = useTournamentStoreContext();

  // Create login state from props
  const loginState: LoginState = {
    userID: props.userID || "",
    username: props.username || "",
    loggedIn: props.loggedIn,
    connID: "",
    connectedToSocket: true,
    path: "",
    perms: [],
  };

  // Use the selector to compute the competitor state
  const competitorState = getCompetitorState(tournamentContext, loginState);

  // Now we can use competitorState just like before
  const renderStatus = () => {
    if (!competitorState.isRegistered) {
      return <div>You are not registered for this tournament.</div>;
    }

    if (!competitorState.isCheckedIn) {
      return <div>Please check in for the tournament.</div>;
    }

    switch (competitorState.status) {
      case TourneyStatus.ROUND_OPEN:
        return <div>Round is open. Get ready to play!</div>;
      case TourneyStatus.ROUND_READY:
        return <div>You are ready. Waiting for your opponent.</div>;
      case TourneyStatus.ROUND_OPPONENT_WAITING:
        return (
          <div>
            Your opponent is ready. Please click "Ready" to start the game.
          </div>
        );
      case TourneyStatus.ROUND_GAME_ACTIVE:
        return <div>Your game is in progress.</div>;
      case TourneyStatus.ROUND_GAME_FINISHED:
        return <div>Your game is finished. Waiting for the next round.</div>;
      case TourneyStatus.ROUND_BYE:
        return <div>You have a bye this round.</div>;
      default:
        return <div>Tournament status: {competitorState.status}</div>;
    }
  };

  return (
    <div className="tournament-status">
      <h3>Tournament Status</h3>
      {renderStatus()}
      {competitorState.division && (
        <div>Division: {competitorState.division}</div>
      )}
      {competitorState.currentRound >= 0 && (
        <div>Current Round: {competitorState.currentRound + 1}</div>
      )}
    </div>
  );
};
