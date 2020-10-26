import React, { useRef } from 'react';
import BoardSpace from './board_space';
import { PlacementArrow } from '../utils/cwgame/tile_placement';
import { BonusType } from '../constants/board_layout';
import { isTouchDevice } from '../utils/cwgame/common';
import { useDrop, XYCoord } from 'react-dnd';
import { TILE_TYPE } from './tile';

type Props = {
  gridDim: number;
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
  gridLayout: Array<string>;
  placementArrow: PlacementArrow;
  squareClicked: (row: number, col: number) => void;
};
const calculatePosition = (
  position: XYCoord,
  boardElement: HTMLElement,
  gridSize: number
) => {
  const boardTop = boardElement.getBoundingClientRect().top;
  const boardLeft = boardElement.getBoundingClientRect().left;
  const tileSize = boardElement.clientHeight / gridSize;
  return {
    col: Math.floor((position.x - boardLeft) / tileSize),
    row: Math.floor((position.y - boardTop) / tileSize),
  };
};

const BoardSpaces = React.memo((props: Props) => {
  const spaces = [];
  const boardRef = useRef(null);

  const [, drop] = useDrop({
    accept: TILE_TYPE,
    drop: (item: any, monitor) => {
      const clientOffset = monitor.getClientOffset();
      const boardElement = document.getElementById('board');
      if (clientOffset && props.handleTileDrop && boardElement) {
        const { row, col } = calculatePosition(
          clientOffset,
          boardElement,
          props.gridDim
        );
        props.handleTileDrop(
          row,
          col,
          parseInt(item.rackIndex, 10),
          parseInt(item.tileIndex, 10)
        );
      }
    },
    collect: (monitor) => ({
      isOver: !!monitor.isOver(),
      canDrop: !!monitor.canDrop(),
    }),
  });

  if (isTouchDevice()) {
    drop(boardRef);
  }
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
          arrow={showArrow}
          arrowHoriz={props.placementArrow.horizontal}
          startingSquare={startingSquare}
          clicked={() => props.squareClicked(y, x)}
          handleTileDrop={(e: any) => {
            if (props.handleTileDrop) {
              props.handleTileDrop(
                y,
                x,
                parseInt(e.dataTransfer.getData('rackIndex'), 10),
                parseInt(e.dataTransfer.getData('tileIndex'), 10)
              );
            }
          }}
        />
      );
    }
  }
  return (
    <div className="board-spaces" ref={boardRef}>
      {spaces}
    </div>
  );
});

export default BoardSpaces;
