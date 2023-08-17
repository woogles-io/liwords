/** @fileoverview business logic for handling blindfold events */

export const letterPronunciations = new Map([
  ['A', 'A'],
  ['B', 'B'],
  ['C', 'C'],
  ['D', 'D'],
  ['E', 'E'],
  ['F', 'F'],
  ['G', 'G'],
  ['H', 'H'],
  ['I', 'I'],
  ['J', 'J'],
  ['K', 'K'],
  ['L', 'L'],
  ['M', 'M'],
  ['N', 'N'],
  ['O', 'O'],
  ['P', 'P'],
  ['Q', 'Q'],
  ['R', 'R'],
  ['S', 'S'],
  ['T', 'T'],
  ['U', 'U'],
  ['V', 'V'],
  ['W', 'W'],
  ['X', 'X'],
  ['Y', 'Y'],
  ['Z', 'Z'],
]);

export const natoPhoneticAlphabet = new Map([
  ['A', 'Alpha'],
  ['B', 'Bravo'],
  ['C', 'Charlie'],
  ['D', 'Delta'],
  ['E', 'Echo'],
  ['F', 'Foxtrot'],
  ['G', 'Golf'],
  ['H', 'Hotel'],
  ['I', 'India'],
  ['J', 'Juliett'],
  ['K', 'Kilo'],
  ['L', 'Lima'],
  ['M', 'Mike'],
  ['N', 'November'],
  ['O', 'Oscar'],
  ['P', 'Papa'],
  ['Q', 'Quebec'],
  ['R', 'Romeo'],
  ['S', 'Sierra'],
  ['T', 'Tango'],
  ['U', 'Uniform'],
  ['V', 'Victor'],
  ['W', 'Whiskey'],
  ['X', 'X-ray'],
  ['Y', 'Yankee'],
  ['Z', 'Zulu'],
]);

export const say = (text: string, moreText: string) => {
  const speech = new SpeechSynthesisUtterance(text);
  const lang = 'en-US';
  const rate = 0.8;
  speech.lang = lang;
  speech.rate = rate;
  window.speechSynthesis.cancel();
  speech.onend = () => {
    if (moreText !== '') {
      const moreSpeech = new SpeechSynthesisUtterance(moreText);
      moreSpeech.lang = lang;
      moreSpeech.rate = rate;
      window.speechSynthesis.cancel();
      speechSynthesis.speak(moreSpeech);
    }
  };
  window.speechSynthesis.speak(speech);
};

export const wordToSayString = (word: string, useNPA: boolean): string => {
  let speech = '';
  let currentNumber = '';
  for (let i = 0; i < word.length; i++) {
    const natoWord = natoPhoneticAlphabet.get(word[i].toUpperCase());
    if (natoWord !== undefined) {
      // Single letters in their own sentences are usually
      // fairly understandable when spoken by TTS. In some cases
      // it is unclear and using the NATO Phonetic Alphabet
      // will remove the ambiguity.
      if (word[i] >= 'a' && word[i] <= 'z') {
        speech += 'blank, ';
      }
      if (useNPA) {
        speech += natoWord + ', ';
      } else {
        const pword = letterPronunciations.get(word[i].toUpperCase());
        speech += `${pword}, `;
      }
    } else if (word[i] === '?') {
      speech += 'blank, ';
    } else {
      // It's a number
      let middleOfNumber = false;
      currentNumber += word[i];
      if (i + 1 < word.length) {
        const natoNextWord = natoPhoneticAlphabet.get(
          word[i + 1].toUpperCase()
        );
        if (natoNextWord === undefined) {
          middleOfNumber = true;
        }
      }
      if (!middleOfNumber) {
        speech += currentNumber + '. ';
        currentNumber = '';
      }
    }
  }
  return speech;
};
