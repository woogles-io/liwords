import React from 'react';
import './topbar.scss';
import { DisconnectOutlined } from '@ant-design/icons/lib';
import { Tooltip } from 'antd';
import { useStoreContext } from '../store/store';

const Menu = (
  <div className="top-header-menu">
    <div className="top-header-left-frame-crossword-game">
      <a href="/">OMGWords</a>
    </div>
    <div className="top-header-left-frame-aerolith">Aerolith</div>
    <div className="top-header-left-frame-blog">Random.Racer</div>
    <div className="top-header-left-frame-special-land">
      <a href="/about">About Us</a>
    </div>
  </div>
);

type Props = {};

export const TopBar = React.memo((props: Props) => {
  const {
    username,
    loggedIn,
    connectedToSocket,
    currentLagMs,
  } = useStoreContext().loginState;
  return (
    <nav className="top-header" id="main-nav">
      <div className="container">
        <a href="/" className="site-icon">
          <div className="top-header-site-icon-rect">
            <div className="top-header-site-icon-m">W</div>
          </div>
          <div className="top-header-left-frame-site-name">Woogles.io</div>
        </a>
        {Menu}
        {loggedIn ? (
          <div className="user-info">
            <Tooltip title={`Latency: ${currentLagMs || '...'} ms.`}>
              <a href={`/profile/${username}`}>{username}</a>
            </Tooltip>
            {!connectedToSocket ? (
              <DisconnectOutlined style={{ color: 'red', marginLeft: 5 }} />
            ) : null}
          </div>
        ) : (
          <div className="user-info">
            <a href="/login">Log In</a>
          </div>
        )}
      </div>
    </nav>
  );
});
