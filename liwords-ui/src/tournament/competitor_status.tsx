import React, { useCallback } from 'react';
import { Card, Button } from 'antd';
import { useTournamentStoreContext } from '../store/store';
import { ClockCircleOutlined } from '@ant-design/icons';
import './competitor_status.scss';
import { TourneyStatus } from '../store/reducers/tournament_reducer';

type Props = {
  sendReady: () => void;
};

export const CompetitorStatus = (props: Props) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { competitorState: competitorContext } = tournamentContext;
  const renderStatus = useCallback(() => {
    //TODO: If they're playing the right game, this should be true.
    // If they've wandered off, we'll render the backToGamePrompt instead
    // of other in-game states
    const inCorrectGame = true;

    //TODO: Calculate this from presences
    const opponentDisconnected = false;

    // TODO: Make return buttton to do the right thing.
    const backToGamePrompt = (
      <>
        <ClockCircleOutlined />
        <p>Your round {competitorContext.currentRound} game is in progress.</p>
        <Button className="primary">Return</Button>
      </>
    );
    // TODO: This needs to calculate if current round is last once division info is in the store
    const isLastRound = false;

    switch (competitorContext.status) {
      case TourneyStatus.PRETOURNEY:
        // TODO: This one should specify tournament start date or time when we have that
        // Round 1 of the [tourney name] starts at [time].
        return (
          <>
            <ClockCircleOutlined />
            <p>
              Thanks for registering for the {tournamentContext.metadata.name}.
            </p>
          </>
        );
      case TourneyStatus.ROUND_FORFEIT_LOSS:
        return (
          <>
            <ClockCircleOutlined />
            <p>
              You forfeited your Round {competitorContext.currentRound} game.
              Please check in with the director.
            </p>
          </>
        );
      case TourneyStatus.ROUND_FORFEIT_WIN:
        return (
          <>
            <ClockCircleOutlined />
            <p>
              Your opponent forfeited their Round{' '}
              {competitorContext.currentRound} game. Please check in with the
              director.
            </p>
          </>
        );
      case TourneyStatus.ROUND_BYE: {
        // TODO: This one should specify next round start time when we have that
        // secondary-message: Please be back by XX:XX for round XX
        return (
          <>
            <ClockCircleOutlined />
            <p>
              You have a bye for round {competitorContext.currentRound}.
              {!isLastRound && (
                <span className="secondary-message">
                  Please return in time for round{' '}
                  {competitorContext.currentRound + 1}.
                </span>
              )}
            </p>
          </>
        );
      }
      case TourneyStatus.ROUND_OPEN: {
        return (
          <>
            <p>Time to start round {competitorContext.currentRound}!</p>
            <Button className="primary" onClick={props.sendReady}>
              I'm ready
            </Button>
          </>
        );
      }
      case TourneyStatus.ROUND_LATE: {
        //Todo: When we have automatic forfeit, make countdown to forfeit
        //the secondary message
        return (
          <>
            <p>
              Start your game!
              <span className="secondary-message">
                Director may assign a forfeit.
              </span>
            </p>
            <Button className="primary" onClick={props.sendReady}>
              I'm ready
            </Button>
          </>
        );
      }
      case TourneyStatus.ROUND_OPPONENT_WAITING: {
        //Todo: Button should send ready message
        return (
          <>
            <p>Your opponent is waiting!</p>
            <Button className="primary" onClick={props.sendReady}>
              I'm ready
            </Button>
          </>
        );
      }
      case TourneyStatus.ROUND_READY: {
        //Todo: Button should send I'm not ready message
        return (
          <>
            <p>Waiting for opponent</p>
            {/*<Button className="secondary">Not ready</Button>*/}
          </>
        );
      }
      case TourneyStatus.ROUND_GAME_ACTIVE: {
        if (!inCorrectGame) {
          return backToGamePrompt;
        }
        if (opponentDisconnected) {
          return (
            <>
              <ClockCircleOutlined />
              <p>
                Your opponent has disconnected from the game. Their clock will
                continue to run.
              </p>
            </>
          );
        }
        return (
          <>
            <ClockCircleOutlined />
            <p>
              Good luck in your round {competitorContext.currentRound} game!
            </p>
          </>
        );
      }
      case TourneyStatus.ROUND_GAME_FINISHED: {
        return (
          <>
            <ClockCircleOutlined />
            <p>
              Your round {competitorContext.currentRound} score has been
              recorded.{!isLastRound ? ' Good luck in your next game!' : ''}
            </p>
          </>
        );
      }
      case TourneyStatus.POSTTOURNEY:
        return (
          <>
            <ClockCircleOutlined />
            return (
            <p>
              Thanks so much for playing in the{' '}
              {tournamentContext.metadata.name}!
            </p>
          </>
        );
      default:
        // We don't understand this status or there isn't one
        return (
          <>
            <ClockCircleOutlined />
            <p>
              Thanks for registering for the {tournamentContext.metadata.name}.
            </p>
          </>
        );
    }
    // Missing status or
  }, [tournamentContext, competitorContext, props.sendReady]);
  return (
    <Card className={`competitor-status ${competitorContext.status}`}>
      {renderStatus()}
    </Card>
  );
};
