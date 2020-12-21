import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import { Card, Input, Tabs } from 'antd';
import { useMountedState } from '../utils/mounted';
import { ChatEntity } from './chat_entity';
import {
  ChatEntityObj,
  ChatEntityType,
  PresenceEntity,
  useChatStoreContext,
  useLoginStateStoreContext,
} from '../store/store';
import { LeftOutlined } from '@ant-design/icons';
import './chat.scss';
import { Presences } from './presences';
import { ChatChannels } from './chat_channels';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import {
  ChatMessageFromJSON,
  chatMessageToChatEntity,
} from '../store/constants';
import { ActiveChatChannels } from '../gen/api/proto/user_service/user_service_pb';

const { TabPane } = Tabs;

type Props = {
  peopleOnlineContext: (n: number) => string; // should return "1 person" or "2 people"
  sendChat: (msg: string, chan: string) => void;
  defaultChannel: string;
  defaultDescription: string;
  presences: { [uuid: string]: PresenceEntity };
  DISCONNECT?: () => void;
  highlight?: Array<string>;
};

export const Chat = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn, userID } = loginState;
  const [curMsg, setCurMsg] = useState('');
  const [hasScroll, setHasScroll] = useState(false);
  const [showChannels, setShowChannels] = useState(false);
  const [selectedChatTab, setSelectedChatTab] = useState('CHAT');
  const [presenceVisible, setPresenceVisible] = useState(false);
  const presenceCount = Object.keys(props.presences).length;
  // We cannot useRef because we need to awoken effects relying on the element.
  // For simplicity, the setter is used directly as a ref callback.
  const [
    tabContainerElement,
    setTabContainerElement,
  ] = useState<HTMLDivElement | null>(null);
  const propsSendChat = useMemo(() => props.sendChat, [props.sendChat]);

  // Chat auto-scrolls when the last entity is visible.
  const [hasUnreadChat, setHasUnreadChat] = useState(false);

  const {
    chat: chatEntities,
    chatChannels,
    clearChat,
    addChats,
    setChatChannels,
  } = useChatStoreContext();
  const lastChannel = useRef(props.defaultChannel);
  const chatTab = selectedChatTab === 'CHAT' ? tabContainerElement : null;
  const [chatAutoScroll, setChatAutoScroll] = useState(true);
  const [channel, setChannel] = useState(props.defaultChannel);
  const [description, setDescription] = useState(props.defaultDescription);
  const [channelSelectedTime, setChannelSelectedTime] = useState(Date.now());
  // Messages that come in for other channels
  const [unseenMessages, setUnseenMessages] = useState(
    new Array<ChatEntityObj>()
  );
  // Channels other than the current that are flagged hasUpdate. Each one is removed
  // if we switch to it
  const [updatedChannels, setUpdatedChannels] = useState(new Set<string>());
  const [decoratedDescription, setDecoratedDescription] = useState('');
  const onChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setCurMsg(e.target.value);
  }, []);
  const knownUsers = useMemo(
    () =>
      Object.keys(props.presences).filter((p) => !props.presences[p].anon) ||
      [],
    [props.presences]
  );

  const doChatAutoScroll = useCallback(() => {
    if (chatAutoScroll && chatTab) {
      // Doing this conditionally may help browser performance.
      // (Not sure.)
      if (chatTab.scrollTop !== chatTab.scrollHeight) {
        chatTab.scrollTop = chatTab.scrollHeight;
      }
    }
  }, [chatAutoScroll, chatTab]);

  const fetchChannels = useCallback(() => {
    axios
      .post(
        toAPIUrl('user_service.SocializeService', 'GetActiveChatChannels'),
        {
          number: 20,
          offset: 0,
        },
        { withCredentials: true }
      )
      .then((res) => {
        const newChannels: ActiveChatChannels.AsObject = {
          channelsList:
            res.data.channels.map((ch: any) => {
              return {
                displayName: ch.display_name,
                lastUpdate: parseInt(ch.last_update, 10),
                hasUpdate: ch.has_update,
                lastMessage: ch.last_message || '',
                name: ch.name,
              };
            }) || [],
        };
        setChatChannels(newChannels);
      });
  }, [setChatChannels]);

  useEffect(() => {
    // Initial load of channels
    fetchChannels();
  }, [fetchChannels]);
  useEffect(() => {
    // Chat channels have changed. Note them if hasUpdate is true
    const changed = chatChannels?.channelsList
      ?.map((ch) => {
        return ch;
      })
      .filter((ch) => ch.hasUpdate)
      .map((ch) => {
        return ch.name;
      });
    setUpdatedChannels(new Set(changed));
  }, [chatChannels]);

  useEffect(() => {
    if (chatTab) {
      // Designs changes when the chat is long enough to scroll
      if (chatTab.scrollHeight > chatTab.clientHeight) {
        setHasScroll(true);
      }
      doChatAutoScroll();
    }
  }, [chatTab, doChatAutoScroll, chatEntities]);

  useEffect(() => {
    setChannel(props.defaultChannel);
    setDescription(props.defaultDescription);
  }, [props.defaultChannel, props.defaultDescription]);

  useEffect(() => {
    const channelType =
      channel.split('.').length > 2 ? channel.split('.')[1] : '';
    switch (channelType) {
      case 'game':
        setDecoratedDescription('Game chat with ' + description);
        break;
      case 'gametv':
        setDecoratedDescription('Game chat for ' + description);
        break;
      default:
        setDecoratedDescription(description);
    }
  }, [channel, description]);

  useEffect(() => {
    if (chatTab || showChannels) {
      // chat entities changed.
      // If there are new messages in this
      // channel and we've scrolled up, mark this chat unread,
      const currentUnread = chatEntities
        .filter((ch) => {
          return (
            ch.channel === channel &&
            ch.timestamp &&
            ch.timestamp > channelSelectedTime
          );
        })
        .sort((chA, chB) => {
          if (chB.timestamp && chA.timestamp) {
            return chB.timestamp - chA.timestamp;
          }
          return 0;
        });
      if (currentUnread.length && chatTab) {
        setHasUnreadChat(
          chatTab.scrollTop < chatTab.scrollHeight - chatTab.clientHeight
        );
      }
      // If they're for other channels add them to the unseenMessage,
      // to indicate unread in the channels component
      const otherUnreads = chatEntities
        .filter(
          // Only the ones since we switched to this channel
          (ch) =>
            ch.channel !== channel &&
            ch.timestamp &&
            ch.timestamp > channelSelectedTime
        )
        .filter(
          (ch) =>
            // And not our own
            ch.senderId !== userID
        );
      if (otherUnreads.length) {
        setUnseenMessages((u) => u.concat(otherUnreads));
      }
    }
  }, [
    chatTab,
    chatEntities,
    channel,
    fetchChannels,
    channelSelectedTime,
    showChannels,
    userID,
  ]);

  // When window is shrunk, auto-scroll may be enabled. This is one-way.
  // Hiding bookmarks bar or downloaded files bar should keep it enabled.
  const enableChatAutoScroll = useCallback(() => {
    if (
      chatTab &&
      chatTab.scrollTop >= chatTab.scrollHeight - chatTab.clientHeight
    ) {
      setChatAutoScroll(true);
    }
    // When window is shrunk, keep the bottom entity instead of the top.
    doChatAutoScroll();
  }, [chatTab, doChatAutoScroll]);
  useEffect(() => {
    window.addEventListener('resize', enableChatAutoScroll);
    return () => {
      window.removeEventListener('resize', enableChatAutoScroll);
    };
  }, [enableChatAutoScroll]);

  useEffect(() => {
    // If we actually changed the channel, get the new messages
    if (loggedIn && channel !== lastChannel.current) {
      lastChannel.current = channel;
      setChannelSelectedTime(Date.now());
      axios
        .post(
          toAPIUrl('user_service.SocializeService', 'GetChatsForChannel'),
          {
            channel: channel,
          },
          { withCredentials: true }
        )
        .then((res) => {
          clearChat();
          const messages: Array<ChatMessageFromJSON> = res.data?.messages;
          addChats(messages.map(chatMessageToChatEntity));
          setHasUnreadChat(false);
          setChatAutoScroll(true);
          // Remove this channel's messages from the unseen list
          setUnseenMessages((u) => u.filter((ch) => ch.channel !== channel));
        });
    }
  }, [channel, addChats, clearChat, loggedIn]);

  // When user is scrolling, auto-scroll may be enabled or disabled.
  // This handler is set through onScroll.
  const handleChatScrolled = useCallback(() => {
    if (chatTab) {
      // Are we within a few pixels of the end? Then go ahead and autoscroll
      // The wiggle room accounts for some throttling of onScroll
      if (
        chatTab.scrollTop >=
        chatTab.scrollHeight - chatTab.clientHeight - 24
      ) {
        setChatAutoScroll(true);
        setHasUnreadChat(false);
      } else {
        setChatAutoScroll(false);
      }
    }
  }, [chatTab]);

  const calculatePMChannel = useCallback(
    (receiverID: string) => {
      let u1 = userID;
      let u2 = receiverID;
      if (u2 < u1) {
        [u1, u2] = [u2, u1];
      }
      return `chat.pm.${u1}_${u2}`;
    },
    [userID]
  );

  const entities = useMemo(
    () =>
      chatEntities
        ?.filter((ent: ChatEntityObj) => {
          if (ent.entityType === ChatEntityType.UserChat) {
            return ent.channel === channel;
          }
          return true;
        })
        .map((ent) => {
          const anon = ent.senderId ? !knownUsers.includes(ent.senderId) : true;
          const specialSender = props.highlight
            ? props.highlight.includes(ent.sender)
            : false;
          return (
            <ChatEntity
              entityType={ent.entityType}
              key={ent.id}
              sender={ent.sender}
              senderId={ent.senderId}
              message={ent.message}
              channel={ent.channel}
              sendChannel={channel}
              timestamp={ent.timestamp}
              anonymous={anon}
              highlight={specialSender}
              sendMessage={
                loggedIn
                  ? (userID: string, username: string) => {
                      setChannel(calculatePMChannel(userID));
                      setDescription(`Chat with ${username}`);
                      setShowChannels(false);
                    }
                  : undefined
              }
            />
          );
        }),
    [
      knownUsers,
      chatEntities,
      props.highlight,
      channel,
      loggedIn,
      calculatePMChannel,
    ]
  );

  const handleTabClick = useCallback((key) => {
    setSelectedChatTab(key);
  }, []);
  const handleHideList = useCallback(() => {
    setPresenceVisible(false);
  }, []);
  const handleShowList = useCallback(() => {
    setPresenceVisible(true);
  }, []);

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        // Send if non-trivial
        const msg = curMsg.trim();
        setCurMsg('');

        if (msg === '') {
          return;
        }
        if (!loggedIn) {
          return;
        }
        propsSendChat(msg, channel);
        // This may not be a good idea. User will miss unread messages.
        setChatAutoScroll(true);
        doChatAutoScroll();
      }
    },
    [curMsg, doChatAutoScroll, loggedIn, propsSendChat, channel]
  );
  return (
    <Card className="chat">
      <Tabs defaultActiveKey="CHAT" centered onTabClick={handleTabClick}>
        {/* TabPane for available players to chat with goes here:
          past chats, friends, all online players.
          It's not the same as the users in this current chat group.
         */}
        <TabPane tab={<>Players{/* Notification dot */}</>} key="PLAYERS">
          Coming soon! This will be your friends list.
        </TabPane>
        <TabPane tab="Chat" key="CHAT">
          {showChannels ? (
            <ChatChannels
              defaultChannel={props.defaultChannel}
              defaultDescription={props.defaultDescription}
              updatedChannels={updatedChannels}
              unseenMessages={unseenMessages}
              onChannelSelect={(ch: string, desc: string) => {
                if (channel !== ch) {
                  setChannel(ch);
                  setDescription(desc);
                }
                setShowChannels(false);
              }}
              sendMessage={
                loggedIn
                  ? (userID: string, username: string) => {
                      setChannel(calculatePMChannel(userID));
                      setDescription(`Chat with ${username}`);
                      setShowChannels(false);
                    }
                  : undefined
              }
            />
          ) : (
            <>
              <div className={`chat-context${hasScroll ? ' scrolling' : ''}`}>
                {loggedIn ? (
                  <p
                    className="breadcrumb clickable"
                    onClick={() => {
                      setShowChannels(!showChannels);
                      fetchChannels();
                    }}
                  >
                    <LeftOutlined /> All Chats
                    {(updatedChannels.size > 0 || unseenMessages.length > 0) &&
                      ' •'}
                  </p>
                ) : null}
                <p>
                  {decoratedDescription}
                  {hasUnreadChat && ' •'}
                </p>
                {presenceCount && !channel.startsWith('chat.pm.') ? (
                  <>
                    <p className="presence-count">
                      <span>{props.peopleOnlineContext(presenceCount)}</span>
                      {presenceVisible ? (
                        <span className="list-trigger" onClick={handleHideList}>
                          Hide list
                        </span>
                      ) : (
                        <span className="list-trigger" onClick={handleShowList}>
                          Show list
                        </span>
                      )}
                    </p>
                    {presenceVisible ? (
                      <p className="presence">
                        <Presences
                          players={props.presences}
                          channel={channel}
                          sendMessage={
                            loggedIn
                              ? (userID: string, username: string) => {
                                  setChannel(calculatePMChannel(userID));
                                  setDescription(`Chat with ${username}`);
                                  setShowChannels(false);
                                }
                              : undefined
                          }
                        />
                      </p>
                    ) : null}
                  </>
                ) : null}
              </div>
              <div
                className="entities"
                ref={setTabContainerElement}
                onScroll={handleChatScrolled}
              >
                {entities}
              </div>
              <Input
                placeholder="chat..."
                disabled={!loggedIn}
                onKeyDown={onKeyDown}
                onChange={onChange}
                value={curMsg}
                spellCheck={false}
              />
            </>
          )}
          {/* <Button onClick={props.DISCONNECT}>DISCONNECT</Button> */}
        </TabPane>
      </Tabs>
    </Card>
  );
});
