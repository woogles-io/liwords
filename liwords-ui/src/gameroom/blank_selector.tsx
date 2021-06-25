import React from 'react';
import Tile from './tile';
import { useExaminableGameContextStoreContext } from '../store/store';
import { Alphabet } from '../constants/alphabets';

type Props = {
  handleSelection: (rune: string) => void;
  alphabet: Alphabet;
};

export const BlankSelector = (props: Props) => {
  const {
    gameContext: examinableGameContext,
  } = useExaminableGameContextStoreContext();

  return (
    <div className="blank-selector">
      {Object.keys(props.alphabet.letterMap).map((rune) => (
        <Tile
          lastPlayed={false}
          playerOfTile={examinableGameContext.onturn}
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
