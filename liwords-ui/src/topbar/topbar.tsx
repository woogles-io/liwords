import React from 'react';
import './topbar.scss';
import { Link } from 'react-router-dom';

const Menu = (
  <div className="top-header-menu">
    <div className="top-header-left-frame-crossword-game">OMGWords</div>
    <div className="top-header-left-frame-aerolith">Aerolith</div>
    <div className="top-header-left-frame-blog">Blog</div>
    <div className="top-header-left-frame-special-land">
      César's Special Land
    </div>
  </div>
);

type Props = {
  username: string;
  loggedIn: boolean;
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
          <div className="user-info">{props.username}</div>
        ) : (
          <div className="user-info">
            <Link to="/login">Log In</Link>
          </div>
        )}
      </div>
    </nav>
  );
});
