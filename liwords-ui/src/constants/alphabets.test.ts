import {
  runesToUint8Array,
  StandardCatalanAlphabet,
  StandardEnglishAlphabet,
  uint8ArrayToRunes,
  runesToRuneArray,
  uint8ArrayToRuneArray,
} from './alphabets';

it('test simple runestoarr', () => {
  const alphabet = StandardEnglishAlphabet;
  expect(runesToUint8Array('COOKIE', alphabet)).toEqual(
    Uint8Array.from([3, 15, 15, 11, 9, 5])
  );
  expect(runesToUint8Array('COoKIE', alphabet)).toEqual(
    Uint8Array.from([3, 15, 15 | 0x80, 11, 9, 5])
  );
  expect(
    uint8ArrayToRunes(Uint8Array.from([3, 15, 15 | 0x80, 11, 9, 5]), alphabet)
  ).toEqual('COoKIE');
  expect(
    uint8ArrayToRuneArray(
      Uint8Array.from([3, 15, 15 | 0x80, 11, 9, 5]),
      alphabet
    )
  ).toEqual(['C', 'O', 'o', 'K', 'I', 'E']);
});

it('test catalan runestoarr', () => {
  // AL·LOQUIMIQUES is only 10 tiles despite being 14 codepoints long.
  // A L·L O QU I M I QU E S

  const alphabet = StandardCatalanAlphabet;
  expect(runesToUint8Array('AL·LOQUIMIQUES', alphabet)).toEqual(
    Uint8Array.from([1, 13, 17, 19, 10, 14, 10, 19, 6, 21])
  );
  expect(runesToUint8Array('Al·lOQUIMIquES', alphabet)).toEqual(
    Uint8Array.from([1, 13 | 0x80, 17, 19, 10, 14, 10, 19 | 0x80, 6, 21])
  );
  expect(runesToRuneArray('Al·lOQUIMIquES', alphabet)).toEqual([
    'A',
    'l·l',
    'O',
    'QU',
    'I',
    'M',
    'I',
    'qu',
    'E',
    'S',
  ]);

  expect(runesToUint8Array('ARQUEGESSIU', alphabet)).toEqual(
    Uint8Array.from([1, 20, 19, 6, 8, 6, 21, 21, 10, 23])
  );
});

it('test catalan uint8ArrayToRunes', () => {
  const alphabet = StandardCatalanAlphabet;
  const arr = Uint8Array.from([
    1,
    13 | 0x80,
    17,
    19,
    10,
    14,
    10,
    19 | 0x80,
    6,
    21,
  ]);
  expect(uint8ArrayToRunes(arr, alphabet)).toEqual('Al·lOQUIMIquES');
  expect(uint8ArrayToRuneArray(arr, alphabet)).toEqual([
    'A',
    'l·l',
    'O',
    'QU',
    'I',
    'M',
    'I',
    'qu',
    'E',
    'S',
  ]);
});

it('test playedtiles through', () => {
  const alphabet = StandardCatalanAlphabet;
  const arr = Uint8Array.from([
    1,
    13 | 0x80,
    0,
    0,
    10,
    14,
    10,
    19 | 0x80,
    6,
    0,
  ]);
  expect(uint8ArrayToRunes(arr, alphabet, true)).toEqual('Al·l..IMIquE.');
  expect(runesToUint8Array('Al·l..IMIquE.', alphabet)).toEqual(arr);
});
