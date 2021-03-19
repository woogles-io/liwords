import { Button } from 'antd';
import React, { useMemo } from 'react';
import {
  useLoginStateStoreContext,
  useTournamentStoreContext,
} from '../store/store';

// I did not find a design for this, but it is trial functionality in order
// to keep the tournament running smoothly.

export const CheckIn = () => {
  const { tournamentContext } = useTournamentStoreContext();

  const { loginState } = useLoginStateStoreContext();

  // Only registered players can check in.
  const checkedIn = useMemo(() => {
    if (!tournamentContext.competitorState.division) {
      return false;
    }
    const division =
      tournamentContext.divisions[tournamentContext.competitorState.division];
    return division.checkedInPlayers.includes(loginState.username); // xxx + userid?
  }, [
    loginState.username,
    tournamentContext.competitorState.division,
    tournamentContext.divisions,
  ]);

  if (!tournamentContext.competitorState.isRegistered) {
    return null;
  }
  if (checkedIn) {
    return null;
  }

  return <Button>Check in</Button>;
};
