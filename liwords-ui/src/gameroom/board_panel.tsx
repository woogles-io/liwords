import React, { useCallback, useState, useEffect, useRef } from 'react';
import { Button, Modal, notification, message, Tooltip } from 'antd';
import { SyncOutlined } from '@ant-design/icons';
import axios from 'axios';

import GameBoard from './board';
import { DrawingHandlersSetterContext } from './drawing';
import GameControls from './game_controls';
import { Rack } from './rack';
import { ExchangeTiles } from './exchange_tiles';
import {
  nextArrowPropertyState,
  handleKeyPress,
  handleDroppedTile,
  returnTileToRack,
  designateBlank,
} from '../utils/cwgame/tile_placement';
import {
  Blank,
  uniqueTileIdx,
  EphemeralTile,
  EmptySpace,
} from '../utils/cwgame/common';

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
import {
  useGameContextStoreContext,
  useGameEndMessageStoreContext,
  useTimerStoreContext,
} from '../store/store';
import { BlankSelector } from './blank_selector';
import { GameEndMessage } from './game_end_message';
import { PlayerMetadata, GCGResponse } from './game_info';
import {
  GameEvent,
  PlayState,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import { toAPIUrl } from '../api/api';

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

const gcgExport = (gameID: string, playerMeta: Array<PlayerMetadata>) => {
  axios
    .post<GCGResponse>(toAPIUrl('game_service.GameMetadataService', 'GetGCG'), {
      gameId: gameID,
    })
    .then((resp) => {
      const url = window.URL.createObjectURL(new Blob([resp.data.gcg]));
      const link = document.createElement('a');
      link.href = url;
      let downloadFilename = `${gameID}.gcg`;
      // TODO: allow more characters as applicable
      if (playerMeta.every((x) => /^[0-9A-Za-z]+$/.test(x.nickname))) {
        const byStarts: Array<Array<string>> = [[], []];
        for (const x of playerMeta) {
          byStarts[+!!x.first].push(x.nickname);
        }
        downloadFilename = `${[...byStarts[1], ...byStarts[0]].join(
          '-'
        )}-${gameID}.gcg`;
      }
      link.setAttribute('download', downloadFilename);
      document.body.appendChild(link);
      link.onclick = () => {
        setTimeout(() => {
          window.URL.revokeObjectURL(url);
        }, 1000);
      };
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
  const [drawingKeyMode, setDrawingKeyMode] = useState(false);
  const {
    drawingCanBeEnabled,
    handleKeyDown: handleDrawingKeyDown,
  } = React.useContext(DrawingHandlersSetterContext);
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
  const { gameContext } = useGameContextStoreContext();
  const { gameEndMessage } = useGameEndMessageStoreContext();
  const { stopClock } = useTimerStoreContext();
  const [exchangeModalVisible, setExchangeModalVisible] = useState(false);
  const [exchangeAllowed, setexchangeAllowed] = useState(true);

  const observer = !props.playerMeta.some((p) => p.nickname === props.username);

  const isMyTurn = useCallback(() => {
    const iam = gameContext.nickToPlayerOrder[props.username];
    return iam && iam === `p${gameContext.onturn}`;
  }, [gameContext.nickToPlayerOrder, props.username, gameContext.onturn]);

  const { board, gameID, playerMeta, sendSocketMsg, username } = props;

  const makeMove = useCallback(
    (move: string, addl?: string) => {
      let moveEvt;
      if (move !== 'resign' && !isMyTurn()) {
        console.log(
          'off turn move attempts',
          gameContext.nickToPlayerOrder,
          username,
          gameContext.onturn
        );
        // It is not my turn. Ignore this event.
        message.warn({
          content: 'It is not your turn.',
          className: 'board-hud-message',
          duration: 1.5,
        });
        return;
      }
      console.log(
        'making move',
        gameContext.nickToPlayerOrder,
        username,
        gameContext.onturn
      );
      switch (move) {
        case 'exchange':
          moveEvt = exchangeMoveEvent(addl!, gameID);
          break;
        case 'pass':
          moveEvt = passMoveEvent(gameID);
          break;
        case 'resign':
          moveEvt = resignMoveEvent(gameID);
          break;
        case 'challenge':
          moveEvt = challengeMoveEvent(gameID);
          break;
        case 'commit':
          moveEvt = tilesetToMoveEvent(placedTiles, board, gameID);
          if (!moveEvt) {
            // this is an invalid play
            return;
          }
          break;
      }
      if (!moveEvt) {
        return;
      }
      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.CLIENT_GAMEPLAY_EVENT,
          moveEvt.serializeBinary()
        )
      );
      // Don't stop the clock; the next user event to come in will change the
      // clock over.
      // stopClock();
    },
    [
      gameContext.nickToPlayerOrder,
      gameContext.onturn,
      isMyTurn,
      placedTiles,
      board,
      gameID,
      sendSocketMsg,
      username,
    ]
  );

  const recallTiles = useCallback(() => {
    setPlacedTilesTempScore(0);
    setPlacedTiles(new Set<EphemeralTile>());
    setDisplayedRack(props.currentRack);
  }, [props.currentRack]);

  const shuffleTiles = useCallback(() => {
    setPlacedTilesTempScore(0);
    setPlacedTiles(new Set<EphemeralTile>());
    setDisplayedRack(shuffleString(props.currentRack));
  }, [props.currentRack]);

  const lastLettersRef = useRef<string>();
  const readOnlyEffectDependenciesRef = useRef<{
    displayedRack: string;
    isMyTurn: () => boolean;
    placedTiles: Set<EphemeralTile>;
    dim: number;
  }>();
  readOnlyEffectDependenciesRef.current = {
    displayedRack,
    isMyTurn,
    placedTiles,
    dim: props.board.dim,
  };

  // Need to sync state to props here whenever the board changes.
  useEffect(() => {
    let fullReset = false;
    const lastLetters = lastLettersRef.current;
    const dep = readOnlyEffectDependenciesRef.current!;
    if (lastLetters === undefined) {
      // First load.
      fullReset = true;
    } else if (
      props.currentRack &&
      !dep.displayedRack &&
      !dep.placedTiles.size
    ) {
      // First load after receiving rack.
      fullReset = true;
    } else if (!dep.isMyTurn()) {
      // Opponent's turn means we have just made a move. (Assumption: there are only two players.)
      fullReset = true;
    } else {
      // Opponent just did something. Check if it affects any premove.
      // TODO: revisit when supporting non-square boards.
      const letterAt = (row: number, col: number, letters: string) =>
        row < 0 || row >= dep.dim || col < 0 || col >= dep.dim
          ? null
          : letters[row * dep.dim + col];
      const letterChanged = (row: number, col: number) =>
        letterAt(row, col, lastLetters) !==
        letterAt(row, col, props.board.letters);
      const hookChanged = (
        row: number,
        col: number,
        drow: number,
        dcol: number
      ) => {
        while (true) {
          row += drow;
          col += dcol;
          if (letterChanged(row, col)) return true;
          const letter = letterAt(row, col, props.board.letters);
          if (letter === null || letter === EmptySpace) {
            return false;
          }
        }
      };
      const placedTileAffected = (row: number, col: number) =>
        letterChanged(row, col) ||
        hookChanged(row, col, -1, 0) ||
        hookChanged(row, col, +1, 0) ||
        hookChanged(row, col, 0, -1) ||
        hookChanged(row, col, 0, +1);
      if (
        Array.from(dep.placedTiles).some(({ row, col }) =>
          placedTileAffected(row, col)
        )
      ) {
        fullReset = true;
      }
    }
    if (fullReset) {
      setDisplayedRack(props.currentRack);
      setPlacedTiles(new Set<EphemeralTile>());
      setPlacedTilesTempScore(0);
      setArrowProperties({ row: 0, col: 0, horizontal: false, show: false });
    }
    lastLettersRef.current = props.board.letters;
  }, [props.board.letters, props.currentRack]);

  useEffect(() => {
    // Stop the clock if we unload the board panel.
    return () => {
      stopClock();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const bag = { ...gameContext.pool };
    for (let i = 0; i < props.currentRack.length; i += 1) {
      bag[props.currentRack[i]] -= 1;
    }
    const tilesRemaining =
      Object.values(bag).reduce((acc, cur) => {
        return acc + cur;
      }, 0) - 7;
    // Subtract 7 for opponent rack, won't matter when the
    // rack is smaller than that because past the threshold by then
    setexchangeAllowed(tilesRemaining >= 7);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameContext.pool]);
  useEffect(() => {
    if (
      gameContext.playState === PlayState.WAITING_FOR_FINAL_PASS &&
      isMyTurn()
    ) {
      const finalAction = (
        <>
          Your opponent has played their final tiles. You must{' '}
          <span
            className="message-action"
            onClick={() => makeMove('pass')}
            role="button"
          >
            pass
          </span>{' '}
          or{' '}
          <span
            className="message-action"
            role="button"
            onClick={() => makeMove('challenge')}
          >
            challenge
          </span>
          .
        </>
      );

      message.info(
        {
          content: finalAction,
          className: 'board-hud-message',
        },
        15
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameContext.playState]);

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
  const squareClicked = useCallback(
    (row: number, col: number) => {
      if (props.board.letterAt(row, col) !== EmptySpace) {
        // If there is a tile on this square, ignore the click.
        return;
      }
      setArrowProperties(nextArrowPropertyState(arrowProperties, row, col));
    },
    [arrowProperties, props.board]
  );
  const keydown = useCallback(
    (key: string) => {
      if (isMyTurn() && !props.gameDone) {
        if (key === '2') {
          makeMove('pass');
          return;
        }
        if (key === '3') {
          makeMove('challenge');
          return;
        }
        if (key === '4' && exchangeAllowed) {
          setExchangeModalVisible(true);
          return;
        }
        if (key === '$' && exchangeAllowed) {
          makeMove('exchange', props.currentRack);
          return;
        }
      }
      if (key === 'ArrowLeft' || key === 'ArrowRight') {
        setArrowProperties({
          ...arrowProperties,
          horizontal: !arrowProperties.horizontal,
        });
        return;
      }
      if (key === 'ArrowDown') {
        recallTiles();
        return;
      }
      if (key === 'ArrowUp') {
        shuffleTiles();
        return;
      }
      if (key === EnterKey && !exchangeModalVisible && !blankModalVisible) {
        makeMove('commit');
        return;
      }
      if (exchangeModalVisible) {
        return;
      }
      if (!arrowProperties.show) {
        return;
      }
      if (key === '?') {
        return;
      }
      // This should return a new set of arrow properties, and also set
      // some state further up (the tiles layout with a "just played" type
      // marker)
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
    },
    [
      arrowProperties,
      blankModalVisible,
      displayedRack,
      exchangeAllowed,
      exchangeModalVisible,
      isMyTurn,
      makeMove,
      placedTiles,
      props.board,
      props.currentRack,
      props.gameDone,
      recallTiles,
      shuffleTiles,
    ]
  );

  const handleTileDrop = useCallback(
    (
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
    },
    [displayedRack, placedTiles, props.board]
  );

  const clickToBoard = useCallback(
    (rackIndex: number) => {
      if (
        !arrowProperties.show ||
        arrowProperties.row >= props.board.dim ||
        arrowProperties.col >= props.board.dim
      ) {
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
    },
    [
      arrowProperties.col,
      arrowProperties.horizontal,
      arrowProperties.row,
      arrowProperties.show,
      displayedRack,
      placedTiles,
      props.board,
    ]
  );

  const handleBoardTileClick = useCallback((rune: string) => {
    if (rune === Blank) {
      setBlankModalVisible(true);
    }
  }, []);

  const handleBlankSelection = useCallback(
    (rune: string) => {
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
    },
    [displayedRack, placedTiles, props.board]
  );

  const handleBlankModalCancel = useCallback(() => {
    setBlankModalVisible(false);
  }, []);

  const returnToRack = useCallback(
    (rackIndex: number | undefined, tileIndex: number | undefined) => {
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
    },
    [displayedRack, placedTiles, props.board]
  );

  const moveRackTile = useCallback(
    (newIndex: number | undefined, oldIndex: number | undefined) => {
      if (typeof newIndex === 'number' && typeof oldIndex === 'number') {
        const newRack = displayedRack.split('');
        newRack.splice(oldIndex, 1);
        newRack.splice(newIndex, 0, displayedRack[oldIndex]);
        setPlacedTilesTempScore(0);
        setDisplayedRack(newRack.join(''));
      }
    },
    [displayedRack]
  );

  const showExchangeModal = useCallback(() => {
    setExchangeModalVisible(true);
  }, []);

  const handleExchangeModalOk = useCallback(
    (exchangedTiles: string) => {
      setExchangeModalVisible(false);
      makeMove('exchange', exchangedTiles);
    },
    [makeMove]
  );

  const rematch = useCallback(() => {
    const evt = new MatchRequest();
    const receiver = new MatchUser();

    let opp = '';
    playerMeta.forEach((p) => {
      if (!(p.nickname === username)) {
        opp = p.nickname;
      }
    });

    if (observer) {
      return;
    }

    receiver.setDisplayName(opp);
    evt.setReceivingUser(receiver);
    evt.setRematchFor(gameID);
    sendSocketMsg(
      encodeToSocketFmt(MessageType.MATCH_REQUEST, evt.serializeBinary())
    );

    notification.info({
      message: 'Rematch',
      description: `Sent rematch request to ${opp}`,
    });
    console.log('rematching');
  }, [observer, gameID, playerMeta, sendSocketMsg, username]);

  const handleKeyDown = useCallback(
    (e) => {
      if (drawingCanBeEnabled) {
        // To activate a drawing hotkey, type 0, then the hotkey.
        if (!exchangeModalVisible && !blankModalVisible) {
          if (drawingKeyMode) {
            e.preventDefault();
            setDrawingKeyMode(false);
            handleDrawingKeyDown(e);
            return;
          } else if (e.key === '0') {
            e.preventDefault();
            setDrawingKeyMode(true);
            console.log(
              'You pressed 0. Now press one of these keys:' +
                '\n0 = Toggle drawing' +
                '\nU = Undo' +
                '\nW = Wipe' +
                '\nR = Red pen' +
                '\nG = Green pen' +
                '\nB = Blue pen' +
                '\nY = Yellow pen' +
                '\nE = Eraser'
            );
            return;
          }
        }
      }
      keydown(e.key);
    },
    [
      blankModalVisible,
      drawingCanBeEnabled,
      drawingKeyMode,
      exchangeModalVisible,
      handleDrawingKeyDown,
      keydown,
    ]
  );
  const handlePass = useCallback(() => makeMove('pass'), [makeMove]);
  const handleResign = useCallback(() => makeMove('resign'), [makeMove]);
  const handleChallenge = useCallback(() => makeMove('challenge'), [makeMove]);
  const handleCommit = useCallback(() => makeMove('commit'), [makeMove]);
  const handleExamine = useCallback(
    () => gcgExport(props.gameID, props.playerMeta),
    [props.gameID, props.playerMeta]
  );
  const handleExchangeTilesCancel = useCallback(() => {
    setExchangeModalVisible(false);
  }, []);

  return (
    <div
      id="board-container"
      className="board-container"
      onKeyDown={handleKeyDown}
      tabIndex={-1}
      role="textbox"
    >
      <GameBoard
        gridSize={props.board.dim}
        gridLayout={props.board.gridLayout}
        handleBoardTileClick={handleBoardTileClick}
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
            title="Reset Rack &darr;"
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
            moveRackTile={moveRackTile}
          />
          <Tooltip
            title="Shuffle &uarr;"
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
        myTurn={isMyTurn()}
        finalPassOrChallenge={
          gameContext.playState === PlayState.WAITING_FOR_FINAL_PASS
        }
        exchangeAllowed={exchangeAllowed}
        observer={observer}
        onRecall={recallTiles}
        showExchangeModal={showExchangeModal}
        onPass={handlePass}
        onResign={handleResign}
        onChallenge={handleChallenge}
        onCommit={handleCommit}
        onRematch={rematch}
        onExamine={handleExamine}
        showRematch={gameEndMessage !== ''}
        gameEndControls={gameEndMessage !== '' || props.gameDone}
        currentRack={props.currentRack}
      />
      <ExchangeTiles
        rack={props.currentRack}
        modalVisible={exchangeModalVisible}
        onOk={handleExchangeModalOk}
        onCancel={handleExchangeTilesCancel}
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
