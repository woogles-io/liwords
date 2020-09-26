import React, { useState, useEffect } from 'react';
import { Button, Modal, notification, message, Tooltip } from 'antd';
import { SyncOutlined } from '@ant-design/icons';
import axios from 'axios';

import GameBoard from './board';
import GameControls from './game_controls';
import { Rack } from './rack';
import {
  nextArrowPropertyState,
  handleKeyPress,
  handleDroppedTile,
  returnTileToRack,
  designateBlank,
} from '../utils/cwgame/tile_placement';
import { uniqueTileIdx } from '../utils/cwgame/common';
import { EphemeralTile, EmptySpace } from '../utils/cwgame/common';
import {
  tilesetToMoveEvent,
  exchangeMoveEvent,
  passMoveEvent,
  resignMoveEvent,
  challengeMoveEvent,
} from '../utils/cwgame/game_event';
import { Board } from '../utils/cwgame/board';
import { encodeToSocketFmt } from '../utils/protobuf';
import {
  MessageType,
  MatchRequest,
  MatchUser,
} from '../gen/api/proto/realtime/realtime_pb';
import { useStoreContext } from '../store/store';
import { BlankSelector } from './blank_selector';
import { GameEndMessage } from './game_end_message';
import { PlayerMetadata, GCGResponse } from './game_info';
import { GameEvent } from '../gen/macondo/api/proto/macondo/macondo_pb';

// The frame atop is 24 height
// The frames on the sides are 24 in width, surrounded by a 14 pix gutter
const EnterKey = 'Enter';
const colors = require('../base.scss');

type Props = {
  username: string;
  currentRack: string;
  events: Array<GameEvent>;
  gameID: string;
  board: Board;
  sendSocketMsg: (msg: Uint8Array) => void;
  gameDone: boolean;
  playerMeta: Array<PlayerMetadata>;
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

const gcgExport = (gameID: string) => {
  axios
    .post<GCGResponse>('/twirp/game_service.GameMetadataService/GetGCG', {
      gameId: gameID,
    })
    .then((resp) => {
      const url = window.URL.createObjectURL(new Blob([resp.data.gcg]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `${gameID}.gcg`);
      document.body.appendChild(link);
      link.click();
    })
    .catch((e) => {
      if (e.response) {
        // From Twirp
        notification.warning({
          message: 'Export Error',
          description: e.response.data.msg,
          duration: 4,
        });
      } else {
        console.log(e);
      }
    });
};

export const BoardPanel = React.memo((props: Props) => {
  const [arrowProperties, setArrowProperties] = useState({
    row: 0,
    col: 0,
    horizontal: false,
    show: false,
  });

  const [displayedRack, setDisplayedRack] = useState(props.currentRack);
  const [placedTiles, setPlacedTiles] = useState(new Set<EphemeralTile>());
  const [placedTilesTempScore, setPlacedTilesTempScore] = useState<number>();
  const [blankModalVisible, setBlankModalVisible] = useState(false);
  const { stopClock, gameContext, gameEndMessage } = useStoreContext();

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

  useEffect(() => {
    if (!props.events.length) {
      return;
    }
    const evt = props.events[props.events.length - 1];
    if (evt.getNickname() === props.username) {
      return;
    }
    let boardMessage = null;
    switch (evt.getType()) {
      case GameEvent.Type.PASS:
        boardMessage = `${evt.getNickname()} passed`;
        break;
      case GameEvent.Type.EXCHANGE:
        boardMessage = `${evt.getNickname()} exchanged ${evt.getExchanged()}`;
        break;
    }
    if (boardMessage) {
      message.info(
        {
          content: boardMessage,
          className: 'board-hud-message',
        },
        3,
        undefined
      );
    }
  }, [props.events, props.username]);
  const squareClicked = (row: number, col: number) => {
    if (props.board.letterAt(row, col) !== EmptySpace) {
      // If there is a tile on this square, ignore the click.
      return;
    }
    setArrowProperties(nextArrowPropertyState(arrowProperties, row, col));
  };

  const keydown = (key: string) => {
    // This should return a new set of arrow properties, and also set
    // some state further up (the tiles layout with a "just played" type
    // marker)
    if (key === 'ArrowDown') {
      recallTiles();
      return;
    }
    if (key === 'ArrowUp') {
      shuffleTiles();
      return;
    }
    if (key === EnterKey) {
      makeMove('commit');
      return;
    }

    if (!arrowProperties.show) {
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

  const handleTileDrop = (
    row: number,
    col: number,
    rackIndex: number = -1,
    tileIndex: number = -1
  ) => {
    const handlerReturn = handleDroppedTile(
      row,
      col,
      props.board,
      displayedRack,
      placedTiles,
      rackIndex,
      tileIndex
    );
    if (handlerReturn === null) {
      return;
    }
    setDisplayedRack(handlerReturn.newDisplayedRack);
    setPlacedTiles(handlerReturn.newPlacedTiles);
    setPlacedTilesTempScore(handlerReturn.playScore);
    setArrowProperties({ row: 0, col: 0, horizontal: false, show: false });
    if (handlerReturn.isUndesignated) {
      setBlankModalVisible(true);
    }
  };

  const clickToBoard = (rackIndex: number) => {
    if (!arrowProperties.show) {
      return null;
    }
    const handlerReturn = handleDroppedTile(
      arrowProperties.row,
      arrowProperties.col,
      props.board,
      displayedRack,
      placedTiles,
      rackIndex,
      uniqueTileIdx(arrowProperties.row, arrowProperties.col)
    );
    if (handlerReturn === null) {
      return;
    }
    setDisplayedRack(handlerReturn.newDisplayedRack);
    setPlacedTiles(handlerReturn.newPlacedTiles);
    setPlacedTilesTempScore(handlerReturn.playScore);
    if (handlerReturn.isUndesignated) {
      setBlankModalVisible(true);
    }
    let newrow = arrowProperties.row;
    let newcol = arrowProperties.col;

    if (arrowProperties.horizontal) {
      do {
        newcol += 1;
      } while (
        newcol < props.board.dim &&
        newcol >= 0 &&
        props.board.letterAt(newrow, newcol) !== EmptySpace
      );
    } else {
      do {
        newrow += 1;
      } while (
        newrow < props.board.dim &&
        newrow >= 0 &&
        props.board.letterAt(newrow, newcol) !== EmptySpace
      );
    }
    setArrowProperties({
      col: newcol,
      horizontal: arrowProperties.horizontal,
      show: !(newcol === props.board.dim || newrow === props.board.dim),
      row: newrow,
    });
  };
  const handleBlankSelection = (rune: string) => {
    const handlerReturn = designateBlank(
      props.board,
      placedTiles,
      displayedRack,
      rune
    );
    if (handlerReturn === null) {
      return;
    }
    setBlankModalVisible(false);
    setPlacedTiles(handlerReturn.newPlacedTiles);
    setPlacedTilesTempScore(handlerReturn.playScore);
  };

  const handleBlankModalCancel = () => {
    setBlankModalVisible(false);
  };

  const returnToRack = (
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => {
    const handlerReturn = returnTileToRack(
      props.board,
      displayedRack,
      placedTiles,
      rackIndex,
      tileIndex
    );
    if (handlerReturn === null) {
      return;
    }
    setDisplayedRack(handlerReturn.newDisplayedRack);
    setPlacedTiles(handlerReturn.newPlacedTiles);
    setPlacedTilesTempScore(handlerReturn.playScore);
    setArrowProperties({ row: 0, col: 0, horizontal: false, show: false });
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

  const moveRackTile = (
    newIndex: number | undefined,
    oldIndex: number | undefined
  ) => {
    if (typeof newIndex === 'number' && typeof oldIndex === 'number') {
      const newRack = displayedRack.split('');
      newRack.splice(oldIndex, 1);
      newRack.splice(newIndex, 0, displayedRack[oldIndex]);
      setPlacedTilesTempScore(0);
      setDisplayedRack(newRack.join(''));
    }
  };

  const makeMove = (move: string, addl?: string) => {
    let moveEvt;
    console.log(
      'making move',
      gameContext.nickToPlayerOrder,
      props.username,
      gameContext.onturn
    );
    const iam = gameContext.nickToPlayerOrder[props.username];
    if (!(iam && iam === `p${gameContext.onturn}`)) {
      // It is not my turn. Ignore this event.
      notification.warning({
        key: 'notyourturn',
        message: 'Attention',
        description: 'It is not your turn.',
        duration: 1.5,
      });
      return;
    }

    switch (move) {
      case 'exchange':
        moveEvt = exchangeMoveEvent(addl!, props.gameID);
        break;
      case 'pass':
        moveEvt = passMoveEvent(props.gameID);
        break;
      case 'resign':
        moveEvt = resignMoveEvent(props.gameID);
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

  const observer = !props.playerMeta.some((p) => p.nickname === props.username);
  const rematch = () => {
    const evt = new MatchRequest();
    const receiver = new MatchUser();

    let opp = '';
    props.playerMeta.forEach((p) => {
      if (!(p.nickname === props.username)) {
        opp = p.nickname;
      }
    });

    if (observer) {
      return;
    }

    receiver.setDisplayName(opp);
    evt.setReceivingUser(receiver);
    evt.setRematchFor(props.gameID);
    props.sendSocketMsg(
      encodeToSocketFmt(MessageType.MATCH_REQUEST, evt.serializeBinary())
    );
    notification.info({
      message: 'Rematch',
      description: `Sent rematch request to ${opp}`,
    });
    console.log('rematching');
  };

  return (
    <div
      id="board-container"
      className="board-container"
      onKeyDown={(e) => {
        keydown(e.key);
      }}
      tabIndex={-1}
      role="textbox"
    >
      <GameBoard
        gridSize={props.board.dim}
        gridLayout={props.board.gridLayout}
        handleTileDrop={handleTileDrop}
        tilesLayout={props.board.letters}
        lastPlayedTiles={gameContext.lastPlayedTiles}
        tentativeTiles={placedTiles}
        tentativeTileScore={placedTilesTempScore}
        currentRack={props.currentRack}
        squareClicked={squareClicked}
        placementArrowProperties={arrowProperties}
      />
      {!gameEndMessage ? (
        <div className="rack-container">
          <Tooltip
            title="Reset Rack"
            placement="bottomRight"
            mouseEnterDelay={0.1}
            mouseLeaveDelay={0.01}
            color={colors.colorPrimary}
          >
            <Button shape="circle" type="primary" onClick={recallTiles}>
              &#8595;
            </Button>
          </Tooltip>
          <Rack
            letters={displayedRack}
            grabbable
            returnToRack={returnToRack}
            onTileClick={clickToBoard}
            moveRackTile={(
              indexA: number | undefined,
              indexB: number | undefined
            ) => {
              moveRackTile(indexA, indexB);
            }}
          />
          <Tooltip
            title="Shuffle"
            placement="bottomLeft"
            mouseEnterDelay={0.1}
            mouseLeaveDelay={0.01}
            color={colors.colorPrimary}
          >
            <Button
              shape="circle"
              icon={<SyncOutlined />}
              type="primary"
              onClick={shuffleTiles}
              autoFocus={true}
            />
          </Tooltip>
        </div>
      ) : (
        <GameEndMessage message={gameEndMessage} />
      )}
      <GameControls
        observer={observer}
        onRecall={recallTiles}
        onExchange={(rack: string) => makeMove('exchange', rack)}
        onPass={() => makeMove('pass')}
        onResign={() => makeMove('resign')}
        onChallenge={() => makeMove('challenge')}
        onCommit={() => makeMove('commit')}
        onRematch={rematch}
        onExamine={() => gcgExport(props.gameID)}
        showRematch={gameEndMessage !== ''}
        gameEndControls={gameEndMessage !== '' || props.gameDone}
        currentRack={props.currentRack}
      />
      <Modal
        className="blank-modal"
        title="Designate your blank"
        visible={blankModalVisible}
        onCancel={handleBlankModalCancel}
        width={360}
        footer={null}
      >
        <BlankSelector handleSelection={handleBlankSelection} />
      </Modal>
    </div>
  );
});
