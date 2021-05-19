import React, { useCallback, useEffect, useRef } from 'react';
import { useMountedState } from '../utils/mounted';

// This magical timer was written by Andy. I am not sure how it works.
export const SimpleTimer = ({
  lastRefreshedPerformanceNow,
  millisAtLastRefresh,
  isRunning,
}: {
  lastRefreshedPerformanceNow: number;
  millisAtLastRefresh: number;
  isRunning: boolean;
}) => {
  const { useState } = useMountedState();
  const [rerender, setRerender] = useState([]);
  void rerender;
  const lastRaf = useRef(0);

  const cb = useCallback(() => {
    // this seems to force a rerender
    setRerender([]);
    lastRaf.current = requestAnimationFrame(cb);
  }, []);

  useEffect(() => {
    cb();
    return () => cancelAnimationFrame(lastRaf.current);
  }, [cb]);

  const currentMillis = isRunning
    ? millisAtLastRefresh - (performance.now() - lastRefreshedPerformanceNow)
    : millisAtLastRefresh;
  const currentSec = Math.ceil(currentMillis / 1000);
  const nonnegativeSec = Math.max(currentSec, 0);
  return <>{`${nonnegativeSec} second${nonnegativeSec === 1 ? '' : 's'}`}</>;
};
