import { InputRef } from "rc-input";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { MachineLetter, MachineWord } from "../utils/cwgame/common";
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
};

const MaxRackLength = 7;

export const RackEditor = (props: Props) => {
  const calculateRackStr = useCallback(
    (rack: MachineWord) => machineWordToRunes(rack, props.alphabet, false),
    [props.alphabet],
  );

  // Enter sends this as it is, so a prefilled rack is capped like typed input.
  const [currentRack, setCurrentRack] = useState(() =>
    props.currentRack.slice(0, MaxRackLength),
  );
  const [curRackStr, setCurRackStr] = useState(() =>
    calculateRackStr(props.currentRack.slice(0, MaxRackLength)),
  );

  const inputRef = useRef<InputRef>(null);
  // a tile clicked joins the rack, as if typed, while it has room
  const addTile = (letter: MachineLetter) => {
    if (currentRack.length < MaxRackLength) {
      const rack = [...currentRack, letter];
      setCurrentRack(rack);
      setCurRackStr(calculateRackStr(rack));
    }
    inputRef.current?.focus();
  };
  // the rack as it stands, a tile of it taken out by a click, or all of it
  const setRack = (rack: MachineWord) => {
    setCurrentRack(rack);
    setCurRackStr(calculateRackStr(rack));
    inputRef.current?.focus();
  };
  const handleKeyDown = (evt: React.KeyboardEvent) => {
    if (evt.key === "Enter") {
      props.rackCallback(currentRack);
    } else if (evt.key === "Escape") {
      props.cancelCallback();
    }
  };
  useEffect(() => {
    inputRef.current?.focus({
      cursor: "all",
    });
  }, [inputRef]);

  const handleRackEditChange = (evt: React.ChangeEvent<HTMLInputElement>) => {
    // strip out any spaces, fix max length, etc.
    const raw = evt.target.value;
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
    <React.Fragment>
      <Input
        ref={inputRef}
        placeholder={
          "Enter rack. Use ? or . for blank." +
          (props.alphabet.name === "spanish" ? " Use [CH] for digraph CH." : "")
        }
        className="rack"
        value={curRackStr}
        onChange={handleRackEditChange}
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
          <Tile
            lastPlayed={false}
            playerOfTile={props.tileColorId ?? 0}
            alphabet={props.alphabet}
            letter={idx}
            value={letter.score}
            grabbable={false}
            key={`rack_${letter.rune}`}
            onClick={() => addTile(idx)}
          />
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
    </React.Fragment>
  );
};
