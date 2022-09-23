// Different pairing methods should show different options to the director.

import {
  PairingMethodMap,
  PairingMethod,
} from '../../gen/api/proto/ipc/tournament_pb';

export type RoundSetting = {
  beginRound: number;
  endRound: number;
  setting: SingleRoundSetting;
};

export type SingleRoundSetting = {
  pairingType: pairingMethod;
  gamesPerRound?: number;
  factor?: number;
  maxRepeats?: number;
  allowOverMaxRepeats?: boolean;
  repeatRelativeWeight?: number;
  winDifferenceRelativeWeight?: number;
};

export const settingsEqual = (
  s1: SingleRoundSetting,
  s2: SingleRoundSetting
): boolean => {
  return (
    s1.pairingType === s2.pairingType &&
    s1.gamesPerRound === s2.gamesPerRound &&
    s1.factor === s2.factor &&
    s1.maxRepeats === s2.maxRepeats &&
    s1.allowOverMaxRepeats === s2.allowOverMaxRepeats &&
    s1.repeatRelativeWeight === s2.repeatRelativeWeight &&
    s1.winDifferenceRelativeWeight === s2.winDifferenceRelativeWeight
  );
};

type valueof<T> = T[keyof T];
export type pairingMethod = valueof<PairingMethodMap>;
export type PairingMethodField = [
  string,
  keyof SingleRoundSetting,
  string,
  string
];

export const fieldsForMethod = (
  m: pairingMethod
): Array<PairingMethodField> => {
  const fields = new Array<PairingMethodField>();
  switch (m) {
    case (PairingMethod.RANDOM,
    PairingMethod.ROUND_ROBIN,
    PairingMethod.KING_OF_THE_HILL,
    PairingMethod.MANUAL,
    PairingMethod.INITIAL_FONTES):
      return [];
    // @ts-expect-error fallthrough is purposeful:
    case PairingMethod.FACTOR:
      fields.push([
        'number',
        'factor',
        'Factor',
        'Your selected factor (use 2 for 1v3, 2v4 for example).',
      ]);
    case PairingMethod.SWISS:
      fields.push(
        [
          'number',
          'maxRepeats',
          'Max Desirable Repeats',
          'Use "1" for no repeats, "2" for 1 max repeat, and so on. ' +
            'The pairing system will try to meet your requirement, but it is not guaranteed.',
        ],
        [
          'number',
          'repeatRelativeWeight',
          'Repeat Relative Weight',
          'The larger this number relative to other weights, the less likely a repeat will be.',
        ],
        [
          'number',
          'winDifferenceRelativeWeight',
          'Win Difference Relative Weight',
          'The smaller this number relative to other weights, the more mismatched your pairings will be, in terms of win difference.',
        ]
      );
      break;

    case PairingMethod.TEAM_ROUND_ROBIN:
      fields.push([
        'number',
        'gamesPerRound',
        'Games per Round',
        'This pairing system is not currently available.',
      ]);
      break;
  }

  return fields;
};
