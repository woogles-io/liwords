import React, { useMemo, useRef, DragEvent, useState } from "react";
import TentativeScore from "./tentative_score";
import {
  Blank,
  EmptyRackSpaceMachineLetter,
  MachineLetter,
  isDesignatedBlankMachineLetter,
  isTouchDevice,
  uniqueTileIdx,
} from "../utils/cwgame/common";
import { Popover } from "antd";
import { Alphabet, machineLetterToDisplayedTile } from "../constants/alphabets";

// just refresh the page when changing the setting...
const bicolorMode = localStorage.getItem("enableBicolorMode") === "true";

type TileLetterProps = {
  letter: MachineLetter;
  alphabet: Alphabet;
};

export const TILE_TYPE = "TILE_TYPE";

export const TileLetter = React.memo((props: TileLetterProps) => {
  const { letter, alphabet } = props;
  let rune = machineLetterToDisplayedTile(letter, alphabet);
  // For display purposes, an empty blank should just look empty and not like a `?`.
  if (rune === Blank) {
    rune = " ";
  }
  return (
    <p className="rune">
      <span>{rune}</span>
    </p>
  );
});

type PointValueProps = {
  value: number;
};

export const PointValue = React.memo((props: PointValueProps) => {
  if (!props.value) {
    return null;
  }
  return <p className="point-value">{props.value}</p>;
});

type TileProps = {
  lastPlayed: boolean;
  playerOfTile: number;
  alphabet: Alphabet;
  letter: MachineLetter;
  value: number;
  tentative?: boolean;
  tentativeScore?: number;
  tentativeScoreIsHorizontal?: boolean | undefined;
  grabbable: boolean;
  rackIndex?: number | undefined;
  selected?: boolean;
  moveRackTile?: (
    indexA: number | undefined,
    indexB: number | undefined,
  ) => void;
  returnToRack?: (
    rackIndex: number | undefined,
    tileIndex: number | undefined,
  ) => void;
  onClick?: (evt: React.MouseEvent<HTMLElement>) => void;
  onContextMenu?: (evt: React.MouseEvent<HTMLElement>) => void;
  onMouseEnter?: (evt: React.MouseEvent<HTMLElement>) => void;
  onMouseLeave?: (evt: React.MouseEvent<HTMLElement>) => void;
  x?: number | undefined;
  y?: number | undefined;
  popoverContent?: React.ReactNode;
  onPopoverClick?: (evt: React.MouseEvent<HTMLElement>) => void;
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined,
  ) => void;
};

const Tile = React.memo((props: TileProps) => {
  const rune = useMemo(
    () => machineLetterToDisplayedTile(props.letter, props.alphabet),
    [props.letter, props.alphabet],
  );
  const bnjyable = useMemo(() => {
    return props.alphabet.letterMap[rune.toUpperCase()]?.bnjyable;
  }, [props.alphabet, rune]);

  const [isMouseDragging, setIsMouseDragging] = useState(false);

  const handleStartDrag = (e: DragEvent<HTMLDivElement>) => {
    setIsMouseDragging(true);
    e.dataTransfer.dropEffect = "move";
    if (
      props.tentative &&
      typeof props.x == "number" &&
      typeof props.y == "number"
    ) {
      e.dataTransfer.setData(
        "tileIndex",
        uniqueTileIdx(props.y, props.x).toString(),
      );
    } else {
      e.dataTransfer.setData("rackIndex", props.rackIndex?.toString() || "");
    }
  };

  const handleEndDrag = () => {
    setIsMouseDragging(false);
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    if (props.handleTileDrop && props.y != null && props.x != null) {
      props.handleTileDrop(
        props.y,
        props.x,
        parseInt(e.dataTransfer.getData("rackIndex"), 10),
        parseInt(e.dataTransfer.getData("tileIndex"), 10),
      );
      return;
    }
    if (props.moveRackTile && e.dataTransfer.getData("rackIndex")) {
      props.moveRackTile(
        props.rackIndex,
        parseInt(e.dataTransfer.getData("rackIndex"), 10),
      );
    } else {
      if (props.returnToRack && e.dataTransfer.getData("tileIndex")) {
        props.returnToRack(
          props.rackIndex,
          parseInt(e.dataTransfer.getData("tileIndex"), 10),
        );
      }
    }
  };

  const handleDropOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const canDrag =
    props.grabbable && props.letter !== EmptyRackSpaceMachineLetter;

  const tileRef = useRef(null);

  const computedClassName = `tile${
    isMouseDragging ? " dragging" : ""
  }${canDrag ? " droppable" : ""}${props.selected ? " selected" : ""}${
    props.tentative ? " tentative" : ""
  }${props.lastPlayed ? " last-played" : ""}${
    isDesignatedBlankMachineLetter(props.letter) ? " blank" : ""
  }${props.playerOfTile ? " tile-p1" : " tile-p0"}${
    (bicolorMode ? props.playerOfTile : props.lastPlayed) ? " second-color" : ""
  }`;
  let ret = (
    <div onDragOver={handleDropOver} onDrop={handleDrop} ref={tileRef}>
      <div
        className={computedClassName}
        data-letter={props.letter}
        data-length={rune.length}
        data-bnjy={bnjyable ? "1" : "0"}
        style={{
          cursor: canDrag ? "grab" : "default",
          ...(props.letter === EmptyRackSpaceMachineLetter
            ? { visibility: "hidden" }
            : null),
        }}
        onClick={props.onClick}
        onContextMenu={props.onContextMenu}
        onMouseEnter={props.onMouseEnter}
        onMouseLeave={props.onMouseLeave}
        onDragStart={canDrag ? handleStartDrag : undefined}
        onDragEnd={handleEndDrag}
        draggable={canDrag}
      >
        {props.letter !== EmptyRackSpaceMachineLetter && (
          <React.Fragment>
            <TileLetter letter={props.letter} alphabet={props.alphabet} />
            <PointValue value={props.value} />
          </React.Fragment>
        )}
        <TentativeScore
          score={props.tentativeScore}
          horizontal={props.tentativeScoreIsHorizontal}
        />
      </div>
    </div>
  );
  ret = (
    <Popover
      content={<div onClick={props.onPopoverClick}>{props.popoverContent}</div>}
      open={props.popoverContent != null}
    >
      {ret}
    </Popover>
  );
  return ret;
});

export default Tile;
