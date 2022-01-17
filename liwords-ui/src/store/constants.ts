import {
  TType,
  TTypeMap,
} from '../gen/api/proto/tournament_service/tournament_service_pb';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { Blank } from '../utils/cwgame/common';
import { ChatEntityObj, ChatEntityType, randomID } from './store';

export type PlayerOrder = 'p0' | 'p1';

// number of turns in a game, this is just an estimate. See `variants.go`
const turnsPerGame = 16;

export const calculateTotalTime = (
  secs: number,
  incrementSecs: number,
  maxOvertime: number
): number => {
  return secs + maxOvertime * 60 + incrementSecs * turnsPerGame;
};

export type valueof<T> = T[keyof T];

export const isPairedMode = (type: valueof<TTypeMap>) => {
  return type === TType.CHILD || type === TType.STANDARD;
};

export const isClubType = (type: valueof<TTypeMap>) => {
  return type === TType.CHILD || type === TType.CLUB;
};
// See cutoffs in variants.go. XXX: Try to tie these together better.
export const timeCtrlToDisplayName = (
  secs: number,
  incrementSecs: number,
  maxOvertime: number
) => {
  const totalTime = calculateTotalTime(secs, incrementSecs, maxOvertime);

  if (totalTime <= 2 * 60) {
    return ['Ultra-Blitz!', 'ultrablitz', 'magenta'];
  }
  if (totalTime <= 6 * 60) {
    return ['Blitz', 'blitz', 'volcano'];
  }
  if (totalTime <= 14 * 60) {
    return ['Rapid', 'rapid', 'gold'];
  }
  return ['Regular', 'regular', 'blue'];
};
export const StartingRating = '1500';
// see ToVariantKey and related functions in ratings.go
export const ratingKey = (
  secs: number,
  incrementSecs: number,
  maxOvertime: number,
  variant: string,
  lexicon: string
) => {
  const a = timeCtrlToDisplayName(secs, incrementSecs, maxOvertime);
  const tfmt = a[1];
  let lexVariant = lexicon;
  // These are just used for the hard-coded rating keys in the profile.
  if (lexicon.startsWith('NWL')) {
    lexVariant = 'NWL18';
  }
  if (lexicon.startsWith('CSW')) {
    lexVariant = 'CSW19';
  }
  if (lexicon.startsWith('ECWL')) {
    lexVariant = 'ECWL';
  }
  return `${lexVariant}.${variant}.${tfmt}`;
};

const initialTimeLabel = (secs: number) => {
  let initTLabel;
  switch (secs) {
    case 15:
      initTLabel = '¼';
      break;
    case 30:
      initTLabel = '½';
      break;
    case 45:
      initTLabel = '¾';
      break;
    default:
      initTLabel = `${secs / 60}`;
  }
  return initTLabel;
};

export const initTimeDiscreteScale = [
  15,
  30,
  45,
  ...Array.from(new Array(25), (_, x) => (x + 1) * 60),
  ...[30, 35, 40, 45, 50, 55, 60].map((x) => x * 60),
].map((seconds) => ({
  seconds,
  label: initialTimeLabel(seconds),
}));

export const initialTimeSecondsToSlider = (secs: number) => {
  const ret = initTimeDiscreteScale.findIndex((x) => x.seconds === secs);
  if (ret >= 0) return ret;
  throw new Error(`bad initial time: ${secs} seconds`);
};

export const initialTimeMinutesToSlider = (mins: number) =>
  initialTimeSecondsToSlider(mins * 60);

export const timeToString = (
  secs: number,
  incrementSecs: number,
  maxOvertimeMinutes: number
) => {
  return `${initialTimeLabel(secs)}${
    maxOvertimeMinutes ? '/' + maxOvertimeMinutes : ''
  }${incrementSecs ? '+' + incrementSecs : ''}`;
};

export type ChatMessageFromJSON = {
  username: string;
  channel: string;
  message: string;
  timestamp: string;
  user_id: string;
  id: string;
};

export const chatMessageToChatEntity = (
  cm: ChatMessageFromJSON
): ChatEntityObj => {
  return {
    entityType: ChatEntityType.UserChat,
    id: cm.id || randomID(),
    sender: cm.username,
    message: cm.message,
    timestamp: parseInt(cm.timestamp, 10),
    senderId: cm.user_id,
    channel: cm.channel,
  };
};

export const ratingToColor = (rating: string): [number, string] => {
  let ratNum;
  if (rating.endsWith('?')) {
    ratNum = parseInt(rating.substring(0, rating.length - 1), 10);
  } else {
    ratNum = parseInt(rating, 10);
  }
  const ratingCutoffs: Array<[number, string]> = [
    [2100, 'pink'],
    [1900, 'volcano'],
    [1700, 'yellow'],
    [1500, 'orange'],
    [1300, 'cyan'],
    [1100, 'green'],
    [900, 'blue'],
    [700, 'purple'],
    [500, 'gold'],
    [300, 'lime'],
    [100, 'gray'],
  ];
  for (let r = 0; r < ratingCutoffs.length; r++) {
    if (ratNum >= ratingCutoffs[r][0]) {
      return [ratNum, ratingCutoffs[r][1]];
    }
  }
  // If you're rated under 100 you're a geek.
  return [ratNum, 'geekblue'];
};

export const challRuleToStr = (n: number): string => {
  switch (n) {
    case ChallengeRule.DOUBLE:
      return 'x2';
    case ChallengeRule.SINGLE:
      return 'x1';
    case ChallengeRule.TRIPLE:
      return 'x3';
    case ChallengeRule.FIVE_POINT:
      return '+5';
    case ChallengeRule.TEN_POINT:
      return '+10';
    case ChallengeRule.VOID:
      return 'Void';
  }
  return 'Unsupported';
};

export let sharedEnableAutoShuffle =
  localStorage.getItem('enableAutoShuffle') === 'true';

export const setSharedEnableAutoShuffle = (value: boolean) => {
  if (value) {
    localStorage.setItem('enableAutoShuffle', 'true');
  } else {
    localStorage.removeItem('enableAutoShuffle');
  }
  sharedEnableAutoShuffle = value;
};

// To expose this and make it more ergonomic to reorder without refreshing.
export let preferredSortOrder = localStorage.getItem('tileOrder');

export const setPreferredSortOrder = (value: string) => {
  if (value) {
    localStorage.setItem('tileOrder', value);
    preferredSortOrder = value;
  } else {
    localStorage.removeItem('tileOrder');
    preferredSortOrder = null;
  }
};

export const sortTiles = (rack: string) => {
  const effectiveSortOrder = preferredSortOrder ?? '';
  return Array.from(rack, (tile) => {
    let index = effectiveSortOrder.indexOf(tile);
    if (index < 0) index = effectiveSortOrder.length + (tile === Blank ? 1 : 0);
    return [index, tile];
  })
    .sort(([aIndex, aTile], [bIndex, bTile]) =>
      aIndex < bIndex
        ? -1
        : aIndex > bIndex
        ? 1
        : aTile < bTile
        ? -1
        : aTile > bTile
        ? 1
        : 0
    )
    .reduce((s, [index, tile]) => s + tile, '');
};

// Can skip error codes for now.
// 1001 - max bye placement - front end doesn't allow this
// 1002 - min gibson placement - front end doesn't allow this
// 1003 - min gibson spread - not allowed in front end
// 1004 - empty round controls - not allowed in front end
export const errorMap: Map<number, string> = new Map<number, string>([
  [
    1001,
    'Max Bye Placement cannot be less than 1: Tournament: $1 Division: $2 Max Bye Placement you entered: $3',
  ],
  [1002, 'Min Gibson Placement cannot be less than 1.'],
  [1005, 'The tournament has already started.'],
  // 1006 - elimination tournament not supported yet.
  [1007, 'You cannot have other pairings preceding Initial Fontes.'],
  // This is major tom to round controls
  [1008, 'Initial Fontes must have an odd number of rounds; you entered $3.'],
  // 1009 - elimination tournament not supported yet.
  [1010, 'Round number must be between 1 and the number of rounds.'],
  [1011, 'You must select a player.'],
  // 1012 - past rounds
  [1012, 'Turn on Amendment to edit an already existing score.'],
  [1013, 'You cannot enter scores for future rounds.'],
  // XXX: 1014 - nil player pairing -- how can this be triggered?
  [1015, 'The players you selected did not play in this round.'],
  // 1016 - mixed void and nonvoid results -- how can this be triggered?
  // 1017 - nonexistent pairing -- how can this be triggered?
  // 1018 - pairing has no games -- how can this be triggered?
  // 1019 - elimination tournament not supported yet
  // 1020 - tournament game index out of range, only for pairings that have multiple
  // numbers of games. Can revisit this later.
  // copypasta
  // 1021 - current rounds
  // XXX - check front end error here for this, when i don't click Amendment:
  [1021, 'Turn on Amendment to edit an already existing score.'],
  // 1022 - TOURNAMENT_NONEXISTENT_RESULT_AMENDMENT -- how to trigger?
  // 1023 - How to trigger this error?
  [1023, 'Gibsonization indexes out of range: [0, $3], $4, $5'],
  [1038, 'This round cannot be opened because round $3 is not complete.'],
  [
    1040,
    'Tournament cannot be started. Please ensure all your divisions have at least two players.',
  ],
  [1053, 'The tournament has not yet started.'],
  [1054, 'The division named $2 does not exist in this tournament.'],
  [
    1056,
    'This round is already over or underway. You are currently in round $3 for division $2.',
  ],
  [1057, 'You cannot add a division after the tournament has started.'],
  [1059, 'The division named $2 already exists in this tournament.'],
  [1060, 'You cannot remove a division after the tournament has started.'],
  [1073, 'You must select one of the options for Suspended game result'],
]);
