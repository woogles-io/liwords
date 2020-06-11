import React, { useState, useEffect } from 'react';
import { Button } from 'antd';
import { ArrowDownOutlined, SyncOutlined } from '@ant-design/icons';
import GameBoard from './board';
import GameControls from './game_controls';
import Rack from './rack';
import {
  nextArrowPropertyState,
  handleKeyPress,
} from '../utils/cwgame/tile_placement';

import { EphemeralTile, EmptySpace } from '../utils/cwgame/common';
import {
  tilesetToMoveEvent,
  exchangeMoveEvent,
  passMoveEvent,
  challengeMoveEvent,
} from '../utils/cwgame/game_event';
import { Board } from '../utils/cwgame/board';
import { encodeToSocketFmt } from '../utils/protobuf';
import { MessageType } from '../gen/api/proto/game_service_pb';
import { useStoreContext } from '../store/store';

// The frame atop is 24 height
// The frames on the sides are 24 in width, surrounded by a 14 pix gutter
const EnterKey = 'Enter';
const sideFrameWidth = 24;
const topFrameHeight = 24;
const sideFrameGutter = 14;
// XXX: Later make the 15 customizable if we want to add other sizes.
const gridSize = 15;

type Props = {
  username: string;
  compWidth: number;
  compHeight: number;
  showBonusLabels: boolean;
  lastPlayedLetters: { [tile: string]: boolean };
  currentRack: string;
  gameID: string;
  board: Board;
  sendSocketMsg: (msg: Uint8Array) => void;
};

const shuffleString = (a: string): string => {
  const alist = a.split('');
  const n = a.length;

  for (let i = n - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    const tmp = alist[i];
    alist[i] = alist[j];
    alist[j] = tmp;
  }
  return alist.join('');
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
  const { stopClock, gameContext } = useStoreContext();

  // Need to sync state to props here whenever the props.currentRack changes.
  // We want to take back all the tiles also if the board changes.
  useEffect(() => {
    setDisplayedRack(props.currentRack);
    setPlacedTiles(new Set<EphemeralTile>());
    setPlacedTilesTempScore(0);
    setArrowProperties({ row: 0, col: 0, horizontal: false, show: false });
  }, [props.currentRack, props.board.letters]);

  useEffect(() => {
    // Stop the clock if we unload the board panel.
    return () => {
      stopClock();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const squareClicked = (row: number, col: number) => {
    if (props.board.letterAt(row, col) !== EmptySpace) {
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

    if (key === EnterKey) {
      makeMove('commit');
      return;
    }

    const handlerReturn = handleKeyPress(
      arrowProperties,
      props.board,
      key,
      displayedRack,
      placedTiles
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

  const shuffleTiles = () => {
    setPlacedTilesTempScore(0);
    setPlacedTiles(new Set<EphemeralTile>());
    setDisplayedRack(shuffleString(props.currentRack));
  };

  const makeMove = (move: string, addl?: string) => {
    let moveEvt;
    const iam = gameContext.nickToPlayerOrder[props.username];
    if (!(iam && iam === `p${gameContext.onturn}`)) {
      // It is not my turn. Ignore this event.
      return;
    }

    switch (move) {
      case 'exchange':
        moveEvt = exchangeMoveEvent(addl!, props.gameID);
        break;
      case 'pass':
        moveEvt = passMoveEvent(props.gameID);
        break;
      case 'challenge':
        moveEvt = challengeMoveEvent(props.gameID);
        break;
      case 'commit':
        moveEvt = tilesetToMoveEvent(placedTiles, props.board, props.gameID);
        if (!moveEvt) {
          // this is an invalid play
          return;
        }
        break;
    }
    if (!moveEvt) {
      return;
    }
    props.sendSocketMsg(
      encodeToSocketFmt(
        MessageType.CLIENT_GAMEPLAY_EVENT,
        moveEvt.serializeBinary()
      )
    );
    // Don't stop the clock; the next user event to come in will change the
    // clock over.
    // stopClock();
  };

  return (
    <div
      className="board-container"
      onKeyDown={(e) => {
        keydown(e.key);
      }}
      tabIndex={-1}
      role="textbox"
    >
      <GameBoard
        compWidth={props.compWidth}
        boardDim={boardDim}
        topFrameHeight={topFrameHeight}
        sideFrameWidth={sideFrameWidth}
        sideFrameGutter={sideFrameGutter}
        sqWidth={sqWidth}
        gridSize={props.board.dim}
        gridLayout={props.board.gridLayout}
        tilesLayout={props.board.letters}
        showBonusLabels={false}
        lastPlayedLetters={props.lastPlayedLetters}
        tentativeTiles={placedTiles}
        tentativeTileScore={placedTilesTempScore}
        currentRack={props.currentRack}
        squareClicked={squareClicked}
        placementArrowProperties={arrowProperties}
      />
      <div className="rack-container">
        <Button
          shape="circle"
          icon={<ArrowDownOutlined />}
          type="primary"
          onClick={recallTiles}
        />
        <Rack letters={displayedRack} tileDim={sqWidth} grabbable />
        <Button
          shape="circle"
          icon={<SyncOutlined />}
          type="primary"
          onClick={shuffleTiles}
        />
      </div>
      <div>
        <GameControls
          onRecall={recallTiles}
          onExchange={(rack: string) => makeMove('exchange', rack)}
          onPass={() => makeMove('pass')}
          onChallenge={() => makeMove('challenge')}
          onCommit={() => makeMove('commit')}
          currentRack={props.currentRack}
        />
      </div>
    </div>
  );
};
