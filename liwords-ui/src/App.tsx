import React, { useLayoutEffect, useState, useRef, useEffect } from 'react';
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom';
import './App.css';
import 'antd/dist/antd.css';

import { Table } from './gameroom/table';
import { Lobby } from './lobby/lobby';
import { useStoreContext, StoreData } from './store/store';

import { getSocketURI, websocket } from './socket/socket';
import { decodeToMsg } from './utils/protobuf';
import { onSocketMsg } from './store/socket_handlers';

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

type Props = {
  username: string;
};

function useSocket(
  socketRef: React.MutableRefObject<WebSocket | null>,
  storeData: StoreData,
  username: string
) {
  const storeRef = useRef(storeData);
  storeRef.current = storeData;

  useEffect(() => {
    websocket(
      getSocketURI(username),
      (socket) => {
        // eslint-disable-next-line no-param-reassign
        socketRef.current = socket;
      },
      (event) => {
        decodeToMsg(event.data, onSocketMsg(storeRef.current));
      }
    );
    return () => {
      if (socketRef.current) {
        console.log('closing socket');
        socketRef.current.close(1000);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
}

const sendSocketMsg = (msg: Uint8Array, socket: WebSocket | null) => {
  if (!socket) {
    // reconnect?
    return;
  }
  socket.send(msg);
};

const App = (props: Props) => {
  const [width, height] = useWindowSize();
  const socketRef = useRef<WebSocket | null>(null);
  const lobbyStoreFns = useStoreContext();
  useSocket(socketRef, lobbyStoreFns, props.username);

  return (
    <div className="App">
      <Router>
        <Switch>
          <Route path="/" exact>
            <Lobby
              username={props.username}
              sendSocketMsg={(msg: Uint8Array) =>
                sendSocketMsg(msg, socketRef.current)
              }
            />
          </Route>
          <Route path="/game/:gameID">
            {/* Table meaning a game table */}
            <Table
              windowWidth={width}
              windowHeight={height}
              sendSocketMsg={(msg: Uint8Array) =>
                sendSocketMsg(msg, socketRef.current)
              }
              username={props.username}
            />
          </Route>
        </Switch>
      </Router>
    </div>
  );
};

export default App;
