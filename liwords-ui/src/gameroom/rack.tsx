import React, { DragEvent, useRef } from "react";
import Tile, { TILE_TYPE } from "./tile";
import { MachineWord } from "../utils/cwgame/common";
import { Alphabet, scoreFor } from "../constants/alphabets";

type XYCoord = { x: number; y: number };
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
  const handleDropOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  };
  const handleDrop = (e: DragEvent<HTMLDivElement>, index: number) => {
    if (e.dataTransfer.getData("rackIndex")) {
      props.moveRackTile(
        index,
        parseInt(e.dataTransfer.getData("rackIndex"), 10),
      );
    } else if (props.returnToRack && e.dataTransfer.getData("tileIndex")) {
      props.returnToRack(
        index,
        parseInt(e.dataTransfer.getData("tileIndex"), 10),
      );
    }
  };
  const rackRef = useRef(null);

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
      <div
        className="empty-rack droppable"
        id="left-empty"
        onDragOver={handleDropOver}
        onDrop={(e) => {
          handleDrop(e, 0);
        }}
      />
      {renderTiles()}
      <div
        className="empty-rack droppable"
        onDragOver={handleDropOver}
        onDrop={(e) => {
          handleDrop(e, props.letters.length);
        }}
      />
    </div>
  );
});

export default Rack;
