import { PlayerStanding } from "../gen/api/proto/ipc/tournament_pb";

export type CompetitionRank = { rank: number; tied: boolean };

// Players who read identically share a place, and the place after a shared one
// skips: 1=, 1=, 3.
//
// The tie is on what the row shows -- W, L and spread -- not on the comparator
// the server sorts by, which is win rate, then losses, then spread. Those
// disagree: equal rate with equal losses and equal spread admits 1-0 sitting
// next to 2-0 once byes leave players on different game counts, and marking
// those two as sharing a place would read as a bug. The "=" is a claim about
// the row in front of you.
//
// Before anybody has played, every row is 0-0 with no spread. That is a tie in
// the arithmetic and noise on the screen, so it goes unmarked.
export const competitionRanks = (
  standings: readonly PlayerStanding[],
): CompetitionRank[] => {
  const plain = standings.map((_, index) => ({
    rank: index + 1,
    tied: false,
  }));

  const played = standings.some((s) => s.wins + s.losses + s.draws > 0);
  if (!played) {
    return plain;
  }

  // Doubled so the half a draw is worth stays an integer.
  const key = (s: PlayerStanding) =>
    `${s.wins * 2 + s.draws}/${s.losses * 2 + s.draws}/${s.spread}`;

  const ranks: CompetitionRank[] = [];
  let currentRank = 1;
  for (let i = 0; i < standings.length; i++) {
    if (i > 0 && key(standings[i]) !== key(standings[i - 1])) {
      currentRank = i + 1;
    }
    // A rank that has fallen behind its own position is one an earlier row
    // already holds.
    ranks.push({ rank: currentRank, tied: currentRank <= i });
  }
  // Mark the first row of each shared place too.
  for (let i = 1; i < ranks.length; i++) {
    if (ranks[i].tied) {
      ranks[i - 1].tied = true;
    }
  }
  return ranks;
};

// Which side the shared-place marker sits on follows how the column is
// aligned, so that the digits stay in a line either way: a left-aligned column
// trails it (1=, as the league standings write it), a right-aligned one leads
// with it (=1, as tournament archives write it).
export type TieMarker = "trailing" | "leading";

export const rankLabel = (
  rank: CompetitionRank | undefined,
  fallback: number,
  marker: TieMarker = "trailing",
) => {
  if (!rank) {
    return `${fallback}`;
  }
  if (!rank.tied) {
    return `${rank.rank}`;
  }
  return marker === "leading" ? `=${rank.rank}` : `${rank.rank}=`;
};
