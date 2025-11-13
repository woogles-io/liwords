// The timer controller should be a mostly stand-alone state, for performance sake.
// This code is heavily based on the AGPL-licensed timer controller code
// for lichess (https://github.com/ornicar/lila)
// You rock Thibault

import { PlayerOrder } from "./constants";
// import { GameState } from './reducers/game_reducer';
import { PlayState } from "../gen/api/vendor/macondo/macondo_pb";

const positiveShowTenthsCutoff = 10000;
const negativeShowTenthsCutoff = -1000;

export type Seconds = number;
export type Centis = number;
export type Millis = number;

export interface ClockData {
  running: boolean;
  initial: Seconds;
  increment: Seconds;
  p1: Seconds; // index 0
  p2: Seconds; // index 1
  emerg: Seconds;
  showTenths: boolean;
  moretime: number;
}

export const millisToTimeStr = (
  ms: number,
  showTenths = true,
  daysDecimalPlaces = 1,
): string => {
  const neg = ms < 0;
  const absms = Math.abs(ms);

  // Calculate total seconds first
  let totalSecs;
  if (!neg) {
    totalSecs = Math.ceil(absms / 1000);
  } else {
    totalSecs = Math.floor(absms / 1000);
  }

  // > 24 hours: show as "X.X days" (calculate directly from seconds for precision)
  const totalHours = totalSecs / 3600;
  if (totalHours >= 24) {
    const days = totalHours / 24;
    const daysStr = days.toFixed(daysDecimalPlaces);
    return `${neg ? "-" : ""}${daysStr} ${daysStr === "1.0" && daysDecimalPlaces === 1 ? "day" : "days"}`;
  }

  // >= 1 hour: show as "hh:mm:ss"
  if (totalHours >= 1) {
    const hours = Math.floor(totalHours);
    const mins = Math.floor((totalSecs % 3600) / 60);
    const secs = totalSecs % 60;
    const hh = hours.toString().padStart(2, "0");
    const mm = mins.toString().padStart(2, "0");
    const ss = secs.toString().padStart(2, "0");
    return `${neg ? "-" : ""}${hh}:${mm}:${ss}`;
  }

  const totalMins = Math.floor(totalSecs / 60);

  // < 1 hour: show as "mm:ss" or "mm:ss.d" (original behavior)
  let secs;
  let secStr;
  let mins;
  // Show tenths for (negativeShowTenthsCutoff, positiveShowTenthsCutoff].
  // Both cases round up, so when counting down the string changes at exact multiples of 100ms or 1000ms.
  // As a special case, (while 1 to 100 is "00:00.1") 0 is "00:00.0" but -1 to -99 is "-00:00.0".
  if (
    ms > positiveShowTenthsCutoff ||
    ms <= negativeShowTenthsCutoff ||
    !showTenths
  ) {
    secs = totalSecs % 60;
    mins = Math.floor(totalSecs / 60);
    secStr = secs.toString().padStart(2, "0");
  } else {
    let totalDecisecs;
    if (!neg) {
      totalDecisecs = Math.ceil(absms / 100);
    } else {
      totalDecisecs = Math.floor(absms / 100);
    }
    secs = totalDecisecs % 600;
    mins = Math.floor(totalDecisecs / 600);
    // Avoid using .toFixed(1), which elicits floating-point off-by-one errors.
    secStr = secs.toString().padStart(3, "0");
    const dot = secStr.length - 1;
    secStr = secStr.substr(0, dot) + "." + secStr.substr(dot);
  }
  const minStr = mins.toString().padStart(2, "0");
  return `${neg ? "-" : ""}${minStr}:${secStr}`;
};

export type Times = {
  p0: Millis;
  p1: Millis;
  p0TimeBank?: Millis; // Time bank remaining for player 0 (correspondence games)
  p1TimeBank?: Millis; // Time bank remaining for player 1 (correspondence games)
  p0UsingTimeBank?: boolean; // True when player 0 is counting from time bank
  p1UsingTimeBank?: boolean; // True when player 1 is counting from time bank
  activePlayer?: PlayerOrder; // the index of the player
  lastUpdate: Millis;
};

const minsToMillis = (m: number) => {
  return (m * 60000) as Millis;
};

export class ClockController {
  times: Times;

  private tickCallback?: number;

  onTimeout: (activePlayer: PlayerOrder) => void;

  onTick: (p: PlayerOrder, t: Millis) => void;

  maxOvertimeMinutes: number;

  constructor(
    ts: Times,
    onTimeout: (activePlayer: PlayerOrder) => void,
    onTick: (p: PlayerOrder, t: Millis) => void,
  ) {
    this.times = { ...ts };
    this.onTimeout = onTimeout;
    this.onTick = onTick;
    this.setClock(PlayState.PLAYING, this.times);
    this.maxOvertimeMinutes = 0;
  }

  setClock = (playState: number, ts: Times, delay: Centis = 0) => {
    const isClockRunning = playState !== PlayState.GAME_OVER;
    const delayMs = delay * 10;

    this.times = {
      ...ts,
      activePlayer: isClockRunning ? ts.activePlayer : undefined,
      lastUpdate: performance.now() + delayMs,
    };

    if (isClockRunning && this.times.activePlayer) {
      // Update the display immediately so we don't show 00:00
      this.onTick(this.times.activePlayer, this.times[this.times.activePlayer]);

      this.scheduleTick(this.times[this.times.activePlayer], delayMs);
    }
  };

  setMaxOvertime = (maxOTMinutes: number | undefined) => {
    this.maxOvertimeMinutes = maxOTMinutes || 0;
  };

  stopClock = (): Millis | null => {
    const { activePlayer } = this.times;
    if (activePlayer) {
      const curElapse = this.elapsed();
      this.times[activePlayer] = Math.max(
        -minsToMillis(this.maxOvertimeMinutes),
        this.times[activePlayer] - curElapse,
      );
      this.times.activePlayer = undefined;
      return curElapse;
    }
    return null;
  };

  private scheduleTick = (time: Millis, extraDelay: Millis) => {
    if (this.tickCallback !== undefined) {
      clearTimeout(this.tickCallback);
    }

    const totalMins = Math.floor(Math.abs(time) / 60000);
    const totalHours = Math.floor(totalMins / 60);

    let delay; // millis to next millisToTimeStr change.

    // For times >= 24 hours (shown as days), tick every ten seconds
    if (totalHours >= 24) {
      delay = 10000;
    } else if (time > positiveShowTenthsCutoff) {
      // 1000ms resolution, non-negative remainder.
      delay = Math.min(
        ((time + 999) % 1000) + 1,
        time - positiveShowTenthsCutoff,
      );
    } else if (time >= 0) {
      // 100ms resolution, non-negative remainder.
      delay = Math.min(((time + 99) % 100) + 1, time - -1);
    } else if (time > negativeShowTenthsCutoff) {
      // 100ms resolution, negative remainder.
      delay = Math.min((time % 100) + 100, time - negativeShowTenthsCutoff);
    } else {
      // 1000ms resolution, negative remainder.
      delay = (time % 1000) + 1000;
    }

    // Some browser versions round down the timeout, this causes the same
    // second to be displayed twice before decrementing by two at once.
    delay = Math.ceil(delay);

    this.tickCallback = window.setTimeout(
      this.tick,
      delay + Math.max(extraDelay, 0),
    );
  };

  // Should only be invoked by scheduleTick.
  private tick = (): void => {
    this.tickCallback = undefined;
    const { activePlayer } = this.times;

    if (activePlayer === undefined) {
      return;
    }

    const now = performance.now();
    const elapsed = this.elapsed(now);
    let millis = Math.max(
      -minsToMillis(this.maxOvertimeMinutes),
      this.times[activePlayer] - elapsed,
    );

    // Check if we're hit 0 and have a time bank
    const timeBankKey = activePlayer === "p0" ? "p0TimeBank" : "p1TimeBank";
    const usingTimeBankKey =
      activePlayer === "p0" ? "p0UsingTimeBank" : "p1UsingTimeBank";
    const timeBank = this.times[timeBankKey];

    // Check if we're already using time bank (persistent state)
    if (this.times[usingTimeBankKey]) {
      // Already in time bank mode - just count down normally
      millis = this.times[activePlayer] - elapsed;
    } else if (millis <= 0 && timeBank && timeBank > 0) {
      // First time entering time bank - reset the clock
      this.times[activePlayer] = timeBank + millis; // Account for overage
      this.times.lastUpdate = now;
      this.times[usingTimeBankKey] = true;
      millis = this.times[activePlayer];
    }

    this.onTick(activePlayer, millis);

    if (millis > -minsToMillis(this.maxOvertimeMinutes)) {
      this.scheduleTick(millis, 0);
    } else {
      // we timed out (both main time and time bank exhausted).
      this.onTimeout(activePlayer);
    }
  };

  elapsed = (now = performance.now()) =>
    Math.max(
      -minsToMillis(this.maxOvertimeMinutes),
      now - this.times.lastUpdate,
    );

  millisOf = (p: PlayerOrder): Millis => {
    if (this.times.activePlayer !== p) {
      return this.times[p];
    }

    const elapsed = this.elapsed();
    let millis = Math.max(
      -minsToMillis(this.maxOvertimeMinutes),
      this.times[p] - elapsed,
    );

    // Check if we've hit 0 and have a time bank
    const timeBankKey = p === "p0" ? "p0TimeBank" : "p1TimeBank";
    const usingTimeBankKey = p === "p0" ? "p0UsingTimeBank" : "p1UsingTimeBank";
    const timeBank = this.times[timeBankKey];

    if (millis <= 0 && timeBank && timeBank > 0) {
      // If we're using time bank, just return the current time
      // (tick() already handles the transition)
      if (this.times[usingTimeBankKey]) {
        millis = this.times[p] - elapsed;
      } else {
        millis = timeBank + millis;
      }
    }

    return millis;
  };
}
