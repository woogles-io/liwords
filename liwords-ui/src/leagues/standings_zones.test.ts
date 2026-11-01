import { describe, expect, it } from "vitest";
import { competitionRanksOf, zoneRowClass } from "./standings_zones";

// 15 players, 5 up and 5 down, in server order: level on points and spread at
// the promotion line (5th, 6th), at the relegation line (10th, 11th), and
// inside the relegation zone (13th, 14th).
const rows = [
  { wins: 11, draws: 0, spread: 605 },
  { wins: 9, draws: 0, spread: 694 },
  { wins: 9, draws: 0, spread: 347 },
  { wins: 9, draws: 0, spread: 319 },
  { wins: 8, draws: 0, spread: 186 },
  { wins: 8, draws: 0, spread: 186 },
  { wins: 7, draws: 0, spread: -131 },
  { wins: 7, draws: 0, spread: -200 },
  { wins: 6, draws: 0, spread: 165 },
  { wins: 6, draws: 0, spread: -18 },
  { wins: 6, draws: 0, spread: -18 },
  { wins: 5, draws: 0, spread: -192 },
  { wins: 4, draws: 0, spread: -729 },
  { wins: 4, draws: 0, spread: -729 },
  { wins: 2, draws: 0, spread: -618 },
];

describe("standings zones", () => {
  it("gives players level on points and spread one rank", () => {
    expect(competitionRanksOf(rows).map((r) => r.rank)).toEqual([
      1, 2, 3, 4, 5, 5, 7, 8, 9, 10, 10, 12, 13, 13, 15,
    ]);
    expect(competitionRanksOf(rows).map((r) => r.tied)).toEqual([
      false,
      false,
      false,
      false,
      true,
      true,
      false,
      false,
      false,
      true,
      true,
      false,
      true,
      true,
      false,
    ]);
  });

  it("puts a tie at the edge of a zone on its better side", () => {
    const zones = competitionRanksOf(rows).map((r) =>
      zoneRowClass(r.rank, 5, 5, rows.length),
    );
    expect(zones).toEqual([
      "promotion-row",
      "promotion-row",
      "promotion-row",
      "promotion-row",
      "promotion-row",
      "promotion-row",
      "",
      "",
      "",
      "",
      "",
      "relegation-row",
      "relegation-row",
      "relegation-row",
      "relegation-row",
    ]);
  });

  it("has no promotion zone in the top division and no relegation zone in the bottom one", () => {
    expect(zoneRowClass(1, 0, 5, 15)).toBe("");
    expect(zoneRowClass(15, 5, 0, 15)).toBe("");
  });
});
