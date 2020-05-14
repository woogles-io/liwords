import React from 'react';

type Props = {
  // component width:
  compWidth: number;
};

export const Board = (props: Props) => {
  const labelRatio = 1 / 2;
  let numCols;

  // The frame atop is 24 height
  // The frames on the sides are 24 in width, surrounded by a 14 pix gutter

  // Keep frames the same size, and shrink or grow the
  // board squares as necessary.

  const rowLabelWidth = boardWidth / (props.gridWidth / labelRatio + 1);
  const colLabelHeight = boardHeight / (props.gridHeight / labelRatio + 1);

  const boardSquareWidth = rowLabelWidth / labelRatio;
  const boardSquareHeight = colLabelHeight / labelRatio;

  return (
    <svg width={boardWidth} height={boardHeight}>
      <g>
        {/* apply transform here to the g */}
        <BoardCoordLabels
          gridWidth={props.gridWidth}
          gridHeight={props.gridHeight}
          rowLabelWidth={rowLabelWidth}
          colLabelHeight={colLabelHeight}
          boardSquareWidth={boardSquareWidth}
          boardSquareHeight={boardSquareHeight}
        />
        <BoardSpaces
          gridWidth={props.gridWidth}
          gridHeight={props.gridHeight}
          rowLabelWidth={rowLabelWidth}
          colLabelHeight={colLabelHeight}
          boardSquareWidth={boardSquareWidth}
          boardSquareHeight={boardSquareHeight}
          gridLayout={props.gridLayout}
          showBonusLabels={props.showBonusLabels}
        />
        <Tiles
          gridWidth={props.gridWidth}
          gridHeight={props.gridHeight}
          rowLabelWidth={rowLabelWidth}
          colLabelHeight={colLabelHeight}
          boardSquareWidth={boardSquareWidth}
          boardSquareHeight={boardSquareHeight}
          tilesLayout={props.tilesLayout}
          lastPlayedLetters={props.lastPlayedLetters}
        />
      </g>
    </svg>
  );
};
