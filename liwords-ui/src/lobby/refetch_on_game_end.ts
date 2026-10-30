import { useEffect, useRef } from "react";

// Fetches the recently ended games again when a game leaves the list of
// ongoing ones, since one that ended belongs under "Recently ended games".
// The lobby hears of an ending just before the game is saved, so the fetch
// waits a moment, and is made once more for a game still missing then. One
// fetch is pending at a time: another game leaving the list starts the wait
// again, so endings close together make one fetch.
export const useRefetchOnGameEnd = (
  ongoingIds: Array<string>,
  // Resolves to the ids of the recently ended games it fetched.
  refetch: () => Promise<Array<string>>,
  delayMs = 1000,
  retryMs = 3000,
) => {
  const listed = useRef<Set<string> | null>(null);
  // The games that left the list and were not among the ended ones yet, each
  // with how many fetches have looked for it.
  const waiting = useRef(new Map<string, number>());
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const alive = useRef(true);

  useEffect(() => {
    const ids = new Set(ongoingIds);
    const before = listed.current;
    listed.current = ids;
    if (!before) return;
    const gone = [...before].filter((id) => !ids.has(id));
    if (gone.length === 0) return;
    for (const id of gone) waiting.current.set(id, 0);
    const fetchAfter = (ms: number) => {
      if (!alive.current) return;
      clearTimeout(timer.current);
      timer.current = setTimeout(async () => {
        timer.current = undefined;
        // Only the games waiting when the fetch starts can be in its answer.
        const lookedFor = [...waiting.current.keys()];
        const ended = new Set(await refetch());
        for (const id of lookedFor) {
          const looked = waiting.current.get(id);
          if (looked === undefined) continue;
          if (ended.has(id) || looked > 0) waiting.current.delete(id);
          else waiting.current.set(id, looked + 1);
        }
        // A game that left meanwhile has its own fetch pending already.
        if (waiting.current.size > 0 && timer.current === undefined) {
          fetchAfter(retryMs);
        }
      }, ms);
    };
    fetchAfter(delayMs);
  }, [ongoingIds, refetch, delayMs, retryMs]);

  // Only leaving the page drops a pending fetch: the list changes with every
  // move in any game, and that must not cancel it.
  useEffect(() => {
    alive.current = true;
    const pending = waiting.current;
    return () => {
      alive.current = false;
      clearTimeout(timer.current);
      pending.clear();
    };
  }, []);
};
