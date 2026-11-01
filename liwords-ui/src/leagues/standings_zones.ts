// Compute competition ranks: tied players (same points + spread) share a rank.
// rows are in server order (sorted by points desc, spread desc, username).
export const competitionRanksOf = (
  rows: ReadonlyArray<{ wins: number; draws: number; spread: number }>,
): Array<{ rank: number; tied: boolean }> => {
  const stats = rows.map((d) => ({
    points: d.wins * 2 + d.draws,
    spread: d.spread,
  }));
  const ranks: Array<{ rank: number; tied: boolean }> = [];
  let currentRank = 1;
  for (let i = 0; i < stats.length; i++) {
    if (
      i > 0 &&
      (stats[i].points !== stats[i - 1].points ||
        stats[i].spread !== stats[i - 1].spread)
    ) {
      currentRank = i + 1;
    }
    ranks.push({ rank: currentRank, tied: currentRank <= i });
  }
  for (let i = 1; i < ranks.length; i++) {
    if (ranks[i].tied) ranks[i - 1].tied = true;
  }
  return ranks;
};

// The zone a rank falls in. Zones go by competition rank, as the season's
// outcomes do: players level on points and spread at the edge of a zone share
// the better outcome.
export const zoneRowClass = (
  rank: number,
  promotionCount: number,
  relegationCount: number,
  divisionSize: number,
): string => {
  if (promotionCount > 0 && rank <= promotionCount) {
    return "promotion-row";
  }
  if (relegationCount > 0 && rank > divisionSize - relegationCount) {
    return "relegation-row";
  }
  return "";
};
