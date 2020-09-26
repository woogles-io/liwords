import React from 'react';
import { BonusType } from '../constants/board_layout';
import {
  ArrowRightOutlined,
  ArrowDownOutlined,
  StarOutlined,
} from '@ant-design/icons/lib';

interface BonusProperties {
  bonusText: string;
  bonusClass: string;
}

function getBonusProperties(bt: BonusType): BonusProperties {
  switch (bt) {
    case BonusType.DoubleWord:
      return { bonusText: '2x word', bonusClass: '2WS' };
    case BonusType.TripleWord:
      return { bonusText: '3x word', bonusClass: '3WS' };
    case BonusType.DoubleLetter:
      return { bonusText: '2x letter', bonusClass: '2LS' };
    case BonusType.TripleLetter:
      return { bonusText: '3x letter', bonusClass: '3LS' };
  }
  return { bonusText: '', bonusClass: '' };
}

type Props = {
  bonusType: BonusType;
  handleTileDrop: (e: any) => void;
  startingSquare: boolean;
  arrow: boolean;
  arrowHoriz: boolean;
  clicked: () => void;
};

const BoardSpace = React.memo((props: Props) => {
  const { bonusText, bonusClass } = getBonusProperties(props.bonusType);
  let bonusLabel = null;
  let startingSquare = null;
  let arrow = null;
  // ✩✪✫
  if (props.startingSquare) {
    startingSquare = <StarOutlined className="center-square" />;
  } else if (bonusText !== '') {
    bonusLabel = <p className="bonus-label">{bonusText}</p>;
  }
  if (props.arrow) {
    if (props.arrowHoriz) {
      arrow = <ArrowRightOutlined />;
    } else {
      arrow = <ArrowDownOutlined />;
    }
  }

  const handleDropOver = (e: any) => {
    e.preventDefault();
    e.stopPropagation();
  };
  return (
    <div
      className={`board-space droppable ${
        props.arrow ? 'selected' : ''
      } bonus-${bonusClass ? bonusClass : 'none'}`}
      onClick={props.clicked}
      onDragOver={handleDropOver}
      onDrop={props.handleTileDrop}
    >
      {bonusLabel}
      {startingSquare}
      {arrow}
    </div>
  );
});

export default BoardSpace;
