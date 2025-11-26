import React, { useContext, useRef } from "react";
import { Link } from "react-router";
import { App, Dropdown } from "antd";
import { BlockerHandle, TheBlocker } from "./blocker";
import {
  useContextMatchContext,
  useLoginStateStoreContext,
} from "../store/store";
import { canMod } from "../mod/perms";
import { DisplayUserFlag } from "./display_flag";
import { SettingOutlined } from "@ant-design/icons";
import { FollowerHandle, TheFollower } from "./follower";
import { PettableContext } from "./player_avatar";
import { HookAPI } from "antd/lib/modal/useModal";
import { DisplayUserBadges } from "../profile/badge";
import { DisplayUserTitle } from "./display_title";

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
  moderate?: (modal: HookAPI, uuid: string, username: string) => void;
  deleteMessage?: () => void;
  iconOnly?: boolean;
  currentActiveGames?: Array<string>;
  currentWatchedGames?: Array<string>;
  currentEditingGames?: Array<string>;
  currentWatchedAnnoGames?: Array<string>;
  currentlyPuzzling?: boolean;
  omitBadges?: boolean;
};

export const UsernameWithContext = (props: UsernameWithContextProps) => {
  const { isPettable, isPetting, setPetting } = useContext(PettableContext);
  const {
    currentActiveGames,
    currentWatchedGames,
    currentEditingGames,
    currentWatchedAnnoGames,
    currentlyPuzzling,
  } = props;
  const { handleContextMatches } = useContextMatchContext();
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn, userID, perms } = loginState;

  const followerRef = useRef<FollowerHandle>(undefined);
  const blockerRef = useRef<BlockerHandle>(undefined);

  const contextItem = React.useMemo(() => {
    if (currentActiveGames && currentActiveGames.length > 0) {
      const gameID =
        currentActiveGames[
          Math.floor(Math.random() * currentActiveGames.length)
        ];
      return {
        label: <Link to={`/game/${encodeURIComponent(gameID)}`}>Watch</Link>,
        key: `watch-${userID}`,
      };
    } else if (currentWatchedGames && currentWatchedGames.length > 0) {
      const gameID =
        currentWatchedGames[
          Math.floor(Math.random() * currentWatchedGames.length)
        ];
      return {
        label: <Link to={`/game/${encodeURIComponent(gameID)}`}>Join</Link>,
        key: `join-${userID}`,
      };
    } else if (currentEditingGames && currentEditingGames.length > 0) {
      const gameID =
        currentEditingGames[
          Math.floor(Math.random() * currentEditingGames.length)
        ];
      return {
        label: <Link to={`/anno/${encodeURIComponent(gameID)}`}>Watch</Link>,
        key: `watch-${userID}`,
      };
    } else if (currentWatchedAnnoGames && currentWatchedAnnoGames.length > 0) {
      const gameID =
        currentWatchedAnnoGames[
          Math.floor(Math.random() * currentWatchedAnnoGames.length)
        ];
      return {
        label: <Link to={`/anno/${encodeURIComponent(gameID)}`}>Join</Link>,
        key: `join-${userID}`,
      };
    } else if (currentlyPuzzling) {
      return {
        label: <Link to="/puzzle">Join</Link>,
        key: `puzzlejoin-${userID}`,
      };
    } else {
      return null;
    }
  }, [
    currentActiveGames,
    userID,
    currentWatchedGames,
    currentlyPuzzling,
    currentEditingGames,
    currentWatchedAnnoGames,
  ]);

  const userMenuOptions = [];
  if (isPettable) {
    userMenuOptions.push({
      key: `pettable-${userID}`,
      label: isPetting ? "Stop petting" : "Pet",
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
      label: "Chat",
    });
  }
  if (!props.omitProfileLink) {
    userMenuOptions.push({
      label: (
        <Link
          to={`/profile/${encodeURIComponent(props.username)}`}
          target="_blank"
        >
          View profile
        </Link>
      ),
      key: `viewprofile-${props.userID}`,
    });
  }
  if (contextItem) {
    userMenuOptions.push(contextItem);
  }
  if (
    loggedIn &&
    props.userID &&
    props.userID !== userID &&
    props.username &&
    handleContextMatches.length > 0
  ) {
    userMenuOptions.push({
      key: `match-${props.userID}`,
      label: "Match user",
    });
  }
  if (
    loggedIn &&
    props.userID &&
    props.userID !== userID &&
    !props.omitFriend
  ) {
    userMenuOptions.push({
      key: `follower-${props.userID}`,
      label: (
        <TheFollower
          friendCallback={props.friendCallback}
          target={props.userID}
          ref={followerRef}
        />
      ),
    });
  }
  if (loggedIn && props.userID && props.userID !== userID && !props.omitBlock) {
    userMenuOptions.push({
      key: `blocker-${props.userID}`,
      label: (
        <TheBlocker
          blockCallback={props.blockCallback}
          target={props.userID}
          userName={props.username}
          ref={blockerRef}
        />
      ),
    });
  }
  if (props.showModTools && canMod(perms)) {
    userMenuOptions.push({
      key: `mod-${props.userID}`,
      label: `Moderate`,
    });
  }
  if (props.showDeleteMessage && canMod(perms) && props.userID !== userID) {
    userMenuOptions.push({
      key: `delete-${props.userID}`,
      label: "Delete this message",
    });
  }
  const { modal } = App.useApp();
  // const userMenu = <ul>{userMenuOptions}</ul>;
  return (
    <Dropdown
      overlayClassName="user-menu"
      destroyPopupOnHide
      menu={{
        items: userMenuOptions,
        onClick: ({ key }) => {
          switch (key) {
            case `delete-${props.userID}`:
              if (props.deleteMessage) props.deleteMessage();
              break;
            case `mod-${props.userID}`:
              if (props.moderate && props.userID)
                props.moderate(modal, props.userID, props.username);
              break;
            case `match-${props.userID}`:
              for (const handleContextMatch of handleContextMatches) {
                handleContextMatch(props.username);
              }
              break;
            case `messageable-${props.userID}`:
              if (props.sendMessage && props.userID) {
                props.sendMessage(props.userID, props.username);
              }
              break;
            case `pettable-${userID}`:
              setPetting((x) => !x);
              break;
            case `follower-${props.userID}`:
              followerRef.current?.friendAction();
              break;
            case `blocker-${props.userID}`:
              blockerRef.current?.blockAction();
              break;
          }
        },
      }}
      getPopupContainer={() => document.getElementById("root") as HTMLElement}
      placement="bottomLeft"
      trigger={userMenuOptions.length > 0 ? ["click"] : []}
    >
      <span
        className={`user-context-menu ${
          userMenuOptions.length > 0 ? "" : "auto-cursor"
        }`}
      >
        {props.iconOnly ? ( // Not yet used
          <SettingOutlined />
        ) : (
          <>
            <DisplayUserTitle uuid={props.userID} />
            {props.fullName || props.username}
            {props.includeFlag && <DisplayUserFlag uuid={props.userID} />}
            {!props.omitBadges && <DisplayUserBadges uuid={props.userID} />}
          </>
        )}
      </span>
    </Dropdown>
  );
};
