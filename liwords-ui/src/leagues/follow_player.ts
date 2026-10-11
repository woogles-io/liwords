// Following one player through a league's seasons: picking a player's season
// in the all-time roster opens it on their division, and from there the
// previous and next buttons step through only the seasons that player was in,
// landing on their division each time.

export type FollowedSeason = {
  seasonNumber: number;
  // 0 for a season the player registered for but was not placed in.
  divisionNumber: number;
};

export type FollowedPlayer = {
  username: string;
  // Ascending by season number.
  seasons: FollowedSeason[];
};

export const followPlayer = (
  username: string,
  seasons: FollowedSeason[],
): FollowedPlayer => ({
  username,
  seasons: seasons
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

// Whether the page shows the followed player's own season and division (for
// a season they were not placed in, any division of it).
export const showsFollowed = (
  followed: FollowedPlayer,
  seasonNumber: number | undefined,
  divisionNumber: number | undefined,
): boolean =>
  followed.seasons.some(
    (s) =>
      s.seasonNumber === seasonNumber &&
      (s.divisionNumber === 0 || s.divisionNumber === divisionNumber),
  );
