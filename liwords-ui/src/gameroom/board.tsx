import React from 'react';

import BoardSpaces from './board_spaces';
import { useDrawing } from './drawing';
import { PlacementArrow } from '../utils/cwgame/tile_placement';
import BoardCoordLabels from './board_coord_labels';
import Tiles from './tiles';
import {
  EphemeralTile,
  PlayedTiles,
  PlayerOfTiles,
} from '../utils/cwgame/common';

type Props = {
  gridLayout: Array<string>;
  gridSize: number;
  handleBoardTileClick: (rune: string) => void;
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
  tilesLayout: string;
  lastPlayedTiles: PlayedTiles;
  playerOfTileAt: PlayerOfTiles;
  currentRack: string;
  squareClicked: (row: number, col: number) => void;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
  placementArrowProperties: PlacementArrow;
  handleSetHover?: (
    x: number,
    y: number,
    words: Array<string> | undefined
  ) => void;
};

const Board = React.memo((props: Props) => {
  // Keep frames the same size, and shrink or grow the
  // board squares as necessary.

  const { outerDivProps, svgDrawing } = useDrawing();

  return (
    <div className="board">
      <BoardCoordLabels gridDim={props.gridSize} />
      <div className="board-spaces-container" id="board" {...outerDivProps}>
        <BoardSpaces
          gridDim={props.gridSize}
          gridLayout={props.gridLayout}
          handleTileDrop={props.handleTileDrop}
          placementArrow={props.placementArrowProperties}
          squareClicked={props.squareClicked}
        />
        <Tiles
          gridDim={props.gridSize}
          onClick={props.handleBoardTileClick}
          tilesLayout={props.tilesLayout}
          lastPlayedTiles={props.lastPlayedTiles}
          playerOfTileAt={props.playerOfTileAt}
          tentativeTiles={props.tentativeTiles}
          scaleTiles={true}
          placementArrow={props.placementArrowProperties}
          tentativeTileScore={props.tentativeTileScore}
          handleSetHover={props.handleSetHover}
        />
        {svgDrawing}
      </div>
    </div>
  );
});

export default Board;
