import React, { useState } from 'react';
import TentativeScore from './tentative_score';
import { Blank, uniqueTileIdx } from '../utils/cwgame/common';
const colors = require('../base.scss');

type TileStyle = {
  backgroundColor: string;
  outline: string;
  color: string;
  blankTextColor: string;
  strokeWidth: string;
};

const TILE_STYLES: { [name: string]: TileStyle } = {
  primary: {
    backgroundColor: colors.colorSecondary,
    outline: '#ffffff',
    color: '#ffffff',
    blankTextColor: '#11fefe',
    strokeWidth: '0px',
  },
  primaryJustPlayed: {
    backgroundColor: colors.colorSecondaryMedium,
    outline: '#ffffff',
    color: '#ffffff',
    blankTextColor: '#11fefe',
    strokeWidth: '0px',
  },
  primaryTentative: {
    backgroundColor: colors.colorSecondary,
    outline: colors.colorSecondaryMedium,
    color: colors.colorSecondaryLight,
    blankTextColor: '#fe1111',
    strokeWidth: '0px',
  },
  primarySelectedForExchange: {
    backgroundColor: colors.colorPrimary,
    outline: colors.colorPrimary,
    color: '#ffffff',
    blankTextColor: '#fe1111',
    strokeWidth: '0px',
  },
};

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

const PointValue = (props: PointValueProps) => {
  if (!props.value) {
    return null;
  }
  return <p className="point-value">{props.value}</p>;
};

type TileProps = {
  lastPlayed: boolean;
  rune: string;
  value: number;
  scale?: boolean;
  tentative?: boolean;
  tentativeScore?: number;
  grabbable: boolean;
  rackIndex?: number | undefined;
  selected?: boolean;
  swapRackTiles?: (
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
  let tileStyle = TILE_STYLES.primary;
  if (props.lastPlayed) {
    tileStyle = TILE_STYLES.primaryJustPlayed;
  }
  if (props.tentative) {
    tileStyle = TILE_STYLES.primaryTentative;
  }
  if (props.selected) {
    tileStyle = TILE_STYLES.primarySelectedForExchange;
  }

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
    if (props.swapRackTiles && e.dataTransfer.getData('rackIndex')) {
      props.swapRackTiles(
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
  }${props.selected ? ' selected' : ''}`;
  return (
    <div
      className={computedClassName}
      data-rune={props.rune}
      style={{ ...tileStyle, cursor: props.grabbable ? 'grab' : 'default' }}
      onClick={props.onClick ? props.onClick : () => {}}
      onDragStart={handleStartDrag}
      onDragEnd={handleEndDrag}
      onDragOver={handleDropOver}
      onDrop={handleDrop}
      draggable={props.grabbable}
    >
      <TileLetter rune={props.rune} />
      <PointValue value={props.value} />
      <TentativeScore score={props.tentativeScore} />
    </div>
  );
});

export default Tile;
