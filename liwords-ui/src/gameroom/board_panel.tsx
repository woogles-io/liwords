import React, { useState, useEffect } from 'react';
import { Button, Row, Col } from 'antd';
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

// The frame atop is 24 height
// The frames on the sides are 24 in width, surrounded by a 14 pix gutter
const EnterKey = 'Enter';
const sideFrameWidth = 24;
const topFrameHeight = 24;
const sideFrameGutter = 14;
// XXX: Later make the 15 customizable if we want to add other sizes.
const gridSize = 15;

type Props = {
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

  // Need to sync state to props here whenever the props.currentRack changes.
  // We want to take back all the tiles also if the board changes.
  useEffect(() => {
    setDisplayedRack(props.currentRack);
    setPlacedTiles(new Set<EphemeralTile>());
    setPlacedTilesTempScore(0);
    setArrowProperties({ row: 0, col: 0, horizontal: false, show: false });
  }, [props.currentRack, props.board.letters]);

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
      commitPlay();
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

  const exchangeTiles = (rack: string) => {
    console.log('exchange ', rack);
    const moveEvt = exchangeMoveEvent(rack, props.gameID);
    props.sendSocketMsg(
      encodeToSocketFmt(
        MessageType.CLIENT_GAMEPLAY_EVENT,
        moveEvt.serializeBinary()
      )
    );
  };

  const passTurn = () => {
    props.sendSocketMsg(
      encodeToSocketFmt(
        MessageType.CLIENT_GAMEPLAY_EVENT,
        passMoveEvent(props.gameID).serializeBinary()
      )
    );

    console.log('pass turn');
  };

  const challengePlay = () => {
    encodeToSocketFmt(
      MessageType.CLIENT_GAMEPLAY_EVENT,
      challengeMoveEvent(props.gameID).serializeBinary()
    );
  };

  const commitPlay = () => {
    const moveEvt = tilesetToMoveEvent(placedTiles, props.board, props.gameID);
    if (moveEvt === null) {
      // Just return. This is an invalid play.
      return;
    }
    props.sendSocketMsg(
      encodeToSocketFmt(
        MessageType.CLIENT_GAMEPLAY_EVENT,
        moveEvt.serializeBinary()
      )
    );
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
      <div style={{ marginTop: 30 }}>
        <Row>
          <Col span={2} offset={4}>
            <Button
              shape="circle"
              icon={<ArrowDownOutlined />}
              type="primary"
              onClick={recallTiles}
            />
          </Col>
          <Col span={12}>
            <Rack letters={displayedRack} tileDim={sqWidth} grabbable />
          </Col>
          <Col span={2}>
            <Button
              shape="circle"
              icon={<SyncOutlined />}
              type="primary"
              onClick={shuffleTiles}
            />
          </Col>
        </Row>
      </div>
      <div style={{ marginTop: 30 }}>
        <GameControls
          onExchange={exchangeTiles}
          onPass={passTurn}
          onChallenge={challengePlay}
          onCommit={commitPlay}
          currentRack={props.currentRack}
        />
      </div>
    </div>
  );
};
