import { Unrace } from '../utils/unrace';

class Booper {
  private audio: HTMLAudioElement;
  private unrace = new Unrace();
  private times = 0;
  private unlocked = false;

  constructor(readonly soundName: string, src: string) {
    this.audio = new Audio(src);
    // On iOS, if not yet loaded, audio.play() will silently play a short
    // silent sound instead, and fire ended event on that.
    // Try to load first, so hopefully it's already loaded when needed.
    this.audio.load();
    this.audio.addEventListener('ended', () => {
      if (this.times > 0) this.unlock();
    });
  }

  callPlay = async () => {
    const isPlaying = this.times > 0;
    try {
      // Use .muted, because iOS does not support .volume.
      this.audio.muted = !isPlaying;
      // On some browsers (desktop included), audio may start midway.
      this.audio.currentTime = isPlaying ? 0 : this.audio.duration;
      await this.audio.play();
      this.unlocked = true;
      if (isPlaying) --this.times;
    } catch (e) {
      console.warn(
        `cannot ${isPlaying ? 'play' : 'initialize'} ${this.soundName}:`,
        e
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

for (const booper of [
  new Booper('makeMoveSound', require('../assets/makemove.mp3')),
  new Booper('oppMoveSound', require('../assets/oppmove.mp3')),
  new Booper('matchReqSound', require('../assets/matchreq.mp3')),
  new Booper('startgameSound', require('../assets/startgame.mp3')),
  new Booper('endgameSound', require('../assets/endgame.mp3')),
  new Booper('woofSound', require('../assets/woof.mp3')),
]) {
  playableSounds[booper.soundName] = booper;
}

const unlockSounds = () => {
  // Browser settings may disallow autoplay until user interacts with document.
  // Use that chance to play() all known sounds muted.
  (async () => {
    if (
      (
        await Promise.all(
          Object.values(playableSounds).map((booper) => booper.unlock())
        )
      ).every((x) => x)
    ) {
      window.removeEventListener('click', unlockSounds, true);
      window.removeEventListener('keydown', unlockSounds, true);
    }
  })();
};

window.addEventListener('click', unlockSounds, true);
window.addEventListener('keydown', unlockSounds, true);

const playSound = (soundName: string) => {
  const booper = playableSounds[soundName];
  if (!booper) {
    throw new TypeError(`unsupported sound: ${soundName}`);
  }
  booper.play();
};

export const BoopSounds = {
  playSound,
};
