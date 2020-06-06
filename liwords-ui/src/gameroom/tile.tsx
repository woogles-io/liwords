import React from 'react';
import TentativeScore from "./tentative_score";

export const Blank = '?';

type TileStyle = {
  backgroundColor: string;
  outline: string;
  color: string;
  blankTextColor: string;
  strokeWidth: string;
};

const TILE_STYLES: { [name: string]: TileStyle } = {
  aeroBlue: {
    backgroundColor: '#2D6A9E',
    outline: '#ffffff',
    color: '#ffffff',
    blankTextColor: '#11fefe',
    strokeWidth: '0px',
  },
  aeroOrange: {
    backgroundColor: '#F0AD4E',
    outline: '#FFFFFF',
    color: '#FFFFFF',
    blankTextColor: '#fe1111',
    strokeWidth: '0px',
  },
  aeroOrangeJustPlayed: {
    backgroundColor: '#925b0c',
    outline: '#FFFFFF',
    color: '#FFFFFF',
    blankTextColor: '#fe1111',
    strokeWidth: '0px',
  },
  aeroOrangeTentative: {
    backgroundColor: '#F0AD4E',
    outline: '#449E2D',
    color: '#FFFFFF',
    blankTextColor: '#fe1111',
    strokeWidth: '0px',
  },
};

type TileLetterProps = {
  rune: string;
};

const TileLetter = (props: TileLetterProps) => {
  let { rune } = props;
  if (rune.toUpperCase() !== rune) {
    rune = rune.toUpperCase();
  }
  if (rune === Blank) {
    rune = ' ';
  }

  return (
    <p className="rune">
      {rune}
    </p>
  );
};

type PointValueProps = {
  value: number;
};

const PointValue = (props: PointValueProps) => {
  if (!props.value) {
    return null;
  }
  return (
    <p className="point-value">
      {props.value}
    </p>
  );
};

type TileProps = {
  lastPlayed: boolean;
  rune: string;
  value: number;
  scale?: boolean;
  tentative?: boolean;
  tentativeScore?: number;
  grabbable: boolean;
  onClick?: () => void;
};

const Tile = (props: TileProps) => {
  let tileStyle = TILE_STYLES.aeroOrange;
  if (props.lastPlayed) {
    tileStyle = TILE_STYLES.aeroOrangeJustPlayed;
  }
  if (props.tentative) {
    tileStyle = TILE_STYLES.aeroOrangeTentative;
  }

  return (
    <div
      className="tile"
      style={Object.assign(
          {},
          tileStyle,
          {
            cursor: props.grabbable ? 'grab' : 'default',
          }
      )}
      onClick={props.onClick ? props.onClick : () => {}}
    >
      <TileLetter
        rune={props.rune}
      />
      <PointValue
        value={props.value}
      />
      <TentativeScore score={props.tentativeScore} />
    </div>
  );
};

export default Tile;
