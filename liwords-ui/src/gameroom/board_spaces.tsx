import React, { useMemo, useRef } from "react";
import BoardSpace from "./board_space";
import { PlacementArrow } from "../utils/cwgame/tile_placement";
import { BonusType } from "../constants/board_layout";
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
const BoardSpaces = React.memo((props: Props) => {
  const { gridDim, gridLayout, placementArrow, squareClicked, handleTileDrop } =
    props;

  const boardRef = useRef<HTMLDivElement>(null);
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
            handleTileDrop={() => {}}
          />,
        );
      }
    }
    return spaces;
  }, [midway, gridDim, gridLayout, placementArrow, squareClicked]);

  const handleBoardDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    if (!handleTileDrop) return;
    let dragData: { rackIndex?: number; tileIndex?: number } = {};
    try {
      dragData = JSON.parse(e.dataTransfer.getData("text/plain"));
    } catch {}
    if (!boardRef.current) return;
    const boardElement = boardRef.current;
    const rect = boardElement.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const tileSize = rect.width / gridDim;
    const col = Math.floor(x / tileSize);
    const row = Math.floor(y / tileSize);
    if (row >= 0 && row < gridDim && col >= 0 && col < gridDim) {
      handleTileDrop(
        row,
        col,
        typeof dragData.rackIndex === "number" ? dragData.rackIndex : undefined,
        typeof dragData.tileIndex === "number" ? dragData.tileIndex : undefined,
      );
    }
  };

  const handleBoardDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
  };

  const handleBoardDragEnter = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
  };

  return (
    <div
      className="board-spaces"
      ref={boardRef}
      id="board-spaces"
      onDrop={handleBoardDrop}
      onDragOver={handleBoardDragOver}
      onDragEnter={handleBoardDragEnter}
    >
      {spaces}
    </div>
  );
});

export default BoardSpaces;
