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
  // Tournament-standings fields — broadcast-slot mode only.
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

// Tournament-standings fields resolve from the broadcast feed, which only
// exists in "slot" mode (a slot tied to a tournament). Hidden in game/user modes.
export const OBS_SLOT_ONLY_SUFFIXES: readonly OBSSuffix[] = [
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
// only "user" mode has (slot mode already exposes both p1_name/p2_name).
export const OBS_USER_ONLY_SUFFIXES: readonly OBSSuffix[] = ["opponent_name"];
