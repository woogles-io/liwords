import React from 'react';

import {
  CrosswordGameTileValues,
  runeToValues,
} from '../constants/tile_values';
import Tile from './tile';
import { EphemeralTile, PlayedTiles } from '../utils/cwgame/common';

type Props = {
  gridDim: number;
  tilesLayout: string;
  lastPlayedTiles: PlayedTiles;
  scaleTiles: boolean;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
};

const Tiles = React.memo((props: Props) => {
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
      const rune = props.tilesLayout[y * 15 + x];
      if (rune !== ' ') {
        const lastPlayed = props.lastPlayedTiles[`R${y}C${x}`] === true;
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
        const tentativeTile = tentativeTiles.find(
          (tile) => tile.col === x && tile.row === y
        );
        if (tentativeTile) {
          tiles.push(
            <Tile
              rune={tentativeTile.letter}
              value={runeToValues(
                tentativeTile.letter,
                CrosswordGameTileValues
              )}
              lastPlayed={false}
              key={`tileT_${tentativeTile.col}_${tentativeTile.row}`}
              scale={false}
              tentative={true}
              tentativeScore={
                tentativeTilesRemaining === tentativeTiles.length
                  ? props.tentativeTileScore
                  : undefined
              }
              grabbable={true}
            />
          );
          tentativeTilesRemaining -= 1;
        } else {
          tiles.push(
            <div className="empty-space" key={`tile_${x}_${y}`}>
              &nbsp;
            </div>
          );
        }
      }
    }
  }

  return <div className="tiles">{tiles}</div>;
});

export default Tiles;
