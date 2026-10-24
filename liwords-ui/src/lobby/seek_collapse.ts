import { SoughtGame } from "../store/reducers/lobby_reducer";
import { GameMode } from "../gen/api/proto/ipc/omgwords_pb";

// Everything a player accepting a seek would care about: two seeks with the
// same key are interchangeable.
export const seekKey = (sg: SoughtGame): string =>
  JSON.stringify([
    sg.seekerID ?? sg.seeker,
    sg.lexicon,
    sg.variant,
    sg.boardLayout ?? "",
    sg.letterDistribution ?? "",
    sg.initialTimeSecs,
    sg.incrementSecs,
    sg.maxOvertimeMinutes,
    sg.timeBankMinutes ?? 0,
    sg.gameMode,
    sg.challengeRule,
    sg.rated,
    sg.minRatingRange,
    sg.maxRatingRange,
    sg.requireEstablishedRating,
    sg.onlyFollowedPlayers,
  ]);

export type SeekGroup = {
  sg: SoughtGame;
  count: number;
  all: Array<SoughtGame>;
};

// One seek of a group, at random: two players accepting the same row at
// once then rarely reach for the same seek (the server lets only one of
// them have any one seek).
export const pickSeek = (
  g: Pick<SeekGroup, "all">,
  rand: () => number = Math.random,
): SoughtGame => g.all[Math.floor(rand() * g.all.length)];

// Someone else's identical open correspondence seeks are shown once, with a
// count; all keeps every seek of the group, so accepting can take any one. Match requests, real-time seeks
// and the viewer's own seeks (each of which they may want to cancel) are
// left as they are. Order is kept, each group where its first seek was.
export const collapseIdenticalSeeks = (
  seeks: Array<SoughtGame>,
  username?: string,
): Array<SeekGroup> => {
  const out: Array<SeekGroup> = [];
  const groups = new Map<string, SeekGroup>();
  for (const sg of seeks) {
    if (
      sg.gameMode !== GameMode.CORRESPONDENCE ||
      sg.receiverIsPermanent ||
      sg.seeker === username
    ) {
      out.push({ sg, count: 1, all: [sg] });
      continue;
    }
    const key = seekKey(sg);
    const g = groups.get(key);
    if (g) {
      g.count++;
      g.all.push(sg);
      continue;
    }
    const entry = { sg, count: 1, all: [sg] };
    groups.set(key, entry);
    out.push(entry);
  }
  return out;
};
