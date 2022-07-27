import React from 'react';

import BoardSpaces from './board_spaces';
import { useDrawing } from './drawing';
import { useExamineStoreContext } from '../store/store';
import { PlacementArrow } from '../utils/cwgame/tile_placement';
import BoardCoordLabels from './board_coord_labels';
import Tiles from './tiles';
import {
  EphemeralTile,
  PlayedTiles,
  PlayerOfTiles,
} from '../utils/cwgame/common';
import { Alphabet } from '../constants/alphabets';
import { LearnOverlay } from '../learn/learn_overlay';

type Props = {
  tileColorId: number;
  gridLayout: Array<string>;
  gridSize: number;
  handleBoardTileClick: (rune: string) => void;
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
  tilesLayout: string;
  alphabet: Alphabet;
  lastPlayedTiles: PlayedTiles;
  playerOfTileAt: PlayerOfTiles;
  currentRack: string;
  squareClicked: (row: number, col: number) => void;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
  placementArrowProperties: PlacementArrow;
  handleSetHover?: (
    x: number,
    y: number,
    words: Array<string> | undefined
  ) => void;
  handleUnsetHover?: () => void;
  definitionPopover?:
    | { x: number; y: number; content: React.ReactNode }
    | undefined;
  recallOneTile?: (row: number, col: number) => void;
};

const Board = React.memo((props: Props) => {
  // Keep frames the same size, and shrink or grow the
  // board squares as necessary.

  const { outerDivProps, svgDrawing } = useDrawing();
  const { isExamining } = useExamineStoreContext();

  return (
    <div className="board">
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
