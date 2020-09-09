export enum PoolFormatType {
  Alphabet,
  VowelConsonant,
  Detail,
}

export type PoolFormat = {
  poolFormatType: PoolFormatType;
  displayName: string;
  format: string; // A comma delimited list of rune sections
};

// When we support other languages, we'll want these coming from the db
export const PoolFormats: PoolFormat[] = [
  {
    poolFormatType: PoolFormatType.Alphabet,
    displayName: 'Alphabetical',
    format: 'ABCDEFGHIJKLMNOPQRSTUVWXYZ?',
  },
  {
    poolFormatType: PoolFormatType.VowelConsonant,
    displayName: 'Vowels First',
    format: 'AEIOU,BCDFGHJKLMNPQRSTVWXYZ?',
  },
  {
    poolFormatType: PoolFormatType.Detail,
    displayName: 'Detailed',
    format: 'AEIOU,DGLNRT,BCFHMPVWY,JKQXZS?',
  },
];
