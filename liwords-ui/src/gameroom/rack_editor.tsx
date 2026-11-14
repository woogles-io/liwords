import { InputRef } from "rc-input";
import React, { useCallback, useEffect, useRef, useState } from "react";
import {
  EmptyBoardSpaceMachineLetter,
  EmptyRackSpaceMachineLetter,
  isDesignatedBlankMachineLetter,
  MachineLetter,
  MachineWord,
} from "../utils/cwgame/common";
import {
  Alphabet,
  machineWordToRunes,
  runesToMachineWord,
} from "../constants/alphabets";
import { Button, Input } from "antd";
import Tile from "./tile";

type Props = {
  rackCallback: (rack: MachineWord) => void;
  cancelCallback: () => void;
  currentRack: MachineWord;
  alphabet: Alphabet;
  // the tiles' color, as the rack's
  tileColorId?: number;
  // the board's tiles, so the grid can say how many of each are left
  boardLetters?: Array<MachineLetter>;
};

const MaxRackLength = 7;

export const RackEditor = (props: Props) => {
  const calculateRackStr = useCallback(
    (rack: MachineWord) => machineWordToRunes(rack, props.alphabet, false),
    [props.alphabet],
  );

  // Enter sends this as it is, so a prefilled rack is capped like typed
  // input; a rack with tiles out on the board has empty places, not tiles.
  const [currentRack, setCurrentRack] = useState(() =>
    props.currentRack
      .filter((ml) => ml !== EmptyRackSpaceMachineLetter)
      .slice(0, MaxRackLength),
  );
  const [curRackStr, setCurRackStr] = useState(() =>
    calculateRackStr(
      props.currentRack
        .filter((ml) => ml !== EmptyRackSpaceMachineLetter)
        .slice(0, MaxRackLength),
    ),
  );

  const inputRef = useRef<InputRef>(null);
  // how many of each tile are left for the rack: the game's, less those on
  // the board (a blank played counts as a blank) and in the rack being set
  const left = props.alphabet.letters.map((l) => l.count);
  for (const ml of props.boardLetters ?? []) {
    if (ml === EmptyBoardSpaceMachineLetter) continue;
    left[isDesignatedBlankMachineLetter(ml) ? 0 : ml]--;
  }
  for (const ml of currentRack) left[ml]--;
  // a tile clicked joins the rack, as if typed, while it has room
  const addTile = (letter: MachineLetter) => {
    if (currentRack.length < MaxRackLength) {
      const rack = [...currentRack, letter];
      setCurrentRack(rack);
      setCurRackStr(calculateRackStr(rack));
    }
  };
  // the rack as it stands, a tile of it taken out by a click, or all of it
  const setRack = (rack: MachineWord) => {
    setCurrentRack(rack);
    setCurRackStr(calculateRackStr(rack));
  };
  // (a tile tapped leaves the focus where it is: moving it to the box would
  // raise a phone's on-screen keyboard at every tap)
  const handleKeyDown = (evt: React.KeyboardEvent) => {
    // a key an input method is still making a letter with is that letter's
    if (composing.current || evt.nativeEvent.isComposing) return;
    if (evt.key === "Enter") {
      props.rackCallback(currentRack);
    }
  };
  useEffect(() => {
    inputRef.current?.focus({
      cursor: "all",
    });
  }, [inputRef]);

  // whether a dead key or an input method is making a letter in the box
  const composing = useRef(false);
  const handleRackEditChange = (evt: React.ChangeEvent<HTMLInputElement>) => {
    if (composing.current) {
      // a dead key's accent, or an input method's, before its letter: left
      // as typed until the letter is made, which takes the accent then
      setCurRackStr(evt.target.value);
      return;
    }
    readRack(evt.target.value);
  };

  const readRack = (raw: string) => {
    // strip out any spaces, fix max length, etc.
    let out = raw;
    out = out.toLocaleUpperCase();

    // letters, interpunct, blank (? or .), and [] for multi-char Spanish tiles.
    const onlyValidTileCharacters = out.match(/[\p{Letter}·\?\.\[\]]/gu);
    // Convert . to ? for blanks
    const normalized =
      onlyValidTileCharacters?.map((c) => (c === "." ? "?" : c)).join("") ?? "";
    try {
      let curRack = runesToMachineWord(normalized, props.alphabet);
      if (curRack.length > MaxRackLength) {
        curRack = curRack.slice(0, MaxRackLength);
      }
      setCurrentRack(curRack);
      setCurRackStr(machineWordToRunes(curRack, props.alphabet, false));
    } catch {
      // Do nothing for now. Maybe the user is not done typing their multi-char tile.
      // Just echo back what they are typing (with . converted to ?).
      setCurRackStr(normalized);
    }
  };

  return (
    // The dialog is drawn in the board's tree, so its keys would reach the
    // board's own handler: they stop here. Esc anywhere in it cancels.
    <div
      className="rack-editor"
      onKeyDown={(evt) => {
        evt.stopPropagation();
        if (evt.key === "Escape" && !evt.nativeEvent.isComposing) {
          props.cancelCallback();
        }
      }}
    >
      <Input
        ref={inputRef}
        // short enough for the box at any width: the tiles below spell out
        // the rest
        placeholder={
          "Rack, ? for a blank" +
          (props.alphabet.name === "spanish" ? ", [CH] for CH" : "")
        }
        className="rack"
        value={curRackStr}
        onChange={handleRackEditChange}
        onCompositionStart={() => {
          composing.current = true;
        }}
        onCompositionEnd={(evt) => {
          composing.current = false;
          readRack(evt.currentTarget.value);
        }}
        onKeyDown={handleKeyDown}
      />
      {/* the rack being set, a tile taken out by a click on it */}
      <div className="rack-editor-rack">
        {currentRack.map((letter, idx) => (
          <Tile
            lastPlayed={false}
            playerOfTile={props.tileColorId ?? 0}
            alphabet={props.alphabet}
            letter={letter}
            value={props.alphabet.letters[letter]?.score ?? 0}
            grabbable={false}
            key={`set_${idx}`}
            onClick={() => setRack(currentRack.filter((_, at) => at !== idx))}
          />
        ))}
      </div>
      {/* every tile, the blank first, for a rack clicked together: a letter
          the keyboard cannot type is a click away */}
      <div className="rack-editor-tiles">
        {props.alphabet.letters.map((letter, idx) => (
          <div
            className={`rack-editor-tile${left[idx] > 0 ? "" : " rack-editor-none"}`}
            key={`rack_${letter.rune}`}
          >
            <Tile
              lastPlayed={false}
              playerOfTile={props.tileColorId ?? 0}
              alphabet={props.alphabet}
              letter={idx}
              value={letter.score}
              grabbable={false}
              onClick={left[idx] > 0 ? () => addTile(idx) : undefined}
            />
            {/* how many are left, as the unseen tiles count them */}
            <span className="rack-editor-left">{Math.max(0, left[idx])}</span>
          </div>
        ))}
      </div>
      {/* what Enter and Esc do, for a pointer */}
      <div className="rack-editor-buttons">
        <Button onClick={() => setRack([])} disabled={!currentRack.length}>
          Clear
        </Button>
        <Button onClick={props.cancelCallback}>Cancel</Button>
        <Button type="primary" onClick={() => props.rackCallback(currentRack)}>
          Use this rack
        </Button>
      </div>
    </div>
  );
};
