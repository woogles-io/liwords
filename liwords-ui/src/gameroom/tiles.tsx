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
  tilesLayout: Array<string>;
  lastPlayedLetters: Record<string, boolean>;
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

  let tentativeTilesRemaining = tentativeTiles.length;

  for (let y = 0; y < props.gridDim; y += 1) {
    for (let x = 0; x < props.gridDim; x += 1) {
      const rune = props.tilesLayout[y][x];
      if (rune !== ' ') {
        const lastPlayed = props.lastPlayedLetters[`R${y}C${x}`] === true;
        tiles.push(
          <Tile
            rune={rune}
            value={runeToValues(rune, CrosswordGameTileValues)}
            lastPlayed={lastPlayed}
            key={`tile_${x}_${y}`}
            scale={props.scaleTiles}
            grabbable={false}
          />
        );
      } else {
        const tentativeTile = tentativeTiles.find(tile => tile.col === x && tile.row === y);
        if (tentativeTile) {
            tiles.push(<Tile
                rune={tentativeTile.letter}
                value={runeToValues(tentativeTile.letter, CrosswordGameTileValues)}
                lastPlayed={false}
                key={`tileT_${tentativeTile.col}_${tentativeTile.row}`}
                scale={false}
                tentative={true}
                tentativeScore={tentativeTilesRemaining === tentativeTiles.length ?
                  props.tentativeTileScore : undefined}
                grabbable={true}
            />);
            tentativeTilesRemaining -= 1;
        } else {
            tiles.push(
                <div
                    className="empty-space"
                    key={`tile_${x}_${y}`}
                >
                    &nbsp;
                </div>
            );
        }
      }
    }
  }
  /*if (tentativeTiles.length > 0 && props.tentativeTileScore !== undefined) {
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
  }*/

  return <div className="tiles">
      {tiles}
  </div>;
};

export default Tiles;
