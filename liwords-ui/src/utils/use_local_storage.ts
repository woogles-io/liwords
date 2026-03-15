import { useState, useEffect, useCallback } from "react";

/**
 * A hook that syncs a localStorage key across browser tabs.
 * When another tab changes the value, this hook updates the local
 * React state immediately via the 'storage' event.
 */
export function useLocalStorage(
  key: string,
  defaultValue: string,
): [string, (value: string) => void] {
  const [value, setValue] = useState(
    () => localStorage.getItem(key) ?? defaultValue,
  );

  useEffect(() => {
    const handler = (e: StorageEvent) => {
      if (e.key === key) {
        setValue(e.newValue ?? defaultValue);
      }
    };
    window.addEventListener("storage", handler);
    return () => window.removeEventListener("storage", handler);
  }, [key, defaultValue]);

  const set = useCallback(
    (newValue: string) => {
      localStorage.setItem(key, newValue);
      setValue(newValue);
    },
    [key],
  );

  return [value, set];
}

/**
 * Boolean variant — stores "true"/"false" strings.
 */
export function useLocalStorageBool(
  key: string,
  defaultValue = false,
): [boolean, (value: boolean) => void] {
  const [raw, setRaw] = useLocalStorage(key, defaultValue ? "true" : "false");
  const set = useCallback(
    (v: boolean) => setRaw(v ? "true" : "false"),
    [setRaw],
  );
  return [raw === "true", set];
}
