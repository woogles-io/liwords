import React from 'react';
import { BonusType } from '../constants/board_layout';
import {
  ArrowRightOutlined,
  ArrowDownOutlined,
  StarOutlined,
} from '@ant-design/icons/lib';

interface BonusProperties {
  bonusText: string;
}

function getBonusProperties(bt: BonusType): BonusProperties {
  switch (bt) {
    case BonusType.DoubleWord:
      return { bonusText: '2WS' };
    case BonusType.TripleWord:
      return { bonusText: '3WS' };
    case BonusType.DoubleLetter:
      return { bonusText: '2LS' };
    case BonusType.TripleLetter:
      return { bonusText: '3LS' };
  }
  return { bonusText: '' };
}

type Props = {
  bonusType: BonusType;
  handleTileDrop: (e: any) => void;
  showBonusLabel: boolean;
  startingSquare: boolean;
  arrow: boolean;
  arrowHoriz: boolean;
  clicked: () => void;
};

const BoardSpace = React.memo((props: Props) => {
  const { bonusText } = getBonusProperties(props.bonusType);
  let bonusLabel = null;
  let startingSquare = null;
  let arrow = null;
  if (props.showBonusLabel && bonusText !== '') {
    bonusLabel = <p className="bonus-label">{bonusText}</p>;
  }
  // ✩✪✫
  if (props.startingSquare) {
    startingSquare = <StarOutlined className="center-square" />;
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
      } bonus-${bonusText ? bonusText : 'none'}`}
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
