import React, { useState } from 'react';
import './topbar.scss';
import { DisconnectOutlined, SettingOutlined } from '@ant-design/icons/lib';
import { notification, Dropdown, Tooltip, Modal } from 'antd';
import {
  useLagStoreContext,
  useLoginStateStoreContext,
  useResetStoreContext,
} from '../store/store';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { Login } from '../lobby/login';

const colors = require('../base.scss');
const topMenu = (
  <div className="top-header-menu">
    <div className="top-header-left-frame-crossword-game">
      <a href="/">OMGWords</a>
    </div>
    <div className="top-header-left-frame-aerolith">
      <a href="https://aerolith.org">Aerolith</a>
    </div>
    <div className="top-header-left-frame-blog">
      <a href="http://randomracer.com">Random.Racer</a>
    </div>
    <div className="top-header-left-frame-special-land">
      <a href="/about">About Us</a>
    </div>
  </div>
);

type Props = {};

export const TopBar = React.memo((props: Props) => {
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
        <a className="plain" href={`/profile/${encodeURIComponent(username)}`}>
          View Profile
        </a>
      </li>
      <li onClick={handleLogout} className="link plain">
        Log out
      </li>
    </ul>
  );
  return (
    <nav className="top-header" id="main-nav">
      <div className="container">
        <Tooltip
          placement="bottomLeft"
          color={colors.colorPrimary}
          title={`Latency: ${currentLagMs || '...'} ms.`}
        >
          <a href="/" className="site-icon">
            <div className="top-header-site-icon-rect">
              <div className="top-header-site-icon-m">W</div>
            </div>
            <div className="top-header-left-frame-site-name">Woogles.io</div>
          </a>
        </Tooltip>
        {topMenu}
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
            <a href="/register">
              <button className="primary">Sign Up</button>
            </a>
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
