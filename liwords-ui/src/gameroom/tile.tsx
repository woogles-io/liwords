import React from 'react';

const fontFamily = "'Roboto', sans-serif";
export const Blank = '?';

type TileStyle = {
  color: string;
  outline: string;
  textColor: string;
  blankTextColor: string;
  strokeWidth: string;
};

const TILE_STYLES: Record<string, TileStyle> = {
  aeroBlue: {
    color: '#2D6A9E',
    outline: '#ffffff',
    textColor: '#ffffff',
    blankTextColor: '#11fefe',
    strokeWidth: '0.5px',
  },
  aeroOrange: {
    color: '#F0AD4E',
    outline: '#FFFFFF',
    textColor: '#FFFFFF',
    blankTextColor: '#fe1111',
    strokeWidth: '0.5px',
  },
  aeroOrangeJustPlayed: {
    color: '#925b0c',
    outline: '#FFFFFF',
    textColor: '#FFFFFF',
    blankTextColor: '#fe1111',
    strokeWidth: '0.5px',
  },
  aeroOrangeTentative: {
    color: '#F0AD4E',
    outline: '#449E2D',
    textColor: '#FFFFFF',
    blankTextColor: '#fe1111',
    strokeWidth: '3px',
  },
};

// Get the desired font size and weights as a function of the width.
function tileProps(width: number) {
  // This formula is not the most scientific. The tiles look optimal at 130%
  // if the tile size is 31.
  const size = (120 / 31) * width;
  let weight = '500';
  if (width < 24) {
    weight = '300';
  }
  const valueSize = (50 / 31) * width;
  return {
    size,
    weight,
    valueSize,
  };
}

type TileLetterProps = {
  tileStyle: TileStyle;
  rune: string;
  width: number;
  height: number;
};

const TileLetter = (props: TileLetterProps) => {
  let letterColor = props.tileStyle.textColor;
  let { rune } = props;
  if (rune.toUpperCase() !== rune) {
    letterColor = props.tileStyle.blankTextColor;
    rune = rune.toUpperCase();
  }
  if (rune === Blank) {
    rune = ' ';
  }

  const font = tileProps(props.width);

  return (
    <text
      x={props.width / 2}
      y={props.height / 2 - props.width / 30}
      textAnchor="middle"
      dominantBaseline="central"
      fontFamily={fontFamily}
      fontWeight={font.weight}
      fontSize={`${font.size}%`}
      fill={letterColor}
      strokeWidth={0}
    >
      {rune}
    </text>
  );
};

type PointValueProps = {
  value: number;
  width: number;
  height: number;
  tileStyle: TileStyle;
};

const PointValue = (props: PointValueProps) => {
  if (!props.value) {
    return null;
  }
  const font = tileProps(props.width);
  return (
    <text
      x={8 * (props.width / 10)}
      y={8 * (props.height / 10)}
      textAnchor="middle"
      dominantBaseline="central"
      fontFamily={fontFamily}
      fontSize={`${font.valueSize}%`}
      stroke={props.tileStyle.textColor}
      strokeWidth="0.03px"
      fill={props.tileStyle.textColor}
    >
      {props.value}
    </text>
  );
};

type TileProps = {
  x: number;
  y: number;
  width: number;
  height: number;
  lastPlayed: boolean;
  rune: string;
  value: number;
  scale?: boolean;
  tentative?: boolean;
  grabbable: boolean;
  onClick?: () => void;
};

const Tile = (props: TileProps) => {
  let scaleFactor = 0.95;
  let realX;
  let realY;
  if (props.scale) {
    realX = props.x + ((1 - scaleFactor) * props.width) / 2;
    realY = props.y + ((1 - scaleFactor) * props.height) / 2;
  } else {
    realX = props.x;
    realY = props.y;
    scaleFactor = 1.0;
  }
  const transform = `translate(${realX}, ${realY})`;

  let tileStyle = TILE_STYLES.aeroOrange;
  if (props.lastPlayed) {
    tileStyle = TILE_STYLES.aeroOrangeJustPlayed;
  }
  if (props.tentative) {
    tileStyle = TILE_STYLES.aeroOrangeTentative;
  }

  return (
    <g
      transform={transform}
      style={{ cursor: props.grabbable ? 'grab' : 'default' }}
      onClick={props.onClick ? props.onClick : () => {}}
    >
      <rect
        width={scaleFactor * props.width}
        height={scaleFactor * props.height}
        strokeWidth={tileStyle.strokeWidth}
        stroke={tileStyle.outline}
        fill={tileStyle.color}
        rx={3}
        ry={3}
      />
      <TileLetter
        tileStyle={tileStyle}
        rune={props.rune}
        width={props.width}
        height={props.height}
      />
      <PointValue
        tileStyle={tileStyle}
        value={props.value}
        width={props.width}
        height={props.height}
      />
    </g>
  );
};

export default Tile;
