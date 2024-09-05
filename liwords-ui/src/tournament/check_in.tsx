import { Button, Col, Divider, message, Row } from 'antd';
import React, { useMemo } from 'react';
import { TournamentService } from '../gen/api/proto/tournament_service/tournament_service_connect';
import { useTournamentStoreContext } from '../store/store';
import { flashError, useClient } from '../utils/hooks/connect';

// I did not find a design for this, but it is trial functionality in order
// to keep the tournament running smoothly.

export const CheckIn = () => {
  const { tournamentContext } = useTournamentStoreContext();

  // const { loginState } = useLoginStateStoreContext();
  const tournamentClient = useClient(TournamentService);

  // Only registered players can check in.
  const checkedIn = useMemo(() => {
    if (!tournamentContext.competitorState.division) {
      return false;
    }
    const division =
      tournamentContext.divisions[tournamentContext.competitorState.division];
    // return division.checkedInPlayers.has(
    //   loginState.userID + ':' + loginState.username
    // );
    // XXX: TEMP CODE so this thing compiles -- FIX ME!!
    return division !== null;
  }, [tournamentContext.competitorState.division, tournamentContext.divisions]);

  if (!tournamentContext.competitorState.isRegistered) {
    return null;
  }
  if (checkedIn) {
    return null;
  }

  const checkin = async () => {
    try {
      await tournamentClient.checkIn({ id: tournamentContext.metadata?.id });
      message.info({
        content: 'You are checked in.',
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Row>
      <Col offset={10}>
        <Button
          onClick={checkin}
          type="primary"
          size="large"
          style={{ marginTop: 54, marginBottom: 58 }}
        >
          CHECK IN
        </Button>
      </Col>
      <Divider />
    </Row>
  );
};
