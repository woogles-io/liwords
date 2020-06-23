import React, { useEffect, useState } from 'react';
import { Row, Col, Button, Modal } from 'antd';
import { Redirect } from 'react-router-dom';

import { TopBar } from '../topbar/topbar';
import {
  SeekRequest,
  GameRequest,
  MessageType,
  RequestingUser,
  GameAcceptedEvent,
  GameRules,
  JoinPath,
  UnjoinRealm,
} from '../gen/api/proto/game_service_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import { SoughtGames } from './sought_games';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { useStoreContext } from '../store/store';
import {
  ChallengeRuleMap,
  ChallengeRule,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import { SeekForm, seekPropVals } from './seek_form';

const JoinSocketDelay = 1000;

const sendSeek = (
  game: SoughtGame,
  sendSocketMsg: (msg: Uint8Array) => void
) => {
  const sr = new SeekRequest();
  const gr = new GameRequest();
  const rules = new GameRules();
  rules.setBoardLayoutName('CrosswordGame');
  rules.setLetterDistributionName('english');

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
  loggedIn: boolean;
  sendSocketMsg: (msg: Uint8Array) => void;
};

export const Lobby = (props: Props) => {
  const { redirGame } = useStoreContext();
  // On render, register lobby realm; deregister when we exit.
  const { sendSocketMsg } = props;
  useEffect(() => {
    console.log('Tryna register with lobby');
    const rr = new JoinPath();
    rr.setPath('/');
    // Note: We send the socket message 1 second after we join. This
    // is to give the socket token time to log us in. Note, however,
    // that the app will not stop working if we are not logged in by the
    // time this fires. The backend will just fire what it fires when
    // one logs in, twice.
    const timeout = setTimeout(() => {
      sendSocketMsg(
        encodeToSocketFmt(MessageType.JOIN_PATH, rr.serializeBinary())
      );
    }, JoinSocketDelay);

    return () => {
      clearTimeout(timeout);
      console.log('cleaning up; deregistering');
      const dr = new UnjoinRealm();
      sendSocketMsg(
        encodeToSocketFmt(MessageType.UNJOIN_REALM, dr.serializeBinary())
      );
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const [seekModalVisible, setSeekModalVisible] = useState(false);
  const [seekSettings, setSeekSettings] = useState<seekPropVals>({
    lexicon: 'CSW19',
    challengerule: ChallengeRule.FIVE_POINT,
    initialtime: 8,
  });

  const showSeekModal = () => {
    setSeekModalVisible(true);
  };

  const handleModalOk = () => {
    setSeekModalVisible(false);
    sendSeek(
      {
        seeker: '',
        lexicon: seekSettings.lexicon as string,
        challengeRule: seekSettings.challengerule as number,
        initialTimeSecs: (seekSettings.initialtime as number) * 60,
        // rating: 0,
        seekID: '', // assigned by server
      },
      props.sendSocketMsg
    );
  };

  const handleModalCancel = () => {
    setSeekModalVisible(false);
  };
  console.log('redirGame', redirGame);

  if (redirGame !== '') {
    return <Redirect push to={`/game/${redirGame}`} />;
  }

  return (
    <div>
      <Row>
        <Col span={24}>
          <TopBar username={props.username} loggedIn={props.loggedIn} />
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
          <Button type="primary" onClick={showSeekModal}>
            New Game
          </Button>
          <Modal
            title="New Game"
            visible={seekModalVisible}
            onOk={handleModalOk}
            onCancel={handleModalCancel}
          >
            <SeekForm vals={seekSettings} onChange={setSeekSettings} />
          </Modal>
        </Col>
      </Row>
    </div>
  );
};
