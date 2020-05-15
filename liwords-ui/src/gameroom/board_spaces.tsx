import React from 'react';

import BoardSpace, { BonusType } from './board_space';

type Props = {
  gridDim: number;
  boardSquareDim: number;
  rowLabelWidth: number;
  colLabelHeight: number;
  showBonusLabels: boolean;
  gridLayout: Array<string>;
}

const BoardSpaces = (props: Props) => {
  const spaces = [];
  for (let y = 0; y < props.gridDim; y += 1) {
    for (let x = 0; x < props.gridDim; x += 1) {
      const sq = props.gridLayout[y][x];
      const startingSquare = x === 7 && y === 7;
      spaces.push(<BoardSpace
        bonusType={sq as BonusType}
        boardSquareDim={props.boardSquareDim}
        x={(x * props.boardSquareDim) + props.rowLabelWidth}
        y={(y * props.boardSquareDim) + props.colLabelHeight}
        key={`sq_${x}_${y}`}
        showBonusLabel={props.showBonusLabels && !startingSquare}
        startingSquare={startingSquare}
      />);
    }
  }
  return <>{spaces}</>;
};

export default BoardSpaces;