import { InputRef } from "rc-input";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { MachineWord } from "../utils/cwgame/common";
import {
  Alphabet,
  machineWordToRunes,
  runesToMachineWord,
} from "../constants/alphabets";
import { Input } from "antd";

type Props = {
  rackCallback: (rack: MachineWord) => void;
  cancelCallback: () => void;
  currentRack: MachineWord;
  alphabet: Alphabet;
};

const MaxRackLength = 7;

export const RackEditor = (props: Props) => {
  const calculateRackStr = useCallback(
    (rack: MachineWord) => machineWordToRunes(rack, props.alphabet, false),
    [props.alphabet],
  );

  const [currentRack, setCurrentRack] = useState(props.currentRack);
  const [curRackStr, setCurRackStr] = useState(
    calculateRackStr(props.currentRack),
  );

  const inputRef = useRef<InputRef>(null);
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

    // letters, interpunct, blank, and [] for multi-char Spanish tiles.
    const onlyValidTileCharacters = out.match(/[\p{Letter}·\?\[\]]/gu);
    try {
      let curRack = runesToMachineWord(
        onlyValidTileCharacters?.join("") ?? "",
        props.alphabet,
      );
      if (curRack.length > MaxRackLength) {
        curRack = curRack.slice(0, MaxRackLength);
      }
      setCurrentRack(curRack);
      setCurRackStr(machineWordToRunes(curRack, props.alphabet, false));
    } catch {
      // Do nothing for now. Maybe the user is not done typing their multi-char tile.
      // Just echo back what they are typing.
      setCurRackStr(onlyValidTileCharacters?.join("") ?? "");
    }
  };

  return (
    <>
      <Input
        ref={inputRef}
        placeholder={
          "Enter rack. Use ? for blank." +
          (props.alphabet.name === "spanish" ? " Use [CH] for digraph CH." : "")
        }
        className="rack"
        value={curRackStr}
        onChange={handleRackEditChange}
        onKeyDown={handleKeyDown}
      />
    </>
  );
};
