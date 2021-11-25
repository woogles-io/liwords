import React, { useCallback, useEffect } from 'react';
import { useMountedState } from '../utils/mounted';

import { TopBar } from '../topbar/topbar';

import { SoughtGame } from '../store/reducers/lobby_reducer';
import { GameLists } from './gameLists';
import { Chat } from '../chat/chat';
import { useLoginStateStoreContext } from '../store/store';
import './lobby.scss';
import { Announcements } from './announcements';
import { sendAccept, sendSeek } from './sought_game_interactions';
import { Sidebar } from '../shared/layoutContainers/sidebar';
import { Main } from '../shared/layoutContainers/main';
import {
  SideMenu,
  SideMenuContextProvider,
} from '../shared/layoutContainers/menu';
import { PanelComponentWrapper } from '../shared/layoutContainers/panelComponentWrapper';
import {
  ContextTab,
  ContextTabs,
} from '../shared/layoutContainers/contextTabs';

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
  DISCONNECT: () => void;
};

export const Lobby = (props: Props) => {
  const { useState } = useMountedState();
  const { sendSocketMsg } = props;
  const { loginState } = useLoginStateStoreContext();

  const { loggedIn, username, userID } = loginState;

  const [selectedGameTab, setSelectedGameTab] = useState(
    loggedIn ? 'PLAY' : 'WATCH'
  );

  useEffect(() => {
    setSelectedGameTab(loggedIn ? 'PLAY' : 'WATCH');
  }, [loggedIn]);

  const handleNewGame = useCallback(
    (seekID: string) => {
      sendAccept(seekID, sendSocketMsg);
    },
    [sendSocketMsg]
  );
  const onSeekSubmit = useCallback(
    (g: SoughtGame) => {
      console.log('sought game', g);
      sendSeek(g, sendSocketMsg);
    },
    [sendSocketMsg]
  );

  return (
    <SideMenuContextProvider defaultActivePanelKey={selectedGameTab}>
      <TopBar />
      <div className="lobby">
        <ContextTabs>
          <ContextTab panelKey={'PLAY'} label="Play" />
        </ContextTabs>
        <SideMenu>
          <PanelComponentWrapper panelKey="CHAT" className="chat-area">
            <Chat
              sendChat={props.sendChat}
              defaultChannel="chat.lobby"
              defaultDescription="Help chat"
              DISCONNECT={props.DISCONNECT}
            />
          </PanelComponentWrapper>
        </SideMenu>
        <Main>
          <GameLists
            loggedIn={loggedIn}
            userID={userID}
            username={username}
            newGame={handleNewGame}
            selectedGameTab={selectedGameTab}
            setSelectedGameTab={setSelectedGameTab}
            onSeekSubmit={onSeekSubmit}
          />
        </Main>
        <Sidebar className="announcements">
          <PanelComponentWrapper panelKey="ANNOUNCEMENTS">
            <Announcements />
          </PanelComponentWrapper>
        </Sidebar>
      </div>
    </SideMenuContextProvider>
  );
};
