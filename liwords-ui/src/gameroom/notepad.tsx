import React, {
  useRef,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { Button, Card } from "antd";
import { DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import {
  useGameContextStoreContext,
  useTentativeTileContext,
} from "../store/store";
import { sortTiles } from "../store/constants";
import {
  contiguousTilesFromTileSet,
  simpletile,
} from "../utils/cwgame/scoring";
import { Direction, isMobile } from "../utils/cwgame/common";
import { BoopSounds } from "../sound/boop";
import {
  machineLetterToRune,
  machineWordToRunes,
} from "../constants/alphabets";

type NotepadProps = {
  style?: React.CSSProperties;
  includeCard?: boolean;
};

const humanReadablePosition = (
  direction: Direction,
  firstLetter: simpletile,
): string => {
  const readableCol = String.fromCodePoint(firstLetter.col + 65);
  const readableRow = (firstLetter.row + 1).toString();
  if (direction === Direction.Horizontal) {
    return readableRow + readableCol;
  }
  return readableCol + readableRow;
};

const NotepadContext = React.createContext({
  curNotepad: "",
  setCurNotepad: ((a: string) => {}) as React.Dispatch<
    React.SetStateAction<string>
  >,
  feRackInfo: true,
});

export const NotepadContextProvider = ({
  children,
  feRackInfo,
  gameID,
}: {
  children: React.ReactNode;
  feRackInfo: boolean;
  gameID?: string;
}) => {
  const [curNotepad, setCurNotepad] = useState("");

  // Load saved notepad from localStorage when gameID changes
  useEffect(() => {
    if (!gameID) return;

    const savedNotes = localStorage.getItem(`notepad_${gameID}`);
    if (savedNotes) {
      setCurNotepad(savedNotes);
    } else {
      setCurNotepad("");
    }
  }, [gameID]);

  // Auto-save notepad to localStorage when content changes (debounced)
  useEffect(() => {
    if (!gameID) return;

    const timer = setTimeout(() => {
      if (curNotepad) {
        localStorage.setItem(`notepad_${gameID}`, curNotepad);
      } else {
        // Remove from localStorage if notepad is cleared
        localStorage.removeItem(`notepad_${gameID}`);
      }
    }, 500); // 500ms debounce

    return () => clearTimeout(timer);
  }, [curNotepad, gameID]);

  const contextValue = useMemo(
    () => ({ curNotepad, setCurNotepad, feRackInfo }),
    [curNotepad, setCurNotepad, feRackInfo],
  );

  return <NotepadContext.Provider value={contextValue} children={children} />;
};
const defaultFunction = () => {};

export const Notepad = React.memo((props: NotepadProps) => {
  const notepadEl = useRef<HTMLTextAreaElement>(null);
  const { curNotepad, setCurNotepad, feRackInfo } = useContext(NotepadContext);
  const { displayedRack, placedTiles, placedTilesTempScore } =
    useTentativeTileContext();
  const { gameContext } = useGameContextStoreContext();
  const board = gameContext.board;
  const addPlay = useCallback(() => {
    const contiguousTiles = contiguousTilesFromTileSet(placedTiles, board);
    let play = "";
    let position = "";
    const leave = machineWordToRunes(
      sortTiles(displayedRack, gameContext.alphabet),
      gameContext.alphabet,
    );
    if (contiguousTiles?.length === 2) {
      position = humanReadablePosition(
        contiguousTiles[1],
        contiguousTiles[0][0],
      );
      let inParen = false;
      for (const tile of contiguousTiles[0]) {
        if (!tile.fresh) {
          if (!inParen) {
            play += "(";
            inParen = true;
          }
        } else {
          if (inParen) {
            play += ")";
            inParen = false;
          }
        }
        play += machineLetterToRune(tile.letter, gameContext.alphabet);
      }
      if (inParen) play += ")";
    }
    setCurNotepad(
      (curNotepad) =>
        `${curNotepad ? curNotepad + "\n" : ""}${
          play
            ? `${position} ${play} ${placedTilesTempScore}${leave ? " " : ""}`
            : ""
        }${leave}`,
    );
    // Return focus to board on all but mobile so the key commands can be used immediately
    if (!isMobile()) {
      document.getElementById("board-container")?.focus();
    }
  }, [
    displayedRack,
    placedTiles,
    placedTilesTempScore,
    setCurNotepad,
    board,
    gameContext.alphabet,
  ]);
  const clearNotepad = useCallback(() => {
    setCurNotepad("");
  }, [setCurNotepad]);
  useEffect(() => {
    if (notepadEl.current && !(notepadEl.current === document.activeElement)) {
      notepadEl.current.scrollTop = notepadEl.current.scrollHeight || 0;
    }
  }, [curNotepad]);
  const handleNotepadChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      setCurNotepad(e.target.value);
    },
    [setCurNotepad],
  );
  const easterEggEnabled = useMemo(
    () => /catcam/i.test(curNotepad),
    [curNotepad],
  );
  const numWolgesWas = useRef(0);
  const numWolges = useMemo(
    () => curNotepad.match(/wolges/gi)?.length || 0,
    [curNotepad],
  );
  useEffect(() => {
    if (numWolges > numWolgesWas.current) {
      BoopSounds.playSound("wolgesSound");
    }
    numWolgesWas.current = numWolges;
  }, [numWolges]);
  const notepadIsNotEmpty = curNotepad.length > 0;
  const controls = useCallback(
    () => (
      <React.Fragment>
        {easterEggEnabled && <EasterEgg />}
        {notepadIsNotEmpty && (
          <Button
            shape="circle"
            icon={<DeleteOutlined />}
            type="primary"
            onClick={clearNotepad}
          />
        )}
        <Button
          shape="circle"
          icon={<PlusOutlined />}
          type="primary"
          onClick={feRackInfo ? addPlay : defaultFunction}
        />
      </React.Fragment>
    ),
    [easterEggEnabled, feRackInfo, notepadIsNotEmpty, clearNotepad, addPlay],
  );
  const notepadContainer = (
    <div className="notepad-container" style={props.style}>
      <textarea
        className="notepad"
        value={curNotepad}
        ref={notepadEl}
        spellCheck={false}
        style={props.style}
        onChange={handleNotepadChange}
      />
      <div className="controls">{controls()}</div>
    </div>
  );
  if (props.includeCard) {
    return (
      <Card title="Notepad" className="notepad-card " extra={controls()}>
        {notepadContainer}
      </Card>
    );
  }
  return notepadContainer;
});

// usage: type catcam on the notepad
const EasterEgg = () => {
  const ctx = useMemo(() => {
    const AudioContext =
      window.AudioContext ||
      (window as unknown as { webkitAudioContext: AudioContext })
        .webkitAudioContext;
    return new AudioContext();
  }, []);

  const muterRef = useRef(ctx.createGain());

  const stopIt = useCallback(() => {
    muterRef.current.gain.setValueAtTime(0, ctx.currentTime);
    muterRef.current = ctx.createGain();
    muterRef.current.gain.setValueAtTime(1, ctx.currentTime);
    muterRef.current.connect(ctx.destination);
  }, [ctx]);

  useEffect(() => {
    return () => {
      stopIt();
    };
  }, [stopIt]);

  const handler = useCallback(
    (song: string) => {
      stopIt();
      if (!BoopSounds.soundIsEnabled(`${song}Song`)) {
        return;
      }

      const chans = [0.125, 0.5, 0.0625].map((mul) => {
        const chan = ctx.createGain();
        chan.gain.setValueAtTime(mul, ctx.currentTime);
        chan.connect(muterRef.current);
        return chan;
      });

      // Hz, sec, sec
      const playFreq = (
        freq: number,
        timeOn: number,
        duration: number,
        type: OscillatorType,
        chan: number,
      ) => {
        const env = ctx.createGain();
        // TODO: find a believable ADSR envelope
        env.gain.setValueAtTime(0, timeOn);
        env.gain.linearRampToValueAtTime(1, timeOn + 0.125 * duration);
        env.gain.linearRampToValueAtTime(0.5, timeOn + 0.875 * duration);
        env.gain.linearRampToValueAtTime(0.25, timeOn + duration);
        env.connect(chans[chan]);
        const osc = ctx.createOscillator();
        osc.frequency.value = freq;
        osc.type = type;
        osc.connect(env);
        osc.start(timeOn);
        osc.stop(timeOn + duration);
        return osc;
      };

      let dur = 60 / 120;
      const t0 = ctx.currentTime;
      let t = 0;
      const playNote = (
        note: number,
        beats: number,
        type: OscillatorType,
        chan: number,
      ) => {
        if (note) {
          playFreq(
            440 * 2 ** ((note - 69) / 12),
            t0 + t * dur,
            beats * dur,
            type,
            chan,
          );
        }
        t += beats;
      };

      if (song === "catcam") {
        // https://www.youtube.com/watch?v=J7UwSVsiwzI
        dur = 60 / 128;
        // 69 is middle A and 60 is middle C
        const song0 = [[0, 4]]; // 4 beats
        const song1a = [
          [83, 3],
          [81, 0.5],
          [83, 1],
          [84, 1],
          [83, 1.5],
          [81, 0.5],
          [79, 2.5],
        ]; // 10 beats
        const song1 = [...song1a, [76, 5], [0, 1]]; // 16 beats
        const song2a = [...song1a, [76, 4], [0, 1]]; // 15 beats
        const song2b = [
          [79, 0.5],
          [76, 0.5],
        ]; // 1 beat
        const song3 = [
          [74, 2.5],
          [72, 0.5],
          [74, 0.5],
          [76, 1],
          [80, 1],
          [81, 1],
          [83, 1.5],
        ]; // 8 beats
        const song4 = [
          [74, 5.5],
          [76, 2.5 / 3],
          [74, 2.5 / 3],
          [69, 2.5 / 3],
        ]; // 8 beats
        const song5a = [[71, 4]]; // 4 beats
        const song5b = [
          [71, 1],
          [0, 3],
        ]; // 4 beats
        const song1_d = song1.map(([note, beats]) => [note - 12, beats]);
        const song2a_d = song2a.map(([note, beats]) => [note - 12, beats]);
        const fullSong = [
          ...song0, // 4 beats
          ...song0, // 4 beats
          ...song1_d, // 16 beats
          ...song2a_d, // 15 beats
          ...song2b, // 1 beat
          ...song3, // 8 beats
          ...song3, // 8 beats
          ...song4, // 8 beats
          ...song5a, // 4 beats
          ...song0, // 4 beats
          ...song1, // 16 beats
          ...song2a, // 15 beats
          ...song2b, // 1 beat
          ...song3, // 8 beats
          ...song3, // 8 beats
          ...song4, // 8 beats
          ...song5b, // 4 beats
        ];

        for (const [note, beats] of fullSong) {
          playNote(note, beats, "sawtooth", 0);
        }

        const accompaniment = [
          40, 40, 45, 45, 40, 40, 45, 45, 40, 40, 38, 40, 38, 40, 38, 38,
        ];

        t = 0;
        for (const note1 of [...accompaniment, ...accompaniment]) {
          const note2 = note1 + 12;
          for (let i = 0; i < 4; ++i) {
            for (const note of [note1, note2]) {
              playNote(note, 0.5, "triangle", 1);
            }
          }
        }
        playNote(40, 1, "triangle", 1);

        for (const t1 of [82, 98]) {
          t = t1;
          for (let i = 0; i < 6; ++i) {
            for (const note of [79, 78, 76, 74]) {
              playNote(note, 1 / 4, "square", 2);
            }
          }
        }
      }
    },
    [stopIt, ctx],
  );

  return (
    <React.Fragment>
      <Button
        shape="circle"
        icon={<React.Fragment>&#x1f63b;</React.Fragment>}
        type="primary"
        onClick={() => handler("catcam")}
      />
    </React.Fragment>
  );
};
