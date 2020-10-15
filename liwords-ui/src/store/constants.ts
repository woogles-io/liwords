import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';

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

export const initialTimeLabel = (secs: number) => {
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

export const timeToString = (
  secs: number,
  incrementSecs: number,
  maxOvertimeMinutes: number
) => {
  return `${initialTimeLabel(secs)}${
    maxOvertimeMinutes ? '/' + maxOvertimeMinutes : ''
  }${incrementSecs ? '+' + incrementSecs : ''}`;
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
    [100, 'geekblue'],
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
