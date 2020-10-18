import React, { useState } from 'react';
import TentativeScore from './tentative_score';
import { Blank, uniqueTileIdx } from '../utils/cwgame/common';

type TileLetterProps = {
  rune: string;
};

const TileLetter = React.memo((props: TileLetterProps) => {
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

const PointValue = React.memo((props: PointValueProps) => {
  if (!props.value) {
    return null;
  }
  return <p className="point-value">{props.value}</p>;
});

type TileProps = {
  lastPlayed: boolean;
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
  onClick?: () => void;
  x?: number | undefined;
  y?: number | undefined;
};

const Tile = React.memo((props: TileProps) => {
  const [isDragging, setIsDragging] = useState(false);

  const handleStartDrag = (e: any) => {
    if (e) {
      setIsDragging(true);
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
    setIsDragging(false);
  };

  const handleDrop = (e: any) => {
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

  const computedClassName = `tile${isDragging ? ' dragging' : ''}${
    props.grabbable ? ' droppable' : ''
  }${props.selected ? ' selected' : ''}${props.tentative ? ' tentative' : ''}${
    props.lastPlayed ? ' last-played' : ''
  }`;
  return (
    <div
      className={computedClassName}
      data-rune={props.rune}
      style={{ cursor: props.grabbable ? 'grab' : 'default' }}
      onClick={props.onClick ? props.onClick : () => {}}
      onDragStart={handleStartDrag}
      onDragEnd={handleEndDrag}
      onDragOver={handleDropOver}
      onDrop={handleDrop}
      draggable={props.grabbable}
    >
      <TileLetter rune={props.rune} />
      <PointValue value={props.value} />
      <TentativeScore
        score={props.tentativeScore}
        horizontal={props.tentativeScoreIsHorizontal}
      />
    </div>
  );
});

export default Tile;
