/**
 * @fileoverview A mapping between lexicon codes and user-visible lexicon names.
 */

type Lexicon = {
  code: string;
  shortDescription: string;
  ratingName: string; // the name that shows up in your profile ratings
  matchName: string; // the name that shows up in a match/seek/watch
  longDescription?: string;
  flagCode?: string;
};

export const AllLexica: { [code: string]: Lexicon } = {
  NWL20: {
    code: 'NWL20',
    shortDescription: 'NWL 20 (North American English)',
    ratingName: 'NWL',
    matchName: 'NWL20',
    longDescription:
      'NASPA Word List, 2020 Edition (NWL20), © 2020 North American Word Game Players Association. All rights reserved.',
    // us canada
    // flag: 'https://woogles-flags.s3.us-east-2.amazonaws.com/us.png',
  },
  NSWL20: {
    code: 'NSWL20',
    shortDescription: 'NSWL 20 (NASPA School Word List)',
    ratingName: 'NSWL',
    matchName: 'NSWL20',
    longDescription:
      'NASPA School Word List 2020 Edition (NSWL20), © 2020 North American Word Game Players Association. All rights reserved.',
  },
  CSW21: {
    code: 'CSW21',
    shortDescription: 'CSW 21 (World English)',
    ratingName: 'CSW',
    matchName: 'CSW21',
    longDescription:
      'Published under license with Collins, an imprint of HarperCollins Publishers Limited',
  },
  ECWL: {
    code: 'ECWL',
    shortDescription: 'CEL (Common English Lexicon)',
    ratingName: 'CEL',
    matchName: 'CEL',
    longDescription:
      'Common English Lexicon, Copyright (c) 2021 Fj00. Used with permission',
  },
  RD28: {
    code: 'RD28',
    shortDescription: 'Deutsch (German)',
    ratingName: 'Deutsch',
    matchName: 'Deutsch',
    longDescription:
      'The “Scrabble®-Turnierliste” used as the German Lexicon is subject to copyright and related rights of Scrabble® Deutschland e.V. With the friendly assistance of Gero Illings SuperDic.',
    flagCode: 'de',
  },
  NSF21: {
    code: 'NSF21',
    shortDescription: 'Norsk (Norwegian)',
    ratingName: 'Norsk',
    matchName: 'Norsk',
    longDescription:
      'The NSF word list is provided by the language committee of the Norwegian Scrabble Player Association. Used with permission.',
    flagCode: 'no',
  },
  FRA20: {
    code: 'FRA20',
    shortDescription: 'Français (French)',
    ratingName: 'Français',
    matchName: 'Français',
    flagCode: 'fr',
  },
  CSW19X: {
    code: 'CSW19X',
    shortDescription: 'CSW19X (School Expurgated)',
    ratingName: 'CSW',
    matchName: 'CSW19X',
  },
};
