import { Unrace } from '../utils/unrace';

class Booper {
  private audio: HTMLAudioElement;
  private unrace = new Unrace();
  private times = 0;
  private unlocked = false;

  constructor(readonly soundName: string, src: string, private volume: number) {
    this.audio = new Audio(src);
    this.callPlay = this.callPlay.bind(this);
    this.audio.addEventListener('ended', () => {
      if (this.times > 0) this.unlock();
    });
  }

  private async callPlay() {
    const isPlaying = this.times > 0;
    try {
      this.audio.volume = isPlaying ? this.volume : 0;
      await this.audio.play();
      this.unlocked = true;
      if (isPlaying) --this.times;
    } catch (e) {
      console.warn(
        `cannot ${isPlaying ? 'play' : 'initialize'} ${this.soundName}:`,
        e
      );
    }
  }

  async unlock() {
    await this.unrace.run(this.callPlay);
    return this.unlocked;
  }

  async play() {
    ++this.times;
    return await this.unlock();
  }
}

const playableSounds: { [key: string]: Booper } = {};

for (const booper of [
  new Booper('makeMoveSound', require('../assets/makemove.mp3'), 1),
  new Booper('oppMoveSound', require('../assets/oppmove.mp3'), 1),
  new Booper('matchReqSound', require('../assets/matchreq.mp3'), 1),
  new Booper('startgameSound', require('../assets/startgame.mp3'), 1),
  new Booper('endgameSound', require('../assets/endgame.mp3'), 1),
  new Booper('woofSound', require('../assets/woof.wav'), 0.25),
]) {
  playableSounds[booper.soundName] = booper;
}

const unlockSounds = () => {
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
