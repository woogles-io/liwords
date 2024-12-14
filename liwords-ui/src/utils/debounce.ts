import { useCallback, useRef } from "react";

// evolved from
// https://www.matthewgerstman.com/tech/throttle-and-debounce/

export function useDebounce(func: (...args: any[]) => void, timeout: number) {
  const timer = useRef<NodeJS.Timeout>();
  const debounced = useCallback(
    (...args: any[]) => {
      if (timer.current != null) clearTimeout(timer.current);
      timer.current = setTimeout(() => {
        func(...args);
      }, timeout);
    },
    [func, timeout],
  );
  return debounced;
}
