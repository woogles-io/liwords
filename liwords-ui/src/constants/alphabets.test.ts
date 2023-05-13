import {
  StandardEnglishAlphabet,
  runesToMachineWord,
  machineWordToRunes,
  machineWordToRuneArray,
} from './alphabets';

it('test simple runestoarr', () => {
  const alphabet = StandardEnglishAlphabet;
  expect(runesToMachineWord('COOKIE', alphabet)).toEqual(
    Array.from([3, 15, 15, 11, 9, 5])
  );
  expect(runesToMachineWord('COoKIE', alphabet)).toEqual(
    Array.from([3, 15, 15 | 0x80, 11, 9, 5])
  );
  expect(
    machineWordToRunes(Array.from([3, 15, 15 | 0x80, 11, 9, 5]), alphabet)
  ).toEqual('COoKIE');
  expect(
    machineWordToRuneArray(Array.from([3, 15, 15 | 0x80, 11, 9, 5]), alphabet)
  ).toEqual(['C', 'O', 'o', 'K', 'I', 'E']);
});
