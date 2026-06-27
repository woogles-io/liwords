import React, { useCallback, useEffect, useState } from "react";
import { Tooltip } from "antd";
import { ClockCircleOutlined, FieldTimeOutlined } from "@ant-design/icons";
import { SimpleTimer } from "../gameroom/simple_timer";
import { millisToTimeStrWithoutDays } from "../store/timer_controller";

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
const LOW_TIME_MS = 86400 * 1000;
// setTimeout caps delays at ~24.8 days (2^31-1 ms); cap our transition timers
// at that and let them reschedule. A correspondence deadline never reaches that
// far out, so the cap is only a defensive backstop.
const MAX_TIMEOUT_MS = 2147483647;

// The shared "d:hh:mm:ss" clock (degrading to hh:mm:ss / mm:ss under a day /
// hour), without subseconds. The same formatter the in-game clock uses, so the
// list and the board read identically.
const fmtClock = (millis: number) => millisToTimeStrWithoutDays(millis, false);

// Perspective controls only the label, colour and the tooltip's possessive
// subject -- never the ticking. "mine"/"opponent" are the participant views (the
// league card and lobby list, which only ever show the viewer's own games);
// "spectator" is the third-person view for the all-players standings popup.
// "bare" shows just the ticking time (no turn label) for a surface that already
// identifies the player and whose turn it is (the season-games modal's Time
// column); the tooltip still names the player, since a floating tooltip does not
// say which row it belongs to.
export type CorrespondencePerspective =
  | { kind: "mine" }
  | { kind: "opponent"; playerName: string }
  | { kind: "spectator"; playerName: string }
  | { kind: "bare"; playerName: string };

type Props = {
  perspective: CorrespondencePerspective;
  // The on-turn player's clock anchor: the epoch ms of their last move, their
  // per-turn allowance (ms) and their remaining time bank (ms). When any is
  // undefined the clock is unknown and only the turn label is shown.
  lastUpdateMs?: number;
  incrementMs?: number;
  bankMs?: number;
};

// Turn / time indicator shared by the lobby correspondence list and the league
// "My League Games" card. The inline text is the label plus a live "d:hh:mm:ss"
// countdown to the hard deadline (X = per-turn allowance + bank - elapsed); the
// tooltip breaks that down into the free window (Y) and the time bank (Z = X-Y),
// each ticking. Exactly one of Y/Z drains at a time: Y until the bank starts
// bleeding, then Z.
export const CorrespondenceTurnIndicator = (props: Props) => {
  const { perspective, lastUpdateMs, incrementMs, bankMs } = props;

  const [, setRerender] = useState(0);
  const requestRerender = useCallback(
    () => setRerender((n) => (n + 1) | 0),
    [],
  );

  // Anchor both clocks at the same instant: Date.now() places us on the
  // server's epoch timeline (lastUpdateMs), while performance.now() is the
  // monotonic reference the SimpleTimers extrapolate from (immune to wall-clock
  // jumps). They are read together, so no conversion between them is needed.
  const perfNow = performance.now();
  const haveClock =
    lastUpdateMs !== undefined &&
    incrementMs !== undefined &&
    bankMs !== undefined;

  const incMs = incrementMs ?? 0;
  const bnkMs = bankMs ?? 0;
  const elapsedMs = lastUpdateMs === undefined ? 0 : Date.now() - lastUpdateMs;
  const expiryMs = incMs + bnkMs - elapsedMs; // X: hard deadline (forfeit)
  const bleedMs = Math.max(0, incMs - elapsedMs); // Y: free window before bleed
  const bankRemMs = expiryMs - bleedMs; // Z: time bank remaining
  const hasBank = haveClock && bnkMs > 0;
  const isBleeding = hasBank && bleedMs <= 0;
  const isLowTime = haveClock && expiryMs < LOW_TIME_MS;

  // Re-render at the next state transition so the icon and tooltip template
  // switch exactly when the state changes; the SimpleTimers handle the
  // per-second number ticking themselves. Schedule the soonest of: bank starts
  // bleeding (Y -> 0), low-time (X crosses 24h) and timeout (X -> 0). The delay
  // is computed off the perfNow anchor so it stays precise, and capped at the
  // setTimeout maximum (it then reschedules harmlessly).
  useEffect(() => {
    if (!haveClock) return;
    const transitions = [expiryMs];
    if (hasBank && !isBleeding) transitions.push(bleedMs);
    if (!isLowTime) transitions.push(expiryMs - LOW_TIME_MS);
    const next = Math.min(...transitions.filter((t) => t > 0));
    if (!Number.isFinite(next)) return;
    const delay = Math.min(
      Math.max(next - (performance.now() - perfNow), 0),
      MAX_TIMEOUT_MS,
    );
    const t = window.setTimeout(requestRerender, delay);
    return () => window.clearTimeout(t);
  });

  let label: string;
  let labelColor: string | undefined;
  let outerOpacity: number | undefined;
  let timeOpacity: number | undefined;
  let subject: string; // possessive tooltip subject
  let turnClass: string;
  switch (perspective.kind) {
    case "mine":
      label = "Your turn";
      labelColor = YOUR_TURN_COLOR;
      timeOpacity = 0.7;
      subject = "Your";
      turnClass = "your-turn";
      break;
    case "opponent":
      label = "Their turn";
      outerOpacity = 0.55;
      subject = `${perspective.playerName}'s`;
      turnClass = "their-turn";
      break;
    case "spectator":
      label = perspective.playerName;
      timeOpacity = 0.7;
      subject = `${perspective.playerName}'s`;
      turnClass = "spectator-turn";
      break;
    case "bare":
      // No inline label -- just the ticking time. The tooltip keeps the
      // player's name so it is clear whose clock it describes.
      label = "";
      subject = `${perspective.playerName}'s`;
      turnClass = "bare-turn";
      break;
  }

  if (!haveClock) {
    return (
      <span
        className={`corres-turn ${turnClass}`}
        style={{ opacity: outerOpacity }}
      >
        <span style={{ color: labelColor }}>{label}</span>
      </span>
    );
  }

  const xTimer = (
    <SimpleTimer
      lastRefreshedPerformanceNow={perfNow}
      millisAtLastRefresh={expiryMs}
      isRunning
      format={fmtClock}
    />
  );
  // Y ticks only before bleeding; Z is steady until the bank bleeds, then ticks
  // (one timer, isRunning = isBleeding) -- so exactly one of them is moving.
  const yTimer = (
    <SimpleTimer
      lastRefreshedPerformanceNow={perfNow}
      millisAtLastRefresh={bleedMs}
      isRunning={!isBleeding}
      format={fmtClock}
    />
  );
  const zTimer = (
    <SimpleTimer
      lastRefreshedPerformanceNow={perfNow}
      millisAtLastRefresh={bankRemMs}
      isRunning={isBleeding}
      format={fmtClock}
    />
  );

  let tooltip: React.ReactNode;
  if (!hasBank) {
    // No bank (typical non-league correspondence): no bank terminology, so a
    // correspondence-only player never sees the "time bank" concept.
    tooltip = (
      <span>
        {subject} clock times out in {xTimer}
      </span>
    );
  } else if (isBleeding) {
    tooltip = (
      <span>
        {subject} time bank: {zTimer} draining
      </span>
    );
  } else {
    tooltip = (
      <span>
        {subject} free time: {yTimer}, time bank: {zTimer}
      </span>
    );
  }

  // Icons escalate the on-turn player's state. They show only on the viewer's
  // own turn for now (the participant lists); the spectator popup will decide
  // its own icon treatment.
  const icon =
    perspective.kind === "mine" ? (
      isBleeding ? (
        <FieldTimeOutlined
          className="corres-turn-icon"
          style={{ color: TIME_BANK_COLOR, marginRight: 4 }}
        />
      ) : isLowTime ? (
        <ClockCircleOutlined
          className="corres-turn-icon"
          style={{ color: YOUR_TURN_COLOR, marginRight: 4 }}
        />
      ) : null
    ) : null;

  return (
    <span
      className={`corres-turn ${turnClass}`}
      style={{ opacity: outerOpacity }}
    >
      <Tooltip title={tooltip}>
        {icon}
        {label ? <span style={{ color: labelColor }}>{label}</span> : null}
        {/* Separate label and clock with a real space, not the time span's
            margin: a space is a line-break opportunity, so a narrow column
            wraps cleanly to label / clock ("Your turn" / "10:09:33:33")
            rather than overflowing as one unbreakable run. A margin gives
            the gap but never breaks, and would indent the clock on wrap. */}
        {label ? " " : null}
        <span className="corres-turn-time" style={{ opacity: timeOpacity }}>
          {xTimer}
        </span>
      </Tooltip>
    </span>
  );
};
