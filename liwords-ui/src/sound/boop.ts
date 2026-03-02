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
    const Ctor = window.AudioContext || window.webkitAudioContext;
    audioCtx = new Ctor();
  }
  return audioCtx;
};

// iOS Safari requires both resuming the AudioContext AND playing a buffer
// through it during a user gesture to fully unlock audio. Calling resume()
// alone is not sufficient — this is the same technique Howler.js and Tone.js
// use internally. We keep the listeners active because iOS can re-suspend the
// context when the tab is backgrounded or the device is locked.
const onUserGesture = () => {
  // Use getAudioContext() so the context is created during the gesture if it
  // doesn't exist yet. Creating during a gesture is ideal on iOS.
  const ctx = getAudioContext();
  if (ctx.state === "running") {
    unlocked = true;
    return;
  }
  // Play a short silent buffer through the context. On iOS Safari this is
  // required in addition to resume() to fully unlock audio output.
  try {
    const silentBuffer = ctx.createBuffer(1, 1, ctx.sampleRate || 44100);
    const source = ctx.createBufferSource();
    source.buffer = silentBuffer;
    source.connect(ctx.destination);
    source.start();
  } catch (_) {
    // Ignore — the resume() below is still valuable on its own.
  }
  ctx.resume().then(() => {
    unlocked = true;
  });
};

window.addEventListener("click", onUserGesture, true);
window.addEventListener("touchstart", onUserGesture, true);
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
    if (ctx.state === "running") {
      this.playBuffer(ctx);
    } else {
      // On iOS, playing into a suspended context silently drops the sound.
      // Wait for resume() to finish before scheduling the buffer.
      ctx.resume().then(() => this.playBuffer(ctx));
    }
  }

  private playBuffer(ctx: AudioContext) {
    if (!this.buffer) return;
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
