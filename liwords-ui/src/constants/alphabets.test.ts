import {
  StandardCatalanAlphabet,
  StandardEnglishAlphabet,
  StandardGermanAlphabet,
  StandardSpanishAlphabet,
  SuperCatalanAlphabet,
  alphabetFromName,
  getMachineLetterForKey,
  labelsToTiles,
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

it("has the super Catalan set: the Catalan letters, 200 of them", () => {
  const letters = SuperCatalanAlphabet.letters;
  expect(letters.map((l) => l.rune)).toEqual(
    StandardCatalanAlphabet.letters.map((l) => l.rune),
  );
  expect(letters.reduce((n, l) => n + l.count, 0)).toBe(200);
  // the cedilla C and the geminated L score more than in the Catalan set
  const score = (rune: string) => SuperCatalanAlphabet.letterMap[rune].score;
  expect([score("Ç"), score("L·L"), score("NY")]).toEqual([12, 15, 10]);
  expect(alphabetFromName("catalan_super")).toBe(SuperCatalanAlphabet);
  expect(runesToMachineWord("AL·LOQUIMIQUES", SuperCatalanAlphabet)).toEqual(
    runesToMachineWord("AL·LOQUIMIQUES", StandardCatalanAlphabet),
  );
});

describe("tiles read by their labels", () => {
  const ml = (rune: string, alphabet = StandardEnglishAlphabet) =>
    alphabet.machineLetterMap[rune];

  it("reads each tile's label, and the one it takes on a blank", () => {
    expect(labelsToTiles("Ab?", StandardEnglishAlphabet)).toEqual([
      ml("A"),
      ml("B"),
      0,
    ]);
    const es = StandardSpanishAlphabet;
    expect(labelsToTiles("[ch]i", es)).toEqual([ml("[CH]", es), ml("I", es)]);
    // a C and an H are another spelling
    expect(labelsToTiles("chi", es)).toEqual([
      ml("C", es),
      ml("H", es),
      ml("I", es),
    ]);
    const ca = StandardCatalanAlphabet;
    expect(labelsToTiles("l·l", ca)).toEqual([ml("L·L", ca)]);
    const de = StandardGermanAlphabet;
    expect(labelsToTiles("äÜ", de)).toEqual([ml("Ä", de), ml("Ü", de)]);
  });

  it("finds no tile where none is named, case never changed", () => {
    // upper-cased, ß would read as two S tiles
    expect(labelsToTiles("ß", StandardGermanAlphabet)).toBeNull();
    // a label as written, or as a blank writes it: never in other case
    expect(labelsToTiles("[Ch]", StandardSpanishAlphabet)).toBeNull();
    expect(labelsToTiles("a1", StandardEnglishAlphabet)).toBeNull();
    expect(labelsToTiles("", StandardEnglishAlphabet)).toEqual([]);
  });

  it("reads a '.' as the blank and a space as an empty slot, as before", () => {
    expect(labelsToTiles("a. b", StandardEnglishAlphabet)).toEqual(
      runesToMachineWord("A. B", StandardEnglishAlphabet),
    );
  });

  it("finds a key's tile by its labels, or the board's shortcut in either case", () => {
    expect(getMachineLetterForKey("a", StandardEnglishAlphabet)).toBe(ml("A"));
    expect(getMachineLetterForKey("A", StandardEnglishAlphabet)).toBe(ml("A"));
    expect(getMachineLetterForKey("?", StandardEnglishAlphabet)).toBe(0);
    expect(getMachineLetterForKey("1", StandardEnglishAlphabet)).toBeNull();
    const es = StandardSpanishAlphabet;
    expect(getMachineLetterForKey("K", es)).toBe(ml("[CH]", es));
    expect(getMachineLetterForKey("k", es)).toBe(ml("[CH]", es));
  });
});
