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
] as const;

export type OBSSuffix = (typeof OBS_SUFFIXES)[number];
