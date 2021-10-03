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
    return ['Ultra-Blitz!', 'magenta'];
  }
  if (totalTime <= 6 * 60) {
    return ['Blitz', 'volcano'];
  }
  if (totalTime <= 14 * 60) {
    return ['Rapid', 'gold'];
  }
  return ['Regular', 'blue'];
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
