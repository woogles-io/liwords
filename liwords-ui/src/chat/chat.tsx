/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-static-element-interactions */
import React, { useCallback, useEffect, useMemo, useRef } from 'react';
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
  const el = useRef<HTMLDivElement>(null);
  const propsSendChat = useMemo(() => props.sendChat, [props.sendChat]);
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
      }
    },
    [curMsg, propsSendChat]
  );

  const onChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setCurMsg(e.target.value);
  }, []);
  const knownUsers = useMemo(
    () =>
      Object.keys(props.presences).filter((p) => !props.presences[p].anon) ||
      [],
    [props.presences]
  );
  useEffect(() => {
    const tabContainer = el.current;
    if (tabContainer && selectedChatTab === 'CHAT') {
      if (tabContainer.scrollHeight > tabContainer.clientHeight) {
        setHasScroll(true);
      }
      tabContainer.scrollTop = tabContainer.scrollHeight || 0;
    }
  }, [props.chatEntities, selectedChatTab]);

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
            <p>{props.description}</p>
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
          <div className="entities" ref={el}>
            {entities}
          </div>
          <Input
            placeholder="chat..."
            disabled={!loggedIn}
            onKeyDown={onKeyDown}
            onChange={onChange}
            value={curMsg}
          />
          {/* <Button onClick={props.DISCONNECT}>DISCONNECT</Button> */}
        </TabPane>
      </Tabs>
    </Card>
  );
});
