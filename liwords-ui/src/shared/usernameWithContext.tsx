import React, { useCallback } from 'react';
import { Link } from 'react-router-dom';
import { Dropdown } from 'antd';
import { TheBlocker } from './blocker';

type UsernameWithContextProps = {
  additionalMenuItems?: React.ReactNode;
  omitProfileLink?: boolean;
  omitSendMessage?: boolean;
  username: string;
  userID?: string;
  sendMessage?: (msg: string, receiver: string) => void;
};

export const UsernameWithContext = (props: UsernameWithContextProps) => {
  const sendMessage = (uid: string, username: string) => {
    // very temporary way to send messages to users
    const msg = window.prompt(`Send a private message to ${username}`);
    if (msg && props.sendMessage) {
      props.sendMessage(msg, uid);
    }
  };

  const userMenu = (
    <ul>
      {!props.omitProfileLink && (
        <li>
          <Link
            className="plain"
            to={`/profile/${encodeURIComponent(props.username)}`}
            target="_blank"
          >
            View Profile
          </Link>
        </li>
      )}
      {props.userID ? (
        <TheBlocker className="link plain" target={props.userID} tagName="li" />
      ) : null}
      {!props.omitSendMessage && props.userID ? (
        <li onClick={() => sendMessage(props.userID!, props.username)}>
          Message
        </li>
      ) : null}
      {props.additionalMenuItems}
    </ul>
  );
  return (
    <Dropdown
      overlayClassName="user-menu"
      overlay={userMenu}
      getPopupContainer={() => document.getElementById('root') as HTMLElement}
      placement="bottomLeft"
      trigger={['click']}
    >
      <span className="user-context-menu">{props.username}</span>
    </Dropdown>
  );
};
