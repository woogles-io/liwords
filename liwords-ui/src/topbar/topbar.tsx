import React from 'react';
import './topbar.scss';
import { Link } from 'react-router-dom';

const Menu = (
  <div className="top-header-menu">
    <div className="top-header-left-frame-crossword-game">Crossword Game</div>
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

export const TopBar = (props: Props) => {
  return (
    <nav className="top-header">
      <Link to="/" className="site-icon">
        <div className="top-header-site-icon-rect">
          <div className="top-header-site-icon-m">M</div>
        </div>
        <div className="top-header-left-frame-site-name">Macondo</div>
      </Link>
      {Menu}
      {props.loggedIn ? (
        <div className="user-info">{props.username}</div>
      ) : (
        <div className="user-info">
          <Link to="/login">Log In</Link>
        </div>
      )}
    </nav>
  );
};
