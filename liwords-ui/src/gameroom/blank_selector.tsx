import React from 'react';
import { CrosswordGameTileValues } from '../constants/tile_values';
import Tile from './tile';

type Props = {
  handleSelection: (rune: string) => void;
};

export const BlankSelector = (props: Props) => {
  return (
    <div className="blank-selector">
      {Object.keys(CrosswordGameTileValues).map((rune) => (
        <Tile
          lastPlayed={false}
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
