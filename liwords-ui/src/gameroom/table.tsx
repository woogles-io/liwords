import React, { useEffect } from 'react';
import { Row, Col, Card } from 'antd';

import { useParams } from 'react-router-dom';
import { BoardPanel } from './board_panel';
import { TopBar } from '../topbar/topbar';
import { Chat } from './chat';
import { useStoreContext } from '../store/store';
import { PlayerCards } from './player_cards';
import Pool from './pool';
import {
  RegisterRealm,
  DeregisterRealm,
  MessageType,
} from '../gen/api/proto/game_service_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import './gameroom.scss';

const gutter = 16;
const boardspan = 12;
const maxspan = 24; // from ant design
const navbarHeightAndGutter = 84; // 72 + 12 spacing

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
  const { setRedirGame, gameContext, chat } = useStoreContext();
  const { gameID } = useParams();

  useEffect(() => {
    // Avoid react-router hijacking the back button.
    // If setRedirGame is not defined, then we're SOL I guess.
    setRedirGame ? setRedirGame('') : (() => {})();
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

  // Figure out what rack we should display.
  // If we are one of the players, display our rack.
  // If we are NOT one of the players (so an observer), display the rack of
  // the player on turn.
  let rack;
  const us = gameContext.players.find((p) => p.nickname === props.username);
  if (us) {
    rack = us.currentRack;
  } else {
    rack = gameContext.players.find((p) => p.onturn)?.currentRack || '';
  }

  return (
    <div>
      <Row>
        <Col span={24}>
          <TopBar username={props.username} />
        </Col>
      </Row>
      <Row gutter={gutter} className="game-table">
        <Col span={6} className="chat-area">
          <Chat chatEntities={chat} />
        </Col>
        <Col span={boardspan} className="play-area">
          <BoardPanel
            username={props.username}
            compWidth={boardPanelWidth}
            compHeight={boardPanelHeight}
            board={gameContext.board}
            showBonusLabels={false}
            currentRack={rack}
            lastPlayedLetters={{}}
            gameID={gameID}
            sendSocketMsg={props.sendSocketMsg}
          />
        </Col>
        <Col span={6} className="data-area">
          {/* maybe some of this info comes from backend */}
          <PlayerCards />
          {/* <GameInfo
            timer="15 0"
            gameType="Classic"
            dictionary="Collins"
            challengeRule="5-pt"
            rated={rated}
          /> */}
          <Pool pool={gameContext?.pool} currentRack={rack} />
          <Card>
            <Row>15 0 - Classic - Collins</Row>
            <Row>5 point challenge - Unrated</Row>
          </Card>
       </Col>
      </Row>
    </div>
  );
};
