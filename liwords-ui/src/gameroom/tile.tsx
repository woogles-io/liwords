import React, { useEffect, useRef } from 'react';
import { useMountedState } from '../utils/mounted';
import { useDrag, useDragLayer, useDrop } from 'react-dnd';
import TentativeScore from './tentative_score';
import {
  Blank,
  EmptySpace,
  isDesignatedBlank,
  isTouchDevice,
  uniqueTileIdx,
} from '../utils/cwgame/common';
import { Popover } from 'antd';

type TileLetterProps = {
  rune: string;
};

export const TILE_TYPE = 'TILE_TYPE';

export const TileLetter = React.memo((props: TileLetterProps) => {
  let { rune } = props;
  // if (rune.toUpperCase() !== rune) {
  //   rune = rune.toUpperCase();
  // }
  if (rune === Blank) {
    rune = ' ';
  }

  return <p className="rune">{rune}</p>;
});

type PointValueProps = {
  value: number;
};

export const PointValue = React.memo((props: PointValueProps) => {
  if (!props.value) {
    return null;
  }
  return <p className="point-value">{props.value}</p>;
});

type TilePreviewProps = {
  gridDim: number;
};

export const TilePreview = React.memo((props: TilePreviewProps) => {
  const { useState } = useMountedState();
  const [updateCount, setUpdateCount] = useState(0);
  const { isDragging, xyPosition, initialPosition, rune, value } = useDragLayer(
    (monitor) => ({
      xyPosition: monitor.getClientOffset(),
      initialPosition: monitor.getInitialClientOffset(),
      isDragging: monitor.isDragging(),
      rune: monitor.getItem()?.rune,
      value: monitor.getItem()?.value,
    })
  );
  const [position, setPosition] = useState(initialPosition);
  const boardElement = document.getElementById('board-spaces');
  useEffect(() => {
    // This makes us only re-render 1/5 of the time, to improve performance
    setUpdateCount(updateCount + 1);
    if (!xyPosition) {
      setPosition(null);
    }
    if (updateCount % 5 === 0 && xyPosition) {
      setPosition(xyPosition);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [xyPosition]);
  if (boardElement && position) {
    const boardTop = boardElement.getBoundingClientRect().top;
    const boardLeft = boardElement.getBoundingClientRect().left;
    const boardWidth = boardElement.getBoundingClientRect().width;
    let top = position.y - boardTop;
    let left = position.x - boardLeft;
    const overBoard =
      boardWidth > position?.y - boardTop &&
      position?.y > boardTop &&
      boardWidth > position?.x - boardLeft &&
      position?.x > boardLeft;
    const tileSize = boardWidth / props.gridDim;
    if (overBoard) {
      const col = Math.floor((position.x - boardLeft) / tileSize);
      const row = Math.floor((position.y - boardTop) / tileSize);
      left = col * tileSize + 6;
      top = row * tileSize + 23;
    }
    const computedStyle = {
      top,
      left,
    };
    const computedClass = `tile preview${overBoard ? ' over-board' : ''}`;
    if (isDragging) {
      return (
        <div className={computedClass} style={computedStyle}>
          <TileLetter rune={rune} />
          <PointValue value={value} />
        </div>
      );
    }
  }

  return null;
});

type TileProps = {
  lastPlayed: boolean;
  playerOfTile: number;
  rune: string;
  value: number;
  scale?: boolean;
  tentative?: boolean;
  tentativeScore?: number;
  tentativeScoreIsHorizontal?: boolean | undefined;
  grabbable: boolean;
  rackIndex?: number | undefined;
  selected?: boolean;
  moveRackTile?: (
    indexA: number | undefined,
    indexB: number | undefined
  ) => void;
  returnToRack?: (
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
  onClick?: (evt: React.MouseEvent<HTMLElement>) => void;
  onMouseEnter?: (evt: React.MouseEvent<HTMLElement>) => void;
  onMouseLeave?: (evt: React.MouseEvent<HTMLElement>) => void;
  x?: number | undefined;
  y?: number | undefined;
  popoverContent?: React.ReactNode;
  onPopoverClick?: (evt: React.MouseEvent<HTMLElement>) => void;
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
};

const Tile = React.memo((props: TileProps) => {
  const { useState } = useMountedState();

  const [isMouseDragging, setIsMouseDragging] = useState(false);

  const handleStartDrag = (e: any) => {
    if (e) {
      setIsMouseDragging(true);
      e.dataTransfer.dropEffect = 'move';
      if (
        props.tentative &&
        typeof props.x == 'number' &&
        typeof props.y == 'number'
      ) {
        e.dataTransfer.setData('tileIndex', uniqueTileIdx(props.y, props.x));
      } else {
        e.dataTransfer.setData('rackIndex', props.rackIndex);
      }
    }
  };

  const handleEndDrag = () => {
    setIsMouseDragging(false);
  };

  const handleDrop = (e: any) => {
    if (props.handleTileDrop && props.y != null && props.x != null) {
      props.handleTileDrop(
        props.y,
        props.x,
        parseInt(e.dataTransfer.getData('rackIndex'), 10),
        parseInt(e.dataTransfer.getData('tileIndex'), 10)
      );
      return;
    }
    if (props.moveRackTile && e.dataTransfer.getData('rackIndex')) {
      props.moveRackTile(
        props.rackIndex,
        parseInt(e.dataTransfer.getData('rackIndex'), 10)
      );
    } else {
      if (props.returnToRack && e.dataTransfer.getData('tileIndex')) {
        props.returnToRack(
          props.rackIndex,
          parseInt(e.dataTransfer.getData('tileIndex'), 10)
        );
      }
    }
  };

  const handleDropOver = (e: any) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const canDrag = props.grabbable && props.rune !== EmptySpace;
  const [{ isDragging }, drag, preview] = useDrag({
    item: {
      type: TILE_TYPE,
      rackIndex:
        typeof props.rackIndex === 'number'
          ? props.rackIndex.toString()
          : undefined,
      tileIndex:
        typeof props.x === 'number' && typeof props.y === 'number'
          ? uniqueTileIdx(props.y, props.x).toString()
          : undefined,
      rune: props.rune,
      value: props.value,
    },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
    canDrag: (monitor) => canDrag,
  });

  useEffect(() => {
    preview(<div></div>);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const [, drop] = useDrop({
    accept: TILE_TYPE,
    drop: (item: any, monitor: any) => {
      if (props.handleTileDrop && props.y != null && props.x != null) {
        props.handleTileDrop(
          props.y,
          props.x,
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

  const tileRef = useRef(null);
  const isTouchDeviceResult = isTouchDevice();
  useEffect(() => {
    if (canDrag && isTouchDeviceResult) {
      drag(tileRef);
    }
  }, [canDrag, isTouchDeviceResult, drag]);
  const canDrop = props.handleTileDrop && props.y != null && props.x != null;
  useEffect(() => {
    if (canDrop && isTouchDeviceResult) {
      drop(tileRef);
    }
  }, [canDrop, isTouchDeviceResult, drop]);

  const computedClassName = `tile${
    isDragging || isMouseDragging ? ' dragging' : ''
  }${canDrag ? ' droppable' : ''}${props.selected ? ' selected' : ''}${
    props.tentative ? ' tentative' : ''
  }${props.lastPlayed ? ' last-played' : ''}${
    isDesignatedBlank(props.rune) ? ' blank' : ''
  }${props.playerOfTile ? ' tile-p1' : ' tile-p0'}`;
  let ret = (
    <div onDragOver={handleDropOver} onDrop={handleDrop} ref={tileRef}>
      <div
        className={computedClassName}
        data-rune={props.rune}
        style={{
          cursor: canDrag ? 'grab' : 'default',
          ...(props.rune === EmptySpace ? { visibility: 'hidden' } : null),
        }}
        onClick={props.onClick}
        onMouseEnter={props.onMouseEnter}
        onMouseLeave={props.onMouseLeave}
        onDragStart={canDrag ? handleStartDrag : undefined}
        onDragEnd={handleEndDrag}
        draggable={canDrag}
      >
        {props.rune !== EmptySpace && (
          <React.Fragment>
            <TileLetter rune={props.rune} />
            <PointValue value={props.value} />
          </React.Fragment>
        )}
        <TentativeScore
          score={props.tentativeScore}
          horizontal={props.tentativeScoreIsHorizontal}
        />
      </div>
    </div>
  );
  ret = (
    <Popover
      content={<div onClick={props.onPopoverClick}>{props.popoverContent}</div>}
      visible={props.popoverContent != null}
    >
      {ret}
    </Popover>
  );
  return ret;
});

export default Tile;
