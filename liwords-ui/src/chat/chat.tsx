/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-static-element-interactions */
import React, { useEffect, useRef, useState } from 'react';
import { Card, Input, Tabs, Popover } from 'antd';
import { ChatEntity } from './chat_entity';
import { ChatEntityObj, PresenceEntity } from '../store/store';
import './chat.scss';
import { Presences } from './presences';

const { TabPane } = Tabs;

type Props = {
  peopleOnlineContext: string; // the name for the people in this chat channel
  chatEntities: Array<ChatEntityObj> | undefined;
  sendChat: (msg: string) => void;
  description: string;
  presences: { [uuid: string]: PresenceEntity };
};

export const Chat = React.memo((props: Props) => {
  const [curMsg, setCurMsg] = useState('');
  const [selectedChatTab, setSelectedChatTab] = useState('CHAT');
  // const presenceCount = Object.keys(props.presences).length;
  const el = useRef<HTMLDivElement>(null);
  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      // Send if non-trivial
      const msg = curMsg.trim();
      setCurMsg('');

      if (msg === '') {
        return;
      }
      props.sendChat(msg);
    }
  };

  const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setCurMsg(e.target.value);
  };

  useEffect(() => {
    const tabContainer = el.current?.closest('.ant-tabs-content-holder');
    if (tabContainer && selectedChatTab === 'CHAT') {
      tabContainer.scrollTop = tabContainer.scrollHeight || 0;
    }
  }, [props.chatEntities, selectedChatTab]);

  const entities = props.chatEntities?.map((ent) => {
    return (
      <ChatEntity
        entityType={ent.entityType}
        key={ent.id}
        sender={ent.sender}
        message={ent.message}
        timestamp={ent.timestamp}
      />
    );
  });
  const renderChatTabHeader = () => (
    <>
      {props.description}
      {Object.keys(props.presences).length ? (
        <Popover
          overlayClassName="presences"
          placement="bottomRight"
          title="Online"
          arrowPointAtCenter
          content={<Presences players={props.presences} />}
          trigger="hover"
        ></Popover>
      ) : null}
    </>
  );
  return (
    <Card className="chat">
      <Tabs
        defaultActiveKey="CHAT"
        centered
        onTabClick={(key) => {
          setSelectedChatTab(key);
        }}
      >
        {/* TabPane for available players to chat with goes here:
          past chats, friends, all online players.
          It's not the same as the users in this current chat group.
         */}
        <TabPane tab={<>Players{/* Notification dot */}</>} key="PLAYERS">
          Coming soon! This will be a list of friends and other players to chat
          with.
        </TabPane>
        <TabPane tab={renderChatTabHeader()} key="CHAT">
          <div className="entities" ref={el}>
            {entities}
          </div>
          <Input
            placeholder="chat..."
            onKeyDown={onKeyDown}
            onChange={onChange}
            value={curMsg}
          />
        </TabPane>
      </Tabs>
    </Card>
  );
});
