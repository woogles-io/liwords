import React, { ReactNode, useCallback } from 'react';
import {
  ChatEntityObj,
  useChatStoreContext,
  useExcludedPlayersStoreContext,
  useLoginStateStoreContext,
} from '../store/store';
import { useMountedState } from '../utils/mounted';
import { AutoComplete } from 'antd';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { debounce } from '../utils/debounce';

type Props = {
  defaultChannel: string;
  defaultDescription: string;
  defaultLastMessage: string;
  onChannelSelect: (name: string, displayName: string) => void;
  unseenMessages: Array<ChatEntityObj>;
  updatedChannels: Set<string>;
  sendMessage?: (uuid: string, username: string) => void;
  tournamentID?: string;
};

export type ChatChannelLabel = {
  avatar?: ReactNode;
  title: string;
  label: string;
};

export const parseChannelLabel = (
  channelName: string,
  currentUser: string
): ChatChannelLabel | undefined => {
  let tokenized = channelName.split(':');
  if (tokenized.length > 1) {
    if (tokenized[0] === 'pm') {
      tokenized.shift();
      tokenized = tokenized.filter((player) => player !== currentUser);
      return {
        title: 'Chat with ' + tokenized.join(', '),
        label: tokenized.join(', '),
      };
    }
    if (tokenized[0] === 'tournament') {
      tokenized.shift();
      return {
        title: tokenized[0] + ' chat',
        label: tokenized[0],
      };
    }
  }
  // Unsupported chat channel format
  return undefined;
};

const getLocationLabel = (defaultChannel: string): string => {
  let tokenized = defaultChannel.split('.');
  if (tokenized.length > 1) {
    if (tokenized[1] === 'game') {
      return 'Game Chat';
    }
    if (tokenized[1] === 'gametv') {
      return 'Observer Chat';
    }
    if (tokenized[1] === 'lobby') {
      return '';
    }
    if (tokenized[1] === 'tournament') {
      return 'Tournament Chat';
    }
  }
  return '';
};
type user = {
  username: string;
  uuid: string;
};

type SearchResponse = {
  users: Array<user>;
};

export const ChatChannels = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const { chatChannels } = useChatStoreContext();
  const { excludedPlayers } = useExcludedPlayersStoreContext();
  const { sendMessage } = props;
  const { loginState } = useLoginStateStoreContext();
  const { username, userID } = loginState;
  const [showSearch, setShowSearch] = useState(false);
  const [usernameOptions, setUsernameOptions] = useState<Array<user>>([]);
  const onUsernameSearch = useCallback((searchText: string) => {
    axios
      .post<SearchResponse>(
        toAPIUrl('user_service.AutocompleteService', 'GetCompletion'),
        {
          prefix: searchText,
        }
      )
      .then((res) => {
        setUsernameOptions(res.data.users);
      });
  }, []);

  const searchUsernameDebounced = debounce(onUsernameSearch, 300);

  const handleUsernameSelect = useCallback(
    (data) => {
      const user = data.split(':');
      if (user.length > 1 && sendMessage) {
        sendMessage(user[0], user[1]);
      }
      setShowSearch(false);
    },
    [sendMessage]
  );

  const channelList = chatChannels?.channelsList
    .sort((chA, chB) => {
      return chB.lastUpdate - chA.lastUpdate;
    })
    .filter((ch) => {
      let keep = true;
      excludedPlayers.forEach((ex) => {
        if (ch.name.includes(ex)) {
          keep = false;
        }
      });
      return keep;
    })
    .filter((ch) => {
      return ch.name !== props.defaultChannel;
    })
    .filter((ch) => {
      // From the lobby, filter out channels we can't get new messages for
      // Todo: Remove this when we send tournament messages to all enrollees
      // regardless of their location
      if (props.defaultChannel === 'chat.lobby') {
        return ch.displayName.startsWith('pm');
      }
      if (props.tournamentID) {
        return (
          ch.displayName.startsWith('pm') ||
          ch.name.includes(props.tournamentID)
        );
      }
      return true;
    })
    .map((ch) => {
      const channelLabel = parseChannelLabel(ch.displayName, username);
      if (!channelLabel) {
        return null;
      }
      const isUnread =
        props.updatedChannels.has(ch.name) ||
        props.unseenMessages.some((uc) => uc.channel === ch.name);
      return (
        <div
          className={`channel-listing${isUnread ? ' unread' : ''}`}
          key={ch.name}
          onClick={() => {
            props.onChannelSelect(ch.name, channelLabel.title);
          }}
        >
          <p className="listing-name">
            {channelLabel.label}
            {isUnread && <span className="unread-marker">•</span>}
          </p>
          <p className="listing-preview">{ch.lastMessage}</p>
        </div>
      );
    });
  const defaultUnread =
    props.updatedChannels.has(props.defaultChannel) ||
    props.unseenMessages.some((uc) => uc.channel === props.defaultChannel);
  const locationLabel = getLocationLabel(props.defaultChannel);
  return (
    <div className="channel-list">
      {locationLabel && <p className="breadcrumb">{locationLabel}</p>}
      <div
        className={`channel-listing default${defaultUnread ? ' unread' : ''}`}
        onClick={() => {
          props.onChannelSelect(props.defaultChannel, props.defaultDescription);
        }}
      >
        <p className="listing-name">
          {props.defaultDescription}
          {defaultUnread && <span className="unread-marker">•</span>}
        </p>
        <p className="listing-preview">{props.defaultLastMessage}</p>
      </div>
      <div className="breadcrumb">
        <p>YOUR CHATS</p>
        <p
          className="link plain"
          onClick={() => {
            setShowSearch((s) => !s);
          }}
        >
          + New chat
        </p>
      </div>
      {showSearch && (
        <AutoComplete
          placeholder="Find player"
          onSearch={searchUsernameDebounced}
          onSelect={handleUsernameSelect}
          filterOption={(inputValue, option) =>
            !option || !(option.value === `${userID}:${username}`)
          }
        >
          {usernameOptions.map((user) => (
            <AutoComplete.Option
              key={user.uuid}
              value={`${user.uuid}:${user.username}`}
            >
              {user.username}
            </AutoComplete.Option>
          ))}
        </AutoComplete>
      )}
      {channelList}
    </div>
  );
});
