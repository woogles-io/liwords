import React from 'react';
import Tile from './tile';
import { Alphabet } from '../constants/alphabets';
import { Blank } from '../utils/cwgame/common';

type Props = {
  tileColorId: number;
  handleSelection: (rune: string) => void;
  alphabet: Alphabet;
};

export const BlankSelector = (props: Props) => {
  return (
    <div className="blank-selector">
      {Object.keys(props.alphabet.letterMap)
        .filter((l) => l !== Blank)
        .map((rune) => (
          <Tile
            lastPlayed={false}
            playerOfTile={props.tileColorId}
            rune={rune}
            value={0}
            grabbable={false}
            key={`blank_${rune}`}
            onClick={() => {
              props.handleSelection(rune);
            }}
          />
        ))}
    </div>
  );
};
