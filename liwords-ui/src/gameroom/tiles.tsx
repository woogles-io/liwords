import React from 'react';

import {
  CrosswordGameTileValues,
  runeToValues,
} from '../constants/tile_values';
import Tile from './tile';
import { EmptySpace, EphemeralTile, PlayedTiles } from '../utils/cwgame/common';
import { PlacementArrow } from '../utils/cwgame/tile_placement';

type Props = {
  gridDim: number;
  tilesLayout: string;
  lastPlayedTiles: PlayedTiles;
  placementArrow: PlacementArrow;
  scaleTiles: boolean;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
  returnToRack?: (
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
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

  let tentativeScoreStyle;
  let isHorizontal;
  if (tentativeTiles.length > 0) {
    let minRow = tentativeTiles[0].row;
    let maxRow = minRow;
    let minCol = tentativeTiles[0].col;
    let maxCol = minCol;
    for (let i = 1; i < tentativeTiles.length; ++i) {
      const { row, col } = tentativeTiles[i];
      minRow = Math.min(minRow, row);
      maxRow = Math.max(maxRow, row);
      minCol = Math.min(minCol, col);
      maxCol = Math.max(maxCol, col);
    }
    if (minRow === maxRow) {
      isHorizontal = true;
      // TODO: put Board in useContext and use letterAt instead of indexing directly into tilesLayout?
      if (
        minCol === maxCol &&
        props.placementArrow?.show &&
        !props.placementArrow.horizontal &&
        props.placementArrow.col === minCol
      ) {
        // only one tile was placed, use vertical iff placementArrow says so
        let y = maxRow;
        do ++y;
        while (
          y < props.gridDim &&
          props.tilesLayout[y * props.gridDim + minCol] !== EmptySpace
        );
        if (y === props.placementArrow.row) isHorizontal = false;
      }
    } else if (minCol === maxCol) {
      isHorizontal = false;
    }

    if (isHorizontal === true) {
      let x = minCol;
      do --x;
      while (
        x >= 0 &&
        props.tilesLayout[minRow * props.gridDim + x] !== EmptySpace
      );
      tentativeScoreStyle = { row: minRow, col: x + 1 };
    } else if (isHorizontal === false) {
      let y = minRow;
      do --y;
      while (
        y >= 0 &&
        props.tilesLayout[y * props.gridDim + minCol] !== EmptySpace
      );
      tentativeScoreStyle = { row: y + 1, col: minCol };
    }
  }

  for (let y = 0; y < props.gridDim; y += 1) {
    for (let x = 0; x < props.gridDim; x += 1) {
      const rune = props.tilesLayout[y * 15 + x];
      const tentativeScoreIsHere =
        tentativeScoreStyle &&
        tentativeScoreStyle.row === y &&
        tentativeScoreStyle.col === x;
      const tentativeScoreHere = tentativeScoreIsHere
        ? props.tentativeTileScore
        : undefined;
      const tentativeScoreHereIsHorizontal = tentativeScoreIsHere
        ? isHorizontal
        : undefined;
      if (rune !== ' ') {
        const lastPlayed = props.lastPlayedTiles[`R${y}C${x}`] === true;
        tiles.push(
          <Tile
            rune={rune}
            value={runeToValues(rune, CrosswordGameTileValues)}
            lastPlayed={lastPlayed}
            key={`tile_${x}_${y}`}
            scale={props.scaleTiles}
            tentativeScore={tentativeScoreHere}
            tentativeScoreIsHorizontal={tentativeScoreHereIsHorizontal}
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
              x={x}
              y={y}
              tentativeScore={tentativeScoreHere}
              tentativeScoreIsHorizontal={tentativeScoreHereIsHorizontal}
              grabbable={true}
            />
          );
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

  return <div className="tiles">{tiles}</div>;
});

export default Tiles;
