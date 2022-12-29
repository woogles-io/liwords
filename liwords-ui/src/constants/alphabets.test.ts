import {
  runesToUint8Array,
  StandardCatalanAlphabet,
  StandardEnglishAlphabet,
} from './alphabets';

it('test simple runestoarr', () => {
  const alphabet = StandardEnglishAlphabet;
  expect(runesToUint8Array('COOKIE', alphabet)).toEqual(
    Uint8Array.from([3, 15, 15, 11, 9, 5])
  );
  expect(runesToUint8Array('COoKIE', alphabet)).toEqual(
    Uint8Array.from([3, 15, 241, 11, 9, 5])
  );
});

it('test catalan runestoarr', () => {
  // AL·LOQUIMIQUES is only 10 tiles despite being 14 codepoints long.
  // A L·L O QU I M I QU E S

  const alphabet = StandardCatalanAlphabet;
  expect(runesToUint8Array('AL·LOQUIMIQUES', alphabet)).toEqual(
    Uint8Array.from([1, 13, 17, 19, 10, 14, 10, 19, 6, 21])
  );
  expect(runesToUint8Array('Al·lOQUIMIquES', alphabet)).toEqual(
    Uint8Array.from([1, 243, 17, 19, 10, 14, 10, 237, 6, 21])
  );
});
