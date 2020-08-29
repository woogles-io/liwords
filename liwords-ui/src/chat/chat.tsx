/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-static-element-interactions */
import React, { useEffect, useRef, useState } from 'react';
import { Card, Input } from 'antd';
import { ChatEntity } from './chat_entity';
import { ChatEntityObj, PresenceEntity } from '../store/store';
import './chat.scss';
import { Presences } from './presences';

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
    const currentEl = el.current;

    if (currentEl) {
      currentEl.scrollTop = currentEl.scrollHeight || 0;
    }
  }, [props.chatEntities]);

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

  return (
    <div className="chat-area">
      <Card
        className="chat"
        style={{ textAlign: 'left' }}
        title={props.description}
      >
        <div className="tabs">
          <div
            onClick={() => {
              setSelectedChatTab('PLAYERS');
            }}
            className={selectedChatTab === 'PLAYERS' ? 'tab active' : 'tab'}
          >
            {props.peopleOnlineContext}
          </div>
          <div
            onClick={() => {
              setSelectedChatTab('CHAT');
            }}
            className={selectedChatTab === 'CHAT' ? 'tab active' : 'tab'}
          >
            Chat
          </div>
        </div>

        {selectedChatTab === 'CHAT' ? (
          <>
            <div className="entities" ref={el}>
              {entities}
            </div>
            <Input
              placeholder="chat..."
              onKeyDown={onKeyDown}
              onChange={onChange}
              value={curMsg}
            />
          </>
        ) : null}

        {selectedChatTab === 'PLAYERS' ? (
          <>
            <Presences players={props.presences} />
          </>
        ) : null}
      </Card>
    </div>
  );
});
