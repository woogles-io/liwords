import React, { useEffect, useState } from 'react';
import { Route, Switch, useLocation, Redirect } from 'react-router-dom';
import './App.scss';
import 'antd/dist/antd.css';
import useWebSocket from 'react-use-websocket';
import axios from 'axios';
import jwt from 'jsonwebtoken';

import { Table } from './gameroom/table';
import { Lobby } from './lobby/lobby';
import { useStoreContext } from './store/store';

import { getSocketURI } from './socket/socket';
import { decodeToMsg, encodeToSocketFmt } from './utils/protobuf';
import { onSocketMsg } from './store/socket_handlers';
import { About } from './about/about';
import { Login } from './lobby/login';
import { Register } from './lobby/register';
import { MessageType, JoinPath } from './gen/api/proto/realtime/realtime_pb';
import { UserProfile } from './profile/profile';
import { PasswordChange } from './lobby/password_change';

type TokenResponse = {
  token: string;
};

type DecodedToken = {
  unn: string;
  uid: string;
  a: boolean; // authed
};

const App = React.memo(() => {
  const socketUrl = getSocketURI();
  const store = useStoreContext();

  const [socketToken, setSocketToken] = useState('');
  const [username, setUsername] = useState('Anonymous');
  const [userID, setUserID] = useState('');
  const [loggedIn, setLoggedIn] = useState(false);
  const [connectedToSocket, setConnectedToSocket] = useState(false);
  const [justDisconnected, setJustDisconnected] = useState(false);

  useEffect(() => {
    if (connectedToSocket) {
      // Only call this function if we are not connected to the socket.
      // If we go from unconnected to connected, there is no need to call
      // it again. If we go from connected to unconnected, then we call it
      // to fetch a new token.
      return;
    }

    axios
      .post<TokenResponse>(
        '/twirp/user_service.AuthenticationService/GetSocketToken',
        {}
      )
      .then((resp) => {
        setSocketToken(resp.data.token);
        const decoded = jwt.decode(resp.data.token) as DecodedToken;
        setUsername(decoded.unn);
        setUserID(decoded.uid);
        setLoggedIn(decoded.a);
        console.log('Got token, setting state');
      })
      .catch((e) => {
        if (e.response) {
          window.console.log(e.response);
        }
      });
  }, [connectedToSocket]);

  const { sendMessage } = useWebSocket(
    `${socketUrl}?token=${socketToken}`,
    {
      onOpen: () => {
        console.log('connected to socket');
        setConnectedToSocket(true);
        setJustDisconnected(false);
      },
      onClose: () => {
        console.log('disconnected from socket :(');
        setConnectedToSocket(false);
        setJustDisconnected(true);
      },
      retryOnError: true,
      // Will attempt to reconnect on all close events, such as server shutting down
      shouldReconnect: (closeEvent) => true,
      onMessage: (event: MessageEvent) =>
        decodeToMsg(event.data, onSocketMsg(username, store)),
    },
    socketToken !== '' /* only connect if the socket token is not null */
  );

  const location = useLocation();

  useEffect(() => {
    console.log('location pathname change, now', location.pathname);
    const rr = new JoinPath();
    rr.setPath(location.pathname);
    console.log('Tryna register with path', location.pathname);
    sendMessage(encodeToSocketFmt(MessageType.JOIN_PATH, rr.serializeBinary()));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname, justDisconnected]);

  if (store.redirGame !== '') {
    store.setRedirGame('');
    return <Redirect push to={`/game/${store.redirGame}`} />;
  }

  return (
    <div className="App">
      <Switch>
        <Route path="/" exact>
          <Lobby
            username={username}
            userID={userID}
            sendSocketMsg={sendMessage}
            loggedIn={loggedIn}
            // connectedToSocket={connectedToSocket}
          />
        </Route>
        <Route path="/game/:gameID">
          {/* Table meaning a game table */}
          <Table
            sendSocketMsg={sendMessage}
            username={username}
            loggedIn={loggedIn}
            // can use some visual indicator to show the user if they disconnected
            // connectedToSocket={connectedToSocket}
          />
        </Route>
        <Route path="/about">
          <About myUsername={username} loggedIn={loggedIn} />
        </Route>
        <Route path="/login">
          <Login />
        </Route>
        <Route path="/secretwoogles">
          <Register />
        </Route>
        <Route path="/password/change">
          <PasswordChange username={username} loggedIn={loggedIn} />
        </Route>
        <Route path="/profile/:username">
          <UserProfile myUsername={username} loggedIn={loggedIn} />
        </Route>
      </Switch>
    </div>
  );
});

export default App;
