import { Unrace } from "../utils/unrace";
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

class Booper {
  private audio: HTMLAudioElement;

  private unrace = new Unrace();

  private times = 0;

  private unlocked = false;

  constructor(
    readonly soundName: string,
    src: string,
  ) {
    this.audio = new Audio(src);
    // On iOS, if not yet loaded, audio.play() will silently play a short
    // silent sound instead, and fire ended event on that.
    // Try to load first, so hopefully it's already loaded when needed.
    this.audio.load();
    this.audio.addEventListener("ended", () => {
      if (this.times > 0) this.unlock();
    });
  }

  callPlay = async () => {
    const isPlaying = this.times > 0;
    try {
      // Use .muted, because iOS does not support .volume.
      const thisAudioMuted = !(isPlaying && soundIsEnabled(this.soundName));
      // Always unlock.
      if (!thisAudioMuted || !this.unlocked) {
        this.audio.muted = thisAudioMuted;
        // On some browsers (desktop included), audio may start midway.
        this.audio.currentTime = isPlaying ? 0 : this.audio.duration;
        await this.audio.play();
        this.unlocked = true;
      }
      if (isPlaying) --this.times;
    } catch (e) {
      console.warn(
        `cannot ${isPlaying ? "play" : "initialize"} ${this.soundName}:`,
        e,
      );
    }
  };

  unlock = async () => {
    await this.unrace.run(this.callPlay);
    return this.unlocked;
  };

  play = async () => {
    ++this.times;
    return await this.unlock();
  };
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

const unlockSounds = () => {
  // Browser settings may disallow autoplay until user interacts with document.
  // Use that chance to play() all known sounds muted.
  (async () => {
    if (
      (
        await Promise.all(
          Object.values(playableSounds).map((booper) => booper.unlock()),
        )
      ).every((x) => x)
    ) {
      window.removeEventListener("click", unlockSounds, true);
      window.removeEventListener("keydown", unlockSounds, true);
    }
  })();
};

window.addEventListener("click", unlockSounds, true);
window.addEventListener("keydown", unlockSounds, true);

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
