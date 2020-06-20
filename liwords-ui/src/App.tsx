import React, { useLayoutEffect, useState, useEffect } from 'react';
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom';
import './App.scss';
import 'antd/dist/antd.css';
import useWebSocket from 'react-use-websocket';
import axios from 'axios';
import jwt from 'jsonwebtoken';

import { Table } from './gameroom/table';
import { Lobby } from './lobby/lobby';
import { useStoreContext } from './store/store';

import { getSocketURI } from './socket/socket';
import { decodeToMsg } from './utils/protobuf';
import { onSocketMsg } from './store/socket_handlers';
import { Login } from './lobby/login';
import { Register } from './lobby/register';

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
  // XXX: change to look at the cookie.
  const [loggedIn, setLoggedIn] = useState(false);
  console.log('rendering ap');
  useEffect(() => {
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
      })
      .catch((e) => {
        if (e.response) {
          setErr(e.response.data.msg);
          console.log(e.response);
        }
      });
  }, [loggedIn]);

  // const loggedIn = props.username !== 'anonymous';
  const store = useStoreContext();
  // XXX: change to JWT
  const socketUrl = getSocketURI('foo');
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
