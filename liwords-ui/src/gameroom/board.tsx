import React from 'react';

import BoardSpaces from './board_spaces';
import { PlacementArrow } from '../utils/cwgame/tile_placement';
import BoardCoordLabels from './board_coord_labels';
import Tiles from './tiles';
import { EphemeralTile, PlayedTiles } from '../utils/cwgame/common';

type Props = {
  gridLayout: Array<string>;
  gridSize: number;
  handleBoardTileClick: (rune: string) => void;
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
  tilesLayout: string;
  lastPlayedTiles: PlayedTiles;
  currentRack: string;
  squareClicked: (row: number, col: number) => void;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
  placementArrowProperties: PlacementArrow;
};

const Board = React.memo((props: Props) => {
  // Drawing functionalities.
  // Right-drag = draw.
  // RightClick several times = clear drawing.
  // Shift+RightClick = clear drawing.
  // Shift+RightClick (when no drawing) = context menu.

  const boardEltRef = React.useRef<HTMLElement>();
  const [boardSize, setBoardSize] = React.useState({
    left: 0,
    top: 0,
    width: 1,
    height: 1,
  });
  const resizeFunc = React.useCallback(() => {
    const boardElt = boardEltRef.current;
    if (boardElt) {
      const { left, top, width, height } = boardElt.getBoundingClientRect();
      setBoardSize({
        left,
        top,
        width: Math.max(1, width),
        height: Math.max(1, height),
      });
    }
  }, []);
  const boardRef = React.useCallback(
    (elt) => {
      boardEltRef.current = elt;
      resizeFunc();
    },
    [resizeFunc]
  );
  React.useEffect(() => {
    window.addEventListener('resize', resizeFunc);
    return () => window.removeEventListener('resize', resizeFunc);
  }, [resizeFunc]);
  const getXY = React.useCallback(
    (evt: React.MouseEvent): { x: number; y: number } => {
      const x = Math.max(
        0,
        Math.min(1, (evt.clientX - boardSize.left) / boardSize.width)
      );
      const y = Math.max(
        0,
        Math.min(1, (evt.clientY - boardSize.top) / boardSize.height)
      );
      return { x, y };
    },
    [boardSize]
  );
  const [picture, setPicture] = React.useState<{
    drawing: boolean;
    picture: Array<Array<{ x: number; y: number }>>;
  }>({ drawing: false, picture: [] });
  const hasPicture = picture.picture.length > 0;
  const handleContextMenu = React.useCallback(
    (evt: React.MouseEvent) => {
      if (!evt.shiftKey) {
        // Draw when not holding shift.
        evt.preventDefault();
      } else if (hasPicture) {
        // Shift+RightClick clears drawing.
        setPicture((pic) => ({ ...pic, drawing: false, picture: [] }));
        evt.preventDefault();
      } else {
        // Shift+RightClick accesses context menu if no drawing.
      }
    },
    [hasPicture]
  );
  const handleMouseDown = React.useCallback(
    (evt: React.MouseEvent) => {
      if (evt.button === 2 && !evt.shiftKey) {
        const newXY = getXY(evt);
        setPicture((pic) => {
          pic.picture.push([newXY]); // mutate
          return { ...pic, drawing: true }; // shallow clone for performance
        });
      }
    },
    [getXY]
  );
  const handleMouseUp = React.useCallback((evt: React.MouseEvent) => {
    setPicture((pic) => {
      if (!pic.drawing) return pic;
      // Right-click this many times to clear drawing.
      const howMany = 3;
      if (pic.picture.length >= howMany) {
        let i = 0;
        for (; i < howMany; ++i) {
          if (
            !(
              pic.picture[pic.picture.length - (i + 1)].length < 2 &&
              pic.picture[pic.picture.length - (i + 1)][0].x ===
                pic.picture[pic.picture.length - 1][0].x &&
              pic.picture[pic.picture.length - (i + 1)][0].y ===
                pic.picture[pic.picture.length - 1][0].y
            )
          )
            break;
        }
        if (i === howMany) {
          return { ...pic, drawing: false, picture: [] };
        }
      }
      return { ...pic, drawing: false };
    });
  }, []);
  const handleMouseMove = React.useCallback(
    (evt: React.MouseEvent) => {
      const newXY = getXY(evt);
      setPicture((pic) => {
        if (!pic.drawing) return pic;
        const lastStroke = pic.picture[pic.picture.length - 1];
        const lastPoint = lastStroke[lastStroke.length - 1];
        if (lastPoint.x === newXY.x && lastPoint.y === newXY.y) return pic;
        lastStroke.push(newXY); // mutate
        return { ...pic }; // shallow clone for performance
      });
    },
    [getXY]
  );
  const handlePointerDown = React.useCallback((evt: React.PointerEvent) => {
    (evt.target as Element).setPointerCapture(evt.pointerId);
  }, []);
  const handlePointerUp = React.useCallback((evt: React.PointerEvent) => {
    (evt.target as Element).releasePointerCapture(evt.pointerId);
  }, []);
  const currentDrawing = React.useMemo(() => {
    let path = '';
    for (const stroke of picture.picture) {
      for (let i = 0; i < stroke.length; ++i) {
        const { x, y } = stroke[i];
        const scaledX = x * boardSize.width;
        const scaledY = y * boardSize.height;
        path += `${i === 0 ? 'M' : 'L'}${scaledX},${scaledY}`;
      }
      if (stroke.length === 1) {
        // Draw a diamond to represent a single point.
        path += 'm-1,0l1,1l1,-1l-1,-1l-1,1l1,1';
      }
    }
    return <path d={path} fill="none" strokeWidth={5} stroke="red" />;
  }, [picture, boardSize]);

  // Keep frames the same size, and shrink or grow the
  // board squares as necessary.

  return (
    <div className="board">
      <BoardCoordLabels gridDim={props.gridSize} />
      <div
        className="board-spaces-container"
        ref={boardRef}
        onContextMenu={handleContextMenu}
        onMouseDown={handleMouseDown}
        onMouseUp={handleMouseUp}
        onMouseMove={handleMouseMove}
        onPointerDown={handlePointerDown}
        onPointerUp={handlePointerUp}
      >
        <BoardSpaces
          gridDim={props.gridSize}
          gridLayout={props.gridLayout}
          handleTileDrop={props.handleTileDrop}
          placementArrow={props.placementArrowProperties}
          squareClicked={props.squareClicked}
        />
        <Tiles
          gridDim={props.gridSize}
          onClick={props.handleBoardTileClick}
          tilesLayout={props.tilesLayout}
          lastPlayedTiles={props.lastPlayedTiles}
          tentativeTiles={props.tentativeTiles}
          scaleTiles={true}
          placementArrow={props.placementArrowProperties}
          tentativeTileScore={props.tentativeTileScore}
        />
        <svg
          viewBox={`0 0 ${boardSize.width} ${boardSize.height}`}
          style={{
            position: 'absolute',
            left: 0,
            top: 0,
            width: boardSize.width,
            height: boardSize.height,
            pointerEvents: 'none',
          }}
        >
          {currentDrawing}
        </svg>
      </div>
    </div>
  );
});

export default Board;
