import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import { Card, Input, Tabs } from 'antd';
import { LeftOutlined } from '@ant-design/icons';
import { useMountedState } from '../utils/mounted';
import { ChatEntity } from './chat_entity';
import {
  ChatEntityObj,
  ChatEntityType,
  useChatStoreContext,
  useLoginStateStoreContext,
  usePresenceStoreContext,
  useTournamentStoreContext,
} from '../store/store';
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

export type Props = {
  peopleOnlineContext: (n: number) => string; // should return "1 person" or "2 people"
  sendChat: (msg: string, chan: string) => void;
  defaultChannel: string;
  defaultDescription: string;
  DISCONNECT?: () => void;
  highlight?: Array<string>;
  highlightText?: string;
  tournamentID?: string;
};

type JSONChatChannel = {
  display_name: string;
  last_update: string;
  has_update: boolean;
  last_message?: string;
  name: string;
};

type JSONActiveChatChannels = {
  channels: Array<JSONChatChannel>;
};

export const Chat = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const { loginState } = useLoginStateStoreContext();
  const { tournamentContext } = useTournamentStoreContext();
  const { competitorState } = tournamentContext;
  const { loggedIn, userID } = loginState;
  const [curMsg, setCurMsg] = useState('');
  const [hasScroll, setHasScroll] = useState(false);
  const [channelsFetched, setChannelsFetched] = useState(false);
  const [showChannels, setShowChannels] = useState(false);
  const [selectedChatTab, setSelectedChatTab] = useState('CHAT');
  const [presenceVisible, setPresenceVisible] = useState(false);
  // We cannot useRef because we need to awoken effects relying on the element.
  const [
    tabContainerElement,
    setTabContainerElement,
  ] = useState<HTMLDivElement | null>(null);
  const propsSendChat = useMemo(() => props.sendChat, [props.sendChat]);
  const chatTab = selectedChatTab === 'CHAT' ? tabContainerElement : null;

  // Chat auto-scrolls when the last entity is visible.
  const [hasUnreadChat, setHasUnreadChat] = useState(false);
  const { defaultChannel, defaultDescription } = props;
  const {
    chat: chatEntities,
    clearChat,
    addChats,
    setChatChannels,
  } = useChatStoreContext();
  const { presences } = usePresenceStoreContext();
  const [presenceCount, setPresenceCount] = useState(0);
  const lastChannel = useRef('');
  const [chatAutoScroll, setChatAutoScroll] = useState(true);
  const [channel, setChannel] = useState<string | undefined>(defaultChannel);
  const [maxEntitiesHeight, setMaxEntitiesHeight] = useState<
    number | undefined
  >(undefined);
  const [description, setDescription] = useState(defaultDescription);
  const [defaultLastMessage, setDefaultLastMessage] = useState('');
  const [channelSelectedTime, setChannelSelectedTime] = useState(Date.now());
  const [channelReadTime, setChannelReadTime] = useState(Date.now());
  // Messages that come in for other channels
  const [unseenMessages, setUnseenMessages] = useState(
    new Array<ChatEntityObj>()
  );
  // Channels other than the current that are flagged hasUpdate. Each one is removed
  // if we switch to it
  const [updatedChannels, setUpdatedChannels] = useState<
    Set<string> | undefined
  >(undefined);
  const onChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setCurMsg(e.target.value);
  }, []);

  const doChatAutoScroll = useCallback(
    (force = false) => {
      if ((chatAutoScroll || force) && chatTab) {
        // Slight delay on this to let entities load, now that they're xhr
        setTimeout(() => {
          if (chatTab.scrollHeight > chatTab.clientHeight) {
            setHasScroll(true);
          }
          const desiredScrollTop = chatTab.scrollHeight - chatTab.clientHeight;
          // Slight wiggle room, since close enough to the bottom is close enough.
          // Otherwise it bounces on rounding errors for later changes to things that call this
          if (chatTab.scrollTop < desiredScrollTop - 6) {
            chatTab.scrollTop = desiredScrollTop;
          }
          setHasUnreadChat(false);
        }, 100);
      }
    },
    [chatAutoScroll, chatTab]
  );

  const setHeight = useCallback(() => {
    const tabPaneHeight = document.getElementById('chat')?.clientHeight;
    const contextPanelHeight = document.getElementById('chat-context')
      ?.clientHeight;
    const calculatedEntitiesHeight =
      contextPanelHeight && tabPaneHeight
        ? tabPaneHeight -
          (contextPanelHeight > 106 ? contextPanelHeight : 106) -
          100
        : undefined;
    if (maxEntitiesHeight !== calculatedEntitiesHeight) {
      setMaxEntitiesHeight(calculatedEntitiesHeight);
      doChatAutoScroll();
    }
  }, [doChatAutoScroll, maxEntitiesHeight]);

  useEffect(() => {
    setHeight();
  }, [
    updatedChannels,
    unseenMessages,
    presenceCount,
    competitorState,
    setHeight,
  ]);

  // When window is shrunk, auto-scroll may be enabled. This is one-way.
  // Hiding bookmarks bar or downloaded files bar should keep it enabled.
  const enableChatAutoScroll = useCallback(() => {
    const tab = document.getElementById('chat');
    if (tab && tab?.scrollTop >= tab?.scrollHeight - tab?.clientHeight) {
      setChatAutoScroll(true);
    }
    // When window is shrunk, keep the bottom entity instead of the top.
    doChatAutoScroll();
    setHeight();
  }, [doChatAutoScroll, setHeight]);

  const fetchChannels = useCallback(() => {
    if (loggedIn) {
      const initial = !channelsFetched;
      if (initial) {
        setChannelsFetched(true);
      }
      axios
        .post<JSONActiveChatChannels>(
          toAPIUrl('user_service.SocializeService', 'GetActiveChatChannels'),
          {
            number: 20,
            offset: 0,
            tournament_id: props.tournamentID || '',
          },
          { withCredentials: true }
        )
        .then((res) => {
          console.log('Fetched channels:', res.data.channels);
          const newChannels: ActiveChatChannels.AsObject = {
            channelsList:
              res.data.channels.map((ch) => {
                return {
                  displayName: ch.display_name,
                  lastUpdate: parseInt(ch.last_update, 10),
                  // Don't trust hasUpdate if this isn't the initial poll.
                  hasUpdate: ch.has_update && initial,
                  lastMessage: ch.last_message || '',
                  name: ch.name,
                };
              }) || [],
          };
          setChatChannels(newChannels);
          enableChatAutoScroll();
          if (initial) {
            // If these were set already, just return that list,
            // otherwise respect the hasUpdate fields

            setUpdatedChannels(
              new Set(
                newChannels?.channelsList
                  ?.filter((ch) => ch.hasUpdate)
                  ?.map((ch) => {
                    return ch.name;
                  })
              )
            );
          }
        });
    }
  }, [
    setChatChannels,
    enableChatAutoScroll,
    channelsFetched,
    loggedIn,
    props.tournamentID,
  ]);

  useEffect(() => {
    // Initial load of channels
    if (!channelsFetched) {
      fetchChannels();
    }
  }, [fetchChannels, channelsFetched]);

  useEffect(() => {
    setPresenceCount(presences.filter((p) => p.channel === channel).length);
  }, [presences, channel]);

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
    setChannel(defaultChannel);
    setDescription(defaultDescription);
  }, [defaultChannel, defaultDescription]);

  const decoratedDescription = useMemo(() => {
    const channelType = channel?.split('.')[1] || '';
    switch (channelType) {
      case 'game':
        return `Game chat with ${description}`;
      case 'gametv':
        return `Game chat for ${description}`;
    }
    return description;
  }, [channel, description]);

  useEffect(() => {
    // Remove this channel's messages from the unseen list when we switch back to message view
    setUnseenMessages((u) => u.filter((ch) => ch.channel !== channel));
    setUpdatedChannels((u) => {
      if (u) {
        const initialUpdated = Array.from(u);
        return new Set(initialUpdated.filter((ch) => ch !== channel));
      }
      return undefined;
    });
  }, [channel, showChannels]);
  useEffect(() => {
    if (chatTab || showChannels) {
      // chat entities changed.
      // Update defaultLastMessage

      const lastMessage = chatEntities.reduce(
        (acc: ChatEntityObj | undefined, ch) =>
          ch.channel === defaultChannel &&
          'timestamp' in ch &&
          (acc === undefined || ch.timestamp! > acc.timestamp!)
            ? ch
            : acc,
        undefined
      );

      setDefaultLastMessage((u) =>
        lastMessage ? `${lastMessage?.sender}: ${lastMessage?.message}` : u
      );
      // If there are new messages in this
      // channel and we've scrolled up, mark this chat unread,
      const currentUnread = chatEntities
        .filter((ch) => {
          return (
            ch.channel === channel &&
            ch.timestamp &&
            ch.timestamp > channelReadTime
          );
        })
        .sort((chA, chB) => {
          if (chB.timestamp && chA.timestamp) {
            return chB.timestamp - chA.timestamp;
          }
          return 0;
        });
      if (currentUnread.length && chatTab) {
        const hasUnread =
          chatTab.scrollTop + 6 < chatTab.scrollHeight - chatTab.clientHeight;
        setHasUnreadChat(hasUnread);
        setChannelReadTime(
          (u) => currentUnread[currentUnread.length - 1].timestamp || u
        );
      }
      // If they're for other channels or we're on the channel screen
      // add them to the unseenMessage,
      // to indicate unread in the channels component
      const otherUnreads = chatEntities
        .filter(
          // Only the ones since we switched to this channel
          (ch) =>
            (ch.channel !== channel || showChannels) &&
            ch.timestamp &&
            ch.timestamp > channelSelectedTime
        )
        .filter(
          (ch) =>
            // And not our own
            ch.senderId !== userID
        );
      if (otherUnreads.length) {
        setUnseenMessages((u) =>
          u.concat(otherUnreads).filter((ch) => ch.channel !== channel)
        );
        if (showChannels) {
          // if we have unread messages while looking at the channels, refetch them
          fetchChannels();
        }
      }
    }
  }, [
    chatTab,
    chatEntities,
    channel,
    defaultChannel,
    fetchChannels,
    channelReadTime,
    channelSelectedTime,
    showChannels,
    userID,
  ]);

  useEffect(() => {
    window.addEventListener('resize', enableChatAutoScroll);
    return () => {
      window.removeEventListener('resize', enableChatAutoScroll);
    };
  }, [enableChatAutoScroll]);

  useEffect(() => {
    // If we actually changed the channel, get the new messages
    if (channel && channel !== lastChannel.current) {
      lastChannel.current = channel || '';
      setChannelSelectedTime(Date.now());
      axios
        .post(
          toAPIUrl('user_service.SocializeService', 'GetChatsForChannel'),
          {
            channel,
          },
          { withCredentials: loggedIn }
        )
        .then((res) => {
          clearChat();
          const messages: Array<ChatMessageFromJSON> = res.data?.messages;
          if (messages) {
            addChats(messages.map(chatMessageToChatEntity));
          }
          setHasUnreadChat(false);
          setChatAutoScroll(true);
          setHeight();
        });
    } else {
      setHeight();
    }
  }, [channel, addChats, clearChat, loggedIn, setHeight]);

  // When user is scrolling, auto-scroll may be enabled or disabled.
  // This handler is set through onScroll.
  const handleChatScrolled = useCallback(() => {
    if (chatTab) {
      // Allow for 12 pixels of wiggle room for enabling auto scroll
      if (
        chatTab.scrollTop + 12 >=
        chatTab.scrollHeight - chatTab.clientHeight
      ) {
        setHasUnreadChat(false);
        setChatAutoScroll(true);
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
              timestamp={ent.timestamp}
              highlight={specialSender}
              highlightText={props.highlightText}
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
      chatEntities,
      props.highlight,
      props.highlightText,
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
      if (e.key === 'Enter' && channel) {
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
    <Card className="chat" id="chat">
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
              defaultChannel={defaultChannel}
              defaultDescription={defaultDescription}
              defaultLastMessage={defaultLastMessage}
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
              tournamentID={props.tournamentID}
            />
          ) : (
            channel && (
              <>
                <div
                  id="chat-context"
                  className={`chat-context${hasScroll ? ' scrolling' : ''}`}
                >
                  {loggedIn ? (
                    <p
                      className={`breadcrumb clickable${
                        (updatedChannels && updatedChannels!.size > 0) ||
                        unseenMessages.length > 0
                          ? ' unread'
                          : ''
                      }`}
                      onClick={() => {
                        setChannel(undefined);
                        setChannelSelectedTime(Date.now());
                        setShowChannels(true);
                        fetchChannels();
                      }}
                    >
                      <LeftOutlined /> All Chats
                      {((updatedChannels && updatedChannels!.size > 0) ||
                        unseenMessages.length > 0) && (
                        <span className="unread-marker">•</span>
                      )}
                    </p>
                  ) : null}
                  <p data-testid="description" className="description">
                    {decoratedDescription}
                    {hasUnreadChat && (
                      <span className="unread-marker" data-testid="description">
                        •
                      </span>
                    )}
                  </p>
                  {presenceCount && !channel.startsWith('chat.pm.') ? (
                    <>
                      <p className="presence-count">
                        <span>{props.peopleOnlineContext(presenceCount)}</span>
                        {presenceVisible ? (
                          <span
                            className="list-trigger"
                            onClick={handleHideList}
                          >
                            Hide list
                          </span>
                        ) : (
                          <span
                            className="list-trigger"
                            onClick={handleShowList}
                          >
                            Show list
                          </span>
                        )}
                      </p>
                      {presenceVisible ? (
                        <p className="presence">
                          <Presences
                            players={presences}
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
                  style={
                    maxEntitiesHeight
                      ? {
                          maxHeight: maxEntitiesHeight,
                        }
                      : undefined
                  }
                  ref={setTabContainerElement}
                  onScroll={handleChatScrolled}
                >
                  {entities}
                </div>
                <Input
                  autoFocus={!defaultChannel.startsWith('chat.game')}
                  placeholder="chat..."
                  disabled={!loggedIn}
                  onKeyDown={onKeyDown}
                  onChange={onChange}
                  value={curMsg}
                  spellCheck={false}
                />
              </>
            )
          )}
          {/* <Button onClick={props.DISCONNECT}>DISCONNECT</Button> */}
        </TabPane>
      </Tabs>
    </Card>
  );
});
