import React from 'react';
import { ReactComponent as Logo } from '../aero.svg';

const fontFamily = 'Arial,Geneva,Helvetica,Helv,sans-serif';

export enum BonusType {
  DoubleWord = '-',
  TripleWord = '=',
  QuadrupleWord = '~',
  DoubleLetter = "'",
  TripleLetter = '"',
  QuadrupleLetter = '^',
  StartingSquare = '*',
  NoBonus = ' ',
}

interface BonusProperties {
  fillColor: string;
  bonusText: string;
}

function getBonusProperties(bt: BonusType): BonusProperties {
  switch (bt) {
    case BonusType.DoubleWord:
      return { fillColor: '#FF75DD', bonusText: '2WS' };
    case BonusType.TripleWord:
      return { fillColor: '#FF5555', bonusText: '3WS' };
    case BonusType.DoubleLetter:
      return { fillColor: '#9ACEFD', bonusText: '2LS' };
    case BonusType.TripleLetter:
      return { fillColor: '#0006BD', bonusText: '3LS' };
  }
  return { fillColor: '#FFFFFF', bonusText: '' };
}

type Props = {
  bonusType: BonusType;
  boardSquareDim: number;
  x: number;
  y: number;
  showBonusLabel: boolean;
  startingSquare: boolean;
};

const BoardSpace = (props: Props) => {
  const transform = `translate(${props.x},${props.y})`;
  const { fillColor, bonusText } = getBonusProperties(props.bonusType);

  let bonusLabel = null;
  let startingSquare = null;
  if (props.showBonusLabel && bonusText !== '') {
    bonusLabel = (
      <text
        x={props.boardSquareDim / 2}
        y={props.boardSquareDim / 2}
        textAnchor="middle"
        dominantBaseline="central"
        fontFamily={fontFamily}
        fontSize="60%"
        stroke="#DADADB"
        fill="#DADADB"
        strokeWidth="0.5px"
      >
        {bonusText}
      </text>
    );
  }
  // ✩✪✫
  if (props.startingSquare) {
    startingSquare = <Logo />;
  }

  return (
    <g transform={transform}>
      <rect
        width={props.boardSquareDim}
        height={props.boardSquareDim}
        strokeWidth="0.5px"
        stroke={'#BEBEBE'}
        fill={fillColor}
      />
      {bonusLabel}
      {startingSquare}
    </g>
  );
};

export default BoardSpace;
