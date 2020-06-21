import React, { useLayoutEffect, useState, useEffect, useRef } from 'react';
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom';
import './App.scss';
import 'antd/dist/antd.css';
import useWebSocket, { SendMessage } from 'react-use-websocket';
import axios from 'axios';
import jwt from 'jsonwebtoken';

import { Table } from './gameroom/table';
import { Lobby } from './lobby/lobby';
import { useStoreContext } from './store/store';

import { getSocketURI } from './socket/socket';
import { decodeToMsg, encodeToSocketFmt } from './utils/protobuf';
import { onSocketMsg } from './store/socket_handlers';
import { Login } from './lobby/login';
import { Register } from './lobby/register';
import { MessageType, TokenSocketLogin } from './gen/api/proto/game_service_pb';

function useWindowSize() {
  const [size, setSize] = useState([0, 0]);
  useLayoutEffect(() => {
    function updateSize() {
      setSize([window.innerWidth, window.innerHeight]);
    }
    window.addEventListener('resize', updateSize);
    updateSize();
    return () => window.removeEventListener('resize', updateSize);
  }, []);
  return size;
}

type TokenResponse = {
  token: string;
};

type DecodedToken = {
  unn: string;
  uid: string;
  iss: string;
};

const App = () => {
  const [width, height] = useWindowSize();
  const [err, setErr] = useState('');
  const [username, setUsername] = useState('');
  const [loggedIn, setLoggedIn] = useState(false);
  console.log('rendering ap');

  useEffect(() => {
    // XXX: This fetches the socket token twice, when we first render,
    // and when the "loggedIn" state changes because of the token.
    // if (loggedIn) {
    //   return;
    // }
    console.log('fetching socket token...');
    axios
      .post<TokenResponse>(
        '/twirp/liwords.AuthenticationService/GetSocketToken',
        {}
      )
      .then((resp) => {
        const decoded = jwt.decode(resp.data.token) as DecodedToken;
        setUsername(decoded.unn);
        setLoggedIn(true);
        const msg = new TokenSocketLogin();
        msg.setToken(resp.data.token);
        // Decoding the token logs us in, and we also send the token to
        // the socket server to identify ourselves there as well.
        sendMessage(
          encodeToSocketFmt(
            MessageType.TOKEN_SOCKET_LOGIN,
            msg.serializeBinary()
          )
        );
      })
      .catch((e) => {
        if (e.response) {
          setErr(e.response.data.msg);
          console.log(e.response);
        }
      });
    // We want to do this when we change the "loggedin" state.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loggedIn]);

  // const loggedIn = props.username !== 'anonymous';
  const store = useStoreContext();
  const socketUrl = getSocketURI();
  const { sendMessage } = useWebSocket(socketUrl, {
    onOpen: () => console.log('connected to socket'),
    // Will attempt to reconnect on all close events, such as server shutting down
    shouldReconnect: (closeEvent) => true,
    onMessage: (event: MessageEvent) =>
      decodeToMsg(event.data, onSocketMsg(store)),
  });

  return (
    <div className="App">
      <Router>
        <Switch>
          <Route path="/" exact>
            <Lobby
              username={username}
              sendSocketMsg={sendMessage}
              loggedIn={loggedIn}
            />
          </Route>
          <Route path="/game/:gameID">
            {/* Table meaning a game table */}
            <Table
              windowWidth={width}
              windowHeight={height}
              sendSocketMsg={sendMessage}
              username={username}
              loggedIn={loggedIn}
            />
          </Route>

          <Route path="/login">
            <Login setLoggedIn={setLoggedIn} loggedIn={loggedIn} />
          </Route>
          <Route path="/register">
            <Register setLoggedIn={setLoggedIn} loggedIn={loggedIn} />
          </Route>
        </Switch>
      </Router>
    </div>
  );
};

export default App;
