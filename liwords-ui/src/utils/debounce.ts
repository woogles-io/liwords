import { useCallback, useRef } from 'react';

// evolved from
// https://www.matthewgerstman.com/tech/throttle-and-debounce/

// eslint-disable-next-line @typescript-eslint/ban-types
export function useDebounce(func: Function, timeout: number) {
  const timer = useRef<NodeJS.Timeout>();
  const debounced = useCallback(
    (...args) => {
      if (timer.current != null) clearTimeout(timer.current);
      timer.current = setTimeout(() => {
        func(...args);
      }, timeout);
    },
    [func, timeout]
  );
  return debounced;
}
