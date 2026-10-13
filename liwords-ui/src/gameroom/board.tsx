import React, { useEffect, useLayoutEffect, useRef, useState } from "react";

import BoardSpaces from "./board_spaces";
import { useDrawing } from "./drawing";
import { useExamineStoreContext } from "../store/store";
import { useLocalStorageBool } from "../utils/use_local_storage";
import { PlacementArrow } from "../utils/cwgame/tile_placement";
import BoardCoordLabels from "./board_coord_labels";
import Tiles from "./tiles";
import {
  EphemeralTile,
  MachineLetter,
  PlayedTiles,
  PlayerOfTiles,
} from "../utils/cwgame/common";
import { Alphabet } from "../constants/alphabets";
import { LearnOverlay } from "../learn/learn_overlay";
import { SuperCrosswordGameGridLayout } from "../constants/board_layout";

type Props = {
  tileColorId: number;
  gridLayout: Array<string>;
  gridSize: number;
  handleBoardTileClick: (letter: MachineLetter) => void;
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined,
  ) => void;
  tilesLayout: Array<MachineLetter>;
  alphabet: Alphabet;
  lastPlayedTiles: PlayedTiles;
  playerOfTileAt: PlayerOfTiles;
  squareClicked: (row: number, col: number) => void;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
  placementArrowProperties: PlacementArrow;
  handleSetHover?: (
    x: number,
    y: number,
    words: Array<string> | undefined,
  ) => void;
  handleUnsetHover?: () => void;
  definitionPopover?:
    | { x: number; y: number; content: React.ReactNode }
    | undefined;
  recallOneTile?: (row: number, col: number) => void;
  // Rotate the board 180 degrees, as if turned to face the opponent.
  flipped?: boolean;
  // Spin into a new orientation rather than jump to it.
  animateFlip?: boolean;
};

const Board = React.memo((props: Props) => {
  // Keep frames the same size, and shrink or grow the
  // board squares as necessary.

  const { outerDivProps, svgDrawing } = useDrawing(props.gridSize);
  const { isExamining } = useExamineStoreContext();

  // Secret setting: when enabled, premium-square labels (2x letter, 3x word,
  // etc.) are always shown instead of only on hover. The gameroom SCSS keys
  // off the "show-bonus-labels" body class.
  const [showBonusLabels] = useLocalStorageBool("showBonusLabels");
  useEffect(() => {
    if (!showBonusLabels) return;
    document.body.classList.add("show-bonus-labels");
    return () => {
      document.body.classList.remove("show-bonus-labels");
    };
  }, [showBonusLabels]);

  let zomgClass = "";
  if (props.gridSize === SuperCrosswordGameGridLayout.length) {
    zomgClass = " zomgboard";
  }
  const flipped = !!props.flipped;
  const flippedClass = flipped ? " board-flipped" : "";

  // Each flip plays one fixed clockwise half-turn: 0 to 180 degrees going
  // upside down, 180 to 360 coming back, so nothing ever winds up. A layout
  // effect, so the new resting angle is never painted before the spin starts.
  const [spinClass, setSpinClass] = useState("");
  const lastFlippedRef = useRef(flipped);
  useLayoutEffect(() => {
    if (lastFlippedRef.current === flipped) return;
    lastFlippedRef.current = flipped;
    if (!props.animateFlip) {
      setSpinClass("");
      return;
    }
    setSpinClass(flipped ? " board-spin-down" : " board-spin-up");
  }, [flipped, props.animateFlip]);

  return (
    <div className={`board${zomgClass}${flippedClass}${spinClass}`}>
      <BoardCoordLabels gridDim={props.gridSize} />
      <div
        className="board-spaces-container"
        id="board"
        onMouseLeave={props.handleUnsetHover}
        {...outerDivProps}
      >
        <BoardSpaces
          gridDim={props.gridSize}
          gridLayout={props.gridLayout}
          handleTileDrop={props.handleTileDrop}
          placementArrow={props.placementArrowProperties}
          squareClicked={props.squareClicked}
        />
        {!isExamining && <LearnOverlay gridDim={props.gridSize} />}
        <Tiles
          tileColorId={props.tileColorId}
          gridDim={props.gridSize}
          onClick={props.handleBoardTileClick}
          tilesLayout={props.tilesLayout}
          lastPlayedTiles={props.lastPlayedTiles}
          playerOfTileAt={props.playerOfTileAt}
          tentativeTiles={props.tentativeTiles}
          placementArrow={props.placementArrowProperties}
          tentativeTileScore={props.tentativeTileScore}
          handleSetHover={props.handleSetHover}
          handleUnsetHover={props.handleUnsetHover}
          definitionPopover={props.definitionPopover}
          handleTileDrop={props.handleTileDrop}
          alphabet={props.alphabet}
          recallOneTile={props.recallOneTile}
        />
        {svgDrawing}
      </div>
    </div>
  );
});

export default Board;
