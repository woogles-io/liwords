import React, { useEffect, useState } from 'react';
import { Route, Switch } from 'react-router-dom';
import './App.scss';
import axios from 'axios';
import 'antd/dist/antd.css';

import { Table } from './gameroom/table';
import { Lobby } from './lobby/lobby';
import { useStoreContext } from './store/store';

import { useLiwordsSocket } from './socket/socket';
import { About } from './about/about';
import { Login } from './lobby/login';
import { Register } from './lobby/register';
import { UserProfile } from './profile/profile';
import { PasswordChange } from './lobby/password_change';
import { PasswordReset } from './lobby/password_reset';
import { NewPassword } from './lobby/new_password';
import { toAPIUrl } from './api/api';

type BasicUser = {
  uuid: string;
  username: string;
};

type Blocks = {
  users: Array<BasicUser>;
};

const App = React.memo(() => {
  const store = useStoreContext();
  const [shouldDisconnect, setShouldDisconnect] = useState(false);
  const { sendMessage } = useLiwordsSocket(shouldDisconnect);

  if (store.redirGame !== '') {
    store.setRedirGame('');
    window.location.replace(`/game/${store.redirGame}`);
  }

  const disconnectSocket = () => {
    setShouldDisconnect(true);
    setTimeout(() => {
      // reconnect after 5 seconds.
      setShouldDisconnect(false);
    }, 5000);
  };

  useEffect(() => {
    axios
      .post<Blocks>(
        toAPIUrl('user_service.SocializeService', 'GetFullBlocks'),
        { foo: 'bar' },
        { withCredentials: true, headers: { contentLength: 0 } }
      )
      .then((resp) => {
        store.setExcludedPlayers(resp.data.users.map((u) => u.uuid));
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="App">
      <Switch>
        <Route path="/" exact>
          <Lobby sendSocketMsg={sendMessage} DISCONNECT={disconnectSocket} />
        </Route>
        <Route path="/game/:gameID">
          {/* Table meaning a game table */}
          <Table sendSocketMsg={sendMessage} />
        </Route>
        <Route path="/about">
          <About />
        </Route>
        <Route path="/login">
          <Login />
        </Route>
        <Route path="/secretwoogles">
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
      </Switch>
    </div>
  );
});

export default App;
