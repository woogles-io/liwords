import React, { ReactNode, ReactNodeArray } from 'react';
import axios from 'axios';

import { Link } from 'react-router-dom';
import './topbar.scss';
import {
  DisconnectOutlined,
  MenuOutlined,
  SettingOutlined,
} from '@ant-design/icons/lib';
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
import { Header } from 'antd/lib/layout/layout';
import { useSideMenuContext } from '../shared/layoutContainers/menu';

const colors = require('../base.scss');

export const aboutMenuItems = [
  <Link className="plain" to="/team">
    Meet the team
  </Link>,
  <Link className="plain" to="/terms">
    Terms of Service
  </Link>,
];

const aboutMenu = (
  <ul>
    {aboutMenuItems.map((item) => (
      <li>{item}</li>
    ))}
  </ul>
);

export const offsiteMenuItems = [
  <Link to="/">OMGWords</Link>,
  <a href="https://aerolith.org" target="_blank" rel="noopener noreferrer">
    Aerolith
  </a>,
  <a href="http://randomracer.com" target="_blank" rel="noopener noreferrer">
    Random.Racer
  </a>,
];

const offsiteLinks = (
  <>
    {offsiteMenuItems.map((item) => (
      <div>{item}</div>
    ))}
  </>
);

type UserMenuProps = {
  username: string;
  handleLogout: (e: React.MouseEvent) => void;
};

const userMenuItems = (props: UserMenuProps): ReactNodeArray => [
  <Link className="plain" to={`/profile/${encodeURIComponent(props.username)}`}>
    View profile
  </Link>,
  <Link className="plain" to={`/settings`}>
    Settings
  </Link>,
  <span onClick={props.handleLogout} className="link plain">
    Log out
  </span>,
];

const userMenu = (props: UserMenuProps) => (
  <ul>
    {userMenuItems(props).map((item) => (
      <li>{item}</li>
    ))}
  </ul>
);

const TopMenu = (
  <div className="top-header-menu">
    {offsiteLinks}
    <div className="top-header-left-frame-special-land">
      <Dropdown
        overlayClassName="user-menu"
        overlay={aboutMenu}
        placement="bottomCenter"
        getPopupContainer={() => document.getElementById('root') as HTMLElement}
      >
        <p>About Us</p>
      </Dropdown>
    </div>
  </div>
);

type Props = {
  tournamentID?: string;
};

export const TopBar = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const { currentLagMs } = useLagStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { resetStore } = useResetStoreContext();
  const { tournamentContext } = useTournamentStoreContext();
  const { isOpen, setIsOpen } = useSideMenuContext();
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

  const homeLink = props.tournamentID
    ? tournamentContext.metadata?.getSlug()
    : '/';

  return (
    <Header className="top-header" id="main-nav">
      <div className="container">
        <MenuOutlined
          className="menu-trigger"
          onClick={() => {
            setIsOpen((x) => !x);
          }}
        />
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
        {TopMenu}
        {loggedIn ? (
          <div className="user-info">
            <Dropdown
              overlayClassName="user-menu"
              overlay={userMenu({
                username,
                handleLogout,
              })}
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
    </Header>
  );
});
