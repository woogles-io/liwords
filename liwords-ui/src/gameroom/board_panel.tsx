import React, { useState, useEffect } from 'react';
import { Button, Modal, notification } from 'antd';
import { ArrowDownOutlined, SyncOutlined } from '@ant-design/icons';
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
import { EphemeralTile, EmptySpace } from '../utils/cwgame/common';
import {
  tilesetToMoveEvent,
  exchangeMoveEvent,
  passMoveEvent,
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
import { PlayerMetadata } from './game_info';

// The frame atop is 24 height
// The frames on the sides are 24 in width, surrounded by a 14 pix gutter
const EnterKey = 'Enter';

type Props = {
  username: string;
  showBonusLabels: boolean;
  currentRack: string;
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

  const swapRackTiles = (
    indexA: number | undefined,
    indexB: number | undefined
  ) => {
    if (typeof indexA === 'number' && typeof indexB === 'number') {
      const newRack = displayedRack.split('');
      newRack[indexA] = displayedRack[indexB];
      newRack[indexB] = displayedRack[indexA];
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

  const rematch = () => {
    const evt = new MatchRequest();
    const receiver = new MatchUser();

    let observer = true;
    let opp = '';
    props.playerMeta.forEach((p) => {
      if (p.nickname === props.username) {
        observer = false;
      } else {
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
        showBonusLabels={false}
        lastPlayedTiles={gameContext.lastPlayedTiles}
        tentativeTiles={placedTiles}
        tentativeTileScore={placedTilesTempScore}
        currentRack={props.currentRack}
        squareClicked={squareClicked}
        placementArrowProperties={arrowProperties}
      />
      {!gameEndMessage ? (
        <div className="rack-container">
          <Button
            shape="circle"
            icon={<ArrowDownOutlined />}
            type="primary"
            onClick={recallTiles}
          />
          <Rack
            letters={displayedRack}
            grabbable
            returnToRack={returnToRack}
            swapRackTiles={(
              indexA: number | undefined,
              indexB: number | undefined
            ) => {
              swapRackTiles(indexA, indexB);
            }}
          />
          <Button
            shape="circle"
            icon={<SyncOutlined />}
            type="primary"
            onClick={shuffleTiles}
          />
        </div>
      ) : (
        <GameEndMessage message={gameEndMessage} />
      )}
      <GameControls
        onRecall={recallTiles}
        onExchange={(rack: string) => makeMove('exchange', rack)}
        onPass={() => makeMove('pass')}
        onChallenge={() => makeMove('challenge')}
        onCommit={() => makeMove('commit')}
        onRematch={rematch}
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
