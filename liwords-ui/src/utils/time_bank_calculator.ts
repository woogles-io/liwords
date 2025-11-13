import { GameEvent } from "../gen/api/vendor/macondo/macondo_pb";

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
