import React from 'react';

import BoardSpaces from './board_spaces';
import { PlacementArrow } from '../utils/cwgame/tile_placement';
import BoardCoordLabels from './board_coord_labels';
import Tiles from './tiles';
import { EphemeralTile, PlayedTiles } from '../utils/cwgame/common';

type Props = {
  gridLayout: Array<string>;
  gridSize: number;
  handleTileDrop?: (row: number, col: number, rackIndex: number | undefined, tileIndex: number | undefined) => void;
  tilesLayout: string;
  showBonusLabels: boolean;
  lastPlayedTiles: PlayedTiles;
  currentRack: string;
  squareClicked: (row: number, col: number) => void;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
  placementArrowProperties: PlacementArrow;
};

const Board = React.memo((props: Props) => {
  // Keep frames the same size, and shrink or grow the
  // board squares as necessary.

  return (
    <div className="board">
      <BoardCoordLabels gridDim={props.gridSize} />
      <div className="board-spaces-container">
        <BoardSpaces
          gridDim={props.gridSize}
          gridLayout={props.gridLayout}
          handleTileDrop={props.handleTileDrop}
          showBonusLabels={props.showBonusLabels}
          placementArrow={props.placementArrowProperties}
          squareClicked={props.squareClicked}
        />
        <Tiles
          gridDim={props.gridSize}
          tilesLayout={props.tilesLayout}
          lastPlayedTiles={props.lastPlayedTiles}
          tentativeTiles={props.tentativeTiles}
          scaleTiles={true}
          tentativeTileScore={props.tentativeTileScore}
        />
      </div>
    </div>
  );
});

export default Board;
