import React, { useCallback } from 'react';
import { Card } from 'antd';
import { useTournamentStoreContext } from '../store/store';
import './competitor_status.scss';
import { TourneyStatus } from './state';

export const CompetitorStatus = () => {
  const { tournamentContext, competitorContext } = useTournamentStoreContext();
  const renderStatus = useCallback(() => {
    switch (competitorContext.status) {
      case TourneyStatus.PRETOURNEY:
        // TODO: This one should specify tournament start date or time when we have that
        // Round 1 of the [tourney name] starts at [time].
        return (
          <p>
            Thanks for registering for the {tournamentContext.metadata.name}.
          </p>
        );
      case TourneyStatus.ROUND_FORFEIT:
        return (
          <p>
            You forfeited your Round {competitorContext.currentRound} game.
            Please check in with the director.
          </p>
        );
      case TourneyStatus.POSTTOURNEY:
        return (
          <p>
            Thanks so much for playing in the {tournamentContext.metadata.name}!
          </p>
        );
      default:
        // We don't know this status or there isn't one
        return (
          <p>
            Thanks for registering for the {tournamentContext.metadata.name}.
          </p>
        );
    }
    // Missing status or
  }, [tournamentContext, competitorContext]);
  return (
    <Card className={`competitor-status ${competitorContext.status}`}>
      {renderStatus()}
    </Card>
  );
};
