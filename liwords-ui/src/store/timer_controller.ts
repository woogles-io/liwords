// The timer controller should be a mostly stand-alone state, for performance sake.
// This code is heavily based on the AGPL-licensed timer controller code
// for lichess (https://github.com/ornicar/lila)
// You rock Thibault

import { PlayerOrder } from './constants';
// import { GameState } from './reducers/game_reducer';
import { PlayState } from '../gen/macondo/api/proto/macondo/macondo_pb';

const showTenthsCutoff = 10000;

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

export const millisToTimeStr = (s: number): string => {
  const neg = s < 0;
  // eslint-disable-next-line no-param-reassign
  s = Math.abs(s);
  const mins = Math.floor(s / 60000);
  let secs;
  let secStr;
  if (s > showTenthsCutoff) {
    secs = Math.floor(s / 1000) % 60;
    secStr = secs.toString().padStart(2, '0');
  } else {
    secs = s / 1000;
    secStr = secs.toFixed(1).padStart(4, '0');
  }
  const minStr = mins.toString().padStart(2, '0');
  return `${neg ? '-' : ''}${minStr}:${secStr}`;
};

export type Times = {
  p0: Millis;
  p1: Millis;
  activePlayer?: PlayerOrder; // the index of the player
  lastUpdate: Millis;
};

export class ClockController {
  showTenths: (millis: Millis) => boolean;

  times: Times;

  private tickCallback?: number;

  onTimeout: () => void;

  onTick: (p: PlayerOrder, t: Millis) => void;

  constructor(
    ts: Times,
    onTimeout: () => void,
    onTick: (p: PlayerOrder, t: Millis) => void
  ) {
    // Show tenths after 10 seconds.
    this.showTenths = (time) => time < showTenthsCutoff;

    this.times = { ...ts };
    this.onTimeout = onTimeout;
    this.onTick = onTick;
    this.setClock(PlayState.PLAYING, this.times);

    console.log('in timer controller constructor', this.times);
  }

  setClock = (playState: number, ts: Times, delay: Centis = 0) => {
    const isClockRunning = playState !== PlayState.GAME_OVER;
    const delayMs = delay * 10;

    this.times = {
      ...ts,
      activePlayer: isClockRunning ? ts.activePlayer : undefined,
      lastUpdate: performance.now() + delayMs,
    };

    console.log('setClock', this.times);

    if (isClockRunning) {
      this.scheduleTick(this.times[this.times.activePlayer!], delayMs);
    }
  };

  stopClock = (): Millis | null => {
    console.log('stopClock');

    const { activePlayer } = this.times;
    if (activePlayer) {
      const curElapse = this.elapsed();
      this.times[activePlayer] = Math.max(
        0,
        this.times[activePlayer] - curElapse
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
    this.tickCallback = window.setTimeout(
      this.tick,
      (time % (this.showTenths(time) ? 100 : 500)) + 1 + extraDelay
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
    const millis = Math.max(0, this.times[activePlayer] - this.elapsed(now));

    this.scheduleTick(millis, 0);
    if (millis === 0) {
      this.onTimeout();
    } else {
      this.onTick(activePlayer, millis);
    }
  };

  elapsed = (now = performance.now()) =>
    Math.max(0, now - this.times.lastUpdate);

  millisOf = (p: PlayerOrder): Millis =>
    this.times.activePlayer === p
      ? Math.max(0, this.times[p] - this.elapsed())
      : this.times[p];
}
