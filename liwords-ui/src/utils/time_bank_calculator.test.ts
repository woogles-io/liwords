import {
  formatCoarseDuration,
  formatCoarseDurationShort,
  nextPin,
  onTurnCountdowns,
  pickNextCorresGame,
} from "./time_bank_calculator";

it("onTurnCountdowns: before per-turn allowance is spent, nothing bleeds", () => {
  // 1-day per-turn allowance, 2-day bank, 6h elapsed.
  const perTurn = 86400;
  const bank = 2 * 86400;
  const elapsed = 6 * 3600;
  const { beforeBleed, beforeExpiry } = onTurnCountdowns(
    perTurn,
    bank,
    elapsed,
  );
  // Free time left until the bank starts draining.
  expect(beforeBleed).toBe(perTurn - elapsed);
  // Hard deadline is per-turn + bank - elapsed.
  expect(beforeExpiry).toBe(perTurn + bank - elapsed);
});

it("onTurnCountdowns: once per-turn is exceeded, bleed is 0 but expiry uses bank", () => {
  const perTurn = 86400;
  const bank = 2 * 86400;
  // Elapsed past the per-turn allowance but within the bank.
  const elapsed = perTurn + 3600;
  const { beforeBleed, beforeExpiry } = onTurnCountdowns(
    perTurn,
    bank,
    elapsed,
  );
  expect(beforeBleed).toBe(0); // clamped, the player is bleeding the bank
  expect(beforeExpiry).toBe(bank - 3600); // still time left on the bank
});

it("onTurnCountdowns: expiry can go negative when overdue", () => {
  const perTurn = 86400;
  const bank = 0;
  const elapsed = perTurn + 100;
  const { beforeBleed, beforeExpiry } = onTurnCountdowns(
    perTurn,
    bank,
    elapsed,
  );
  expect(beforeBleed).toBe(0);
  expect(beforeExpiry).toBe(-100);
});

it("onTurnCountdowns: the bank changes sort order vs the bank-blind proxy", () => {
  // Two games, same per-turn allowance and same elapsed. The bank-blind proxy
  // (perTurn - elapsed) would tie them; the bank must break the tie so the one
  // with less bank is more urgent (smaller beforeExpiry).
  const perTurn = 86400;
  const elapsed = 1000;
  const lowBank = onTurnCountdowns(perTurn, 3600, elapsed).beforeExpiry;
  const highBank = onTurnCountdowns(perTurn, 10 * 3600, elapsed).beforeExpiry;
  expect(lowBank).toBeLessThan(highBank);
});

it("formatCoarseDuration: renders coarse two-unit-max labels", () => {
  expect(formatCoarseDuration(2 * 86400 + 3 * 3600)).toBe("2d 3h");
  expect(formatCoarseDuration(2 * 86400)).toBe("2d");
  expect(formatCoarseDuration(5 * 3600 + 12 * 60)).toBe("5h 12m");
  expect(formatCoarseDuration(5 * 3600)).toBe("5h");
  expect(formatCoarseDuration(47 * 60)).toBe("47m");
  expect(formatCoarseDuration(30)).toBe("30s");
});

it("formatCoarseDuration: non-positive and non-finite render as 'now'", () => {
  expect(formatCoarseDuration(0)).toBe("now");
  expect(formatCoarseDuration(-100)).toBe("now");
  expect(formatCoarseDuration(Infinity)).toBe("now");
  expect(formatCoarseDuration(NaN)).toBe("now");
});

it("formatCoarseDurationShort: renders only the single largest unit", () => {
  expect(formatCoarseDurationShort(2 * 86400 + 3 * 3600)).toBe("2d");
  expect(formatCoarseDurationShort(5 * 3600 + 12 * 60)).toBe("5h");
  expect(formatCoarseDurationShort(47 * 60 + 30)).toBe("47m");
  expect(formatCoarseDurationShort(30)).toBe("30s");
  expect(formatCoarseDurationShort(0)).toBe("now");
  expect(formatCoarseDurationShort(-5)).toBe("now");
  expect(formatCoarseDurationShort(Infinity)).toBe("now");
});

it("pickNextCorresGame: clicking Next once per game cycles through all and returns to the first", () => {
  // Five on-turn games with distinct urgencies, supplied unsorted.
  const candidates = [
    { game: "c", gameID: "c", beforeExpiry: 300 },
    { game: "a", gameID: "a", beforeExpiry: 100 },
    { game: "e", gameID: "e", beforeExpiry: 500 },
    { game: "b", gameID: "b", beforeExpiry: 200 },
    { game: "d", gameID: "d", beforeExpiry: 400 },
  ];
  // Most-urgent-first order: a, b, c, d, e.
  const order = ["a", "b", "c", "d", "e"];
  let current = order[0];
  const visited = [current];
  for (let i = 0; i < order.length - 1; i++) {
    const { next } = pickNextCorresGame(candidates, current);
    expect(next).not.toBeNull();
    current = next as string;
    visited.push(current);
  }
  // Walked every on-turn game exactly once, in urgency order.
  expect(visited).toEqual(order);
  // The n-th click wraps back to the first.
  expect(pickNextCorresGame(candidates, current).next).toBe("a");
});

it("pickNextCorresGame: waiting excludes the current game only when it is on turn", () => {
  const candidates = [
    { game: "a", gameID: "a", beforeExpiry: 100 },
    { game: "b", gameID: "b", beforeExpiry: 200 },
    { game: "c", gameID: "c", beforeExpiry: 300 },
  ];
  // Current game is itself on turn: it is excluded from the waiting count.
  expect(pickNextCorresGame(candidates, "a").waiting).toBe(2);
  // Current game is not on turn (opponent's move / different game): every
  // on-turn game is waiting, and Next points at the most urgent.
  const off = pickNextCorresGame(candidates, "not-on-turn");
  expect(off.waiting).toBe(3);
  expect(off.next).toBe("a");
});

it("pickNextCorresGame: a lone on-turn game (or none) has nowhere to cycle", () => {
  const lone = [{ game: "a", gameID: "a", beforeExpiry: 100 }];
  expect(pickNextCorresGame(lone, "a")).toEqual({ next: null, waiting: 0 });
  expect(pickNextCorresGame([], "a")).toEqual({ next: null, waiting: 0 });
});

it("pickNextCorresGame: equal urgency is broken stably by gameID", () => {
  const candidates = [
    { game: "b", gameID: "b", beforeExpiry: 100 },
    { game: "a", gameID: "a", beforeExpiry: 100 },
    { game: "c", gameID: "c", beforeExpiry: 100 },
  ];
  // Deterministic cycle a -> b -> c -> a regardless of input order.
  expect(pickNextCorresGame(candidates, "a").next).toBe("b");
  expect(pickNextCorresGame(candidates, "b").next).toBe("c");
  expect(pickNextCorresGame(candidates, "c").next).toBe("a");
});

it("nextPin: pins on first sight and holds it across refreshes", () => {
  const onTurn = new Set(["a", "b", "c"]);
  // First evaluation on the page (undefined) takes the immediate-next.
  expect(nextPin(undefined, onTurn, "b")).toBe("b");
  // A later refresh keeps the pinned target even though the urgency-ranked
  // immediate-next has changed (this is the bug fix: no bouncing to [0]).
  expect(nextPin("b", onTurn, "c")).toBe("b");
  expect(nextPin("b", onTurn, "a")).toBe("b");
});

it("nextPin: pins a next that appears only after a later refresh", () => {
  // No game was on the user's turn at load.
  expect(nextPin(null, new Set<string>(), null)).toBe(null);
  // An opponent moved -> a next appeared -> pin it.
  expect(nextPin(null, new Set(["x"]), "x")).toBe("x");
});

it("nextPin: repicks when the pinned game leaves the on-turn set", () => {
  // Pinned "b" was played in another tab, so it is no longer on the user's
  // turn -> fall back to the freshly-computed immediate-next.
  expect(nextPin("b", new Set(["a", "c"]), "c")).toBe("c");
  // Nothing left to pin.
  expect(nextPin("b", new Set<string>(), null)).toBe(null);
});
