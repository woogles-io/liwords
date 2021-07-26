import { Alphabet } from './alphabets';

export enum PoolFormatType {
  Alphabet,
  VowelConsonant,
  Detail,
}

export type PoolFormat = {
  poolFormatType: PoolFormatType;
  displayName: string;
  format: (alphabet: Alphabet) => string; // A function that returns a comma delimited list of rune sections
};

export const PoolFormats: PoolFormat[] = [
  {
    poolFormatType: PoolFormatType.Alphabet,
    displayName: 'Alphabetical',
    format: (alphabet: Alphabet) =>
      alphabet.letters.map((l) => (l.count > 0 ? l.rune : '')).join(''),
  },
  {
    poolFormatType: PoolFormatType.VowelConsonant,
    displayName: 'Vowels first',
    format: (alphabet: Alphabet) => {
      const vowels = new Array<string>();
      const consonants = new Array<string>();
      alphabet.letters.forEach((l) => {
        if (l.count === 0) {
          return;
        }
        if (l.vowel) {
          vowels.push(l.rune);
        } else {
          consonants.push(l.rune);
        }
      });
      return vowels.join('') + ',' + consonants.join('');
    },
  },
  {
    poolFormatType: PoolFormatType.Detail,
    displayName: 'Detailed',
    // format: 'AEIOU,DGLNRT,BCFHMPVWY,JKQXZS?',
    format: (alphabet: Alphabet) => {
      const categories: { [k: number]: string } = {};
      alphabet.letters.forEach((l) => {
        if (l.category < 0) {
          // Not in the detailed list.
          return;
        }
        if (!categories[l.category]) {
          categories[l.category] = '';
        }
        categories[l.category] += l.rune;
      });
      return Object.keys(categories)
        .sort()
        .map((c) => categories[parseInt(c, 10)])
        .join(',');
    },
  },
];
