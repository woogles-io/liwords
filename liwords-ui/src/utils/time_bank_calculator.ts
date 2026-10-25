import {
  GameEvent,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";

export type TimeBankState = {
  p0Time?: number;
  p1Time?: number;
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
  // Last known time for each player; undefined until their clock shows up,
  // so a player who has not moved yet is not "using" their bank.
  let p0Time: number | undefined;
  let p1Time: number | undefined;

  for (let i = 0; i < events.length; i++) {
    const evt = events[i];
    const millisRemaining = evt.millisRemaining;
    let owner = evt.playerIndex;
    // Whether the event ends its owner's turn, which is when the server
    // takes an overrun out of the bank.
    let endsTurn = true;
    switch (evt.type) {
      case GameEvent_Type.CHALLENGE_BONUS:
      case GameEvent_Type.PHONY_TILES_RETURNED:
        // These name the challenged player but carry the challenger's
        // clock, and the challenger's turn goes on.
        owner = 1 - evt.playerIndex;
        endsTurn = false;
        break;
      case GameEvent_Type.CHALLENGE:
        endsTurn = false;
        break;
      case GameEvent_Type.END_RACK_PTS:
      case GameEvent_Type.END_RACK_PENALTY:
        // No clock of their own.
        owner = -1;
        break;
    }

    if (owner === 0) {
      p0Time = millisRemaining;
    } else if (owner === 1) {
      p1Time = millisRemaining;
    }

    // An overrun is taken out of the bank once, when the turn ends.
    if (endsTurn && owner >= 0 && millisRemaining < 0) {
      const deficit = Math.abs(millisRemaining);

      if (owner === 0) {
        p0TimeBank = Math.max(0, p0TimeBank - deficit);
      } else {
        p1TimeBank = Math.max(0, p1TimeBank - deficit);
      }
    }

    // Determine if using time bank (negative time and still have bank remaining)
    const p0UsingTimeBank =
      p0Time !== undefined && p0Time <= 0 && p0TimeBank > 0;
    const p1UsingTimeBank =
      p1Time !== undefined && p1Time <= 0 && p1TimeBank > 0;

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

/**
 * Decide which on-turn game the "Next" affordance should point at, given the
 * previously pinned target. The target is pinned on first sight and then held
 * across live refreshes -- a refresh must not move it. Otherwise, once the
 * current game leaves the on-turn set (you moved, or an opponent's move
 * arrived), pickNextCorresGame falls back to the most-urgent game and yanks you
 * back to the top of the list, so you can never play once through all your
 * games. The pin is replaced only when it goes invalid (the pinned game is no
 * longer on the user's turn -- e.g. it was played in another tab) or when there
 * was no target and one has now appeared. It resets on navigation (a new page
 * is a fresh mount with prevPinned === undefined).
 *
 * - prevPinned: the currently pinned game id, null (no target), or undefined
 *   (nothing pinned yet -- first evaluation on this page).
 * - onTurnGameIDs: the games where it is currently the user's turn.
 * - immediateNextGameID: the freshly-computed urgency-ranked next, or null.
 */
export function nextPin(
  prevPinned: string | null | undefined,
  onTurnGameIDs: ReadonlySet<string>,
  immediateNextGameID: string | null,
): string | null {
  // Keep a still-valid pin: a live refresh changes the "(n)" count, not the
  // target.
  if (prevPinned != null && onTurnGameIDs.has(prevPinned)) {
    return prevPinned;
  }
  // First sight, a next that has just appeared, or a repick after the pinned
  // game went invalid.
  return immediateNextGameID;
}
