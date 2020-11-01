import React from 'react';
import { Link } from 'react-router-dom';
import { useMountedState } from '../utils/mounted';
import './topbar.scss';
import { DisconnectOutlined, SettingOutlined } from '@ant-design/icons/lib';
import { notification, Dropdown, Tooltip, Modal, Badge } from 'antd';
import axios from 'axios';
import {
  useLagStoreContext,
  useLoginStateStoreContext,
  useResetStoreContext,
} from '../store/store';
import { toAPIUrl } from '../api/api';
import { Login } from '../lobby/login';

const colors = require('../base.scss');

const TopMenu = React.memo((props: Props) => {
  const { resetStore } = useResetStoreContext();

  return (
    <div className="top-header-menu">
      <div className="top-header-left-frame-crossword-game">
        <Link to="/" onClick={resetStore}>
          OMGWords
        </Link>
      </div>
      <div className="top-header-left-frame-aerolith">
        <a href="https://aerolith.org">Aerolith</a>
      </div>
      <div className="top-header-left-frame-blog">
        <a href="http://randomracer.com">Random.Racer</a>
      </div>
      <div className="top-header-left-frame-special-land">
        <Link to="/about" onClick={resetStore}>
          About Us
        </Link>
      </div>
    </div>
  );
});

type Props = {
  tournamentID?: string;
};

export const TopBar = React.memo((props: Props) => {
  const { useState } = useMountedState();

  const { currentLagMs } = useLagStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { resetStore } = useResetStoreContext();
  const { username, loggedIn, connectedToSocket } = loginState;
  const [loginModalVisible, setLoginModalVisible] = useState(false);

  const handleLogout = (e: React.MouseEvent) => {
    e.preventDefault();
    axios
      .post(toAPIUrl('user_service.AuthenticationService', 'Logout'), {
        withCredentials: true,
      })
      .then(() => {
        notification.info({
          message: 'Success',
          description: 'You have been logged out.',
        });
        resetStore();
      })
      .catch((e) => {
        console.log(e);
      });
  };
  const userMenu = (
    <ul>
      <li>
        <Link
          className="plain"
          to={`/profile/${encodeURIComponent(username)}`}
          onClick={resetStore}
        >
          View Profile
        </Link>
      </li>
      <li onClick={handleLogout} className="link plain">
        Log out
      </li>
    </ul>
  );

  const homeLink = props.tournamentID
    ? `/tournament/${props.tournamentID}`
    : '/';

  return (
    <nav className="top-header" id="main-nav">
      <div className="container">
        <Tooltip
          placement="bottomLeft"
          color={colors.colorPrimary}
          title={`Latency: ${currentLagMs || '...'} ms.`}
        >
          <Link to={homeLink} className="site-icon" onClick={resetStore}>
            <div className="top-header-site-icon-rect">
              <div className="top-header-site-icon-m">W</div>
            </div>

            <div className="top-header-left-frame-site-name">Woogles.io</div>
          </Link>
        </Tooltip>
        <TopMenu />
        {loggedIn ? (
          <div className="user-info">
            <Dropdown
              overlayClassName="user-menu"
              overlay={userMenu}
              placement="bottomRight"
            >
              <button className="link">
                {username}
                <SettingOutlined />
              </button>
            </Dropdown>
            {!connectedToSocket ? (
              <DisconnectOutlined style={{ color: 'red', marginLeft: 5 }} />
            ) : null}
          </div>
        ) : (
          <div className="user-info">
            <button className="link" onClick={() => setLoginModalVisible(true)}>
              Log In
            </button>
            <Link to="/register" onClick={resetStore}>
              <button className="primary">Sign Up</button>
            </Link>
            <Modal
              className="login-modal"
              title="Welcome back, friend!"
              visible={loginModalVisible}
              onCancel={() => {
                setLoginModalVisible(false);
              }}
              footer={null}
              width={332}
            >
              <Login />
            </Modal>
          </div>
        )}
      </div>
    </nav>
  );
});
