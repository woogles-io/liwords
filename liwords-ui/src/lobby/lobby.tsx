import React, { useState, useEffect, useRef } from 'react';
import { Row, Col, Button, Card, Input } from 'antd';
// import { SoughtGame, LobbyStore } from '../store/lobby_store';
import { getSocketURI, websocket } from '../socket/socket';
import {
  useLobbyContext,
  LobbyStoreData,
  SoughtGame,
} from '../store/lobby_store';

import { TopBar } from '../topbar/topbar';
import {
  SeekRequest,
  GameRequest,
  ChallengeRuleMap,
  ChallengeRule,
  MessageType,
  RequestingUser,
} from '../gen/api/proto/game_service_pb';
import { encodeToSocketFmt, decodeToMsg } from '../utils/protobuf';
import { SoughtGames } from './sought_games';

const onSocketMsg = (storeData: LobbyStoreData) => {
  return (reader: FileReader) => {
    if (!reader.result) {
      return;
    }
    const msg = new Uint8Array(reader.result as ArrayBuffer);
    let sr;
    let gameReq;
    let user;
    // msg type:
    switch (msg[0]) {
      case MessageType.SEEK_REQUEST:
        sr = SeekRequest.deserializeBinary(msg.slice(1));
        gameReq = sr.getGameRequest();
        user = sr.getUser();
        if (!gameReq || !user) {
          return;
        }
        storeData.addSoughtGame({
          seeker: user.getUsername(),
          lexicon: gameReq.getLexicon(),
          initialTimeSecs: gameReq.getInitialTimeSeconds(),
          challengeRule: gameReq.getChallengeRule(),
        });
    }
  };
};

function useLobbySocket(
  socketRef: React.MutableRefObject<WebSocket | null>,
  storeData: LobbyStoreData
) {
  useEffect(() => {
    websocket(
      getSocketURI(),
      (socket) => {
        // eslint-disable-next-line no-param-reassign
        socketRef.current = socket;
      },
      (event) => {
        decodeToMsg(event.data, onSocketMsg(storeData));
      }
    );
    return () => {
      if (socketRef.current) {
        console.log('closing lobby socket');
        socketRef.current.close();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
}

const sendSeek = (game: SoughtGame, socket: WebSocket | null) => {
  if (!socket) {
    return;
  }
  const sr = new SeekRequest();
  const gr = new GameRequest();
  const user = new RequestingUser();

  user.setUsername(game.seeker);
  // rating comes from backend.

  sr.setUser(user);
  gr.setChallengeRule(
    game.challengeRule as ChallengeRuleMap[keyof ChallengeRuleMap]
  );
  gr.setLexicon(game.lexicon);
  gr.setInitialTimeSeconds(game.initialTimeSecs);
  sr.setGameRequest(gr);
  socket.send(
    encodeToSocketFmt(MessageType.SEEK_REQUEST, sr.serializeBinary())
  );
};

type Props = {
  username: string;
};

export const Lobby = (props: Props) => {
  // const [store] = React.useState(() => new LobbyStore());
  const socketRef = useRef<WebSocket | null>(null);

  const storeFns = useLobbyContext();
  useLobbySocket(socketRef, storeFns);

  return (
    <div>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>

      <Row>
        <Col span={24}>
          <h3>Lobby</h3>
        </Col>
      </Row>
      <Row>
        <Col span={8} offset={8}>
          <SoughtGames />
        </Col>
      </Row>

      <Row>
        <Col span={24}>
          <Button
            onClick={() =>
              // Eventually will replace with a modal that has selections
              // like lexicon / challenge rule/ etc.
              sendSeek(
                {
                  seeker: props.username,
                  lexicon: 'CSW19',
                  challengeRule: ChallengeRule.FIVE_POINT,
                  initialTimeSecs: 900,
                  // rating: 0,
                  // soughtID: username,
                },
                socketRef.current
              )
            }
          >
            New Game
          </Button>
        </Col>
      </Row>
    </div>
  );
};
