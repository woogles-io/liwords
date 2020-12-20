import React, { ReactNode } from 'react';
import {
  ChatEntityObj,
  useChatStoreContext,
  useExcludedPlayersStoreContext,
  useLoginStateStoreContext,
} from '../store/store';

type Props = {
  defaultChannel: string;
  defaultDescription: string;
  onChannelSelect: (name: string, displayName: string) => void;
  unseenMessages: Array<ChatEntityObj>;
  updatedChannels: Set<string>;
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
  }
  return '';
};

export const ChatChannels = React.memo((props: Props) => {
  const { chatChannels } = useChatStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { excludedPlayers } = useExcludedPlayersStoreContext();
  const { username } = loginState;
  const channelList = chatChannels
    ?.toObject()
    .channelsList.sort((chA, chB) => {
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
      // todo: remove this filter when we can receive messages regardless
      // of our location
      return ch.displayName.startsWith('pm');
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
        <p
          className={`channel-listing${isUnread ? ' unread' : ''}`}
          key={ch.name}
          onClick={() => {
            props.onChannelSelect(ch.name, channelLabel.title);
          }}
        >
          {channelLabel.label}
        </p>
      );
    });
  const locationLabel = getLocationLabel(props.defaultChannel);
  return (
    <div className="channel-list">
      <p className="breadcrumb">{locationLabel}</p>
      <p
        className="channel-listing"
        onClick={() => {
          props.onChannelSelect(props.defaultChannel, props.defaultDescription);
        }}
      >
        {props.defaultDescription}
      </p>
      <p className="breadcrumb">YOUR CHATS</p>
      {channelList}
    </div>
  );
});
