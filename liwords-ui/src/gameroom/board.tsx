import React from 'react';

import BoardSpaces from './board_spaces';
import { PlacementArrow } from '../utils/cwgame/tile_placement';
import BoardCoordLabels from './board_coord_labels';
import Tiles from './tiles';
import { EphemeralTile } from '../utils/cwgame/common';

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
  tilesLayout: string;
  showBonusLabels: boolean;
  lastPlayedLetters: { [tile: string]: boolean };
  currentRack: string;
  squareClicked: (row: number, col: number) => void;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
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
          tentativeTileScore={props.tentativeTileScore}
        />
      </g>
    </svg>
  );
};

export default Board;
