import React from 'react';
import { useChatStoreContext } from '../store/store';

type Props = {
  onChannelSelect: (name: string, displayName: string) => void;
};

export const ChatChannels = React.memo((props: Props) => {
  const { chatChannels } = useChatStoreContext();

  const channelList = chatChannels?.toObject().channelsList.map((ch) => {
    return (
      <p
        key={ch.name}
        onClick={() => {
          props.onChannelSelect(ch.name, ch.displayName);
        }}
      >
        {ch.displayName}
      </p>
    );
  });

  return (
    <div className="channel-list">
      <p className="breadcrumb">YOUR CHATS</p>
      {channelList}
    </div>
  );
});
