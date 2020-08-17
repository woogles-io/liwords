import React from 'react';
import './topbar.scss';
import { Link } from 'react-router-dom';
import { DisconnectOutlined } from '@ant-design/icons/lib';

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

type Props = {
  username: string;
  loggedIn: boolean;
  connectedToSocket: boolean;
};

export const TopBar = React.memo((props: Props) => {
  return (
    <nav className="top-header">
      <div className="container">
        <Link to="/" className="site-icon">
          <div className="top-header-site-icon-rect">
            <div className="top-header-site-icon-m">W</div>
          </div>
          <div className="top-header-left-frame-site-name">Woogles.io</div>
        </Link>
        {Menu}
        {props.loggedIn ? (
          <div className="user-info">
            <Link to={`/profile/${props.username}`}>{props.username}</Link>
            {!props.connectedToSocket ? (
              <DisconnectOutlined style={{ color: 'red', marginLeft: 5 }} />
            ) : null}
          </div>
        ) : (
          <div className="user-info">
            <Link to="/login">Log In</Link>
          </div>
        )}
      </div>
    </nav>
  );
});
