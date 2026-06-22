import { formatCoarseDuration, onTurnCountdowns } from "./time_bank_calculator";

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
