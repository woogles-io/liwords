import React from 'react';

import BoardSpaces from './board_spaces';
import { PlacementArrow, EphemeralTile } from './tile_placement';
import BoardCoordLabels from './board_coord_labels';
import Tiles from './tiles';

type Props = {
  // component width:
  compWidth: number;
  boardDim: number;
  topFrameHeight: number;
  gridLayout: Array<string>;
  gridSize: number;
  sqWidth: number;
  sideFrameWidth: number;
  sideFrameGutter: number;
  tilesLayout: Array<string>;
  showBonusLabels: boolean;
  lastPlayedLetters: Record<string, boolean>;
  currentRack: string;
  squareClicked: (row: number, col: number) => void;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number;
  placementArrowProperties: PlacementArrow;
};

const Board = (props: Props) => {
  // Keep frames the same size, and shrink or grow the
  // board squares as necessary.

  return (
    <svg width={props.compWidth} height={props.boardDim + props.topFrameHeight}>
      <g>
        {/* apply transform here to the g */}
        <BoardCoordLabels
          gridDim={props.gridSize}
          boardSquareDim={props.sqWidth}
          rowLabelWidth={props.sideFrameWidth}
          colLabelHeight={props.topFrameHeight}
          rowLabelGutter={props.sideFrameGutter}
          colLabelGutter={0}
        />
        <BoardSpaces
          gridDim={props.gridSize}
          boardSquareDim={props.sqWidth}
          rowLabelWidth={props.sideFrameWidth + props.sideFrameGutter * 2}
          colLabelHeight={props.topFrameHeight}
          gridLayout={props.gridLayout}
          showBonusLabels={props.showBonusLabels}
          placementArrow={props.placementArrowProperties}
          squareClicked={props.squareClicked}
        />
        <Tiles
          gridDim={props.gridSize}
          rowLabelWidth={props.sideFrameWidth + props.sideFrameGutter * 2}
          colLabelHeight={props.topFrameHeight}
          boardSquareDim={props.sqWidth}
          tilesLayout={props.tilesLayout}
          lastPlayedLetters={props.lastPlayedLetters}
          tentativeTiles={props.tentativeTiles}
          scaleTiles={true}
        />
      </g>
    </svg>
  );
};

export default Board;
