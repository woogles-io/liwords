import React, { useEffect, useMemo, useRef } from "react";
import { useDrag, useDragLayer, useDrop } from "react-dnd";
import TentativeScore from "./tentative_score";
import {
  Blank,
  EmptyRackSpaceMachineLetter,
  MachineLetter,
  isDesignatedBlankMachineLetter,
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

type TilePreviewProps = {
  gridDim: number;
};

export const TilePreview = React.memo((props: TilePreviewProps) => {
  const {
    isDragging,
    xyPosition: position,
    letter,
    alphabet,
    value,
    playerOfTile,
  } = useDragLayer((monitor) => ({
    xyPosition: monitor.getClientOffset(),
    initialPosition: monitor.getInitialClientOffset(),
    isDragging: monitor.isDragging(),
    letter: monitor.getItem()?.letter as MachineLetter,
    alphabet: monitor.getItem()?.alphabet as Alphabet,
    value: monitor.getItem()?.value,
    playerOfTile: monitor.getItem()?.playerOfTile,
  }));
  const boardElement = document.getElementById("board-spaces");
  if (isDragging && boardElement && position) {
    const boardTop = boardElement.getBoundingClientRect().top;
    const boardLeft = boardElement.getBoundingClientRect().left;
    const boardWidth = boardElement.getBoundingClientRect().width;
    let top = position.y - boardTop;
    let left = position.x - boardLeft;
    const overBoard =
      boardWidth > position?.y - boardTop &&
      position?.y > boardTop &&
      boardWidth > position?.x - boardLeft &&
      position?.x > boardLeft;
    const tileSize = boardWidth / props.gridDim;
    if (overBoard) {
      const col = Math.floor((position.x - boardLeft) / tileSize);
      const row = Math.floor((position.y - boardTop) / tileSize);
      left = col * tileSize + 6;
      top = row * tileSize + 23;
    }
    const computedStyle = {
      top,
      left,
    };
    const computedClass = `tile preview${overBoard ? " over-board" : ""}${
      letter && isDesignatedBlankMachineLetter(letter) ? " blank" : ""
    }${playerOfTile ? " tile-p1" : " tile-p0"}`;
    return (
      <div className={computedClass} style={computedStyle}>
        <TileLetter letter={letter} alphabet={alphabet} />
        <PointValue value={value} />
      </div>
    );
  }

  return null;
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

  const canDrag =
    props.grabbable && props.letter !== EmptyRackSpaceMachineLetter;
  const [{ isDragging }, drag, preview] = useDrag({
    type: TILE_TYPE,
    item: {
      rackIndex:
        typeof props.rackIndex === "number"
          ? props.rackIndex.toString()
          : undefined,
      tileIndex:
        typeof props.x === "number" && typeof props.y === "number"
          ? uniqueTileIdx(props.y, props.x).toString()
          : undefined,
      letter: props.letter,
      alphabet: props.alphabet,
      value: props.value,
      playerOfTile: props.playerOfTile,
    },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
    canDrag: (monitor) => canDrag,
  });

  useEffect(() => {
    preview(<div></div>);
  }, [preview]);

  const [, drop] = useDrop({
    accept: TILE_TYPE,
    drop: (item: { rackIndex: string; tileIndex: string }) => {
      if (props.handleTileDrop && props.y != null && props.x != null) {
        props.handleTileDrop(
          props.y,
          props.x,
          parseInt(item.rackIndex, 10),
          parseInt(item.tileIndex, 10),
        );
      }
    },
    collect: (monitor) => ({
      isOver: !!monitor.isOver(),
      canDrop: !!monitor.canDrop(),
    }),
  });

  const computedClassName = `tile${
    isDragging ? " dragging" : ""
  }${canDrag ? " droppable" : ""}${props.selected ? " selected" : ""}${
    props.tentative ? " tentative" : ""
  }${props.lastPlayed ? " last-played" : ""}${
    isDesignatedBlankMachineLetter(props.letter) ? " blank" : ""
  }${props.playerOfTile ? " tile-p1" : " tile-p0"}${
    (bicolorMode ? props.playerOfTile : props.lastPlayed) ? " second-color" : ""
  }`;
  let ret = (
    <div ref={(node) => drag(drop(node))}>
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
