import React, { useCallback, useEffect, useRef, useState } from "react";

import { TopBar } from "../navigation/topbar";

import { SoughtGame } from "../store/reducers/lobby_reducer";
import { GameLists } from "./gameLists";
import { Chat } from "../chat/chat";
import { useLoginStateStoreContext } from "../store/store";
import "./lobby.scss";
import { AnnouncementsWidget } from "./announcements";
import { sendAccept, sendSeek } from "./sought_game_interactions";
import { PuzzlePreview } from "../puzzles/puzzle_preview";
import { ConfigProvider } from "antd";

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
  DISCONNECT: () => void;
};

const LOBBY_TAB_STORAGE_KEY = "lastLobbyTab";

const getInitialTab = (loggedIn: boolean): string => {
  if (!loggedIn) return "WATCH";

  const savedTab = localStorage.getItem(LOBBY_TAB_STORAGE_KEY);
  if (savedTab === "CORRESPONDENCE") {
    return "CORRESPONDENCE";
  }

  return "PLAY";
};

export const Lobby = (props: Props) => {
  const { sendSocketMsg } = props;
  const { loginState } = useLoginStateStoreContext();

  const { loggedIn, username, userID } = loginState;

  const [selectedGameTab, setSelectedGameTabState] = useState(
    getInitialTab(loggedIn),
  );
  const prevLoggedIn = useRef(loggedIn);

  // Wrapper that saves to localStorage when tab changes
  const setSelectedGameTab = useCallback((tab: string) => {
    setSelectedGameTabState(tab);
    localStorage.setItem(LOBBY_TAB_STORAGE_KEY, tab);
  }, []);

  // Update tab when login state changes
  useEffect(() => {
    const wasLoggedIn = prevLoggedIn.current;
    prevLoggedIn.current = loggedIn;

    if (!wasLoggedIn && loggedIn) {
      // Login state just loaded - restore from localStorage
      setSelectedGameTabState(getInitialTab(true));
    } else if (wasLoggedIn && !loggedIn) {
      // User logged out - switch to WATCH
      setSelectedGameTabState("WATCH");
    }
  }, [loggedIn]);

  const handleNewGame = useCallback(
    (seekID: string) => {
      sendAccept(seekID, sendSocketMsg);
    },
    [sendSocketMsg],
  );
  const onSeekSubmit = useCallback(
    (g: SoughtGame) => {
      console.log("sought game", g);
      sendSeek(g, sendSocketMsg);
    },
    [sendSocketMsg],
  );

  return (
    <>
      <TopBar />
      <div className="lobby">
        <div className="chat-area">
          <Chat
            sendChat={props.sendChat}
            defaultChannel="chat.lobby"
            defaultDescription="Help chat"
            DISCONNECT={props.DISCONNECT}
          />
        </div>
        <ConfigProvider
          theme={{
            components: {
              Dropdown: {
                paddingBlock: 5,
                paddingXS: 0,
                paddingXXS: 0,
              },
            },
          }}
        >
          <GameLists
            loggedIn={loggedIn}
            userID={userID}
            username={username}
            newGame={handleNewGame}
            selectedGameTab={selectedGameTab}
            setSelectedGameTab={setSelectedGameTab}
            onSeekSubmit={onSeekSubmit}
          />
        </ConfigProvider>
        <div className="announcements">
          <AnnouncementsWidget />
          <PuzzlePreview />
        </div>
      </div>
    </>
  );
};
