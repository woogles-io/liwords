import React from "react";
import { Tooltip } from "antd";
import { ClockCircleOutlined, FieldTimeOutlined } from "@ant-design/icons";
import {
  OnTurnCountdowns,
  formatCoarseDuration,
  formatCoarseDurationShort,
} from "../utils/time_bank_calculator";

// "Your turn" is always shown in this red so a glance at the list (especially
// on mobile) tells you which games need a move. The low-time and time-bank
// escalations ride on icons rather than on toggling the red on and off -- that
// toggling was the regression: the bank-aware deadline almost never dipped
// under the 24h flag, so the red (which players relied on) effectively vanished.
const YOUR_TURN_COLOR = "#ff4d4f";
// Matches the in-game clock's time-bank color, so "draining your bank" reads
// the same here as it does on the board.
const TIME_BANK_COLOR = "#52c41a";
// Under a day to a forced timeout: surface the urgency with a clock icon.
const LOW_TIME_SECS = 86400;

type Props = {
  // Whether it is the logged-in user's turn in this game.
  onTurn: boolean;
  // Bank-aware countdowns for whoever is on turn; undefined when unknown.
  countdowns?: OnTurnCountdowns;
  // Whether the on-turn player still has time bank left to drain. Non-league
  // correspondence games typically have no bank, so they never "bleed" -- they
  // simply time out when the per-turn allowance runs out.
  hasTimeBank?: boolean;
};

// Condensed turn / time indicator shared by the lobby correspondence list and
// the league "My League Games" card. The inline text stays terse -- "Your turn"
// (red) plus a single coarse time unit -- while the verbose "expires in X, free
// for Y" detail lives in the tooltip.
export const CorrespondenceTurnIndicator = (props: Props) => {
  const { onTurn, countdowns, hasTimeBank } = props;

  if (!onTurn) {
    // Opponent's clock -- greyed, since it is not the user's deadline.
    const short = countdowns
      ? formatCoarseDurationShort(countdowns.beforeExpiry)
      : null;
    const tooltip = countdowns
      ? `Opponent's clock (not your deadline). Times out in ${formatCoarseDuration(
          countdowns.beforeExpiry,
        )}.`
      : "Opponent's turn.";
    return (
      <span className="corres-turn their-turn" style={{ opacity: 0.55 }}>
        <Tooltip title={tooltip}>
          Their turn
          {short ? (
            <span className="corres-turn-time" style={{ marginLeft: 6 }}>
              {short}
            </span>
          ) : null}
        </Tooltip>
      </span>
    );
  }

  // Only a game with bank still to spend can "bleed"; a no-bank correspondence
  // game just times out when its per-turn allowance runs out.
  const isBleeding =
    !!countdowns && !!hasTimeBank && countdowns.beforeBleed <= 0;
  const isLowTime = countdowns
    ? countdowns.beforeExpiry < LOW_TIME_SECS
    : false;
  const short = countdowns
    ? formatCoarseDurationShort(countdowns.beforeExpiry)
    : null;
  const tooltip = !countdowns
    ? "Your turn."
    : isBleeding
      ? `Using your time bank -- times out in ${formatCoarseDuration(
          countdowns.beforeExpiry,
        )}.`
      : hasTimeBank
        ? `Times out in ${formatCoarseDuration(
            countdowns.beforeExpiry,
          )} (free for ${formatCoarseDuration(
            countdowns.beforeBleed,
          )} before your time bank starts draining).`
        : `Times out in ${formatCoarseDuration(countdowns.beforeExpiry)}.`;

  return (
    <span className="corres-turn your-turn">
      <Tooltip title={tooltip}>
        {isBleeding ? (
          <FieldTimeOutlined
            className="corres-turn-icon"
            style={{ color: TIME_BANK_COLOR, marginRight: 4 }}
          />
        ) : isLowTime ? (
          <ClockCircleOutlined
            className="corres-turn-icon"
            style={{ color: YOUR_TURN_COLOR, marginRight: 4 }}
          />
        ) : null}
        <span style={{ color: YOUR_TURN_COLOR }}>Your turn</span>
        {short ? (
          <span
            className="corres-turn-time"
            style={{ marginLeft: 6, opacity: 0.7 }}
          >
            {short}
          </span>
        ) : null}
      </Tooltip>
    </span>
  );
};
