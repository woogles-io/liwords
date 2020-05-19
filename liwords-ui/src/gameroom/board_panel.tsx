import React, { useState } from 'react';

import Board from './board';
import GameControls from './game_controls';
import Rack from './rack';

import {
  nextArrowPropertyState,
  handleKeyPress,
} from '../utils/cwgame/tile_placement';
import { EphemeralTile } from '../utils/cwgame/common';

// The frame atop is 24 height
// The frames on the sides are 24 in width, surrounded by a 14 pix gutter

const sideFrameWidth = 24;
const topFrameHeight = 24;
const sideFrameGutter = 14;
// XXX: Later make the 15 customizable if we want to add other sizes.
const gridSize = 15;

type Props = {
  compWidth: number;
  compHeight: number;
  gridLayout: Array<string>;
  tilesLayout: Array<string>;
  showBonusLabels: boolean;
  lastPlayedLetters: Record<string, boolean>;
  currentRack: string;
};

export const BoardPanel = (props: Props) => {
  const [arrowProperties, setArrowProperties] = useState({
    row: 0,
    col: 0,
    horizontal: false,
    show: false,
  });

  const [displayedRack, setDisplayedRack] = useState(props.currentRack);
  const [placedTiles, setPlacedTiles] = useState(new Set<EphemeralTile>());
  const [placedTilesTempScore, setPlacedTilesTempScore] = useState<number>();

  const squareClicked = (row: number, col: number) => {
    if (props.tilesLayout[row][col] !== ' ') {
      // If there is a tile on this square, ignore the click.
      return;
    }
    setArrowProperties(nextArrowPropertyState(arrowProperties, row, col));
  };

  if (props.compWidth < 100) {
    return null;
  }

  const sideFrames = (sideFrameWidth + sideFrameGutter * 2) * 2;
  const boardDim = props.compWidth - sideFrames;
  const sqWidth = boardDim / gridSize;

  const keydown = (key: string) => {
    // This should return a new set of arrow properties, and also set
    // some state further up (the tiles layout with a "just played" type
    // marker)

    if (!arrowProperties.show) {
      return;
    }

    const handlerReturn = handleKeyPress(
      arrowProperties,
      props.tilesLayout,
      key,
      displayedRack,
      placedTiles,
      props.gridLayout
    );

    if (handlerReturn === null) {
      return;
    }
    setDisplayedRack(handlerReturn.newDisplayedRack);
    setArrowProperties(handlerReturn.newArrow);
    setPlacedTiles(handlerReturn.newPlacedTiles);
    setPlacedTilesTempScore(handlerReturn.playScore);
  };

  const recallTiles = () => {
    setPlacedTilesTempScore(0);
    setPlacedTiles(new Set<EphemeralTile>());
    setDisplayedRack(props.currentRack);
  };

  return (
    <div
      style={{
        width: props.compWidth,
        height: props.compHeight,
        background: 'linear-gradient(180deg, #E2F8FF 0%, #FFFFFF 100%)',
        boxShadow: '0px 0px 30px rgba(0, 0, 0, 0.1)',
        borderRadius: '4px',
        lineHeight: '0px',
        textAlign: 'center',
        outlineStyle: 'none',
      }}
      onKeyDown={(e) => {
        keydown(e.key);
      }}
      tabIndex={-1}
      role="textbox"
    >
      <Board
        compWidth={props.compWidth}
        boardDim={boardDim}
        topFrameHeight={topFrameHeight}
        sideFrameWidth={sideFrameWidth}
        sideFrameGutter={sideFrameGutter}
        sqWidth={sqWidth}
        gridSize={props.gridLayout[0].length}
        gridLayout={props.gridLayout}
        tilesLayout={props.tilesLayout}
        showBonusLabels={false}
        lastPlayedLetters={props.lastPlayedLetters}
        tentativeTiles={placedTiles}
        tentativeTileScore={placedTilesTempScore}
        currentRack={props.currentRack}
        squareClicked={squareClicked}
        placementArrowProperties={arrowProperties}
      />
      <div style={{ marginTop: 30 }}>
        <Rack letters={displayedRack} tileDim={sqWidth} />
      </div>
      <div style={{ marginTop: 30 }}>
        <GameControls onRecall={recallTiles} />
      </div>
    </div>
  );
};
