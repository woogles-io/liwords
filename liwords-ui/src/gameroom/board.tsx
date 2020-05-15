import React from 'react';

import BoardSpaces from './board_spaces';
import BoardCoordLabels from './board_coord_labels';
// The frame atop is 24 height
// The frames on the sides are 24 in width, surrounded by a 14 pix gutter

const sideFrameWidth = 24;
const topFrameHeight = 24;
const sideFrameGutter = 14;
// XXX: Later make the 15 customizable if we want to add other sizes.
const gridSize = 15;

type Props = {
  // component width:
  compWidth: number;
  gridLayout: Array<string>;
  showBonusLabels: boolean;
};

export const Board = (props: Props) => {
  // Keep frames the same size, and shrink or grow the
  // board squares as necessary.
  const sideFrames = (sideFrameWidth + sideFrameGutter * 2) * 2;
  const boardDim = props.compWidth - sideFrames;
  const sqWidth = boardDim / gridSize;

  return (
    <svg width={props.compWidth} height={boardDim + topFrameHeight}>
      <g>
        {/* apply transform here to the g */}
        <BoardCoordLabels
          gridDim={gridSize}
          boardSquareDim={sqWidth}
          rowLabelWidth={sideFrameWidth}
          colLabelHeight={topFrameHeight}
          rowLabelGutter={sideFrameGutter}
          colLabelGutter={0}
        />
        <BoardSpaces
          gridDim={gridSize}
          boardSquareDim={sqWidth}
          rowLabelWidth={sideFrameWidth + sideFrameGutter * 2}
          colLabelHeight={topFrameHeight}
          gridLayout={props.gridLayout}
          showBonusLabels={props.showBonusLabels}
        />
        {/* <Tiles
          gridWidth={props.gridWidth}
          gridHeight={props.gridHeight}
          rowLabelWidth={rowLabelWidth}
          colLabelHeight={colLabelHeight}
          boardSquareWidth={boardSquareWidth}
          boardSquareHeight={boardSquareHeight}
          tilesLayout={props.tilesLayout}
          lastPlayedLetters={props.lastPlayedLetters}
        /> */}
      </g>
    </svg>
  );
};
