import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { PlayerStandingSchema } from "../gen/api/proto/ipc/tournament_pb";
import { competitionRanks, rankLabel } from "./ranks";

const standing = (wins: number, losses: number, spread: number, draws = 0) =>
  create(PlayerStandingSchema, {
    playerId: `p${wins}${losses}${spread}${draws}`,
    wins,
    losses,
    draws,
    spread,
  });

const labels = (...rows: ReturnType<typeof standing>[]) => {
  const ranks = competitionRanks(rows);
  return rows.map((_, i) => rankLabel(ranks[i], i + 1));
};

const leadingLabels = (...rows: ReturnType<typeof standing>[]) => {
  const ranks = competitionRanks(rows);
  return rows.map((_, i) => rankLabel(ranks[i], i + 1, "leading"));
};

describe("competition ranks", () => {
  it("leaves an untied table alone", () => {
    expect(labels(standing(3, 0, 300), standing(2, 1, 100))).toEqual([
      "1",
      "2",
    ]);
  });

  it("shares a place and skips the next one", () => {
    expect(
      labels(standing(3, 0, 100), standing(3, 0, 100), standing(1, 2, 50)),
    ).toEqual(["1=", "1=", "3"]);
  });

  it("splits players who differ only on spread", () => {
    expect(labels(standing(3, 0, 200), standing(3, 0, 100))).toEqual([
      "1",
      "2",
    ]);
  });

  it("ties on the half a draw is worth, not on the raw counts", () => {
    // 2.5-0.5 either way: two wins and a draw, or two and a half wins.
    expect(
      labels(standing(2, 0, 100, 1), standing(2, 0, 100, 1), standing(1, 2, 0)),
    ).toEqual(["1=", "1=", "3"]);
  });

  it("shares a place in the middle of the table", () => {
    expect(
      labels(
        standing(4, 0, 400),
        standing(2, 2, 100),
        standing(2, 2, 100),
        standing(0, 4, -500),
      ),
    ).toEqual(["1", "2=", "2=", "4"]);
  });

  it("leads with the marker where the column is right-aligned", () => {
    expect(
      leadingLabels(
        standing(3, 0, 100),
        standing(3, 0, 100),
        standing(1, 2, 50),
      ),
    ).toEqual(["=1", "=1", "3"]);
  });

  it("says nothing before anybody has played", () => {
    expect(labels(standing(0, 0, 0), standing(0, 0, 0))).toEqual(["1", "2"]);
  });

  it("marks a tie at the bottom once one game has landed", () => {
    expect(
      labels(standing(1, 0, 50), standing(0, 0, 0), standing(0, 0, 0)),
    ).toEqual(["1", "2=", "2="]);
  });
});
