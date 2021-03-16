// a helper that defines a timer hook.

import { useCallback, useRef, useState } from 'react';
import { PlayerOrder } from './constants';
import { ClockController, Millis, Times } from './timer_controller';

export const defaultTimerContext = {
  p0: 0,
  p1: 0,
  activePlayer: 'p0' as PlayerOrder,
  lastUpdate: 0,
};

export const useTimer = () => {
  const clockController = useRef<ClockController | null>(null);

  const [timerContext, setTimerContext] = useState<Times>(defaultTimerContext);
  const [pTimedOut, setPTimedOut] = useState<PlayerOrder | undefined>(
    undefined
  );

  const onClockTick = useCallback((p: PlayerOrder, t: Millis) => {
    if (!clockController || !clockController.current) {
      return;
    }
    const newCtx = { ...clockController.current!.times, [p]: t };
    setTimerContext(newCtx);
  }, []);

  const onClockTimeout = useCallback((p: PlayerOrder) => {
    setPTimedOut(p);
  }, []);

  const stopClock = useCallback(() => {
    if (!clockController.current) {
      return;
    }
    clockController.current.stopClock();
    setTimerContext({ ...clockController.current.times });
  }, []);

  return {
    clockController,
    stopClock,
    timerContext,
    pTimedOut,
    setPTimedOut,
    onClockTick,
    onClockTimeout,
  };
};
