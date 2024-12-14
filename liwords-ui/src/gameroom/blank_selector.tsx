import React from "react";
import Tile from "./tile";
import { Alphabet } from "../constants/alphabets";
import { Blank, MachineLetter } from "../utils/cwgame/common";

type Props = {
  tileColorId: number;
  handleSelection: (letter: MachineLetter) => void;
  alphabet: Alphabet;
};

export const BlankSelector = (props: Props) => {
  return (
    <div className="blank-selector">
      {props.alphabet.letters
        .filter((l) => l.rune !== Blank)
        .map((letter, idx) => (
          <Tile
            lastPlayed={false}
            playerOfTile={props.tileColorId}
            alphabet={props.alphabet}
            letter={idx + 1} // assumes blank was filtered out and was zero
            value={0}
            grabbable={false}
            key={`blank_${letter.rune}`}
            onClick={() => {
              props.handleSelection(idx + 1);
            }}
          />
        ))}
    </div>
  );
};
