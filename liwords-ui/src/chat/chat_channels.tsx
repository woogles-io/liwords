import React, { ReactNode, useCallback, useEffect } from 'react';
import { AutoComplete } from 'antd';
import axios from 'axios';
import {
  ChatEntityObj,
  useChatStoreContext,
  useExcludedPlayersStoreContext,
  useLoginStateStoreContext,
} from '../store/store';
import { useBriefProfile } from '../utils/brief_profiles';
import { useMountedState } from '../utils/mounted';
import { toAPIUrl } from '../api/api';
import { debounce } from '../utils/debounce';
import { ActiveChatChannels } from '../gen/api/proto/user_service/user_service_pb';
import { PlayerAvatar } from '../shared/player_avatar';
import { DisplayFlag } from '../shared/display_flag';
import { TrophyOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';

type Props = {
  defaultChannel: string;
  defaultDescription: string;
  defaultLastMessage: string;
  onChannelSelect: (name: string, displayName: string) => void;
  unseenMessages: Array<ChatEntityObj>;
  updatedChannels?: Set<string>;
  sendMessage?: (uuid: string, username: string) => void;
  tournamentID?: string;
  maxHeight?: number;
};

export type ChatChannelLabel = {
  avatar?: ReactNode;
  title: string;
  label: string;
};

const getChannelType = (channel: string): string => {
  return channel?.split('.')[1] || '';
};

const getChannelIcon = (channelType: string): ReactNode => {
  // Note: We may allow tournaments and clubs to have their own avatar going forward
  switch (channelType) {
    case 'lobby':
      return (
        <div className={`player-avatar channel-icon ch-${channelType}`}>?</div>
      );
    case 'game':
      return (
        <div className={`player-avatar channel-icon ch${channelType}`}>
          <UserOutlined />
        </div>
      );
    case 'gametv':
      return (
        <div className={`player-avatar channel-icon ch-${channelType}`}>
          <TeamOutlined />
        </div>
      );
    case 'tournament':
      return (
        <div className={`player-avatar channel-icon ch-${channelType}`}>
          <TrophyOutlined />
        </div>
      );
  }
  return <div className={`player-avatar channel-icon ch-unknown`}>?</div>;
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
        title: `Chat with ${tokenized.join(', ')}`,
        label: tokenized.join(', '),
      };
    }
    if (tokenized[0] === 'tournament') {
      tokenized.shift();
      return {
        title: `${tokenized[0]} chat`,
        label: tokenized[0],
      };
    }
  }
  // Unsupported chat channel format
  return undefined;
};

const getLocationLabel = (defaultChannel: string): string => {
  const channelType = getChannelType(defaultChannel);
  switch (channelType) {
    case 'game':
      return 'Game Chat';
    case 'gametv':
      return 'Observer Chat';
    case 'lobby':
      return '';
    case 'tournament':
      return 'Tournament/Club Chat';
  }
  return '';
};

type user = {
  username?: string;
  uuid?: string;
};

type SearchResponse = {
  users: Array<user>;
};

const extractUser = (
  ch: ActiveChatChannels.Channel.AsObject,
  userId: string,
  username: string
): user => {
  const nameTokens = ch.displayName.split(':');
  if (nameTokens[0] === 'pm' || 'game') {
    const chatUsername =
      nameTokens[1] === username ? nameTokens[2] : nameTokens[1];
    const idTokens = ch.name.split('.')[2]?.split('_');
    const chatUserId = idTokens[0] === userId ? idTokens[1] : idTokens[0];
    return { uuid: chatUserId, username: chatUsername };
  }
  return {};
};

export const ChatChannels = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const { chatChannels } = useChatStoreContext();
  const { excludedPlayers } = useExcludedPlayersStoreContext();
  const { sendMessage } = props;
  const { loginState } = useLoginStateStoreContext();
  const { username, userID } = loginState;
  const [showSearch, setShowSearch] = useState(false);
  const [maxHeight, setMaxHeight] = useState<number | undefined>(0);
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

  const setHeight = useCallback(() => {
    const tabPaneHeight = document.getElementById('chat')?.clientHeight;
    setMaxHeight(tabPaneHeight ? tabPaneHeight - 48 : undefined);
  }, []);

  useEffect(() => {
    setHeight();
  }, [setHeight]);

  useEffect(() => {
    window.addEventListener('resize', setHeight);
    return () => {
      window.removeEventListener('resize', setHeight);
    };
  }, [setHeight]);

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
          ch.name === `chat.tournament.${props.tournamentID}`
        );
      } else {
        return ch.displayName.startsWith('pm');
      }
    })
    .map((ch) => {
      const channelLabel = parseChannelLabel(ch.displayName, username);
      if (!channelLabel) {
        return null;
      }
      const lastUnread = props.unseenMessages.reduce(
        (acc: ChatEntityObj | undefined, m) =>
          m.channel === ch.name &&
          'timestamp' in m &&
          (acc === undefined || m.timestamp! > acc.timestamp!)
            ? m
            : acc,
        undefined
      );
      const isUnread = props.updatedChannels?.has(ch.name) || lastUnread;
      const chatUser = extractUser(ch, userID, username);
      const briefProfile = useBriefProfile(chatUser.uuid);
      const channelType = getChannelType(ch.name);
      return (
        <div
          className={`channel-listing${isUnread ? ' unread' : ''}`}
          key={ch.name}
          onClick={() => {
            props.onChannelSelect(ch.name, channelLabel.title);
          }}
        >
          {channelType === 'pm' && chatUser.username ? (
            <PlayerAvatar
              player={{
                user_id: chatUser.uuid,
                nickname: chatUser.username,
              }}
            />
          ) : (
            getChannelIcon(getChannelType(ch.name))
          )}
          <div>
            <p className="listing-name">
              {channelLabel.label}
              {briefProfile && (
                <DisplayFlag countryCode={briefProfile.getCountryCode()} />
              )}
              {isUnread && <span className="unread-marker">•</span>}
            </p>
            <p className={`listing-preview`}>
              {lastUnread?.message || ch.lastMessage}
            </p>
          </div>
        </div>
      );
    });
  const defaultUnread =
    (props.updatedChannels &&
      props.updatedChannels!.has(props.defaultChannel)) ||
    props.unseenMessages.some((uc) => uc.channel === props.defaultChannel);
  const locationLabel = getLocationLabel(props.defaultChannel);
  return (
    <div
      className="channel-list"
      style={
        maxHeight
          ? {
              maxHeight: maxHeight,
            }
          : undefined
      }
    >
      {locationLabel && <p className="breadcrumb">{locationLabel}</p>}
      <div
        className={`channel-listing default${defaultUnread ? ' unread' : ''}`}
        onClick={() => {
          props.onChannelSelect(props.defaultChannel, props.defaultDescription);
        }}
      >
        {getChannelIcon(getChannelType(props.defaultChannel))}
        <div>
          <p className="listing-name">
            {props.defaultDescription}
            {defaultUnread && <span className="unread-marker">•</span>}
          </p>
          <p className="listing-preview">{props.defaultLastMessage}</p>
        </div>
      </div>
      <div className="breadcrumb">
        <p>YOUR CHATS</p>
        <p
          className="link plain"
          onClick={() => {
            setShowSearch((s) => !s);
          }}
        >
          + Start new chat
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
