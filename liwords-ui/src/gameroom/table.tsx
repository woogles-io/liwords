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
  TimedOut,
} from '../gen/api/proto/game_service_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import './scss/gameroom.scss';
import { ScoreCard } from './scorecard';

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
  const { gameID } = useParams();
  const {
    setRedirGame,
    gameContext,
    chat,
    clearChat,
    pTimedOut,
    poolFormat,
    setPoolFormat,
    setPTimedOut,
  } = useStoreContext();
  const { username, sendSocketMsg } = props;

  useEffect(() => {
    // Avoid react-router hijacking the back button.
    // If setRedirGame is not defined, then we're SOL I guess.
    setRedirGame ? setRedirGame('') : (() => {})();
  }, [setRedirGame]);

  useEffect(() => {
    const rr = new RegisterRealm();
    rr.setRealm(gameID);
    sendSocketMsg(
      encodeToSocketFmt(MessageType.REGISTER_REALM, rr.serializeBinary())
    );
    // XXX: Fetch some info via XHR about the game itself (timer, tourney, etc) here.

    return () => {
      const dr = new DeregisterRealm();
      dr.setRealm(gameID);
      sendSocketMsg(
        encodeToSocketFmt(MessageType.DEREGISTER_REALM, dr.serializeBinary())
      );
      clearChat();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (pTimedOut === undefined) return;
    // Otherwise, player timed out. This will only send once.
    // Send the time out if we're either of both players that are in the game.
    let send = false;
    let timedout = '';

    for (let idx = 0; idx < gameContext.players.length; idx++) {
      const nick = gameContext.players[idx].nickname;
      if (gameContext.nickToPlayerOrder[nick] === pTimedOut) {
        timedout = nick;
      }
      if (username === nick) {
        send = true;
      }
    }

    if (!send) return;

    const to = new TimedOut();
    to.setGameId(gameID);
    to.setUsername(timedout);
    console.log('sending timeout to socket');
    sendSocketMsg(
      encodeToSocketFmt(MessageType.TIMED_OUT, to.serializeBinary())
    );
    setPTimedOut(undefined);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pTimedOut, gameContext.nickToPlayerOrder, gameID]);

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
        <TopBar username={props.username} />
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
          <Card>
            <Row>15 0 - Classic - Collins</Row>
            <Row>5 point challenge - Unrated</Row>
          </Card>
          <Pool
            pool={gameContext?.pool}
            currentRack={rack}
            poolFormat={poolFormat}
          />
          <ScoreCard
            username={props.username}
            playing={us !== undefined}
            turns={gameContext.turns}
            currentTurn={gameContext.currentTurn}
            board={gameContext.board}
          />
        </Col>
      </Row>
    </div>
  );
};
