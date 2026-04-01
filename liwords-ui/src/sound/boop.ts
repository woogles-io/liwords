import makemoveSound from "../assets/makemove.mp3";
import oppmoveSound from "../assets/oppmove.mp3";
import matchreqSound from "../assets/matchreq.mp3";
import startgameSound from "../assets/startgame.mp3";
import endgameSound from "../assets/endgame.mp3";
import woofSound from "../assets/eagle-screech.mp3";
import meowSound from "../assets/meow.mp3";
import receivechatSound from "../assets/receivechat.mp3";
import newtourneyroundSound from "../assets/newtourneyround.mp3";
import wolgesSound from "../assets/wolges.wav";
import abortnudgeSound from "../assets/abortnudge.mp3";
import newpuzzleSound from "../assets/newpuzzle.mp3";
import puzzlecorrectSound from "../assets/puzzlecorrect-acoustic-fast.mp3";
import puzzlewrongSound from "../assets/puzzlewrong.mp3";

const soundToggleCache: { all: boolean | undefined } = { all: undefined };

// Invalidate cache when another tab changes the setting.
window.addEventListener("storage", (e) => {
  if (e.key === "enableSilentSite") {
    soundToggleCache["all"] = undefined;
  }
});

const soundIsEnabled = (soundName: string) => {
  void soundName;
  let cachedToggle = soundToggleCache["all"];
  if (cachedToggle === undefined) {
    // localStorage is synchronous, try not to do it too often.
    cachedToggle = localStorage.getItem("enableSilentSite") !== "true";
    soundToggleCache["all"] = cachedToggle;
    setTimeout(() => {
      soundToggleCache["all"] = undefined;
    }, 1000);
  }
  return cachedToggle;
};

class Booper {
  private audio: HTMLAudioElement;
  private unlocked = false;

  constructor(
    readonly soundName: string,
    src: string,
  ) {
    this.audio = new Audio(src);
    // Preload so the buffer is ready before it's needed.
    this.audio.load();
  }

  // Must be called from within a user gesture event handler.
  // Plays the element muted so iOS will allow programmatic plays later.
  async unlock(): Promise<boolean> {
    if (this.unlocked) return true;
    try {
      this.audio.muted = true;
      this.audio.currentTime = 0;
      await this.audio.play();
      this.audio.pause();
      this.audio.muted = false;
      this.unlocked = true;
    } catch {
      // Audio not yet loaded or blocked — will retry on the next gesture.
    }
    return this.unlocked;
  }

  play() {
    if (!soundIsEnabled(this.soundName)) return;
    if (!this.unlocked) return;
    this.audio.currentTime = 0;
    this.audio.play().catch((e) => {
      console.warn(`cannot play ${this.soundName}:`, e);
    });
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

// On the first user gesture, pre-unlock all audio elements so they can be
// played later from non-gesture contexts (e.g. WebSocket events).
//
// We guard with soundIsEnabled so that when sounds are turned off we never
// call audio.play() — even muted — which would activate the iOS audio session
// and interrupt background music.
const unlockAll = () => {
  if (!soundIsEnabled("")) return;
  Promise.all(Object.values(playableSounds).map((b) => b.unlock())).then(
    (results) => {
      if (results.every(Boolean)) {
        // All elements unlocked; no need to keep trying on every gesture.
        window.removeEventListener("click", unlockAll, true);
        window.removeEventListener("touchend", unlockAll, true);
        window.removeEventListener("keydown", unlockAll, true);
      }
    },
  );
};

window.addEventListener("click", unlockAll, true);
window.addEventListener("touchend", unlockAll, true);
window.addEventListener("keydown", unlockAll, true);

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
