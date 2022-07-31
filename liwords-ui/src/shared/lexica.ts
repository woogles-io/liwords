/**
 * @fileoverview A mapping between lexicon codes and user-visible lexicon names.
 */

// internal rating name is saved in db (select ratings->'Data' from profiles).
// it follows pkg/entity/ratings.go ToVariantKey.
// XXX: foreign ratings aren't grouped by family.
const lexiconCodeToInternalRatingName = (code: string) => {
  if (code.startsWith('NWL')) return 'NWL18';
  if (code.startsWith('CSW')) return 'CSW19';
  if (code.startsWith('ECWL')) return 'ECWL';
  return code;
};

// profile rating name is the name that shows up in your profile ratings.
const InternalRatingNameToProfileRatingName: {
  [code: string]: string;
} = {
  NWL18: 'NWL',
  NSWL20: 'NSWL',
  CSW19: 'CSW',
  ECWL: 'CEL',
  RD28: 'Deutsch',
  NSF21: 'Norsk',
  FRA20: 'Français',
};

export const lexiconCodeToProfileRatingName = (code: string) => {
  const internalRatingName = lexiconCodeToInternalRatingName(code);
  return (
    InternalRatingNameToProfileRatingName[internalRatingName] ??
    internalRatingName
  );
};

type Lexicon = {
  code: string;
  shortDescription: string;
  matchName: string; // the name that shows up in a match/seek/watch
  longDescription?: string;
  flagCode?: string;
};

export const AllLexica: { [code: string]: Lexicon } = {
  NWL20: {
    code: 'NWL20',
    shortDescription: 'NWL 20 (North American English)',
    matchName: 'NWL20',
    longDescription:
      'NASPA Word List, 2020 Edition (NWL20), © 2020 North American Word Game Players Association. All rights reserved.',
    // us canada
    // flag: 'https://woogles-flags.s3.us-east-2.amazonaws.com/us.png',
  },
  NSWL20: {
    code: 'NSWL20',
    shortDescription: 'NSWL 20 (NASPA School Word List)',
    matchName: 'NSWL20',
    longDescription:
      'NASPA School Word List 2020 Edition (NSWL20), © 2020 North American Word Game Players Association. All rights reserved.',
  },
  CSW21: {
    code: 'CSW21',
    shortDescription: 'CSW 21 (World English)',
    matchName: 'CSW21',
    longDescription:
      'Published under license with Collins, an imprint of HarperCollins Publishers Limited',
  },
  ECWL: {
    code: 'ECWL',
    shortDescription: 'CEL (Common English Lexicon)',
    matchName: 'CEL',
    longDescription:
      'Common English Lexicon, Copyright (c) 2021-2022 Fj00. Used with permission',
  },
  RD28: {
    code: 'RD28',
    shortDescription: 'Deutsch (German)',
    matchName: 'Deutsch',
    longDescription:
      'The “Scrabble®-Turnierliste” used as the German Lexicon is subject to copyright and related rights of Scrabble® Deutschland e.V. With the friendly assistance of Gero Illings SuperDic.',
    flagCode: 'de',
  },
  NSF21: {
    code: 'NSF21',
    shortDescription: 'Norsk (Norwegian)',
    matchName: 'Norsk',
    longDescription:
      'The NSF word list is provided by the language committee of the Norwegian Scrabble Player Association. Used with permission.',
    flagCode: 'no',
  },
  FRA20: {
    code: 'FRA20',
    shortDescription: 'Français (French)',
    matchName: 'Français',
    flagCode: 'fr',
  },
  CSW19X: {
    code: 'CSW19X',
    shortDescription: 'CSW19X (School Expurgated)',
    matchName: 'CSW19X',
  },
};
