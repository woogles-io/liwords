import React from 'react';
import axios from 'axios';

import { Link } from 'react-router-dom';
import './topbar.scss';
import { DisconnectOutlined, SettingOutlined } from '@ant-design/icons/lib';
import { notification, Dropdown, Tooltip } from 'antd';
import {
  useLagStoreContext,
  useLoginStateStoreContext,
  useResetStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { toAPIUrl } from '../api/api';
import { LoginModal } from '../lobby/login';
import { useMountedState } from '../utils/mounted';
import { isClubType } from '../store/constants';

const colors = require('../base.scss');

const TopMenu = React.memo((props: Props) => {
  const aboutMenu = (
    <ul>
      <li>
        <Link className="plain" to="/team">
          Meet the team
        </Link>
      </li>
      <li>
        <Link className="plain" to="/terms">
          Terms of Service
        </Link>
      </li>
    </ul>
  );
  return (
    <div className="top-header-menu">
      <div className="top-header-left-frame-crossword-game">
        <Link to="/">OMGWords</Link>
      </div>
      <div className="top-header-left-frame-aerolith">
        <a
          href="https://aerolith.org"
          target="_blank"
          rel="noopener noreferrer"
        >
          Aerolith
        </a>
      </div>
      <div className="top-header-left-frame-blog">
        <a
          href="http://randomracer.com"
          target="_blank"
          rel="noopener noreferrer"
        >
          Random.Racer
        </a>
      </div>
      <div className="top-header-left-frame-special-land">
        <Dropdown
          overlayClassName="user-menu"
          overlay={aboutMenu}
          placement="bottomCenter"
          getPopupContainer={() =>
            document.getElementById('root') as HTMLElement
          }
        >
          <p>About Us</p>
        </Dropdown>
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
  const { resetLoginStateStore } = useResetStoreContext();
  const { tournamentContext } = useTournamentStoreContext();
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
        resetLoginStateStore();
      })
      .catch((e) => {
        console.log(e);
      });
  };
  const userMenu = (
    <ul>
      <li>
        <Link className="plain" to={`/profile/${encodeURIComponent(username)}`}>
          View profile
        </Link>
      </li>
      <li>
        <Link className="plain" to={`/settings`}>
          Settings
        </Link>
      </li>
      <li onClick={handleLogout} className="link plain">
        Log out
      </li>
    </ul>
  );

  const homeLink = props.tournamentID
    ? tournamentContext.metadata?.getSlug()
    : '/';

  return (
    <nav className="top-header" id="main-nav">
      <div className="container">
        <Tooltip
          placement="bottomLeft"
          color={colors.colorPrimary}
          title={`Latency: ${currentLagMs || '...'} ms.`}
        >
          <Link
            to={homeLink}
            className={`site-icon${
              props.tournamentID ? ' tournament-mode' : ''
            }`}
          >
            <div className="top-header-site-icon-rect">
              <div className="top-header-site-icon-m">W</div>
            </div>

            <div className="top-header-left-frame-site-name">Woogles.io</div>
            {props.tournamentID ? (
              <div className="tournament">
                Back to
                {isClubType(tournamentContext.metadata?.getType())
                  ? ' Club'
                  : ' Tournament'}
              </div>
            ) : null}
          </Link>
        </Tooltip>
        <TopMenu />
        {loggedIn ? (
          <div className="user-info">
            <Dropdown
              overlayClassName="user-menu"
              overlay={userMenu}
              placement="bottomRight"
              getPopupContainer={() =>
                document.getElementById('root') as HTMLElement
              }
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
            <Link to="/register">
              <button className="primary">Sign Up</button>
            </Link>
            <LoginModal {...{ loginModalVisible, setLoginModalVisible }} />
          </div>
        )}
      </div>
    </nav>
  );
});
