import {
  EmptyRackSpaceMachineLetter,
  MachineLetter,
} from "../utils/cwgame/common";
import { Alphabet } from "./alphabets";

export enum PoolFormatType {
  Alphabet,
  VowelConsonant,
  Detail,
}

export type PoolFormat = {
  poolFormatType: PoolFormatType;
  displayName: string;
  format: (alphabet: Alphabet) => Array<Array<MachineLetter>>;
};

export const PoolFormats: PoolFormat[] = [
  {
    poolFormatType: PoolFormatType.Alphabet,
    displayName: "Alphabetical",
    format: (alphabet: Alphabet) => [
      alphabet.letters
        .map((l, idx) => (l.count > 0 ? idx : EmptyRackSpaceMachineLetter))
        .filter((ml) => ml !== EmptyRackSpaceMachineLetter),
    ],
  },
  {
    poolFormatType: PoolFormatType.VowelConsonant,
    displayName: "Vowels first",
    format: (alphabet: Alphabet) => {
      const vowels = new Array<MachineLetter>();
      const consonants = new Array<MachineLetter>();
      alphabet.letters.forEach((l, idx) => {
        if (l.count === 0) {
          return;
        }
        if (l.vowel) {
          vowels.push(idx);
        } else {
          consonants.push(idx);
        }
      });
      return [vowels, consonants];
    },
  },
  {
    poolFormatType: PoolFormatType.Detail,
    displayName: "Detailed",
    format: (alphabet: Alphabet) => {
      const categories: { [k: number]: Array<MachineLetter> } = {};
      alphabet.letters.forEach((l, idx) => {
        if (l.category < 0) {
          // Not in the detailed list.
          return;
        }
        if (!categories[l.category]) {
          categories[l.category] = new Array<MachineLetter>();
        }
        categories[l.category].push(idx);
      });
      return Object.keys(categories)
        .sort()
        .map((c) => categories[parseInt(c, 10)]);
    },
  },
];
