import React, { useLayoutEffect, useState } from 'react';
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom';
import { SoughtGame, Lobby } from './lobby/lobby';

import './App.css';
import 'antd/dist/antd.css';

import { Table } from './gameroom/table';

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

const App = () => {
  const [width, height] = useWindowSize();
  const [soughtGames, setSoughtGames] = useState(new Array<SoughtGame>());

  const onConnect = (username: string) => {
    console.log('connecting with username', username);
    const socket = new WebSocket('ws://vm.xword.club/ws');
    socket.addEventListener('open', (event) => {
      console.log('connected!');
    });

    // Listen for messages
    socket.addEventListener('message', (event) => {
      console.log('Message from server ', event.data);

      if (event.data instanceof Blob) {
        const reader = new FileReader();

        reader.onload = () => {
          console.log(`Result: ${reader.result}`);
          const result = JSON.parse(reader.result as string) as SoughtGame;

          const soughtGamesCopy = [...soughtGames];
          console.log(result, typeof result);
          soughtGamesCopy.push(result);
          console.log(soughtGamesCopy);
          setSoughtGames(soughtGamesCopy);
        };

        reader.readAsText(event.data);
      } else {
        console.log(`Result: ${event.data}`);
      }
    });
  };
  return (
    <div className="App">
      <Router>
        <Switch>
          <Route path="/" exact>
            <Lobby soughtGames={soughtGames} onConnect={onConnect} />
          </Route>

          <Route path="/game/:gameID">
            <Table windowWidth={width} windowHeight={height} />
          </Route>
        </Switch>
      </Router>
    </div>
  );
};

export default App;
