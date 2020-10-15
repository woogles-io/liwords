import React from 'react';
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
          <a
            className="plain"
            href={`/profile/${props.username}`}
            rel="noopener noreferrer"
            target="_blank"
          >
            View Profile
          </a>
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
      placement="bottomLeft"
      trigger={['click']}
    >
      <span className="user-context-menu">{props.username}</span>
    </Dropdown>
  );
};
