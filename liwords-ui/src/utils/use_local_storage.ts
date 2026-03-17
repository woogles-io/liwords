import { useCallback, useSyncExternalStore } from "react";

// Single set of subscribers notified on any localStorage change.
// useSyncExternalStore's snapshot comparison ensures only components
// whose key actually changed will re-render.
const subscribers = new Set<() => void>();

function subscribe(callback: () => void) {
  subscribers.add(callback);
  return () => {
    subscribers.delete(callback);
  };
}

function notifySubscribers() {
  subscribers.forEach((cb) => cb());
}

// Single global listener for cross-tab changes.
if (typeof window !== "undefined") {
  window.addEventListener("storage", () => {
    notifySubscribers();
  });
}

/**
 * A hook that syncs a localStorage key across browser tabs using
 * useSyncExternalStore. Changes from other tabs arrive via the
 * 'storage' event; same-tab changes notify subscribers directly.
 */
export function useLocalStorage(
  key: string,
  defaultValue: string,
): [string, (value: string) => void] {
  const getSnapshot = () => localStorage.getItem(key) ?? defaultValue;

  const value = useSyncExternalStore(subscribe, getSnapshot);

  const set = useCallback(
    (newValue: string) => {
      localStorage.setItem(key, newValue);
      notifySubscribers();
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
