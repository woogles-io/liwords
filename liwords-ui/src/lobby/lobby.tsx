import React, { useEffect } from 'react';
import { Row, Col, Button } from 'antd';
import { Redirect } from 'react-router-dom';

import { TopBar } from '../topbar/topbar';
import {
  SeekRequest,
  GameRequest,
  MessageType,
  RequestingUser,
  GameAcceptedEvent,
  GameRules,
  RegisterRealm,
  DeregisterRealm,
} from '../gen/api/proto/game_service_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import { SoughtGames } from './sought_games';
import { SoughtGame, useStoreContext } from '../store/store';
import {
  ChallengeRuleMap,
  ChallengeRule,
} from '../gen/macondo/api/proto/macondo/macondo_pb';

const sendSeek = (
  game: SoughtGame,
  sendSocketMsg: (msg: Uint8Array) => void
) => {
  const sr = new SeekRequest();
  const gr = new GameRequest();
  const rules = new GameRules();
  rules.setBoardLayoutName('CrosswordGame');
  rules.setLetterDistributionName('english');

  const user = new RequestingUser();
  user.setUsername(game.seeker);
  // rating comes from backend.
  sr.setUser(user);

  gr.setChallengeRule(
    game.challengeRule as ChallengeRuleMap[keyof ChallengeRuleMap]
  );
  gr.setLexicon(game.lexicon);
  gr.setInitialTimeSeconds(game.initialTimeSecs);
  gr.setRules(rules);

  sr.setGameRequest(gr);

  sendSocketMsg(
    encodeToSocketFmt(MessageType.SEEK_REQUEST, sr.serializeBinary())
  );
};

const sendAccept = (
  seekID: string,
  sendSocketMsg: (msg: Uint8Array) => void
) => {
  // Eventually use the ID.
  const sa = new GameAcceptedEvent();
  sa.setRequestId(seekID);
  sendSocketMsg(
    encodeToSocketFmt(MessageType.GAME_ACCEPTED_EVENT, sa.serializeBinary())
  );
};

type Props = {
  username: string;
  sendSocketMsg: (msg: Uint8Array) => void;
};

export const Lobby = (props: Props) => {
  const { redirGame } = useStoreContext();
  // On render, register lobby realm; deregister when we exit.
  const { sendSocketMsg } = props;
  useEffect(() => {
    console.log('Tryna register with lobby');
    const rr = new RegisterRealm();
    rr.setRealm('lobby');
    sendSocketMsg(
      encodeToSocketFmt(MessageType.REGISTER_REALM, rr.serializeBinary())
    );

    return () => {
      console.log('cleaning up; deregistering');
      const dr = new DeregisterRealm();
      dr.setRealm('lobby');
      sendSocketMsg(
        encodeToSocketFmt(MessageType.DEREGISTER_REALM, dr.serializeBinary())
      );
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (redirGame !== '') {
    return <Redirect push to={`/game/${redirGame}`} />;
  }

  return (
    <div>
      <Row>
        <Col span={24}>
          <TopBar username={props.username} />
        </Col>
      </Row>

      <Row>
        <Col span={24}>
          <h3>Lobby</h3>
        </Col>
      </Row>
      <Row>
        <Col span={8} offset={8}>
          <SoughtGames
            newGame={(seekID: string) =>
              sendAccept(seekID, props.sendSocketMsg)
            }
          />
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
                  seekID: '', // assigned by server
                },
                props.sendSocketMsg
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
