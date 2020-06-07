import React, { useEffect } from 'react';
import { Row, Col } from 'antd';

import { useParams } from 'react-router-dom';
import { BoardPanel } from './board_panel';
import { TopBar } from '../topbar/topbar';
import { Chat } from './chat';
import { useStoreContext } from '../store/store';
import { PlayerCards } from './player_cards';
import Pool from './pool';
import { fullPlayerInfo } from '../utils/cwgame/game';
import {
  RegisterRealm,
  DeregisterRealm,
  MessageType,
} from '../gen/api/proto/game_service_pb';
import { encodeToSocketFmt } from '../utils/protobuf';

const gutter = 16;
const boardspan = 12;
const maxspan = 24; // from ant design
const navbarHeightAndGutter = 84; // 72 + 12 spacing

type RouterProps = {
  gameID: string;
};

type Props = {
  windowWidth: number;
  windowHeight: number;
  sendSocketMsg: (msg: Uint8Array) => void;
  username: string;
};

export const Table = (props: Props) => {
  // Calculate the width of the board.
  // If the pixel width is 1440,
  // The width of the drawable part is 12/24 * 1440 = 720
  // Minus gutters makes it 704

  // The height is more important, as the buttons and tiles go down
  // so don't make the board so tall that these elements become invisible.

  let boardPanelWidth = (boardspan / maxspan) * props.windowWidth - gutter;
  // Shrug; determine this better:
  let boardPanelHeight = boardPanelWidth + 96;
  const viewableHeight = props.windowHeight - navbarHeightAndGutter;

  // XXX: this all needs to be tweaked.
  if (boardPanelHeight > viewableHeight) {
    boardPanelHeight = viewableHeight;
    boardPanelWidth = boardPanelHeight - 96;
  }
  const { setRedirGame, gameState, chat } = useStoreContext();
  const { gameID } = useParams();

  useEffect(() => {
    // Avoid react-router hijacking the back button.
    setRedirGame('');
  }, [setRedirGame]);

  useEffect(() => {
    console.log('Tryna register with gameID', gameID);
    const rr = new RegisterRealm();
    rr.setRealm(gameID);
    props.sendSocketMsg(
      encodeToSocketFmt(MessageType.REGISTER_REALM, rr.serializeBinary())
    );

    return () => {
      console.log('cleaning up; deregistering', gameID);
      const dr = new DeregisterRealm();
      dr.setRealm(gameID);
      props.sendSocketMsg(
        encodeToSocketFmt(MessageType.DEREGISTER_REALM, dr.serializeBinary())
      );
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const player1 = fullPlayerInfo(0, gameState);
  const player2 = fullPlayerInfo(1, gameState);

  return (
    <div>
      <Row>
        <Col span={24}>
          <TopBar username={props.username} />
        </Col>
      </Row>
      <Row
      gutter={gutter}
      className="game-table"
      >
        <Col span={6}>
          <Chat chatEntities={chat} />
        </Col>
        <Col span={boardspan}>
          <BoardPanel
            compWidth={boardPanelWidth}
            compHeight={boardPanelHeight}
            board={gameState.board}
            showBonusLabels={false}
            currentRack={gameState.currentRacks[props.username] || ''}
            lastPlayedLetters={{}}
            gameID={gameID}
            sendSocketMsg={props.sendSocketMsg}
          />
        </Col>
        <Col span={6}>
          {/* maybe some of this info comes from backend */}
          <PlayerCards player1={player1} player2={player2} />
          {/* <GameInfo
            timer="15 0"
            gameType="Classic"
            dictionary="Collins"
            challengeRule="5-pt"
            rated={rated}
          /> */}
          <Row>15 0 - Classic - Collins</Row>
          <Row>5 point challenge - Unrated</Row>

          <Pool
            pool={gameState.pool}
            currentRack={gameState.currentRacks[props.username] || ''}
          />
        </Col>
      </Row>
    </div>
  );
};
