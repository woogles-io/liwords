export const OBS_SUFFIXES = [
  "score",
  "p1_score",
  "p2_score",
  "unseen_tiles",
  "unseen_count",
  "last_play",
  "blank1",
  "blank2",
] as const;

export type OBSSuffix = (typeof OBS_SUFFIXES)[number];
