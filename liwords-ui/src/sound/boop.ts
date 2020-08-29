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

export const BoopSounds = {
  makeMoveSound,
  oppMoveSound,
  matchReqSound,
  startgameSound,
  endgameSound,
  woofSound,
};
