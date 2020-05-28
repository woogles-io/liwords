import React, { useLayoutEffect, useState } from 'react';
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom';
import './App.css';
import 'antd/dist/antd.css';

import { Table } from './gameroom/table';
import { Lobby } from './lobby/lobby';
import { LobbyStore } from './store/lobby_store';

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

const App = (props: Props) => {
  const [width, height] = useWindowSize();

  return (
    <div className="App">
      <Router>
        <Switch>
          {/* Note that the Lobby and Table routes will have their own websockets */}
          <Route path="/" exact>
            <LobbyStore>
              <Lobby username={props.username} />
            </LobbyStore>
          </Route>
          <Route path="/game/:gameID">
            {/* Table meaning a game table */}
            <Table windowWidth={width} windowHeight={height} />
          </Route>
        </Switch>
      </Router>
    </div>
  );
};

export default App;
