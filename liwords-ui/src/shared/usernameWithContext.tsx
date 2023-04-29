import React, { useContext } from 'react';
import { Link } from 'react-router-dom';
import { Dropdown } from 'antd';
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
  const contextualLink = React.useMemo(() => {
    if (currentActiveGames && currentActiveGames.length > 0) {
      const gameID =
        currentActiveGames[
          Math.floor(Math.random() * currentActiveGames.length)
        ];
      return {
        key: `watch-${userID}`,
        label: (
          <Link className="plain" to={`/game/${encodeURIComponent(gameID)}`}>
            Watch
          </Link>
        ),
      };
    } else if (currentWatchedGames && currentWatchedGames.length > 0) {
      const gameID =
        currentWatchedGames[
          Math.floor(Math.random() * currentWatchedGames.length)
        ];
      return {
        key: `join-${userID}`,
        label: (
          <Link className="plain" to={`/game/${encodeURIComponent(gameID)}`}>
            Join
          </Link>
        ),
      };
    } else if (currentlyPuzzling) {
      return {
        key: `puzzlejoin-${userID}`,
        label: (
          <Link className="plain" to="/puzzle">
            Join
          </Link>
        ),
      };
    } else {
      return null;
    }
  }, [currentActiveGames, currentWatchedGames, currentlyPuzzling, userID]);

  const userMenuOptions = [];
  if (isPettable) {
    userMenuOptions.push({
      key: `pettable-${userID}`,
      label: (
        <span
          onClick={() => {
            setPetting((x) => !x);
          }}
        >
          {!isPetting && 'Pet'}
          {isPetting && 'Stop petting'}
        </span>
      ),
    });
  }
  if (
    loggedIn &&
    !props.omitSendMessage &&
    props.userID &&
    props.userID !== userID &&
    props.sendMessage
  ) {
    userMenuOptions.push({
      key: `messageable-${props.userID}`,
      label: (
        <span
          onClick={() => {
            if (props.sendMessage && props.userID) {
              props.sendMessage(props.userID, props.username);
            }
          }}
        >
          Chat
        </span>
      ),
    });
  }
  if (!props.omitProfileLink) {
    userMenuOptions.push({
      key: `viewprofile-${props.userID}`,
      label: (
        <Link
          className="plain"
          to={`/profile/${encodeURIComponent(props.username)}`}
          target="_blank"
        >
          View profile
        </Link>
      ),
    });
  }
  if (contextualLink) {
    userMenuOptions.push(contextualLink);
  }
  if (
    loggedIn &&
    props.userID &&
    props.userID !== userID &&
    props.username &&
    handleContextMatches.length > 0
  ) {
    userMenuOptions.push({
      key: `idontknowwhatthisdoes-${props.userID}`,
      label: (
        <span
          onClick={() => {
            for (const handleContextMatch of handleContextMatches) {
              handleContextMatch(props.username);
            }
          }}
        >
          Match user
        </span>
      ),
    });
  }
  if (loggedIn && props.userID && !props.omitFriend) {
    userMenuOptions.push({
      key: `follower-${props.userID}`,
      label: (
        <TheFollower
          friendCallback={props.friendCallback}
          className="link plain"
          target={props.userID}
          tagName="li"
        />
      ),
    });
  }
  if (loggedIn && props.userID && !props.omitBlock) {
    userMenuOptions.push({
      key: `blocker-${props.userID}`,
      label: (
        <TheBlocker
          blockCallback={props.blockCallback}
          className="link plain"
          target={props.userID}
          tagName="li"
          userName={props.username}
        />
      ),
    });
  }
  if (props.showModTools && canMod(perms) && props.userID !== userID) {
    userMenuOptions.push({
      key: `mod-${props.userID}`,
      label: (
        <span
          onClick={() =>
            props.moderate && props.userID
              ? props.moderate(props.userID, props.username)
              : void 0
          }
        >
          Moderate
        </span>
      ),
    });
  }
  if (props.showDeleteMessage && canMod(perms) && props.userID !== userID) {
    userMenuOptions.push({
      key: `delete-${props.userID}`,
      label: <span onClick={props.deleteMessage}>Delete this message</span>,
    });
  }
  return (
    <Dropdown
      overlayClassName="user-menu"
      menu={{ items: userMenuOptions }}
      getPopupContainer={() => document.getElementById('root') as HTMLElement}
      placement="bottomLeft"
      trigger={userMenuOptions.length > 0 ? ['click'] : []}
    >
      <span
        className={`user-context-menu ${
          userMenuOptions.length > 0 ? '' : 'auto-cursor'
        }`}
      >
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
