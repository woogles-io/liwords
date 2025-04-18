import {
  StandardCatalanAlphabet,
  StandardEnglishAlphabet,
  runesToRuneArray,
  runesToMachineWord,
  machineWordToRunes,
  machineWordToRuneArray,
} from "./alphabets";

it("test simple runestoarr", () => {
  const alphabet = StandardEnglishAlphabet;
  expect(runesToMachineWord("COOKIE", alphabet)).toEqual(
    Array.from([3, 15, 15, 11, 9, 5]),
  );
  expect(runesToMachineWord("COoKIE", alphabet)).toEqual(
    Array.from([3, 15, 15 | 0x80, 11, 9, 5]),
  );
  expect(
    machineWordToRunes(Array.from([3, 15, 15 | 0x80, 11, 9, 5]), alphabet),
  ).toEqual("COoKIE");
  expect(
    machineWordToRuneArray(Array.from([3, 15, 15 | 0x80, 11, 9, 5]), alphabet),
  ).toEqual(["C", "O", "o", "K", "I", "E"]);
});

it("test catalan runestoarr", () => {
  // AL·LOQUIMIQUES is only 10 tiles despite being 14 codepoints long.
  // A L·L O QU I M I QU E S

  const alphabet = StandardCatalanAlphabet;
  expect(runesToMachineWord("AL·LOQUIMIQUES", alphabet)).toEqual(
    Array.from([1, 13, 17, 19, 10, 14, 10, 19, 6, 21]),
  );
  expect(runesToMachineWord("Al·lOQUIMIquES", alphabet)).toEqual(
    Array.from([1, 13 | 0x80, 17, 19, 10, 14, 10, 19 | 0x80, 6, 21]),
  );
  expect(runesToRuneArray("Al·lOQUIMIquES", alphabet)).toEqual([
    "A",
    "l·l",
    "O",
    "QU",
    "I",
    "M",
    "I",
    "qu",
    "E",
    "S",
  ]);

  expect(runesToMachineWord("ARQUEGESSIU", alphabet)).toEqual(
    Array.from([1, 20, 19, 6, 8, 6, 21, 21, 10, 23]),
  );
});

it("test catalan uint8ArrayToRunes", () => {
  const alphabet = StandardCatalanAlphabet;
  const arr = Array.from([1, 13 | 0x80, 17, 19, 10, 14, 10, 19 | 0x80, 6, 21]);
  expect(machineWordToRunes(arr, alphabet)).toEqual("Al·lOQUIMIquES");
  expect(machineWordToRuneArray(arr, alphabet)).toEqual([
    "A",
    "l·l",
    "O",
    "QU",
    "I",
    "M",
    "I",
    "qu",
    "E",
    "S",
  ]);
});

it("test playedtiles through", () => {
  const alphabet = StandardCatalanAlphabet;
  const arr = Array.from([1, 13 | 0x80, 0, 0, 10, 14, 10, 19 | 0x80, 6, 0]);
  expect(machineWordToRunes(arr, alphabet, true)).toEqual("Al·l..IMIquE.");
  expect(runesToMachineWord("Al·l..IMIquE.", alphabet)).toEqual(arr);
});
