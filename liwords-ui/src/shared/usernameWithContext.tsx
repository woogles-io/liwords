import React from 'react';
import { Link } from 'react-router-dom';
import { Dropdown } from 'antd';
import { TheBlocker } from './blocker';
import { useBriefProfile } from '../utils/brief_profiles';
import { useLoginStateStoreContext } from '../store/store';
import { canMod } from '../mod/perms';
import { DisplayFlag } from './display_flag';
import { SettingOutlined } from '@ant-design/icons';

type UsernameWithContextProps = {
  additionalMenuItems?: React.ReactNode;
  includeFlag?: boolean;
  omitProfileLink?: boolean;
  omitSendMessage?: boolean;
  omitBlock?: boolean;
  username: string;
  userID?: string;
  sendMessage?: (uuid: string, username: string) => void;
  blockCallback?: () => void;
  showModTools?: boolean;
  showDeleteMessage?: boolean;
  moderate?: (uuid: string, username: string) => void;
  deleteMessage?: () => void;
  iconOnly?: boolean;
};

export const UsernameWithContext = (props: UsernameWithContextProps) => {
  const { loginState } = useLoginStateStoreContext();
  const { userID, perms } = loginState;
  const briefProfile = useBriefProfile(props.userID);

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
      {props.userID && !props.omitBlock ? (
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
          Delete this Message
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
            {props.username}
            {briefProfile && props.includeFlag && (
              <DisplayFlag countryCode={briefProfile!.getCountryCode()} />
            )}
          </>
        )}
      </span>
    </Dropdown>
  );
};
