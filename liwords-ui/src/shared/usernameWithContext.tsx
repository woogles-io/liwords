import React from 'react';
import { Link } from 'react-router-dom';
import { Dropdown } from 'antd';
import { TheBlocker } from './blocker';
import { useLoginStateStoreContext } from '../store/store';
import { canMod } from '../mod/perms';
import { DisplayUserFlag } from './display_flag';
import { SettingOutlined } from '@ant-design/icons';
import { TheFollower } from './follower';

type UsernameWithContextProps = {
  additionalMenuItems?: React.ReactNode;
  includeFlag?: boolean;
  fullName?: string;
  omitProfileLink?: boolean;
  omitSendMessage?: boolean;
  omitFriend?: boolean;
  omitBlock?: boolean;
  username: string;
  userID?: string;
  sendMessage?: (uuid: string, username: string) => void;
  friendCallback?: () => void;
  blockCallback?: () => void;
  showModTools?: boolean;
  showDeleteMessage?: boolean;
  moderate?: (uuid: string, username: string) => void;
  deleteMessage?: () => void;
  iconOnly?: boolean;
};

export const UsernameWithContext = (props: UsernameWithContextProps) => {
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn, userID, perms } = loginState;
  const userMenu = (
    <ul>
      {loggedIn &&
      !props.omitSendMessage &&
      props.userID &&
      props.userID !== userID &&
      props.sendMessage ? (
        <li
          className="link plain"
          onClick={() => {
            props.sendMessage!(props.userID!, props.username);
          }}
        >
          Chat
        </li>
      ) : null}
      {!props.omitProfileLink && (
        <li>
          <Link
            className="plain"
            to={`/profile/${encodeURIComponent(props.username)}`}
            target="_blank"
          >
            View profile
          </Link>
        </li>
      )}
      {loggedIn && props.userID && !props.omitFriend ? (
        <TheFollower
          friendCallback={props.friendCallback}
          className="link plain"
          target={props.userID}
          tagName="li"
        />
      ) : null}

      {loggedIn && props.userID && !props.omitBlock ? (
        <TheBlocker
          blockCallback={props.blockCallback}
          className="link plain"
          target={props.userID}
          tagName="li"
          userName={props.username}
        />
      ) : null}

      {props.showModTools && canMod(perms) && props.userID !== userID ? (
        <li
          className="link plain"
          onClick={() =>
            props.moderate
              ? props.moderate(props.userID!, props.username)
              : void 0
          }
        >
          Moderate
        </li>
      ) : null}
      {props.showDeleteMessage && canMod(perms) && props.userID !== userID ? (
        <li className="link plain" onClick={props.deleteMessage}>
          Delete this message
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
      <span className="user-context-menu">
        {props.iconOnly ? ( // Not yet used
          <SettingOutlined />
        ) : (
          <>
            {props.fullName || props.username}
            {props.includeFlag && <DisplayUserFlag uuid={props.userID} />}
          </>
        )}
      </span>
    </Dropdown>
  );
};
