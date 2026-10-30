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
import {
  isLaggedRound,
  pairingBasisRound,
} from "../store/selectors/round_selectors";

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

    // The round the player is playing now. It is behind the division's
    // current round when a lagged round opened before their previous game
    // finished.
    const round = competitorState.activeRound;
    const division = competitorState.division
      ? tournamentContext.divisions[competitorState.division]
      : undefined;

    // A lagged round that is open while this player's game in the round
    // before it is unfinished.
    const finishPreviousNote =
      division && round < division.currentRound ? (
        <span className="secondary-message">
          Finish your round {round + 1} game to start round{" "}
          {division.currentRound + 1}.
        </span>
      ) : null;

    // The player's round is lagged, so its pairings do not include the
    // results of the round they just finished.
    let laggedBasisNote = null;
    if (division && isLaggedRound(division, round)) {
      const basis = pairingBasisRound(division, round);
      laggedBasisNote = (
        <span className="secondary-message">
          {basis < 0
            ? "Its pairings are based on the initial seeding."
            : `Its pairings are based on results through round ${basis + 1}.`}
        </span>
      );
    }

    // TODO: Make return buttton to do the right thing.
    const backToGamePrompt = (
      <>
        <ClockCircleOutlined />
        <p>Your round {round + 1} game is in progress.</p>
        <Button className="primary">Return</Button>
      </>
    );
    const isLastRound = division ? round === division.numRounds - 1 : false;

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
              You forfeited your Round {round + 1} game.
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
              Your opponent forfeited their Round {round + 1} game.{" "}
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
              You have a bye for round {round + 1}.
              {!isLastRound && (
                <span className="secondary-message">
                  Please return in time for round {round + 2}.
                </span>
              )}
            </p>
          </>
        );
      }
      case TourneyStatus.ROUND_OPEN: {
        return (
          <>
            {finishPreviousNote || !laggedBasisNote ? (
              <p>
                Time to start round {round + 1}!{finishPreviousNote}
              </p>
            ) : (
              <p>
                Round {round + 1} is open. {laggedBasisNote}
              </p>
            )}
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
            <p>Your opponent is waiting!{finishPreviousNote}</p>
            <ReadyButton sendReady={props.sendReady} />
          </>
        );
      }
      case TourneyStatus.ROUND_READY: {
        //Todo: Button should send I'm not ready message
        return (
          <>
            <p>Waiting for opponent{finishPreviousNote}</p>
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
              Good luck in your round {round + 1} game!{finishPreviousNote}
            </p>
          </>
        );
      }
      case TourneyStatus.ROUND_GAME_FINISHED: {
        return (
          <>
            <ClockCircleOutlined />
            <p>
              Your round {round + 1} score has been recorded.
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
