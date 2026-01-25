import React, { useState } from "react";
import { Link } from "react-router";
import { Card, Row, Button, Tooltip, Modal } from "antd";
import { HourglassOutlined } from "@ant-design/icons";
import { RawPlayerInfo } from "../store/reducers/game_reducer";
import {
  useExaminableGameContextStoreContext,
  useExaminableTimerStoreContext,
  useExamineStoreContext,
} from "../store/store";
import {
  Millis,
  millisToTimeStr,
  millisToTimeStrWithoutDays,
} from "../store/timer_controller";
import { PlayerAvatar } from "../shared/player_avatar";
import "./scss/playerCards.scss";
import { PlayState } from "../gen/api/vendor/macondo/macondo_pb";
import { DisplayUserFlag } from "../shared/display_flag";
import { useBriefProfile } from "../utils/brief_profiles";
import {
  GameInfoResponse,
  GameMode,
  PlayerInfo,
} from "../gen/api/proto/ipc/omgwords_pb";
import { MachineLetter } from "../utils/cwgame/common";

import variables from "../base.module.scss";
import { DisplayUserBadges } from "../profile/badge";
import { DisplayUserTitle } from "../shared/display_title";
const { colorPrimary } = variables;

type CardProps = {
  player: RawPlayerInfo | undefined;
  time: Millis;
  initialTimeSeconds: Millis;
  meta: Array<PlayerInfo>;
  playing: boolean;
  score: number;
  spread: number;
  hideProfileLink?: boolean;
  timeBank?: number | bigint;
  usingTimeBank?: boolean;
};

const timepenalty = (time: Millis) => {
  // Calculate a timepenalty for display purposes only. The backend will
  // also properly calculate this.

  if (time >= 0) {
    return 0;
  }

  const minsOvertime = Math.ceil(Math.abs(time) / 60000);
  return minsOvertime * 10;
};

// same as scorecard.tsx
const makeTimeRemainingFragment = (
  timeRemainingMillis: number,
  showTenths = true,
) => {
  const timeRemainingWithDays = millisToTimeStr(
    timeRemainingMillis,
    showTenths,
  );
  return (
    <Tooltip
      title={
        timeRemainingWithDays.includes("day")
          ? millisToTimeStrWithoutDays(timeRemainingMillis, showTenths)
          : null
      }
    >
      {timeRemainingWithDays}
    </Tooltip>
  );
};

const PlayerCard = React.memo((props: CardProps) => {
  const { isExamining } = useExamineStoreContext();
  const briefProfile = useBriefProfile(props.player?.userID);
  const [showTimeBankModal, setShowTimeBankModal] = useState(false);

  if (!props.player) {
    return <Card />;
  }
  // Find the metadata for this player.
  const meta = props.meta.find((pi) => pi.userId === props.player?.userID);
  const timeRemainingFragment =
    isExamining || props.playing
      ? makeTimeRemainingFragment(props.time)
      : "--:--";

  // Check if we have a time bank
  const hasTimeBank = props.timeBank !== undefined && props.timeBank > 0;
  const timeBankMs =
    typeof props.timeBank === "bigint"
      ? Number(props.timeBank)
      : props.timeBank || 0;

  // Format time bank for tooltip
  const formatTimeBankTooltip = (ms: number): React.ReactNode => {
    return makeTimeRemainingFragment(ms, false);
  };

  // Check if we're counting from time bank (tracked by ClockController)
  const inTimeBank = props.usingTimeBank ?? false;

  // TODO: what we consider low time likely be set somewhere and not a magic number
  const timeLowCutoff = Math.max(props.initialTimeSeconds / 5, 30000);
  const timeLow = props.time <= timeLowCutoff && props.time > 0 && !inTimeBank;
  const timeOut = props.time <= 0;

  const nickname = meta?.nickname ?? "";
  const shownName = briefProfile?.fullName || meta?.fullName || nickname;

  return (
    <div
      className={`player-card${props.player.onturn ? " on-turn" : ""}
      ${timeLow ? " time-low" : ""}${timeOut ? " time-out" : ""}${inTimeBank ? " using-time-bank" : ""}`}
    >
      <Row className="player">
        <PlayerAvatar player={meta} />
        <div className="player-info">
          <p className="player-name">
            <Tooltip title={nickname === shownName ? null : nickname}>
              {shownName}
            </Tooltip>
            <DisplayUserTitle uuid={props.player.userID} />
            <DisplayUserBadges uuid={props.player.userID} />
          </p>
          <div className="player-details">
            <DisplayUserFlag uuid={props.player.userID} />
            {meta?.rating || "Unrated"}
            {props.hideProfileLink ? null : (
              <>
                {" "}
                •{" "}
                <Link
                  target="_blank"
                  to={`/profile/${encodeURIComponent(nickname)}`}
                >
                  {nickname === shownName ? "View profile" : nickname}
                </Link>
              </>
            )}
          </div>
        </div>
      </Row>
      <Row className="score-timer">
        <Tooltip
          placement="left"
          color={colorPrimary}
          title={`${props.spread >= 0 ? "+" : ""}${props.spread}`}
        >
          <Button className="score" type="primary">
            {props.score}
          </Button>
        </Tooltip>
        <div className="timer-container">
          <Button className="timer" type="primary">
            {timeRemainingFragment}
            {(() => {
              const shouldShow = hasTimeBank && !inTimeBank && props.time > 0;

              return shouldShow ? (
                <>
                  <Tooltip
                    title={
                      <span>
                        Time bank: {formatTimeBankTooltip(timeBankMs)}
                        <br />
                        <a
                          onClick={(e) => {
                            e.preventDefault();
                            setShowTimeBankModal(true);
                          }}
                          style={{
                            color: "#40a9ff",
                            textDecoration: "underline",
                            cursor: "pointer",
                          }}
                        >
                          How does time bank work?
                        </a>
                      </span>
                    }
                    trigger="hover"
                  >
                    <span>
                      <HourglassOutlined className="time-bank-indicator" />
                    </span>
                  </Tooltip>
                  <Modal
                    title="How does time bank work?"
                    open={showTimeBankModal}
                    onCancel={() => setShowTimeBankModal(false)}
                    footer={null}
                    width={600}
                  >
                    <div style={{ lineHeight: "1.6" }}>
                      <p>
                        <strong>Time bank</strong> is an additional time reserve
                        available for correspondence games.
                      </p>
                      <p>
                        When you use up your main thinking time for a turn, the
                        game will automatically start using your time bank. Your
                        time bank depletes only when you exceed the per-turn
                        time.
                      </p>
                      <p>
                        <strong>Example:</strong> If you have 8 hours of
                        per-turn time and take 12 hours to make a move, only 4
                        hours will be deducted from your time bank.
                      </p>
                      <p>
                        The <HourglassOutlined style={{ color: "#15803d" }} />{" "}
                        icon appears when you still have time bank available but
                        are not currently using it. When you're actively using
                        your time bank, you'll see "using time bank" displayed
                        below your timer.
                      </p>
                      <p>
                        <strong>Important:</strong> If your time bank runs out,
                        you will lose the game on time.
                      </p>
                    </div>
                  </Modal>
                </>
              ) : null;
            })()}
          </Button>
          {inTimeBank && <div className="time-bank-label">using time bank</div>}
        </div>
      </Row>
    </div>
  );
});

type Props = {
  gameMeta: GameInfoResponse;
  playerMeta: Array<PlayerInfo>;
  horizontal?: boolean;
  hideProfileLink?: boolean;
};

export const PlayerCards = React.memo((props: Props) => {
  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { timerContext: examinableTimerContext } =
    useExaminableTimerStoreContext();
  const { isExamining } = useExamineStoreContext();

  // If the gameContext is not yet available, we should try displaying player cards
  // from the meta information, until the information comes in.
  let p0 = examinableGameContext?.players[0];
  let p1 = examinableGameContext?.players[1];
  if (!p0) {
    if (props.playerMeta[0]) {
      p0 = {
        userID: props.playerMeta[0].userId,
        score: 0,
        onturn: false,
        currentRack: new Array<MachineLetter>(),
      };
    }
  }

  if (!p1) {
    if (props.playerMeta[1]) {
      p1 = {
        userID: props.playerMeta[1].userId,
        score: 0,
        onturn: false,
        currentRack: new Array<MachineLetter>(),
      };
    }
  }

  const initialTimeSeconds =
    (props.gameMeta.gameRequest?.initialTimeSeconds ?? 0) * 1000;

  const playing = examinableGameContext.playState !== PlayState.GAME_OVER;
  const isCorrespondence =
    props.gameMeta.gameRequest?.gameMode === GameMode.CORRESPONDENCE;

  // Use timer context for both real-time and correspondence games
  // setClock updates this on every move event, so it stays current
  let p0Time: number = examinableTimerContext.p0;
  if (p0Time === Infinity) p0Time = initialTimeSeconds;
  let p1Time: number = examinableTimerContext.p1;
  if (p1Time === Infinity) p1Time = initialTimeSeconds;

  // Get time bank values for correspondence games
  const p0TimeBank = examinableTimerContext.p0TimeBank;
  const p1TimeBank = examinableTimerContext.p1TimeBank;
  const p0UsingTimeBank = examinableTimerContext.p0UsingTimeBank;
  const p1UsingTimeBank = examinableTimerContext.p1UsingTimeBank;

  const applyTimePenalty = !isExamining && playing && !isCorrespondence;
  let p0Score = p0?.score ?? 0;
  if (applyTimePenalty) p0Score -= timepenalty(p0Time);
  let p1Score = p1?.score ?? 0;
  if (applyTimePenalty) p1Score -= timepenalty(p1Time);
  const p0Spread = p0Score - p1Score;

  return (
    <Card
      className={`player-cards${
        props.horizontal ? " horizontal" : " vertical"
      }`}
      id={`player-cards-${props.horizontal ? "horizontal" : "vertical"}`}
    >
      <PlayerCard
        player={p0}
        meta={props.playerMeta}
        time={p0Time}
        initialTimeSeconds={initialTimeSeconds}
        score={p0Score}
        spread={p0Spread}
        playing={playing}
        hideProfileLink={props.hideProfileLink}
        timeBank={p0TimeBank}
        usingTimeBank={p0UsingTimeBank}
      />
      <PlayerCard
        player={p1}
        meta={props.playerMeta}
        time={p1Time}
        initialTimeSeconds={initialTimeSeconds}
        score={p1Score}
        spread={-p0Spread}
        playing={playing}
        hideProfileLink={props.hideProfileLink}
        timeBank={p1TimeBank}
        usingTimeBank={p1UsingTimeBank}
      />
    </Card>
  );
});
