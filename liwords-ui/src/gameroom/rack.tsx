import React from 'react';

import {
  CrosswordGameTileValues,
  runeToValues,
} from '../constants/tile_values';
import Tile from './tile';

// const TileSpacing = 6;

type Props = {
  letters: string;
  grabbable: boolean;
  onTileClick?: (idx: number) => void;
  swapRackTiles: (indexA: number | undefined, indexB: number | undefined) => void;
};

export const Rack = React.memo((props: Props) => {
  const renderTiles = () => {
    const tiles = [];
    if (!props.letters || props.letters.length === 0) {
      return null;
    }

    for (let n = 0; n < props.letters.length; n += 1) {
      const rune = props.letters[n];
      tiles.push(
        <Tile
          rune={rune}
          value={runeToValues(rune, CrosswordGameTileValues)}
          lastPlayed={false}
          key={`tile_${n}`}
          scale={false}
          grabbable={props.grabbable}
          rackIndex={n}
          swapRackTiles={props.swapRackTiles}
          onClick={() => {
            if (props.onTileClick) {
              props.onTileClick(n);
            }
          }}
        />
      );
    }
    return <>{tiles}</>;
  }

  return <div className="rack">{renderTiles()}</div>;
});

export default Rack;
