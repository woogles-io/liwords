import React, { useRef, useEffect } from "react";
import { Tooltip } from "antd";
import { useDrop } from "react-dnd";
import { TILE_TYPE } from "./tile";
import { BonusType } from "../constants/board_layout";
import {
  ArrowRightOutlined,
  ArrowDownOutlined,
  StarOutlined,
} from "@ant-design/icons";
interface BonusProperties {
  bonusText: string;
  bonusClass: string;
}

function getBonusProperties(bt: BonusType): BonusProperties {
  switch (bt) {
    case BonusType.DoubleWord:
      return { bonusText: "2x word", bonusClass: "2WS" };
    case BonusType.TripleWord:
      return { bonusText: "3x word", bonusClass: "3WS" };
    case BonusType.QuadrupleWord:
      return { bonusText: "4x word", bonusClass: "4WS" };
    case BonusType.DoubleLetter:
      return { bonusText: "2x letter", bonusClass: "2LS" };
    case BonusType.TripleLetter:
      return { bonusText: "3x letter", bonusClass: "3LS" };
    case BonusType.QuadrupleLetter:
      return { bonusText: "4x letter", bonusClass: "4LS" };
  }
  return { bonusText: "", bonusClass: "" };
}

type Props = {
  bonusType: BonusType;
  handleTileDrop: (
    rackIndex: number | undefined,
    tileIndex: number | undefined,
  ) => void;
  startingSquare: boolean;
  arrow: boolean;
  arrowHoriz: boolean;
  clicked: () => void;
  // the analyzer lab's hooks for the square, a line each way, shown on hover
  hookLines?: Array<string>;
};

const BoardSpace = React.memo((props: Props) => {
  const { bonusText, bonusClass } = getBonusProperties(props.bonusType);
  let bonusLabel = null;
  let startingSquare = null;
  let arrow = null;
  // ✩✪✫
  if (props.startingSquare) {
    startingSquare = <StarOutlined className="center-square" />;
  } else if (bonusText !== "") {
    bonusLabel = <p className="bonus-label">{bonusText}</p>;
  }
  if (props.arrow) {
    if (props.arrowHoriz) {
      arrow = <ArrowRightOutlined />;
    } else {
      arrow = <ArrowDownOutlined />;
    }
  }

  const spaceRef = useRef(null);

  const [, drop] = useDrop({
    accept: TILE_TYPE,
    drop: (item: { rackIndex: string; tileIndex: string }) => {
      props.handleTileDrop(
        item.rackIndex ? parseInt(item.rackIndex, 10) : undefined,
        item.tileIndex ? parseInt(item.tileIndex, 10) : undefined,
      );
    },
    collect: (monitor) => ({
      isOver: !!monitor.isOver(),
      canDrop: !!monitor.canDrop(),
    }),
  });

  useEffect(() => {
    drop(spaceRef);
  }, [drop]);

  const square = (
    <div
      ref={spaceRef}
      className={`board-space droppable ${
        props.arrow ? "selected" : ""
      } bonus-${bonusClass ? bonusClass : "none"}`}
      onClick={props.clicked}
    >
      {bonusLabel}
      {startingSquare}
      {arrow}
    </div>
  );
  return props.hookLines ? (
    <Tooltip
      title={props.hookLines.map((line) => (
        <div key={line}>{line}</div>
      ))}
    >
      {square}
    </Tooltip>
  ) : (
    square
  );
});

export default BoardSpace;
