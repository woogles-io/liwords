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
  selected?: Set<number>;
  swapRackTiles: (
    indexA: number | undefined,
    indexB: number | undefined
  ) => void;
  returnToRack?: (
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
};

export const Rack = React.memo((props: Props) => {
  const handleDropOver = (e: any) => {
    e.preventDefault();
    e.stopPropagation();
  };
  const handleDrop = (e: any, index: number) => {
    if (props.swapRackTiles && e.dataTransfer.getData('rackIndex')) {
      props.swapRackTiles(
        index,
        parseInt(e.dataTransfer.getData('rackIndex'), 10)
      );
    } else {
      if (props.returnToRack && e.dataTransfer.getData('tileIndex')) {
        props.returnToRack(
          index,
          parseInt(e.dataTransfer.getData('tileIndex'), 10)
        );
      }
    }
  };
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
          selected={props.selected && props.selected.has(n)}
          grabbable={props.grabbable}
          rackIndex={n}
          returnToRack={props.returnToRack}
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
  };

  return (
    <div className="rack">
      <div
        className="empty-rack droppable"
        onDragOver={handleDropOver}
        onDrop={(e) => {
          handleDrop(e, 0);
        }}
      />
      {renderTiles()}
      <div
        className="empty-rack droppable"
        onDragOver={handleDropOver}
        onDrop={(e) => {
          handleDrop(e, props.letters.length);
        }}
      />
    </div>
  );
});

export default Rack;
