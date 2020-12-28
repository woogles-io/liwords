import React from 'react';
import { Link } from 'react-router-dom';
import { Dropdown } from 'antd';
import { TheBlocker } from './blocker';
import { useLoginStateStoreContext } from '../store/store';

type UsernameWithContextProps = {
  additionalMenuItems?: React.ReactNode;
  omitProfileLink?: boolean;
  omitSendMessage?: boolean;
  username: string;
  userID?: string;
  sendMessage?: (uuid: string, username: string) => void;
  blockCallback?: () => void;
};

export const UsernameWithContext = (props: UsernameWithContextProps) => {
  const { loginState } = useLoginStateStoreContext();
  const { userID } = loginState;

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
        <TheBlocker
          blockCallback={props.blockCallback}
          className="link plain"
          target={props.userID}
          tagName="li"
        />
      ) : null}
      {!props.omitSendMessage && props.userID && props.userID !== userID ? (
        <li
          className="link plain"
          onClick={() => {
            if (props.sendMessage) {
              props.sendMessage(props.userID!, props.username);
            }
          }}
        >
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
