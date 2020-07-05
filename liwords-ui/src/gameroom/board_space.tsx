import React, { useRef } from 'react';
import { ReactComponent as Logo } from '../assets/aero.svg';
import { BonusType } from '../constants/board_layout';
import { useDrop } from 'react-dnd';
import { ArrowRightOutlined, ArrowDownOutlined } from '@ant-design/icons/lib';
import { DragType } from './tile';

const colors = require('../base.scss');

interface BonusProperties {
  fillColor: string;
  bonusText: string;
}

function getBonusProperties(bt: BonusType): BonusProperties {
  switch (bt) {
    case BonusType.DoubleWord:
      return { fillColor: colors.colorBoardDWS, bonusText: '2WS' };
    case BonusType.TripleWord:
      return { fillColor: colors.colorBoardTWS, bonusText: '3WS' };
    case BonusType.DoubleLetter:
      return { fillColor: colors.colorBoardDLS, bonusText: '2LS' };
    case BonusType.TripleLetter:
      return { fillColor: colors.colorBoardTLS, bonusText: '3LS' };
  }
  return { fillColor: 'hsl(35, 30%, 98%)', bonusText: '' };
}

type Props = {
  bonusType: BonusType;
  showBonusLabel: boolean;
  startingSquare: boolean;
  arrow: boolean;
  arrowHoriz: boolean;
  clicked: () => void;
};

const BoardSpace = (props: Props) => {
  const { fillColor, bonusText } = getBonusProperties(props.bonusType);
  const ref = useRef(null);
  let bonusLabel = null;
  let startingSquare = null;
  let arrow = null;
  if (props.showBonusLabel && bonusText !== '') {
    bonusLabel = <p className="bonus-label">{bonusText}</p>;
  }
  // ✩✪✫
  if (props.startingSquare) {
    startingSquare = (
      <Logo
        className="logo"
      />
    );
  }
  if (props.arrow) {
    if (props.arrowHoriz) {
      arrow = <ArrowRightOutlined />;
    } else {
      arrow = <ArrowDownOutlined />;
    }
  }

  const styleOverrides = {
    backgroundColor: fillColor,
  };

  const [{ isOver, canDrop }, drop] = useDrop({
    accept: DragType,
    drop: () => { console.log('drop!')},
    collect: (monitor) => ({
      isOver: !!monitor.isOver(),
      canDrop: !!monitor.canDrop()
    })
  });

  drop(ref);

  return (
    <div
      className="board-space"
      onClick={props.clicked}
      ref={ref}
      style={styleOverrides}
    >
      {bonusLabel}
      {startingSquare}
      {arrow}
    </div>
  );
};

export default BoardSpace;
