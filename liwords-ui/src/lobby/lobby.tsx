import React, { useState, useEffect } from 'react';
import { Card } from 'antd';

import { TopBar } from '../topbar/topbar';
import {
  SeekRequest,
  GameRequest,
  MessageType,
  GameRules,
  RatingMode,
  MatchRequest,
  SoughtGameProcessEvent,
  ChatMessage,
} from '../gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { ChallengeRuleMap } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { GameLists } from './gameLists';
import { Chat } from '../chat/chat';
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
    props.loggedIn ? 'PLAY' : 'WATCH'
  );

  const { chat, presences } = useStoreContext();

  useEffect(() => {
    setSelectedGameTab(props.loggedIn ? 'PLAY' : 'WATCH');
  }, [props.loggedIn]);

  const showSeekModal = () => {
    setSeekModalVisible(true);
  };

  const showMatchModal = () => {
    setMatchModalVisible(true);
  };

  const handleSeekModalCancel = () => {
    setSeekModalVisible(false);
  };

  const onSeekSubmit = (g: SoughtGame) => {
    sendSeek(g, props.sendSocketMsg);
    setMatchModalVisible(false);
    setSeekModalVisible(false);
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
    <>
      <TopBar
        username={props.username}
        loggedIn={props.loggedIn}
        connectedToSocket={props.connectedToSocket}
      />
      <div className="lobby">
        <div className="chat-area">
          <Chat
            chatEntities={chat}
            sendChat={sendChat}
            description="Lobby chat"
            peopleOnlineContext="Players"
            presences={presences}
          />
        </div>
        <GameLists
          loggedIn={props.loggedIn}
          userID={props.userID}
          username={props.username}
          newGame={(seekID: string) => sendAccept(seekID, props.sendSocketMsg)}
          selectedGameTab={selectedGameTab}
          setSelectedGameTab={setSelectedGameTab}
          showSeekModal={showSeekModal}
          showMatchModal={showMatchModal}
          matchModalVisible={matchModalVisible}
          seekModalVisible={seekModalVisible}
          handleMatchModalCancel={handleMatchModalCancel}
          handleSeekModalCancel={handleSeekModalCancel}
          onSeekSubmit={onSeekSubmit}
        />
        <div className="announcements">
          <Card>
            <h3>Woogles is coming soon!</h3>
            <p>
              In the meantime, why not watch some of our alpha testers play?
            </p>
          </Card>
        </div>
      </div>
    </>
  );
};
