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
