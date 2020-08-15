import React, { useState, useEffect } from 'react';
import { Card, Row, Col } from 'antd';

import { TopBar } from '../topbar/topbar';
import {
  SeekRequest,
  GameRequest,
  MessageType,
  GameRules,
  RatingMode,
  MatchRequest,
  MatchUser,
  SoughtGameProcessEvent,
  ChatMessage,
} from '../gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import {
  ChallengeRuleMap,
  ChallengeRule,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import { seekPropVals } from './seek_form';
import { GameLists } from './gameLists';
import { Chat } from '../gameroom/chat';
import { useStoreContext } from '../store/store';
import './lobby.scss';

const sendSeek = (
  game: SoughtGame,
  sendSocketMsg: (msg: Uint8Array) => void
) => {
  const sr = new SeekRequest();
  const mr = new MatchRequest();
  const gr = new GameRequest();
  const rules = new GameRules();
  rules.setBoardLayoutName('CrosswordGame');
  rules.setLetterDistributionName('english');

  gr.setChallengeRule(
    game.challengeRule as ChallengeRuleMap[keyof ChallengeRuleMap]
  );
  gr.setLexicon(game.lexicon);
  gr.setInitialTimeSeconds(game.initialTimeSecs);
  gr.setMaxOvertimeMinutes(game.maxOvertimeMinutes);
  gr.setIncrementSeconds(game.incrementSecs);
  gr.setRules(rules);
  gr.setRatingMode(game.rated ? RatingMode.RATED : RatingMode.CASUAL);
  if (game.receiver.getDisplayName() === '') {
    sr.setGameRequest(gr);
    sendSocketMsg(
      encodeToSocketFmt(MessageType.SEEK_REQUEST, sr.serializeBinary())
    );
  } else {
    mr.setGameRequest(gr);
    mr.setReceivingUser(game.receiver);
    sendSocketMsg(
      encodeToSocketFmt(MessageType.MATCH_REQUEST, mr.serializeBinary())
    );
  }
};

const sendAccept = (
  seekID: string,
  sendSocketMsg: (msg: Uint8Array) => void
) => {
  // Eventually use the ID.
  const sa = new SoughtGameProcessEvent();
  sa.setRequestId(seekID);
  sendSocketMsg(
    encodeToSocketFmt(
      MessageType.SOUGHT_GAME_PROCESS_EVENT,
      sa.serializeBinary()
    )
  );
};

type Props = {
  username: string;
  userID: string;
  loggedIn: boolean;
  sendSocketMsg: (msg: Uint8Array) => void;
  connectedToSocket: boolean;
};

export const Lobby = (props: Props) => {
  const [seekModalVisible, setSeekModalVisible] = useState(false);
  const [matchModalVisible, setMatchModalVisible] = useState(false);
  const [selectedGameTab, setSelectedGameTab] = useState(
    props.loggedIn ? 'WATCH' : 'PLAY'
  );
  const [seekSettings, setSeekSettings] = useState<seekPropVals>({
    lexicon: 'CSW19',
    challengerule: ChallengeRule.FIVE_POINT,
    initialtime: 8,
    rated: false,
    maxovertime: 1,
    friend: '',
    incrementsecs: 0,
  });
  const { chat } = useStoreContext();

  useEffect(() => {
    setSeekSettings((s) => ({
      ...s,
      rated: props.loggedIn,
    }));
  }, [props.loggedIn]);

  const showSeekModal = () => {
    setSeekModalVisible(true);
  };

  const showMatchModal = () => {
    setMatchModalVisible(true);
  };

  const handleSeekModalOk = () => {
    setSeekModalVisible(false);
    sendSeek(
      {
        // These items are assigned by the server:
        seeker: '',
        userRating: '',
        seekID: '',

        lexicon: seekSettings.lexicon as string,
        challengeRule: seekSettings.challengerule as number,
        initialTimeSecs: Math.round((seekSettings.initialtime as number) * 60),
        incrementSecs: Math.round(seekSettings.incrementsecs as number),
        rated: seekSettings.rated as boolean,
        maxOvertimeMinutes: Math.round(seekSettings.maxovertime as number),
        receiver: new MatchUser(),
        rematchFor: '',
      },
      props.sendSocketMsg
    );
  };

  const handleSeekModalCancel = () => {
    setSeekModalVisible(false);
  };

  const handleMatchModalOk = () => {
    setMatchModalVisible(false);
    const matchUser = new MatchUser();
    matchUser.setDisplayName(seekSettings.friend as string);
    sendSeek(
      {
        // These items are assigned by the server:
        seeker: '',
        userRating: '',
        seekID: '',

        lexicon: seekSettings.lexicon as string,
        challengeRule: seekSettings.challengerule as number,
        initialTimeSecs: Math.round((seekSettings.initialtime as number) * 60),
        incrementSecs: Math.round(seekSettings.incrementsecs as number),
        rated: seekSettings.rated as boolean,
        maxOvertimeMinutes: Math.round(seekSettings.maxovertime as number),
        receiver: matchUser,
        rematchFor: '',
      },
      props.sendSocketMsg
    );
  };

  const handleMatchModalCancel = () => {
    setMatchModalVisible(false);
  };

  const sendChat = (msg: string) => {
    const evt = new ChatMessage();
    evt.setMessage(msg);
    evt.setChannel('lobby.chat');
    props.sendSocketMsg(
      encodeToSocketFmt(MessageType.CHAT_MESSAGE, evt.serializeBinary())
    );
  };

  return (
    <div>
      <Row>
        <Col span={24}>
          <TopBar
            username={props.username}
            loggedIn={props.loggedIn}
            connectedToSocket={props.connectedToSocket}
          />
        </Col>
      </Row>
      <Row className="lobby">
        <Col span={6} className="chat-area">
          <Chat
            chatEntities={chat}
            sendChat={sendChat}
            description="Lobby chat"
          />
        </Col>
        <Col span={12} className="game-lists">
          <GameLists
            loggedIn={props.loggedIn}
            userID={props.userID}
            username={props.username}
            newGame={(seekID: string) =>
              sendAccept(seekID, props.sendSocketMsg)
            }
            selectedGameTab={selectedGameTab}
            setSelectedGameTab={setSelectedGameTab}
            showSeekModal={showSeekModal}
            showMatchModal={showMatchModal}
            matchModalVisible={matchModalVisible}
            seekModalVisible={seekModalVisible}
            handleMatchModalCancel={handleMatchModalCancel}
            handleMatchModalOk={handleMatchModalOk}
            handleSeekModalCancel={handleSeekModalCancel}
            handleSeekModalOk={handleSeekModalOk}
            seekSettings={seekSettings}
            setSeekSettings={setSeekSettings}
          />
        </Col>
        <Col span={6} className="news-area">
          <Card className="announcements">
            <h3>Woogles is coming soon!</h3>
            <p>
              Please back our{' '}
              <a href="https://www.kickstarter.com/projects/woogles/woogles">
                {' '}
                Kickstarter.
              </a>{' '}
              We're a nonprofit and are counting on you.
            </p>
          </Card>
        </Col>
      </Row>
    </div>
  );
};
