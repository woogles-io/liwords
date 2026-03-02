import makemoveSound from "../assets/makemove.mp3";
import oppmoveSound from "../assets/oppmove.mp3";
import matchreqSound from "../assets/matchreq.mp3";
import startgameSound from "../assets/startgame.mp3";
import endgameSound from "../assets/endgame.mp3";
import woofSound from "../assets/woof.wav";
import meowSound from "../assets/meow.mp3";
import receivechatSound from "../assets/receivechat.mp3";
import newtourneyroundSound from "../assets/newtourneyround.mp3";
import wolgesSound from "../assets/wolges.wav";
import abortnudgeSound from "../assets/abortnudge.mp3";
import newpuzzleSound from "../assets/newpuzzle.mp3";
import puzzlecorrectSound from "../assets/puzzlecorrect-acoustic-fast.mp3";
import puzzlewrongSound from "../assets/puzzlewrong.mp3";

const soundToggleCache: { all: boolean | undefined } = { all: undefined };

const soundIsEnabled = (soundName: string) => {
  void soundName;
  let cachedToggle = soundToggleCache["all"];
  if (cachedToggle === undefined) {
    // localStorage is synchronous, try not to do it too often.
    // Not sure if this is helpful...
    cachedToggle = localStorage.getItem("enableSilentSite") !== "true";
    soundToggleCache["all"] = cachedToggle;
    setTimeout(() => {
      soundToggleCache["all"] = undefined;
    }, 1000);
  }
  return cachedToggle;
};

let audioCtx: AudioContext | null = null;
let unlocked = false;

const getAudioContext = (): AudioContext => {
  if (!audioCtx) {
    audioCtx = new AudioContext();
  }
  return audioCtx;
};

// iOS Safari requires a user gesture to unlock the AudioContext. We listen
// for gestures and resume the context. Once running, we mark it as unlocked
// but keep the listeners — iOS can re-suspend the context when the tab is
// backgrounded or the device is locked, so we need to be able to resume again.
const onUserGesture = () => {
  const ctx = audioCtx;
  if (!ctx) return;
  if (ctx.state === "running") {
    unlocked = true;
    return;
  }
  ctx.resume().then(() => {
    unlocked = true;
  });
};

window.addEventListener("click", onUserGesture, true);
window.addEventListener("touchend", onUserGesture, true);
window.addEventListener("keydown", onUserGesture, true);

// Eagerly resume when returning from background.
document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible") {
    onUserGesture();
  }
});

class Booper {
  private buffer: AudioBuffer | null = null;

  constructor(
    readonly soundName: string,
    src: string,
  ) {
    this.load(src);
  }

  private async load(src: string) {
    try {
      const response = await fetch(src);
      const arrayBuffer = await response.arrayBuffer();
      this.buffer = await getAudioContext().decodeAudioData(arrayBuffer);
    } catch (e) {
      console.warn(`cannot load ${this.soundName}:`, e);
    }
  }

  play() {
    if (!soundIsEnabled(this.soundName)) return;
    if (!this.buffer) return;
    const ctx = getAudioContext();
    // If the context is suspended, try to resume it. On iOS after the initial
    // unlock, programmatic resume() works even outside user gestures.
    if (ctx.state !== "running") {
      ctx.resume();
    }
    const source = ctx.createBufferSource();
    source.buffer = this.buffer;
    source.connect(ctx.destination);
    source.start();
  }
}

const playableSounds: { [key: string]: Booper } = {};

// Only load sounds if this is not an embed page. This is a bit of a hack.

if (!window.location.pathname.startsWith("/embed/")) {
  const booperArray = [
    new Booper("makeMoveSound", makemoveSound),
    new Booper("oppMoveSound", oppmoveSound),
    new Booper("matchReqSound", matchreqSound),
    new Booper("startgameSound", startgameSound),
    new Booper("endgameSound", endgameSound),
    new Booper("woofSound", woofSound),
    new Booper("meowSound", meowSound),
    new Booper("receiveMsgSound", receivechatSound),
    new Booper("startTourneyRoundSound", newtourneyroundSound),
    new Booper("wolgesSound", wolgesSound),
    new Booper("abortnudgeSound", abortnudgeSound),
    new Booper("puzzleStartSound", newpuzzleSound),
    new Booper("puzzleCorrectSound", puzzlecorrectSound),
    new Booper("puzzleWrongSound", puzzlewrongSound),
  ];

  for (const booper of booperArray) {
    playableSounds[booper.soundName] = booper;
  }
}

const playSound = (soundName: string) => {
  const booper = playableSounds[soundName];
  if (!booper) {
    throw new TypeError(`unsupported sound: ${soundName}`);
  }
  booper.play();
};

export const BoopSounds = {
  playSound,
  soundIsEnabled,
};
