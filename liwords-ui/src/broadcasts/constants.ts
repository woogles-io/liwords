export const OBS_SUFFIXES = [
  "score",
  "p1_score",
  "p2_score",
  "unseen_tiles",
  "unseen_count",
  "last_play",
  "blank1",
  "blank2",
  "p1_name",
  "p2_name",
  "combined_names",
  // Tournament-standings fields — need a broadcast feed.
  "p1_record",
  "p2_record",
  "p1_place",
  "p2_place",
  "p1_spread",
  "p2_spread",
  "p1_rating",
  "p2_rating",
  "division",
  "tournament",
  "round",
  "table",
  // User-alias mode only.
  "opponent_name",
] as const;

export type OBSSuffix = (typeof OBS_SUFFIXES)[number];

// Tournament-standings fields resolve from the broadcast feed, so they need a
// mode that reaches a broadcast: a broadcast slot, or a stream slot (which
// points at one). Hidden in game mode and in the legacy user-alias mode, which
// resolve to a bare annotated game with no tournament behind it.
export const OBS_FEED_SUFFIXES: readonly OBSSuffix[] = [
  "p1_record",
  "p2_record",
  "p1_place",
  "p2_place",
  "p1_spread",
  "p2_spread",
  "p1_rating",
  "p2_rating",
  "division",
  "tournament",
  "round",
  "table",
];

// opponent_name only makes sense relative to a single tracked player, which
// only the legacy user-alias mode has (slot modes already expose both
// p1_name/p2_name, and a stream slot's owner need not be playing at all).
export const OBS_USER_ONLY_SUFFIXES: readonly OBSSuffix[] = ["opponent_name"];
