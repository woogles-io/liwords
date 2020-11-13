import React from 'react';
import { Link } from 'react-router-dom';
import { Dropdown } from 'antd';
import { TheBlocker } from './blocker';

type UsernameWithContextProps = {
  additionalMenuItems?: React.ReactNode;
  omitProfileLink?: boolean;
  username: string;
  userID?: string;
};

export const UsernameWithContext = (props: UsernameWithContextProps) => {
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
