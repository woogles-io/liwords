import React, { useMemo, useState } from "react";
import { useTournamentStoreContext } from "../store/store";
import {
  Division,
  SinglePairing,
  TournamentState,
} from "../store/reducers/tournament_reducer";
import { Button, Divider, Form, InputNumber, message } from "antd";
import { flashError, useClient } from "../utils/hooks/connect";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { TournamentGameResult } from "../gen/api/proto/ipc/tournament_pb";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { ActionsPanel } from "./actions_panel";

type Props = {
  truncatedID: string;
};

function findPlayersByTruncatedId(data: TournamentState, truncatedId: string) {
  // Scorecard QR codes hold the shortest player-ID prefix that was unique
  // when they were printed. A player added after printing can share that
  // prefix, and a removed player's code matches nobody, so return every
  // match rather than assuming there is exactly one.
  const matches = [];
  for (const [divisionName, division] of Object.entries(data.divisions)) {
    for (let idx = 0; idx < division.players.length; idx++) {
      const player = division.players[idx];
      if (player.id.startsWith(truncatedId)) {
        matches.push({ division: divisionName, index: idx });
      }
    }
  }
  return matches;
}

type ShowResultsProps = {
  tournamentID: string;
  userID: string;
  autoshow?: boolean;
};

const ShowResults = (props: ShowResultsProps) => {
  const [show, setShow] = useState(props.autoshow ?? false);
  const [selectedGameTab, setSelectedGameTab] = useState("GAMES");

  return (
    <>
      <a className="link" onClick={() => setShow((v) => !v)}>
        See pairings and standings
      </a>
      <div hidden={!show}>
        <ActionsPanel
          selectedGameTab={selectedGameTab}
          setSelectedGameTab={setSelectedGameTab}
          isDirector={false}
          canManageTournaments={false}
          tournamentID={props.tournamentID}
          onSeekSubmit={() => {}}
          loggedIn={false}
          newGame={() => {}}
          username=""
          userID={props.userID}
          sendReady={() => false}
          showFirst
        />
      </div>
    </>
  );
};

type SelfPairingProps = {
  currentRound: number;
  outcome: TournamentGameResult;
};

const SelfPairingDisplay = (props: SelfPairingProps) => {
  // self-pairing shouldn't show a form.
  let toshow;
  switch (props.outcome) {
    case TournamentGameResult.BYE:
      toshow = `You have a bye for round ${props.currentRound + 1}`;
      break;
    case TournamentGameResult.FORFEIT_LOSS:
      toshow = `You have a forfeit loss for round ${props.currentRound + 1}`;
      break;
    case TournamentGameResult.FORFEIT_WIN:
      toshow = `You have a forfeit win for round ${props.currentRound + 1}`;
      break;
    default:
      toshow = `Unhandled game result for round ${props.currentRound + 1}: ${
        props.outcome
      }`;
  }
  return <h4 className="readable-text-color">{toshow}</h4>;
};

type ScoreFormProps = {
  opponentName: string;
  currentRound: number;
  first: boolean;
  division: Division;
  pairing: SinglePairing;
  ourName: string;
  ourSeed: number;
  opponentSeed?: number;
};

const ScoreForm = (props: ScoreFormProps) => {
  const tClient = useClient(TournamentService);
  const [submitDisabled, setSubmitDisabled] = useState(true);
  return (
    <>
      <h4 className="readable-text-color">
        You are playing {props.opponentName} ({props.opponentSeed}) in round{" "}
        {props.currentRound + 1}.
      </h4>

      <h4>
        You are going <strong>{props.first ? "first" : "second"}</strong>.
      </h4>
      <Divider />

      <h4 className="readable-text-color">
        When your game is over, enter your final scores:
      </h4>
      <Form
        layout="vertical"
        size="large"
        wrapperCol={{ span: 6 }}
        labelCol={{ span: 6 }}
        onValuesChange={(changedValues, values) => {
          setSubmitDisabled(
            values.ourscore == undefined || values.theirscore == undefined,
          );
        }}
        onFinish={async (values) => {
          // if we're first, then we are player 1 (the one first listed)
          const p1score = props.first ? values.ourscore : values.theirscore;
          const p2score = props.first ? values.theirscore : values.ourscore;
          const p1Result =
            p1score > p2score
              ? TournamentGameResult.WIN
              : p1score < p2score
                ? TournamentGameResult.LOSS
                : TournamentGameResult.DRAW;
          const p2Result =
            p1score > p2score
              ? TournamentGameResult.LOSS
              : p1score < p2score
                ? TournamentGameResult.WIN
                : TournamentGameResult.DRAW;

          const obj = {
            id: props.division.tournamentID,
            division: props.division.divisionID,
            playerOneId: props.pairing.players[0].id.split(":")[0],
            playerTwoId: props.pairing.players[1].id.split(":")[0],
            round: props.division.currentRound,
            playerOneScore: p1score,
            playerTwoScore: p2score,
            playerOneResult: p1Result,
            playerTwoResult: p2Result,
            gameEndReason: GameEndReason.STANDARD,
            amendment: false,
          };
          try {
            await tClient.setResult(obj);
            message.info({
              content: "Result submitted",
              duration: 3,
            });
          } catch (e) {
            flashError(e);
          }
        }}
      >
        <Form.Item
          label={`Score for ${props.ourName}`}
          name="ourscore"
          required
        >
          <InputNumber
            inputMode="numeric"
            style={{ width: 200, height: 45, fontSize: 24 }}
            placeholder="Click / touch..."
          />
        </Form.Item>
        <Divider />
        <Form.Item
          label={`Score for ${props.opponentName}`}
          name="theirscore"
          required
        >
          <InputNumber
            inputMode="numeric"
            style={{ width: 200, height: 45, fontSize: 24 }}
            placeholder="Click / touch..."
          />
        </Form.Item>

        <Form.Item label="Please have opponent verify scores.">
          <Button
            style={{ margin: 0 }}
            type="primary"
            htmlType="submit"
            disabled={submitDisabled}
          >
            Submit scores
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};

export const OwnScoreEnterer = (props: Props) => {
  const { tournamentContext } = useTournamentStoreContext();

  const matches = useMemo(
    () => findPlayersByTruncatedId(tournamentContext, props.truncatedID),
    [tournamentContext, props.truncatedID],
  );

  if (Object.keys(tournamentContext.divisions).length === 0) {
    // Still loading.
    return <></>;
  }
  if (matches.length !== 1) {
    return (
      <h4 className="readable-text-color" style={{ marginLeft: 20 }}>
        {matches.length === 0
          ? "This scorecard link is no longer valid."
          : "This scorecard link matches more than one player."}{" "}
        Please ask a director for a new scorecard.
      </h4>
    );
  }
  const player = matches[0];
  const division = tournamentContext.divisions[player.division];
  const foundPlayer = division.players[player.index];
  const fullName = foundPlayer.id.split(":")[1];
  const md5ID = foundPlayer.id.split(":")[0];
  // determine context of what to show.
  // 1) Tourney hasn't started yet
  // 2) We are in round X, but we have not entered a score yet
  // 3) We are in round X, and have already entered a score for it.

  if (!tournamentContext.started) {
    return (
      <h4 className="readable-text-color">Tournament has not yet started.</h4>
    );
  }
  // 0-indexed of course:
  const currentRound = division.currentRound;
  const pairing = division.pairings[currentRound].roundPairings.find(
    (pairing) => {
      if (!pairing || !pairing.players) {
        return false;
      }
      return (
        pairing.players[0].id === foundPlayer.id ||
        pairing.players[1].id === foundPlayer.id
      );
    },
  );
  let opponent, opponentName, opponentSeed;
  const ourSeed = player.index + 1;
  if (!pairing) {
    return (
      <h4 className="readable-text-color">
        Pairing not found for round {division.currentRound + 1} and player{" "}
        {fullName}
      </h4>
    );
  }
  let first = false;
  if (pairing.players[0].id === foundPlayer.id) {
    opponent = pairing.players[1].id;
    opponentName = opponent.split(":")[1];
    opponentSeed = division.playerIndexMap[pairing.players[1].id] + 1;

    first = true;
  } else if (pairing.players[1].id === foundPlayer.id) {
    opponent = pairing.players[0].id;
    opponentName = opponent.split(":")[1];
    opponentSeed = division.playerIndexMap[pairing.players[0].id] + 1;
  }

  let display = null;
  if (!opponentName) {
    return <>Unexpected opponentName not found</>;
  }

  if (pairing.games[0].gameEndReason === 0) {
    // game is ongoing
    if (pairing.players[0].id === pairing.players[1].id) {
      display = (
        <SelfPairingDisplay
          outcome={pairing.outcomes[0]}
          currentRound={currentRound}
        />
      );
    } else {
      display = (
        <ScoreForm
          opponentName={opponentName}
          opponentSeed={opponentSeed}
          ourName={fullName}
          ourSeed={ourSeed}
          currentRound={currentRound}
          first={first}
          division={division}
          pairing={pairing}
        />
      );
    }

    return (
      <div style={{ marginLeft: 20, marginTop: 10 }}>
        <h4 className="readable-text-color">
          Hi, {fullName} ({ourSeed}).
        </h4>
        {display}
        <Divider />
        <ShowResults
          tournamentID={tournamentContext.metadata.id}
          userID={md5ID}
        />
      </div>
    );
  }
  // otherwise the game is over
  const myscore = first
    ? pairing.games[0].scores[0]
    : pairing.games[0].scores[1];
  const theirscore = first
    ? pairing.games[0].scores[1]
    : pairing.games[0].scores[0];
  return (
    <div style={{ marginLeft: 20, marginTop: 10 }}>
      <h4 className="readable-text-color">
        Hi, {fullName} ({ourSeed}).
      </h4>

      <h4 className="readable-text-color">
        You played {opponent?.split(":")[1]} ({opponentSeed}) in round{" "}
        {division.currentRound + 1}.
      </h4>
      <h4>
        You scored {myscore} and your opponent scored {theirscore}.
      </h4>

      <h4 className="readable-text-color">
        If this is not right, please contact a director.
      </h4>

      <Divider />
      <ShowResults
        tournamentID={tournamentContext.metadata.id}
        userID={md5ID}
        autoshow
      />
    </div>
  );
};
