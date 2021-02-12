import React from 'react';
import { CrosswordGameTileValues } from '../constants/tile_values';
import Tile from './tile';
import { useExaminableGameContextStoreContext } from '../store/store';

type Props = {
  handleSelection: (rune: string) => void;
};

export const BlankSelector = (props: Props) => {
  const {
    gameContext: examinableGameContext,
  } = useExaminableGameContextStoreContext();

  return (
    <div className="blank-selector">
      {Object.keys(CrosswordGameTileValues).map((rune) => (
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
