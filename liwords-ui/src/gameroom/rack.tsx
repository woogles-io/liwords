import React, { useEffect, useRef } from "react";
import { useDrop, XYCoord } from "react-dnd";
import Tile, { TILE_TYPE } from "./tile";
import { MachineWord } from "../utils/cwgame/common";
import { Alphabet, scoreFor } from "../constants/alphabets";

// const TileSpacing = 6;

const calculatePosition = (
  position: XYCoord,
  rackElement: HTMLElement,
  rackEmptyLeftElement: HTMLElement,
  rackSize: number,
) => {
  const rackLeft = rackElement.getBoundingClientRect().left;
  const rackWidth = rackElement.clientWidth;
  const rackEmptyWidth = rackEmptyLeftElement.clientWidth;
  const tileSize = (rackWidth - 2 * rackEmptyWidth) / rackSize;
  const relativePosition = position.x - rackLeft;
  if (relativePosition < rackEmptyWidth) return 0;
  if (relativePosition > rackWidth - rackEmptyWidth) return rackSize;
  return Math.floor((relativePosition - rackEmptyWidth) / tileSize);
};

type Props = {
  tileColorId: number;
  letters: MachineWord;
  grabbable: boolean;
  alphabet: Alphabet;
  onTileClick?: (idx: number) => void;
  selected?: Set<number>;
  moveRackTile: (
    indexA: number | undefined,
    indexB: number | undefined,
  ) => void;
  returnToRack?: (
    rackIndex: number | undefined,
    tileIndex: number | undefined,
  ) => void;
};

export const Rack = React.memo((props: Props) => {
  const [, drop] = useDrop(
    {
      accept: TILE_TYPE,
      drop: (item: { rackIndex: string; tileIndex: number }, monitor) => {
        const clientOffset = monitor.getClientOffset();
        const rackElement = document.getElementById("rack");
        const rackEmptyElement = document.getElementById("left-empty");
        let rackPosition = 0;
        console.log(
          "clientOffset",
          clientOffset,
          "item",
          item,
          monitor.getItemType(),
          monitor.getSourceClientOffset(),
          monitor.getInitialClientOffset(),
          monitor.getInitialSourceClientOffset(),
        );
        if (clientOffset && rackElement && rackEmptyElement) {
          rackPosition = calculatePosition(
            clientOffset,
            rackElement,
            rackEmptyElement,
            props.letters.length,
          );
        }
        if (item.rackIndex) {
          props.moveRackTile(rackPosition, parseInt(item.rackIndex, 10));
        }
        if (props.returnToRack && item.tileIndex) {
          props.returnToRack(rackPosition, item.tileIndex);
        }
      },
      hover: (item, monitor) => {
        const clientOffset = monitor.getClientOffset();
        console.log("hover clientOffset", clientOffset);
        // You can store this in state if needed for use in drop
      },
      collect: (monitor) => ({
        isOver: !!monitor.isOver(),
        canDrop: !!monitor.canDrop(),
      }),
    },
    [props.letters.length, props.moveRackTile, props.returnToRack],
  );
  const rackRef = useRef(null);
  useEffect(() => {
    drop(rackRef);
  }, [drop]);
  const renderTiles = () => {
    const tiles = [];
    if (props.letters.length === 0) {
      return null;
    }

    for (let n = 0; n < props.letters.length; n += 1) {
      const letter = props.letters[n];
      tiles.push(
        <Tile
          letter={letter}
          alphabet={props.alphabet}
          value={scoreFor(props.alphabet, letter)}
          lastPlayed={false}
          playerOfTile={props.tileColorId}
          key={`tile_${n}`}
          selected={props.selected && props.selected.has(n)}
          grabbable={props.grabbable}
          rackIndex={n}
          returnToRack={props.returnToRack}
          moveRackTile={props.moveRackTile}
          onClick={() => {
            if (props.onTileClick) {
              props.onTileClick(n);
            }
          }}
        />,
      );
    }
    return <>{tiles}</>;
  };

  return (
    <div className="rack" ref={rackRef} id="rack">
      <div className="empty-rack droppable" id="left-empty" />
      {renderTiles()}
      <div className="empty-rack droppable" />
    </div>
  );
});

export default Rack;
