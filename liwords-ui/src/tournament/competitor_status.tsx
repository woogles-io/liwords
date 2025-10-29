import React, { useCallback } from "react";
import { Card, Button } from "antd";
import { useTournamentStoreContext } from "../store/store";
import {
  ClockCircleOutlined,
  ExclamationCircleOutlined,
} from "@ant-design/icons";
import "./competitor_status.scss";
import { ReadyButton } from "./ready_button";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { TourneyStatus } from "../store/selectors/tournament_selectors";

type Props = {
  sendReady: () => boolean;
};

export const CompetitorStatus = (props: Props) => {
  const { tournamentContext } = useTournamentStoreContext();
  const competitorState = useTournamentCompetitorState();
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
        <p>
          Your round {competitorState.currentRound + 1} game is in progress.
        </p>
        <Button className="primary">Return</Button>
      </>
    );
    const isLastRound = competitorState.division
      ? competitorState.currentRound ===
        tournamentContext.divisions[competitorState.division]?.numRounds - 1
      : false;

    switch (competitorState.status) {
      case TourneyStatus.NOT_CHECKED_IN:
        return (
          <>
            <ExclamationCircleOutlined />
            <p>
              Check-ins are now open for the {tournamentContext.metadata.name}.
              Please check in as soon as you can!
            </p>
          </>
        );
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
              You forfeited your Round {competitorState.currentRound + 1} game.
              <span className="optional">
                Please check in with the director.
              </span>
            </p>
          </>
        );
      case TourneyStatus.ROUND_FORFEIT_WIN:
        return (
          <>
            <ClockCircleOutlined />
            <p>
              Your opponent forfeited their Round{" "}
              {competitorState.currentRound + 1} game.{" "}
              <span className="optional">
                Please check in with the director.
              </span>
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
              You have a bye for round {competitorState.currentRound + 1}.
              {!isLastRound && (
                <span className="secondary-message">
                  Please return in time for round{" "}
                  {competitorState.currentRound + 2}.
                </span>
              )}
            </p>
          </>
        );
      }
      case TourneyStatus.ROUND_OPEN: {
        return (
          <>
            <p>Time to start round {competitorState.currentRound + 1}!</p>
            <ReadyButton sendReady={props.sendReady} />
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
            <ReadyButton sendReady={props.sendReady} />
          </>
        );
      }
      case TourneyStatus.ROUND_OPPONENT_WAITING: {
        //Todo: Button should send ready message
        return (
          <>
            <p>Your opponent is waiting!</p>
            <ReadyButton sendReady={props.sendReady} />
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
              Good luck in your round {competitorState.currentRound + 1} game!
            </p>
          </>
        );
      }
      case TourneyStatus.ROUND_GAME_FINISHED: {
        return (
          <>
            <ClockCircleOutlined />
            <p>
              Your round {competitorState.currentRound + 1} score has been
              recorded.
              <span className="optional">
                {!isLastRound ? " Good luck in your next game!" : ""}
              </span>
            </p>
          </>
        );
      }
      case TourneyStatus.POSTTOURNEY:
        return (
          <>
            <ClockCircleOutlined />
            <p>
              Thanks so much for playing in the{" "}
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
  }, [tournamentContext, competitorState, props.sendReady]);
  return (
    <Card className={`competitor-status ${competitorState.status}`}>
      {renderStatus()}
    </Card>
  );
};
