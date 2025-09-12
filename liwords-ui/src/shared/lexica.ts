/**
 * @fileoverview A mapping between lexicon codes and user-visible lexicon names.
 */

// internal rating name is saved in db (select ratings->'Data' from profiles).
// it follows pkg/entity/ratings.go ToVariantKey.
const lexiconCodeToInternalRatingName = (code: string) => {
  if (code.startsWith("NWL")) return "NWL18";
  if (code.startsWith("CSW")) return "CSW19";
  if (code.startsWith("ECWL")) return "ECWL";
  if (code.startsWith("NSF")) return "NSF21";
  if (code.startsWith("RD")) return "RD28";
  if (code.startsWith("FRA")) return "FRA20";
  if (code.startsWith("DISC")) return "DISC";
  if (code.startsWith("OSPS")) return "OSPS";
  if (code.startsWith("FILE")) return "FILE";
  return code;
};

// profile rating name is the name that shows up in your profile ratings.
const InternalRatingNameToProfileRatingName: {
  [code: string]: string;
} = {
  // Internal rating names are old versions of the lexica. Not ideal,
  // but we can redo this later, maybe.
  NWL18: "NWL",
  NSWL20: "NSWL",
  CSW19: "CSW",
  ECWL: "CEL",
  RD28: "Deutsch",
  NSF21: "Norsk",
  FRA20: "Français",
  DISC: "Català",
  OSPS: "Polski",
  FILE: "Español",
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
  NWL23: {
    code: "NWL23",
    shortDescription: "NWL 23 (North American English)",
    matchName: "NWL23",
    longDescription:
      "NASPA Word List, 2023 Edition (NWL23), © 2023 North American Word Game Players Association. All rights reserved.",
    // us canada
    // flag: 'https://woogles-flags.s3.us-east-2.amazonaws.com/us.png',
  },
  NSWL20: {
    code: "NSWL20",
    shortDescription: "NSWL 20 (NASPA School Word List)",
    matchName: "NSWL20",
    longDescription:
      "NASPA School Word List 2020 Edition (NSWL20), © 2020 North American Word Game Players Association. All rights reserved.",
  },
  CSW24: {
    code: "CSW24",
    shortDescription: "CSW 24 (World English)",
    matchName: "CSW24",
    longDescription:
      "Published under license with Collins, an imprint of HarperCollins Publishers Limited",
  },
  ECWL: {
    code: "ECWL",
    shortDescription: "CEL (Common English Lexicon)",
    matchName: "CEL",
    longDescription:
      "Common English Lexicon, Copyright (c) 2021-2022 Fj00. Used with permission",
  },
  RD29: {
    code: "RD29",
    shortDescription: "Deutsch (German)",
    matchName: "Deutsch",
    longDescription:
      "The “Scrabble®-Turnierliste” used as the German Lexicon is subject to copyright and related rights of Scrabble® Deutschland e.V. With the friendly assistance of Gero Illings SuperDic.",
    flagCode: "de",
  },
  NSF25: {
    code: "NSF25",
    shortDescription: "Norsk (Norwegian)",
    matchName: "Norsk",
    longDescription:
      "The NSF word list is provided by the language committee of the Norwegian Scrabble Player Association. Used with permission.",
    flagCode: "no",
  },
  FRA24: {
    code: "FRA24",
    shortDescription: "Français (French)",
    matchName: "Français",
    flagCode: "fr",
  },
  FILE2017: {
    code: "FILE2017",
    shortDescription: "Español (Spanish)",
    matchName: "Español",
    longDescription:
      "Copyright 2017 Federación Internacional de Léxico en Español",
    flagCode: "es",
  },
  OSPS50: {
    code: "OSPS50",
    shortDescription: "Polski (Polish)",
    matchName: "Polski",
    longDescription:
      "Copyright 2025 Polska Federacja Scrabble. Used with permission.",
    flagCode: "pl",
  },
  CSW24X: {
    code: "CSW24X",
    shortDescription: "CSW24X (School Expurgated)",
    matchName: "CSW24X",
  },
  DISC2: {
    code: "DISC2",
    shortDescription: "Català (Catalan)",
    matchName: "Català",
    longDescription:
      "«Diccionari Informatitzat de l'Scrabble en Català» (DISC) is GPLv3-licensed. Copyright 2012 - 2022 Joan Montané.",
  },
};
