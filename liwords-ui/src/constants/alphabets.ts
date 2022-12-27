/** @fileoverview This should subsume most of the files in this directory.
 * Whilst there is a case to be made for fetching most of this stuff from
 * the backend, it is significantly simpler to deal with this in the frontend
 * for now.
 */

import { Blank } from '../utils/cwgame/common';

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
};

export const StandardEnglishAlphabet: Alphabet = {
  letters: [
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
    { rune: Blank, score: 0, count: 2, vowel: false, category: 3 },
  ],
  letterMap: {},
};

export const StandardGermanAlphabet: Alphabet = {
  letters: [
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
    { rune: Blank, score: 0, count: 2, vowel: false, category: 3 },
  ],
  letterMap: {},
};

export const StandardNorwegianAlphabet: Alphabet = {
  letters: [
    { rune: 'A', score: 1, count: 7, vowel: true, category: 0 },
    // Norwegian has several letters in its alphabet that there are zero of
    // (we need to show them here for the blank designation panel)
    { rune: 'Ä', score: 0, count: 0, vowel: true, category: -1 },
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
    { rune: 'Ö', score: 0, count: 0, vowel: true, category: -1 },
    { rune: 'P', score: 4, count: 2, vowel: false, category: 2 },
    { rune: 'Q', score: 0, count: 0, vowel: false, category: -1 },
    { rune: 'R', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'S', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'T', score: 1, count: 6, vowel: false, category: 1 },
    { rune: 'U', score: 4, count: 3, vowel: true, category: 0 },
    { rune: 'Ü', score: 0, count: 0, vowel: true, category: -1 },
    { rune: 'V', score: 4, count: 3, vowel: false, category: 2 },
    { rune: 'W', score: 8, count: 1, vowel: false, category: 3 },
    { rune: 'X', score: 0, count: 0, vowel: false, category: -1 },
    { rune: 'Y', score: 6, count: 1, vowel: false, category: 3 },
    { rune: 'Z', score: 0, count: 0, vowel: false, category: -1 },
    { rune: 'Æ', score: 6, count: 1, vowel: true, category: 3 },
    { rune: 'Ø', score: 5, count: 2, vowel: true, category: 3 },
    { rune: 'Å', score: 4, count: 2, vowel: true, category: 2 },
    { rune: Blank, score: 0, count: 2, vowel: false, category: 3 },
  ],
  letterMap: {},
};

export const StandardFrenchAlphabet: Alphabet = {
  letters: [
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
    { rune: Blank, score: 0, count: 2, vowel: false, category: 3 },
  ],
  letterMap: {},
};

export const SuperEnglishAlphabet: Alphabet = {
  letters: [
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
    { rune: Blank, score: 0, count: 4, vowel: false, category: 3 },
  ],
  letterMap: {},
};

// Create letter maps for faster access.
[
  StandardEnglishAlphabet,
  StandardGermanAlphabet,
  StandardNorwegianAlphabet,
  StandardFrenchAlphabet,
  SuperEnglishAlphabet,
].forEach((alph) => {
  alph.letters.forEach((letter) => {
    alph.letterMap[letter.rune] = letter;
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

export const uint8ToRune = (i: number, alphabet: Alphabet): string => {
  // Our internal encoding has the blank at 0 and everything begins at 1.
  // This is not the order the runes are listed in above; let's make the
  // change here.
  if (i === 0) {
    return Blank;
  }
  return alphabet.letters[i + 1]?.rune ?? '';
};

export const uint8ArrayToRunes = (
  arr: Uint8Array,
  alphabet: Alphabet
): string => {
  let s = '';
  arr.forEach((v) => {
    s += uint8ToRune(v, alphabet);
  });
  return s;
};
