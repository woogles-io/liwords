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

export type SeekGroup = { sg: SoughtGame; count: number };

// The other open seeks identical to this one, sent along with an accept: if
// the seek is gone by the time the server reads it (someone else took it,
// or it was canceled), the server takes the first of these still there.
export const identicalSeekIDs = (
  sg: SoughtGame,
  seeks: Array<SoughtGame>,
): Array<string> => {
  const key = seekKey(sg);
  return seeks
    .filter((o) => o.seekID !== sg.seekID && seekKey(o) === key)
    .map((o) => o.seekID);
};

// Someone else's identical open correspondence seeks are shown once, with a
// count. Match requests, real-time seeks
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
      out.push({ sg, count: 1 });
      continue;
    }
    const key = seekKey(sg);
    const g = groups.get(key);
    if (g) {
      g.count++;
      continue;
    }
    const entry = { sg, count: 1 };
    groups.set(key, entry);
    out.push(entry);
  }
  return out;
};
