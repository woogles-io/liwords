import React from 'react';

import {
  CrosswordGameTileValues,
  runeToValues,
} from '../constants/tile_values';
import Tile from './tile';
import { EphemeralTile } from './tile_placement';

type Props = {
  gridDim: number;
  tilesLayout: Array<string>;
  lastPlayedLetters: Record<string, boolean>;
  boardSquareDim: number;
  rowLabelWidth: number;
  colLabelHeight: number;
  scaleTiles: boolean;
  tentativeTiles: Set<EphemeralTile>;
};

const Tiles = (props: Props) => {
  const tiles = [];
  if (!props.tilesLayout || props.tilesLayout.length === 0) {
    return null;
  }

  for (let y = 0; y < props.gridDim; y += 1) {
    for (let x = 0; x < props.gridDim; x += 1) {
      const rune = props.tilesLayout[y][x];
      if (rune !== ' ') {
        const lastPlayed = props.lastPlayedLetters[`R${y}C${x}`] === true;
        tiles.push(
          <Tile
            rune={rune}
            value={runeToValues(rune, CrosswordGameTileValues)}
            width={props.boardSquareDim}
            height={props.boardSquareDim}
            x={x * props.boardSquareDim + props.rowLabelWidth}
            y={y * props.boardSquareDim + props.colLabelHeight}
            lastPlayed={lastPlayed}
            key={`tile_${x}_${y}`}
            scale={props.scaleTiles}
          />
        );
      }
    }
  }

  // The "tentative tiles" should be displayed slightly differently.
  props.tentativeTiles.forEach((t) => {
    tiles.push(
      <Tile
        rune={t.letter}
        value={runeToValues(t.letter, CrosswordGameTileValues)}
        width={props.boardSquareDim}
        height={props.boardSquareDim}
        x={t.col * props.boardSquareDim + props.rowLabelWidth}
        y={t.row * props.boardSquareDim + props.colLabelHeight}
        lastPlayed={false}
        key={`tile_${t.col}_${t.row}`}
        scale={false}
        tentative={true}
      />
    );
  });

  return <>{tiles}</>;
};

export default Tiles;
