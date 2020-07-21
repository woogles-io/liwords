import React from 'react';

import BoardSpace from './board_space';
import { PlacementArrow } from '../utils/cwgame/tile_placement';
import { BonusType } from '../constants/board_layout';

type Props = {
  gridDim: number;
  handleTileDrop?: (row: number, col: number, rackIndex: number | undefined, tileIndex: number | undefined) => void;
  showBonusLabels: boolean;
  gridLayout: Array<string>;
  placementArrow: PlacementArrow;
  squareClicked: (row: number, col: number) => void;
};

const BoardSpaces = React.memo((props: Props) => {
  const spaces = [];
  // y row, x col
  for (let y = 0; y < props.gridDim; y += 1) {
    for (let x = 0; x < props.gridDim; x += 1) {
      const sq = props.gridLayout[y][x];
      const startingSquare = x === 7 && y === 7;
      const showArrow =
        props.placementArrow.show &&
        props.placementArrow.row === y &&
        props.placementArrow.col === x;
      spaces.push(
        <BoardSpace
          bonusType={sq as BonusType}
          key={`sq_${x}_${y}`}
          showBonusLabel={props.showBonusLabels && !startingSquare}
          arrow={showArrow}
          arrowHoriz={props.placementArrow.horizontal}
          startingSquare={startingSquare}
          clicked={() => props.squareClicked(y, x)}
          handleTileDrop={ (e : any) => {
            if (props.handleTileDrop) {
              props.handleTileDrop(
                y,
                x,
                parseInt(e.dataTransfer.getData('rackIndex'), 10),
                parseInt(e.dataTransfer.getData('tileIndex'), 10),
              );
            }
          }}
        />
      );
    }
  }
  return <div className="board-spaces">{spaces}</div>;
});

export default BoardSpaces;
