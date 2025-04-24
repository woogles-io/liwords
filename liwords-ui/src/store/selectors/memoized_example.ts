import { LoginState } from "../login_state";
import {
  CompetitorState,
  TournamentState,
  defaultCompetitorState,
} from "../reducers/tournament_reducer";
import { getCompetitorState } from "./tournament_selectors";

/**
 * This is an example of how to implement a simple memoization for the selector
 * to improve performance if the selector is called frequently.
 *
 * Note: This is just a simple implementation. For production use, consider
 * using a library like 'reselect' which handles more complex cases.
 */

// Simple memoization for the getCompetitorState selector
export const createMemoizedCompetitorStateSelector = () => {
  // Cache the last input and output
  let lastTournamentState: TournamentState | null = null;
  let lastLoginState: LoginState | null = null;
  let lastResult: CompetitorState = defaultCompetitorState;

  // Return a function that checks if inputs have changed
  return (
    tournamentState: TournamentState,
    loginState: LoginState,
  ): CompetitorState => {
    // Simple equality check - in a real implementation, you might want
    // to do a deeper comparison of relevant parts of the state
    const tournamentStateChanged = tournamentState !== lastTournamentState;
    const loginStateChanged = loginState !== lastLoginState;

    // If either input has changed, recompute the result
    if (tournamentStateChanged || loginStateChanged) {
      lastTournamentState = tournamentState;
      lastLoginState = loginState;
      lastResult = getCompetitorState(tournamentState, loginState);
    }

    return lastResult;
  };
};

// Create an instance of the memoized selector
export const memoizedGetCompetitorState =
  createMemoizedCompetitorStateSelector();

/**
 * Usage example:
 *
 * import { memoizedGetCompetitorState } from "../store/selectors/memoized_example";
 *
 * // In your component:
 * const { tournamentContext } = useTournamentStoreContext();
 * const loginState = { ... };
 * const competitorState = memoizedGetCompetitorState(tournamentContext, loginState);
 */

/**
 * For more complex selectors or when you need to compose selectors,
 * consider using the 'reselect' library:
 *
 * Example with reselect:
 *
 * import { createSelector } from 'reselect';
 *
 * const getTournamentState = (state: TournamentState) => state;
 * const getLoginState = (_: TournamentState, loginState: LoginState) => loginState;
 *
 * export const getCompetitorStateSelector = createSelector(
 *   [getTournamentState, getLoginState],
 *   (tournamentState, loginState) => {
 *     // Compute competitor state here
 *     // This function will only run when tournamentState or loginState changes
 *   }
 * );
 */
