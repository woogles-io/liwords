import React from 'react';

import {
  CrosswordGameTileValues,
  runeToValues,
} from '../constants/tile_values';
import Tile from './tile';
import { EphemeralTile } from '../utils/cwgame/common';
import TentativeScore from './tentative_score';

type Props = {
  gridDim: number;
  tilesLayout: string;
  lastPlayedLetters: { [tile: string]: boolean };
  boardSquareDim: number;
  rowLabelWidth: number;
  colLabelHeight: number;
  scaleTiles: boolean;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
};

const Tiles = (props: Props) => {
  const tiles = [];
  if (!props.tilesLayout || props.tilesLayout.length === 0) {
    return null;
  }

  // Sort the tentative tiles
  const tentativeTiles = Array.from(props.tentativeTiles.values());
  tentativeTiles.sort((a, b) => {
    if (a.col === b.col) {
      return a.row - b.row;
    }
    return a.col - b.col;
  });

  for (let y = 0; y < props.gridDim; y += 1) {
    for (let x = 0; x < props.gridDim; x += 1) {
      const rune = props.tilesLayout[y * 15 + x];
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
            grabbable={false}
          />
        );
      }
    }
  }

  // The "tentative tiles" should be displayed slightly differently.

  tentativeTiles.forEach((t) => {
    tiles.push(
      <Tile
        rune={t.letter}
        value={runeToValues(t.letter, CrosswordGameTileValues)}
        width={props.boardSquareDim}
        height={props.boardSquareDim}
        x={t.col * props.boardSquareDim + props.rowLabelWidth}
        y={t.row * props.boardSquareDim + props.colLabelHeight}
        lastPlayed={false}
        key={`ttile_${t.col}_${t.row}`}
        scale={false}
        tentative={true}
        grabbable={true}
      />
    );
  });
  if (tentativeTiles.length > 0 && props.tentativeTileScore !== undefined) {
    tiles.push(
      <TentativeScore
        score={props.tentativeTileScore}
        width={props.boardSquareDim / 2}
        height={props.boardSquareDim / 3}
        x={tentativeTiles[0].col * props.boardSquareDim + props.rowLabelWidth}
        y={tentativeTiles[0].row * props.boardSquareDim + props.colLabelHeight}
        key="tentativescore"
      />
    );
  }

  return <>{tiles}</>;
};

export default Tiles;
