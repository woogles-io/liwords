import React, { useCallback, useEffect } from 'react';
import { Millis } from '../store/timer_controller';
import { useMountedState } from '../utils/mounted';

// This magical timer was written by Andy. I am not sure how it works.
export const SimpleTimer = ({
  lastRefreshedPerformanceNow,
  millisAtLastRefresh,
  isRunning,
}: {
  lastRefreshedPerformanceNow: Millis;
  millisAtLastRefresh: Millis;
  isRunning: boolean;
}) => {
  const { useState } = useMountedState();
  const [, setRerender] = useState(0);
  const requestRerender = useCallback(
    () => setRerender((n) => (n + 1) | 0),
    []
  );

  const currentMillis = isRunning
    ? millisAtLastRefresh - (performance.now() - lastRefreshedPerformanceNow)
    : millisAtLastRefresh;

  useEffect(() => {
    if (isRunning && currentMillis > 0) {
      // compute when the display would change.
      const validForMillis = ((currentMillis - 1) % 1000) + 1;
      const t = setTimeout(() => {
        requestRerender();
      }, validForMillis);
      return () => clearTimeout(t);
    }
  }); // no dependency list, this effect should run on every render.

  const currentSec = Math.ceil(currentMillis / 1000);
  const nonnegativeSec = Math.max(currentSec, 0);
  return <>{`${nonnegativeSec} second${nonnegativeSec === 1 ? '' : 's'}`}</>;
};
