// Different pairing methods should show different options to the director.

import {
  PairingMethod,
  PairingMethodMap,
} from '../gen/api/proto/realtime/realtime_pb';

export type RoundSetting = {
  beginRound: number;
  endRound: number;
  pairingType: pairingMethod;
  gamesPerRound?: number;
  factor?: number;
  maxRepeats?: number;
  allowOverMaxRepeats?: boolean;
  repeatRelativeWeight?: number;
  winDifferenceRelativeWeight?: number;
};

type valueof<T> = T[keyof T];
export type pairingMethod = valueof<PairingMethodMap>;
export type PairingMethodField = [string, keyof RoundSetting, string];

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

    case PairingMethod.FACTOR:
      fields.push(['number', 'factor', 'Factor']);
    // fallthrough
    case PairingMethod.SWISS:
      fields.push(
        ['number', 'maxRepeats', 'Max Desirable Repeats'],
        ['bool', 'allowOverMaxRepeats', 'Allow over Max Desirable Repeats'],
        ['number', 'repeatRelativeWeight', 'Repeat Relative Weight'],
        [
          'number',
          'winDifferenceRelativeWeight',
          'Win Difference Relative Weight',
        ]
      );
      break;

    case PairingMethod.TEAM_ROUND_ROBIN:
      fields.push(['number', 'gamesPerRound', 'Games per Round']);
      break;
  }

  return fields;
};
