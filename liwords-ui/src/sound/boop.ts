const makemoveMP3 = require('../assets/makemove.mp3');
const startgameMP3 = require('../assets/startgame.mp3');
const oppMoveMP3 = require('../assets/oppmove.mp3');
const matchReqMP3 = require('../assets/matchreq.mp3');
const woofWav = require('../assets/woof.wav');
const endgameMP3 = require('../assets/endgame.mp3');

const makeMoveSound = new Audio(makemoveMP3);
const oppMoveSound = new Audio(oppMoveMP3);
const matchReqSound = new Audio(matchReqMP3);

const startgameSound = new Audio(startgameMP3);
const endgameSound = new Audio(endgameMP3);
const woofSound = new Audio(woofWav);
woofSound.volume = 0.25;

const playableSounds: { [key: string]: HTMLAudioElement } = {
  makeMoveSound,
  oppMoveSound,
  matchReqSound,
  startgameSound,
  endgameSound,
  woofSound,
};

const playSound = (soundName: string) => {
  const audio = playableSounds[soundName];
  if (!audio) {
    throw new TypeError(`unsupported sound: ${soundName}`);
  }
  (async () => {
    try {
      await audio.play();
    } catch (e) {
      console.warn(`cannot play ${soundName}:`, e);
    }
  })();
};

export const BoopSounds = {
  playSound,
};
