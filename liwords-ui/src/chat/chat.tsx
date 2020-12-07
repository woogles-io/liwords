import React, { useCallback, useEffect, useMemo } from 'react';
import { Card, Input, Tabs } from 'antd';
import { useMountedState } from '../utils/mounted';
import { ChatEntity } from './chat_entity';
import {
  ChatEntityObj,
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

  const onChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setCurMsg(e.target.value);
  }, []);
  const knownUsers = useMemo(
    () =>
      Object.keys(props.presences).filter((p) => !props.presences[p].anon) ||
      [],
    [props.presences]
  );

  // Chat auto-scrolls when the last entity is visible.
  const [hasUnreadChat, setHasUnreadChat] = useState(false);
  const { chat: chatEntities, clearChat, addChats } = useChatStoreContext();
  const chatTab = selectedChatTab === 'CHAT' ? tabContainerElement : null;
  const [chatAutoScroll, setChatAutoScroll] = useState(true);
  const [channel, setChannel] = useState(props.defaultChannel);
  const [description, setDescription] = useState(props.defaultDescription);
  const doChatAutoScroll = useCallback(() => {
    if (chatAutoScroll && chatTab) {
      const desiredScrollTop = chatTab.scrollHeight - chatTab.clientHeight;
      // Doing this conditionally may help browser performance.
      // (Not sure.)
      if (chatTab.scrollTop !== desiredScrollTop) {
        chatTab.scrollTop = desiredScrollTop;
      }
    }
  }, [chatAutoScroll, chatTab]);

  useEffect(() => {
    if (chatTab) {
      // This code needs comments.
      // - Why do we need a hasScroll at all?
      // - What will set hasScroll back to false?
      if (chatTab.scrollHeight > chatTab.clientHeight) {
        setHasScroll(true);
      }
      doChatAutoScroll();
    }
  }, [chatTab, doChatAutoScroll, chatEntities]);

  useEffect(() => {
    if (chatTab) {
      // chat entities changed. Assume it is because something was added.
      // Mark as new if the newest things are unseen.
      setHasUnreadChat(
        chatTab.scrollTop < chatTab.scrollHeight - chatTab.clientHeight
      );
    }
  }, [chatTab, chatEntities]);

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
    axios.post(
      toAPIUrl('user_service.SocializeService', 'GetChatsForChannel'),
      {
        channel: channel,
      },
      { withCredentials: true }
    );
  }, [channel]);

  // When user is scrolling, auto-scroll may be enabled or disabled.
  // This handler is set through onScroll.
  const handleChatScrolled = useCallback(() => {
    if (chatTab) {
      if (chatTab.scrollTop >= chatTab.scrollHeight - chatTab.clientHeight) {
        setChatAutoScroll(true);
        setHasUnreadChat(false);
      } else {
        setChatAutoScroll(false);
      }
    }
  }, [chatTab]);

  const sendPrivateMessage = useCallback(
    (msg: string, receiver: string) => {
      let u1 = userID;
      let u2 = receiver;
      if (u2 < u1) {
        [u1, u2] = [u2, u1];
      }
      propsSendChat(msg, `chat.pm.${u1}_${u2}`);
    },
    [propsSendChat, userID]
  );

  const entities = useMemo(
    () =>
      chatEntities?.map((ent) => {
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
            sendMessage={sendPrivateMessage}
          />
        );
      }),
    [knownUsers, chatEntities, props.highlight, sendPrivateMessage, channel]
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
              onChannelSelect={(channel: string, description: string) => {
                setChannel(channel);
                setDescription(description);
                clearChat();
                setShowChannels(false);
              }}
            />
          ) : (
            <>
              <div className={`chat-context${hasScroll ? ' scrolling' : ''}`}>
                <p
                  className="breadcrumb"
                  onClick={() => {
                    setShowChannels(!showChannels);
                  }}
                >
                  <LeftOutlined /> All Chats
                </p>
                <p>
                  {description}
                  {hasUnreadChat && ' •'}
                </p>
                {presenceCount ? (
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
                          sendMessage={sendPrivateMessage}
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
