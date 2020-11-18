import React, { useCallback, useEffect, useMemo } from 'react';
import { Card, Input, Tabs } from 'antd';
import { useMountedState } from '../utils/mounted';
import { ChatEntity } from './chat_entity';
import {
  ChatEntityObj,
  PresenceEntity,
  useLoginStateStoreContext,
} from '../store/store';
import './chat.scss';
import { Presences } from './presences';

const { TabPane } = Tabs;

type Props = {
  peopleOnlineContext: (n: number) => string; // should return "1 person" or "2 people"
  chatEntities: Array<ChatEntityObj> | undefined;
  sendChat: (msg: string) => void;
  description: string;
  presences: { [uuid: string]: PresenceEntity };
  DISCONNECT?: () => void;
  highlight?: Array<string>;
};

export const Chat = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn } = loginState;
  const [curMsg, setCurMsg] = useState('');
  const [hasScroll, setHasScroll] = useState(false);
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
  const chatTab = selectedChatTab === 'CHAT' ? tabContainerElement : null;
  const [chatAutoScroll, setChatAutoScroll] = useState(true);
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
    // Not sure if props.chatEntities should be a dependency.
  }, [chatTab, doChatAutoScroll, props.chatEntities]);

  useEffect(() => {
    if (chatTab) {
      // chatEntities changed. Assume it is because something was added.
      // Mark as new if the newest things are unseen.
      setHasUnreadChat(
        chatTab.scrollTop < chatTab.scrollHeight - chatTab.clientHeight
      );
    }
  }, [chatTab, props.chatEntities]);

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

  const entities = useMemo(
    () =>
      props.chatEntities?.map((ent) => {
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
            timestamp={ent.timestamp}
            anonymous={anon}
            highlight={specialSender}
          />
        );
      }),
    [knownUsers, props.chatEntities, props.highlight]
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
        propsSendChat(msg);
        // This may not be a good idea. User will miss unread messages.
        setChatAutoScroll(true);
        doChatAutoScroll();
      }
    },
    [curMsg, doChatAutoScroll, propsSendChat]
  );

  return (
    <Card className="chat">
      <Tabs defaultActiveKey="CHAT" centered onTabClick={handleTabClick}>
        {/* TabPane for available players to chat with goes here:
          past chats, friends, all online players.
          It's not the same as the users in this current chat group.
         */}
        <TabPane tab={<>Players{/* Notification dot */}</>} key="PLAYERS">
          Coming soon! This will be a list of friends and other players to chat
          with.
        </TabPane>
        <TabPane tab="Chat" key="CHAT">
          <div className={`chat-context${hasScroll ? ' scrolling' : ''}`}>
            <p>
              {props.description}
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
                    <Presences players={props.presences} />
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
          {/* <Button onClick={props.DISCONNECT}>DISCONNECT</Button> */}
        </TabPane>
      </Tabs>
    </Card>
  );
});
