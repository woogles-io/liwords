// Following one player through a league's seasons: picking a player's season
// in the all-time roster opens it on their division, and from there the
// previous and next buttons step through only the seasons that player was in,
// landing on their division each time. While following, they take the
// viewer's place on the page: their division is starred, their name is in
// gold, and any season shown opens on their division (or on division 1 if
// they sat it out).

export type FollowedSeason = {
  seasonNumber: number;
  // 0 for a season the player registered for but was not placed in.
  divisionNumber: number;
};

export type FollowedPlayer = {
  userId: string;
  username: string;
  // Ascending by season number.
  seasons: FollowedSeason[];
};

export const followPlayer = (player: {
  userId: string;
  username: string;
  seasons: FollowedSeason[];
}): FollowedPlayer => ({
  userId: player.userId,
  username: player.username,
  seasons: player.seasons
    .map((s) => ({
      seasonNumber: s.seasonNumber,
      divisionNumber: s.divisionNumber,
    }))
    .sort((a, b) => a.seasonNumber - b.seasonNumber),
});

// The followed player's seasons just before and just after the one shown,
// skipping the seasons they were not in.
export const followNeighbors = (
  followed: FollowedPlayer,
  seasonNumber: number,
): { prev?: FollowedSeason; next?: FollowedSeason } => {
  const earlier = followed.seasons.filter((s) => s.seasonNumber < seasonNumber);
  return {
    prev: earlier[earlier.length - 1],
    next: followed.seasons.find((s) => s.seasonNumber > seasonNumber),
  };
};

// The followed player's division in a season, or undefined if they sat it
// out or were not placed in a division.
export const followedDivision = (
  followed: FollowedPlayer,
  seasonNumber: number,
): number | undefined => {
  const d = followed.seasons.find(
    (s) => s.seasonNumber === seasonNumber,
  )?.divisionNumber;
  return d ? d : undefined;
};
