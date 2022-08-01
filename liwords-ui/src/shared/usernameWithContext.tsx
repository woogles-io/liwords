import React, { useContext } from 'react';
import { Link } from 'react-router-dom';
import { Dropdown, Tooltip } from 'antd';
import { TheBlocker } from './blocker';
import {
  useContextMatchContext,
  useLoginStateStoreContext,
} from '../store/store';
import { canMod } from '../mod/perms';
import { DisplayUserFlag } from './display_flag';
import { SettingOutlined } from '@ant-design/icons';
import { TheFollower } from './follower';
import { PettableContext } from './player_avatar';
import { useBriefProfile } from '../utils/brief_profiles';

type UsernameWithContextProps = {
  additionalMenuItems?: React.ReactNode;
  includeFlag?: boolean;
  showSilentMode?: boolean;
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
  currentActiveGames?: Array<string>;
  currentWatchedGames?: Array<string>;
  currentlyPuzzling?: boolean;
};

export const UsernameWithContext = (props: UsernameWithContextProps) => {
  const { isPettable, isPetting, setPetting } = useContext(PettableContext);
  const { currentActiveGames, currentWatchedGames, currentlyPuzzling } = props;
  const { handleContextMatches } = useContextMatchContext();
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn, userID, perms } = loginState;
  const briefProfile = useBriefProfile(props.userID);
  const contextualLink = React.useMemo(() => {
    if (currentActiveGames && currentActiveGames.length > 0) {
      const gameID =
        currentActiveGames[
          Math.floor(Math.random() * currentActiveGames.length)
        ];
      return (
        <li>
          <Link className="plain" to={`/game/${encodeURIComponent(gameID)}`}>
            Watch
          </Link>
        </li>
      );
    } else if (currentWatchedGames && currentWatchedGames.length > 0) {
      const gameID =
        currentWatchedGames[
          Math.floor(Math.random() * currentWatchedGames.length)
        ];
      return (
        <li>
          <Link className="plain" to={`/game/${encodeURIComponent(gameID)}`}>
            Join
          </Link>
        </li>
      );
    } else if (currentlyPuzzling) {
      return (
        <li>
          <Link className="plain" to="/puzzle">
            Join
          </Link>
        </li>
      );
    } else {
      return null;
    }
  }, [currentActiveGames, currentWatchedGames, currentlyPuzzling]);
  const userMenu = (
    <ul>
      {isPettable && (
        <li
          className="link plain"
          onClick={() => {
            setPetting((x) => !x);
          }}
        >
          {!isPetting && 'Pet'}
          {isPetting && 'Stop petting'}
        </li>
      )}
      {loggedIn &&
      !props.omitSendMessage &&
      props.userID &&
      props.userID !== userID &&
      props.sendMessage &&
      !briefProfile?.getSilentMode() ? (
        <li
          className="link plain"
          onClick={() => {
            if (props.sendMessage && props.userID) {
              props.sendMessage(props.userID, props.username);
            }
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
      {contextualLink}
      {loggedIn &&
        props.userID &&
        props.userID !== userID &&
        props.username &&
        handleContextMatches.length > 0 && (
          <li
            className="link plain"
            onClick={() => {
              for (const handleContextMatch of handleContextMatches) {
                handleContextMatch(props.username);
              }
            }}
          >
            Match user
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
            props.moderate && props.userID
              ? props.moderate(props.userID, props.username)
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
  console.warn(briefProfile?.toObject());
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
            {props.showSilentMode && briefProfile?.getSilentMode() && (
              <Tooltip
                title={`${props.fullName || props.username} is in silent mode`}
              >
                <i className="fa-solid fa-comment-slash"></i>
              </Tooltip>
            )}
            {props.includeFlag && <DisplayUserFlag uuid={props.userID} />}
          </>
        )}
      </span>
    </Dropdown>
  );
};
