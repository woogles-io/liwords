/** @fileoverview This should subsume most of the files in this directory.
 * Whilst there is a case to be made for fetching most of this stuff from
 * the backend, it is significantly simpler to deal with this in the frontend
 * for now.
 */

import { Blank, EmptySpace } from '../utils/cwgame/common';
import { ThroughTileMarker } from '../utils/cwgame/game_event';

type AlphabetLetter = {
  rune: string; // the physical displayed character(s)
  score: number;
  count: number; // how many of these there are in the bag
  vowel: boolean;
  category: number; // for detailed view, we split letters into groups.
  // for example:  'AEIOU,DGLNRT,BCFHMPVWY,JKQXZS?'
};

export type Alphabet = {
  // Order of letters should be in the desired sorting order for that lexicon.
  letters: Array<AlphabetLetter>;
  // letterMap creates a structure that is faster to access than a list
  letterMap: { [key: string]: AlphabetLetter };
  machineLetterMap: { [key: string]: number };
  // For Catalan we will have L·L, NY, etc. Spanish also has a couple of two-
  // character tiles.
  longestPossibleTileRune: number;
};

export const StandardEnglishAlphabet: Alphabet = {
  letters: [
    { rune: Blank, score: 0, count: 2, vowel: false, category: 3 },
    { rune: 'A', score: 1, count: 9, vowel: true, category: 0 },
    { rune: 'B', score: 3, count: 2, vowel: false, category: 2 },
    { rune: 'C', score: 3, count: 2, vowel: false, category: 2 },
    { rune: 'D', score: 2, count: 4, vowel: false, category: 1 },
    { rune: 'E', score: 1, count: 12, vowel: true, category: 0 },
    { rune: 'F', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'G', score: 2, count: 3, vowel: false, category: 1 },
    { rune: 'H', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'I', score: 1, count: 9, vowel: true, category: 0 },
    { rune: 'J', score: 8, count: 1, vowel: false, category: 3 },
    { rune: 'K', score: 5, count: 1, vowel: false, category: 3 },
    { rune: 'L', score: 1, count: 4, vowel: false, category: 1 },
    { rune: 'M', score: 3, count: 2, vowel: false, category: 2 },
    { rune: 'N', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'O', score: 1, count: 8, vowel: true, category: 0 },
    { rune: 'P', score: 3, count: 2, vowel: false, category: 2 },
    { rune: 'Q', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'R', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'S', score: 1, count: 4, vowel: false, category: 3 },
    { rune: 'T', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'U', score: 1, count: 4, vowel: true, category: 0 },
    { rune: 'V', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'W', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'X', score: 8, count: 1, vowel: false, category: 3 },
    { rune: 'Y', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'Z', score: 10, count: 1, vowel: false, category: 3 },
  ],
  letterMap: {},
  machineLetterMap: {},
  longestPossibleTileRune: 1,
};

export const StandardGermanAlphabet: Alphabet = {
  letters: [
    { rune: Blank, score: 0, count: 2, vowel: false, category: 3 },
    { rune: 'A', score: 1, count: 5, vowel: true, category: 0 },
    { rune: 'Ä', score: 6, count: 1, vowel: true, category: 3 },
    { rune: 'B', score: 3, count: 2, vowel: false, category: 2 },
    { rune: 'C', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'D', score: 1, count: 4, vowel: false, category: 1 },
    { rune: 'E', score: 1, count: 15, vowel: true, category: 0 },
    { rune: 'F', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'G', score: 2, count: 3, vowel: false, category: 1 },
    { rune: 'H', score: 2, count: 4, vowel: false, category: 1 },
    { rune: 'I', score: 1, count: 6, vowel: true, category: 0 },
    { rune: 'J', score: 6, count: 1, vowel: false, category: 3 },
    { rune: 'K', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'L', score: 2, count: 3, vowel: false, category: 1 },
    { rune: 'M', score: 3, count: 4, vowel: false, category: 2 },
    { rune: 'N', score: 1, count: 9, vowel: false, category: 1 },
    { rune: 'O', score: 2, count: 3, vowel: true, category: 0 },
    { rune: 'Ö', score: 8, count: 1, vowel: true, category: 3 },
    { rune: 'P', score: 4, count: 1, vowel: false, category: 2 },
    { rune: 'Q', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'R', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'S', score: 1, count: 7, vowel: false, category: 1 },
    { rune: 'T', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'U', score: 1, count: 6, vowel: true, category: 0 },
    { rune: 'Ü', score: 6, count: 1, vowel: true, category: 3 },
    { rune: 'V', score: 6, count: 1, vowel: false, category: 3 },
    { rune: 'W', score: 3, count: 1, vowel: false, category: 2 },
    { rune: 'X', score: 8, count: 1, vowel: false, category: 3 },
    { rune: 'Y', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'Z', score: 3, count: 1, vowel: false, category: 2 },
  ],
  letterMap: {},
  machineLetterMap: {},

  longestPossibleTileRune: 1,
};

export const StandardNorwegianAlphabet: Alphabet = {
  letters: [
    { rune: Blank, score: 0, count: 2, vowel: false, category: 3 },
    { rune: 'A', score: 1, count: 7, vowel: true, category: 0 },
    { rune: 'B', score: 4, count: 3, vowel: false, category: 2 },
    { rune: 'C', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'D', score: 1, count: 5, vowel: false, category: 1 },
    { rune: 'E', score: 1, count: 9, vowel: true, category: 0 },
    { rune: 'F', score: 2, count: 4, vowel: false, category: 1 },
    { rune: 'G', score: 2, count: 4, vowel: false, category: 1 },
    { rune: 'H', score: 3, count: 3, vowel: false, category: 2 },
    { rune: 'I', score: 1, count: 5, vowel: true, category: 0 },
    { rune: 'J', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'K', score: 2, count: 4, vowel: false, category: 1 },
    { rune: 'L', score: 1, count: 5, vowel: false, category: 1 },
    { rune: 'M', score: 2, count: 3, vowel: false, category: 1 },
    { rune: 'N', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'O', score: 2, count: 4, vowel: true, category: 0 },
    { rune: 'P', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'Q', score: 0, count: 0, vowel: false, category: -1 },
    { rune: 'R', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'S', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'T', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'U', score: 4, count: 3, vowel: true, category: 0 },
    { rune: 'V', score: 4, count: 3, vowel: false, category: 2 },
    { rune: 'W', score: 8, count: 1, vowel: false, category: 3 },
    { rune: 'X', score: 0, count: 0, vowel: false, category: -1 },
    { rune: 'Y', score: 6, count: 1, vowel: false, category: 3 },
    { rune: 'Ü', score: 0, count: 0, vowel: true, category: -1 },
    { rune: 'Z', score: 0, count: 0, vowel: false, category: -1 },
    { rune: 'Æ', score: 6, count: 1, vowel: true, category: 3 },
    // Norwegian has several letters in its alphabet that there are zero of
    // (we need to show them here for the blank designation panel)
    { rune: 'Ä', score: 0, count: 0, vowel: true, category: -1 },
    { rune: 'Ø', score: 5, count: 2, vowel: true, category: 3 },
    { rune: 'Ö', score: 0, count: 0, vowel: true, category: -1 },
    { rune: 'Å', score: 4, count: 2, vowel: true, category: 2 },
  ],
  letterMap: {},
  machineLetterMap: {},

  longestPossibleTileRune: 1,
};

export const StandardFrenchAlphabet: Alphabet = {
  letters: [
    { rune: Blank, score: 0, count: 2, vowel: false, category: 3 },
    { rune: 'A', score: 1, count: 9, vowel: true, category: 0 },
    { rune: 'B', score: 3, count: 2, vowel: false, category: 2 },
    { rune: 'C', score: 3, count: 2, vowel: false, category: 2 },
    { rune: 'D', score: 2, count: 3, vowel: false, category: 1 },
    { rune: 'E', score: 1, count: 15, vowel: true, category: 0 },
    { rune: 'F', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'G', score: 2, count: 2, vowel: false, category: 1 },
    { rune: 'H', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'I', score: 1, count: 8, vowel: true, category: 0 },
    { rune: 'J', score: 8, count: 1, vowel: false, category: 3 },
    { rune: 'K', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'L', score: 1, count: 5, vowel: false, category: 1 },
    { rune: 'M', score: 2, count: 3, vowel: false, category: 1 },
    { rune: 'N', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'O', score: 1, count: 6, vowel: true, category: 0 },
    { rune: 'P', score: 3, count: 2, vowel: false, category: 1 },
    { rune: 'Q', score: 8, count: 1, vowel: false, category: 3 },
    { rune: 'R', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'S', score: 1, count: 6, vowel: false, category: 3 },
    { rune: 'T', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'U', score: 1, count: 6, vowel: true, category: 0 },
    { rune: 'V', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'W', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'X', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'Y', score: 10, count: 1, vowel: true, category: 3 },
    { rune: 'Z', score: 10, count: 1, vowel: false, category: 3 },
  ],
  letterMap: {},
  machineLetterMap: {},

  longestPossibleTileRune: 1,
};

export const SuperEnglishAlphabet: Alphabet = {
  letters: [
    { rune: Blank, score: 0, count: 4, vowel: false, category: 3 },
    { rune: 'A', score: 1, count: 16, vowel: true, category: 0 },
    { rune: 'B', score: 3, count: 4, vowel: false, category: 2 },
    { rune: 'C', score: 3, count: 6, vowel: false, category: 2 },
    { rune: 'D', score: 2, count: 8, vowel: false, category: 1 },
    { rune: 'E', score: 1, count: 24, vowel: true, category: 0 },
    { rune: 'F', score: 4, count: 4, vowel: false, category: 2 },
    { rune: 'G', score: 2, count: 5, vowel: false, category: 1 },
    { rune: 'H', score: 4, count: 5, vowel: false, category: 2 },
    { rune: 'I', score: 1, count: 13, vowel: true, category: 0 },
    { rune: 'J', score: 8, count: 2, vowel: false, category: 3 },
    { rune: 'K', score: 5, count: 2, vowel: false, category: 3 },
    { rune: 'L', score: 1, count: 7, vowel: false, category: 1 },
    { rune: 'M', score: 3, count: 6, vowel: false, category: 2 },
    { rune: 'N', score: 1, count: 13, vowel: false, category: 1 },
    { rune: 'O', score: 1, count: 15, vowel: true, category: 0 },
    { rune: 'P', score: 3, count: 4, vowel: false, category: 2 },
    { rune: 'Q', score: 10, count: 2, vowel: false, category: 3 },
    { rune: 'R', score: 1, count: 13, vowel: false, category: 1 },
    { rune: 'S', score: 1, count: 10, vowel: false, category: 3 },
    { rune: 'T', score: 1, count: 15, vowel: false, category: 1 },
    { rune: 'U', score: 1, count: 7, vowel: true, category: 0 },
    { rune: 'V', score: 4, count: 3, vowel: false, category: 2 },
    { rune: 'W', score: 4, count: 4, vowel: false, category: 2 },
    { rune: 'X', score: 8, count: 2, vowel: false, category: 3 },
    { rune: 'Y', score: 4, count: 4, vowel: false, category: 2 },
    { rune: 'Z', score: 10, count: 2, vowel: false, category: 3 },
  ],
  letterMap: {},
  machineLetterMap: {},

  longestPossibleTileRune: 1,
};

export const StandardCatalanAlphabet: Alphabet = {
  letters: [
    { rune: Blank, score: 0, count: 2, vowel: false, category: 3 },
    { rune: 'A', score: 1, count: 12, vowel: true, category: 0 },
    { rune: 'B', score: 3, count: 2, vowel: false, category: 2 },
    { rune: 'C', score: 2, count: 3, vowel: false, category: 2 },
    { rune: 'Ç', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'D', score: 2, count: 3, vowel: false, category: 1 },
    { rune: 'E', score: 1, count: 13, vowel: true, category: 0 },
    { rune: 'F', score: 4, count: 1, vowel: false, category: 2 },
    { rune: 'G', score: 3, count: 2, vowel: false, category: 1 },
    { rune: 'H', score: 8, count: 1, vowel: false, category: 2 },
    { rune: 'I', score: 1, count: 8, vowel: true, category: 0 },
    { rune: 'J', score: 8, count: 1, vowel: false, category: 3 },
    { rune: 'L', score: 1, count: 4, vowel: false, category: 1 },
    { rune: 'L·L', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'M', score: 2, count: 3, vowel: false, category: 2 },
    { rune: 'N', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'NY', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'O', score: 1, count: 5, vowel: true, category: 0 },
    { rune: 'P', score: 3, count: 2, vowel: false, category: 2 },
    { rune: 'QU', score: 8, count: 1, vowel: false, category: 3 },
    { rune: 'R', score: 1, count: 8, vowel: false, category: 1 },
    { rune: 'S', score: 1, count: 8, vowel: false, category: 3 },
    { rune: 'T', score: 1, count: 5, vowel: false, category: 1 },
    { rune: 'U', score: 1, count: 4, vowel: true, category: 0 },
    { rune: 'V', score: 4, count: 1, vowel: false, category: 2 },
    { rune: 'X', score: 10, count: 1, vowel: false, category: 3 },
    { rune: 'Z', score: 8, count: 1, vowel: false, category: 3 },
  ],
  letterMap: {},
  machineLetterMap: {},
  longestPossibleTileRune: 3,
};

// Create letter maps for faster access.
[
  StandardEnglishAlphabet,
  StandardGermanAlphabet,
  StandardNorwegianAlphabet,
  StandardFrenchAlphabet,
  SuperEnglishAlphabet,
  StandardCatalanAlphabet,
].forEach((alph) => {
  alph.letters.forEach((letter, idx) => {
    alph.letterMap[letter.rune] = letter;
    alph.machineLetterMap[letter.rune] = idx;
  });
});

export const alphabetFromName = (
  letterDistribution: string | undefined
): Alphabet => {
  switch (letterDistribution) {
    case 'norwegian':
      return StandardNorwegianAlphabet;
    case 'german':
      return StandardGermanAlphabet;
    case 'english':
      return StandardEnglishAlphabet;
    case 'french':
      return StandardFrenchAlphabet;
    case 'english_super':
      return SuperEnglishAlphabet;
    default:
      return StandardEnglishAlphabet;
  }
};

export const runeToValues = (
  alphabet: Alphabet,
  rune: string | null
): number => {
  if (rune === null) {
    return 0;
  }
  if (alphabet.letterMap[rune]) {
    return alphabet.letterMap[rune].score;
  }
  return 0;
};

export const uint8ToRune = (
  i: number,
  alphabet: Alphabet,
  usePlaythrough?: boolean
): string => {
  if (i === 0) {
    return usePlaythrough ? ThroughTileMarker : Blank;
  }
  if (i > 0x80) {
    return alphabet.letters[i & 0x7f]?.rune?.toLowerCase() ?? '';
  }
  return alphabet.letters[i]?.rune ?? '';
};

export const uint8ArrayToRunes = (
  arr: Uint8Array,
  alphabet: Alphabet,
  usePlaythrough?: boolean
): string => {
  let s = '';
  arr.forEach((v) => {
    s += uint8ToRune(v, alphabet, usePlaythrough);
  });
  return s;
};

export const runesToUint8Array = (
  runes: string,
  alphabet: Alphabet
): Uint8Array => {
  const bts = [];
  const chars = Array.from(runes);
  let i = 0;
  let match;
  while (i < chars.length) {
    match = false;
    for (let j = i + alphabet.longestPossibleTileRune; j > i; j--) {
      const rune = chars.slice(i, j).join('');
      if (alphabet.machineLetterMap[rune] != undefined) {
        bts.push(alphabet.machineLetterMap[rune]);
        i = j;
        match = true;
        break;
      } else if (alphabet.machineLetterMap[rune.toUpperCase()] != undefined) {
        bts.push(0x80 | alphabet.machineLetterMap[rune.toUpperCase()]);
        i = j;
        match = true;
        break;
      }
    }
    if (!match) {
      // Check if it's a through play.
      // This is not very clean.
      if (chars[i] == ThroughTileMarker) {
        bts.push(0);
        i++;
      } else {
        throw new Error('cannot convert ' + runes + ' to uint8array');
      }
    }
  }

  return Uint8Array.from(bts);
};
