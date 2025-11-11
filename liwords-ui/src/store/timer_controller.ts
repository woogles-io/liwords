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

export const millisToTimeStr = (ms: number, showTenths = true): string => {
  const neg = ms < 0;
  const absms = Math.abs(ms);
  // const mins = Math.floor(ms / 60000);
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
    let totalSecs;
    if (!neg) {
      totalSecs = Math.ceil(absms / 1000);
    } else {
      totalSecs = Math.floor(absms / 1000);
    }
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
    console.log("in timer controller constructor", this.times);
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
      this.scheduleTick(this.times[this.times.activePlayer], delayMs);
    }
  };

  setMaxOvertime = (maxOTMinutes: number | undefined) => {
    console.log("Set max overtime mins", maxOTMinutes);
    this.maxOvertimeMinutes = maxOTMinutes || 0;
  };

  stopClock = (): Millis | null => {
    console.log("stopClock");

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

    let delay; // millis to next millisToTimeStr change.
    if (time > positiveShowTenthsCutoff) {
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
    const millis = Math.max(
      -minsToMillis(this.maxOvertimeMinutes),
      this.times[activePlayer] - this.elapsed(now),
    );
    this.onTick(activePlayer, millis);

    if (millis !== -minsToMillis(this.maxOvertimeMinutes)) {
      this.scheduleTick(millis, 0);
    } else {
      // we timed out.
      this.onTimeout(activePlayer);
    }
  };

  elapsed = (now = performance.now()) =>
    Math.max(
      -minsToMillis(this.maxOvertimeMinutes),
      now - this.times.lastUpdate,
    );

  millisOf = (p: PlayerOrder): Millis =>
    this.times.activePlayer === p
      ? Math.max(
          -minsToMillis(this.maxOvertimeMinutes),
          this.times[p] - this.elapsed(),
        )
      : this.times[p];
}
