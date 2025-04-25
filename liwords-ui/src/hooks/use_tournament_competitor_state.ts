import { useMemo } from "react";
import {
  useLoginStateStoreContext,
  useTournamentStoreContext,
} from "../store/store";
import {
  getCompetitorState,
  CompetitorState,
} from "../store/selectors/tournament_selectors";

export function useTournamentCompetitorState(): CompetitorState {
  const { tournamentContext } = useTournamentStoreContext();
  const { loginState } = useLoginStateStoreContext();

  return useMemo(
    () => getCompetitorState(tournamentContext, loginState),
    [tournamentContext, loginState],
  );
}
