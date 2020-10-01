import React from 'react';
import { Route, Switch } from 'react-router-dom';
import './App.scss';
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

const App = React.memo(() => {
  const store = useStoreContext();
  const {
    username,
    userID,
    connID,
    sendMessage,
    loggedIn,
    connectedToSocket,
  } = useLiwordsSocket();

  if (store.redirGame !== '') {
    store.setRedirGame('');
    window.location.replace(`/game/${store.redirGame}`);
  }

  return (
    <div className="App">
      <Switch>
        <Route path="/" exact>
          <Lobby
            username={username}
            userID={userID}
            connID={connID}
            sendSocketMsg={sendMessage}
            loggedIn={loggedIn}
            connectedToSocket={connectedToSocket}
          />
        </Route>
        <Route path="/game/:gameID">
          {/* Table meaning a game table */}
          <Table
            sendSocketMsg={sendMessage}
            username={username}
            connID={connID}
            loggedIn={loggedIn}
            // can use some visual indicator to show the user if they disconnected
            connectedToSocket={connectedToSocket}
          />
        </Route>
        <Route path="/about">
          <About
            myUsername={username}
            loggedIn={loggedIn}
            connectedToSocket={connectedToSocket}
          />
        </Route>
        <Route path="/login">
          <Login />
        </Route>
        <Route path="/secretwoogles">
          <Register />
        </Route>
        <Route path="/password/change">
          <PasswordChange
            username={username}
            loggedIn={loggedIn}
            connectedToSocket={connectedToSocket}
          />
        </Route>
        <Route path="/password/reset">
          <PasswordReset />
        </Route>

        <Route path="/password/new">
          <NewPassword
            username={username}
            loggedIn={loggedIn}
            connectedToSocket={connectedToSocket}
          />
        </Route>

        <Route path="/profile/:username">
          <UserProfile
            myUsername={username}
            loggedIn={loggedIn}
            connectedToSocket={connectedToSocket}
          />
        </Route>
      </Switch>
    </div>
  );
});

export default App;
