import { GameEvent } from "../gen/api/proto/vendored/macondo/macondo_pb";

export type TimeBankState = {
  p0Time: number;
  p1Time: number;
  p0TimeBank?: number;
  p1TimeBank?: number;
  p0UsingTimeBank?: boolean;
  p1UsingTimeBank?: boolean;
};

/**
 * Calculate historical time bank values by back-calculating from game events.
 * When a player's millis_remaining is negative, it means they went into their time bank.
 * We can reconstruct the time bank state at each turn by deducting the negative time.
 *
 * @param events Array of game events
 * @param initialTimeBankP0 Initial time bank for player 0 in milliseconds
 * @param initialTimeBankP1 Initial time bank for player 1 in milliseconds
 * @returns Map of event index to time bank state at that point
 */
export function calculateHistoricalTimeBanks(
  events: Array<GameEvent>,
  initialTimeBankP0: number,
  initialTimeBankP1: number,
): Map<number, TimeBankState> {
  const history = new Map<number, TimeBankState>();

  let p0TimeBank = initialTimeBankP0;
  let p1TimeBank = initialTimeBankP1;
  let p0Time = 0; // Track last known time for each player
  let p1Time = 0;

  for (let i = 0; i < events.length; i++) {
    const evt = events[i];
    const playerIndex = evt.playerIndex;
    const millisRemaining = evt.millisRemaining;

    // Update the time for the player who just moved
    if (playerIndex === 0) {
      p0Time = millisRemaining;
    } else {
      p1Time = millisRemaining;
    }

    // If time went negative, it means the player used their time bank
    if (millisRemaining < 0) {
      const deficit = Math.abs(millisRemaining);

      if (playerIndex === 0) {
        p0TimeBank = Math.max(0, p0TimeBank - deficit);
      } else {
        p1TimeBank = Math.max(0, p1TimeBank - deficit);
      }
    }

    // Determine if using time bank (negative time and still have bank remaining)
    const p0UsingTimeBank = p0Time <= 0 && p0TimeBank > 0;
    const p1UsingTimeBank = p1Time <= 0 && p1TimeBank > 0;

    history.set(i, {
      p0Time,
      p1Time,
      p0TimeBank: initialTimeBankP0 > 0 ? p0TimeBank : undefined,
      p1TimeBank: initialTimeBankP1 > 0 ? p1TimeBank : undefined,
      p0UsingTimeBank,
      p1UsingTimeBank,
    });
  }

  return history;
}

/**
 * Two countdowns (in seconds) for a player who is on turn in a
 * correspondence/league game.
 *
 * The time model is "reset-to-increment": each turn the player gets a fresh
 * per-turn allowance of `incrementSecs`, which does NOT bank. Separately there
 * is a consumable time bank (`bankSecs`) that is only drained when the per-turn
 * allowance is exceeded, and is never refilled. So the player is fine until
 * `elapsed > incrementSecs` (they start bleeding the bank), and they only time
 * out once `elapsed > incrementSecs + bankSecs`.
 *
 * - beforeBleed: free-time window remaining = max(0, perTurn - elapsed).
 *   When this hits 0 the player has begun consuming the bank ("bleeding").
 * - beforeExpiry: hard deadline = perTurn + bank - elapsed. When this hits 0
 *   the player times out. Can go negative if already overdue.
 *
 * This is the real time-remaining the lists sort and flag by, as opposed to the
 * bank-blind `incrementSecs - elapsed` proxy.
 */
export type OnTurnCountdowns = {
  beforeBleed: number;
  beforeExpiry: number;
};

export function onTurnCountdowns(
  incrementSecs: number,
  bankSecs: number,
  elapsedSecs: number,
): OnTurnCountdowns {
  return {
    beforeBleed: Math.max(0, incrementSecs - elapsedSecs),
    beforeExpiry: incrementSecs + bankSecs - elapsedSecs,
  };
}

/**
 * Format a duration (in seconds) coarsely for the correspondence/league game
 * lists. Granularity is intentionally low (at most two adjacent units, never
 * per-second) since these are multi-hour/day countdowns that only refresh on
 * list updates. Examples: "2d 3h", "5h 12m", "47m", "30s", "now".
 *
 * Non-positive inputs render as "now" (the deadline has passed / is imminent).
 */
export function formatCoarseDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return "now";
  }
  const totalSeconds = Math.floor(seconds);
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const secs = totalSeconds % 60;

  if (days > 0) {
    return hours > 0 ? `${days}d ${hours}h` : `${days}d`;
  }
  if (hours > 0) {
    return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
  }
  if (minutes > 0) {
    return `${minutes}m`;
  }
  return `${secs}s`;
}

/**
 * Like formatCoarseDuration but renders only the single largest unit, for the
 * compact at-a-glance time shown inline in the correspondence/league game
 * lists. The full two-unit value (and the free-time window) belong in a
 * tooltip. Examples: "2d", "5h", "47m", "30s", "now".
 *
 * Non-positive inputs render as "now" (the deadline has passed / is imminent).
 */
export function formatCoarseDurationShort(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return "now";
  }
  const totalSeconds = Math.floor(seconds);
  const days = Math.floor(totalSeconds / 86400);
  if (days > 0) {
    return `${days}d`;
  }
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  if (hours > 0) {
    return `${hours}h`;
  }
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  if (minutes > 0) {
    return `${minutes}m`;
  }
  return `${totalSeconds % 60}s`;
}

export type NextCorresPick<T> = {
  // The game to jump to next, or null when there is nowhere to cycle to.
  next: T | null;
  // How many OTHER on-turn games are waiting (excludes the current game when it
  // is itself on turn). Drives the "(x)" count on the Next button.
  waiting: number;
};

/**
 * Pick the next on-turn correspondence game to jump to, cycling.
 *
 * The candidates are every game where it is the user's turn (the current game
 * included if it is on turn). They are sorted by the bank-aware time-until-
 * expiry (least first, gameID as a stable tiebreak) -- the same "most urgent
 * first" order the correspondence/league lists use -- so this is the single
 * source of truth shared by the in-game Next button and the gameroom's top-bar
 * next-game cycler. Because the order is deterministic, clicking Next once per
 * game walks every on-turn game exactly once and returns to the first after n
 * clicks.
 *
 * The caller is expected to recompute this whenever the live correspondence-game
 * set changes (e.g. a notification that another game became the user's turn), so
 * the cycle stays current "as at last refresh".
 */
export function pickNextCorresGame<T>(
  candidates: ReadonlyArray<{
    game: T;
    gameID: string;
    beforeExpiry: number;
  }>,
  currentGameID: string | undefined,
): NextCorresPick<T> {
  const sorted = [...candidates].sort((a, b) => {
    // Do not use a-b even if it should not overflow.
    if (a.beforeExpiry < b.beforeExpiry) return -1;
    if (a.beforeExpiry > b.beforeExpiry) return 1;
    // Stable tiebreak so the cycle order is deterministic.
    if (a.gameID < b.gameID) return -1;
    if (a.gameID > b.gameID) return 1;
    return 0;
  });

  const currentIndex = sorted.findIndex((c) => c.gameID === currentGameID);

  if (currentIndex >= 0) {
    // The current game is itself on the user's turn: advance to the next game
    // in the cycle (null when this is the only on-turn game).
    return {
      next:
        sorted.length > 1
          ? sorted[(currentIndex + 1) % sorted.length].game
          : null,
      waiting: sorted.length - 1,
    };
  }

  // The current game is not on the user's turn (opponent's move, a finished
  // game, or a different game entirely): jump to the most urgent on-turn game.
  return {
    next: sorted.length > 0 ? sorted[0].game : null,
    waiting: sorted.length,
  };
}
