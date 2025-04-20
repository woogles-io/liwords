import React, { useEffect, useMemo, useRef } from "react";
import BoardSpace from "./board_space";
import { PlacementArrow } from "../utils/cwgame/tile_placement";
import { BonusType } from "../constants/board_layout";
import { isTouchDevice } from "../utils/cwgame/common";
import { useDrop, XYCoord } from "react-dnd";
import { TILE_TYPE } from "./tile";

type Props = {
  gridDim: number;
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined,
  ) => void;
  gridLayout: Array<string>;
  placementArrow: PlacementArrow;
  squareClicked: (row: number, col: number) => void;
};
const calculatePosition = (
  position: XYCoord,
  boardElement: HTMLElement,
  gridSize: number,
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
  const { gridDim, gridLayout, placementArrow, squareClicked, handleTileDrop } =
    props;

  const boardRef = useRef(null);

  const [, drop] = useDrop({
    accept: TILE_TYPE,
    drop: (item: { rackIndex: string; tileIndex: string }, monitor) => {
      const clientOffset = monitor.getClientOffset();
      const boardElement = document.getElementById("board");
      if (clientOffset && handleTileDrop && boardElement) {
        const { row, col } = calculatePosition(
          clientOffset,
          boardElement,
          gridDim,
        );
        handleTileDrop(
          row,
          col,
          parseInt(item.rackIndex, 10),
          parseInt(item.tileIndex, 10),
        );
      }
    },
    collect: (monitor) => ({
      isOver: !!monitor.isOver(),
      canDrop: !!monitor.canDrop(),
    }),
  });

  // Always attach the drop ref, regardless of device
  drop(boardRef);
  // y row, x col
  const midway = Math.trunc(gridDim / 2);

  const spaces = useMemo(() => {
    const spaces = [];
    for (let y = 0; y < gridDim; y += 1) {
      for (let x = 0; x < gridDim; x += 1) {
        const sq = gridLayout[y][x];
        const startingSquare = x === midway && y === midway;
        const showArrow =
          placementArrow.show &&
          placementArrow.row === y &&
          placementArrow.col === x;
        spaces.push(
          <BoardSpace
            bonusType={sq as BonusType}
            key={`sq_${x}_${y}`}
            arrow={showArrow}
            arrowHoriz={placementArrow.horizontal}
            startingSquare={startingSquare}
            clicked={() => squareClicked(y, x)}
          />,
        );
      }
    }
    return spaces;
  }, [midway, gridDim, gridLayout, placementArrow, squareClicked]);

  return (
    <div className="board-spaces" ref={boardRef} id="board-spaces">
      {spaces}
    </div>
  );
});

export default BoardSpaces;
