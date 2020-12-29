import React, { useCallback, useEffect, useRef } from 'react';
import { Route, Switch, useLocation } from 'react-router-dom';
import { useMountedState } from './utils/mounted';
import './App.scss';
import axios from 'axios';
import 'antd/dist/antd.css';

import { Table as GameTable } from './gameroom/table';
import TileImages from './gameroom/tile_images';
import { Lobby } from './lobby/lobby';
import {
  useExcludedPlayersStoreContext,
  useResetStoreContext,
} from './store/store';

import { LiwordsSocket } from './socket/socket';
import { About } from './about/about';
import { Register } from './lobby/register';
import { UserProfile } from './profile/profile';
import { PasswordChange } from './lobby/password_change';
import { PasswordReset } from './lobby/password_reset';
import { NewPassword } from './lobby/new_password';
import { toAPIUrl } from './api/api';
import { ChatMessage, MessageType } from './gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from './utils/protobuf';
import { Clubs } from './clubs';
import { TournamentRoom } from './tournament/room';

type Blocks = {
  user_ids: Array<string>;
};

const useDarkMode = localStorage?.getItem('darkMode') === 'true';

document?.body?.classList?.add(`mode--${useDarkMode ? 'dark' : 'default'}`);

const App = React.memo(() => {
  const { useState } = useMountedState();

  const {
    setExcludedPlayers,
    setExcludedPlayersFetched,
    pendingBlockRefresh,
    setPendingBlockRefresh,
  } = useExcludedPlayersStoreContext();
  const { resetStore } = useResetStoreContext();

  // See store.tsx for how this works.
  const [socketId, setSocketId] = useState(0);
  const resetSocket = useCallback(() => setSocketId((n) => (n + 1) | 0), []);

  const [liwordsSocketValues, setLiwordsSocketValues] = useState({
    sendMessage: (msg: Uint8Array) => {},
    justDisconnected: false,
  });
  const { sendMessage } = liwordsSocketValues;

  const location = useLocation();
  const knownLocation = useRef(location.pathname); // Remember the location on first render.
  const isCurrentLocation = knownLocation.current === location.pathname;
  useEffect(() => {
    if (!isCurrentLocation) {
      resetStore();
    }
  }, [isCurrentLocation, resetStore]);

  const getFullBlocks = useCallback(() => {
    axios
      .post<Blocks>(
        toAPIUrl('user_service.SocializeService', 'GetFullBlocks'),
        {},
        { withCredentials: true }
      )
      .then((resp) => {
        setExcludedPlayers(new Set<string>(resp.data.user_ids));
      })
      .catch((e) => {
        console.log(e);
      })
      .finally(() => {
        setExcludedPlayersFetched(true);
        setPendingBlockRefresh(false);
      });
  }, [setExcludedPlayers, setExcludedPlayersFetched, setPendingBlockRefresh]);

  useEffect(() => {
    getFullBlocks();
  }, [getFullBlocks]);

  useEffect(() => {
    if (pendingBlockRefresh) {
      getFullBlocks();
    }
  }, [getFullBlocks, pendingBlockRefresh]);

  const sendChat = useCallback(
    (msg: string, chan: string) => {
      const evt = new ChatMessage();
      evt.setMessage(msg);

      // const chan = isObserver ? 'gametv' : 'game';
      // evt.setChannel(`chat.${chan}.${gameID}`);
      evt.setChannel(chan);
      sendMessage(
        encodeToSocketFmt(MessageType.CHAT_MESSAGE, evt.serializeBinary())
      );
    },
    [sendMessage]
  );

  // Avoid useEffect in the new path triggering xhr twice.
  if (!isCurrentLocation) return null;

  return (
    <div className="App">
      <LiwordsSocket
        key={socketId}
        resetSocket={resetSocket}
        setValues={setLiwordsSocketValues}
      />
      <Switch>
        <Route path="/" exact>
          <Lobby
            sendSocketMsg={sendMessage}
            sendChat={sendChat}
            DISCONNECT={resetSocket}
          />
        </Route>
        <Route path="/tournament/:partialSlug">
          <TournamentRoom sendSocketMsg={sendMessage} sendChat={sendChat} />
        </Route>
        <Route path="/club/:partialSlug">
          <TournamentRoom sendSocketMsg={sendMessage} sendChat={sendChat} />
        </Route>
        <Route path="/clubs">
          <Clubs />
        </Route>
        <Route path="/game/:gameID">
          {/* Table meaning a game table */}
          <GameTable sendSocketMsg={sendMessage} sendChat={sendChat} />
        </Route>
        <Route path="/about">
          <About />
        </Route>
        <Route path="/register">
          <Register />
        </Route>
        <Route path="/password/change">
          <PasswordChange />
        </Route>
        <Route path="/password/reset">
          <PasswordReset />
        </Route>

        <Route path="/password/new">
          <NewPassword />
        </Route>

        <Route path="/profile/:username">
          <UserProfile />
        </Route>
        <Route path="/tile_images">
          <TileImages />
        </Route>
      </Switch>
    </div>
  );
});

export default App;
