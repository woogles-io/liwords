import React, { useEffect, useRef, useState } from 'react';
import { Card, Input } from 'antd';
import { ChatEntity } from './chat_entity';
import { ChatEntityObj } from '../store/store';
import './chat.scss';

type Props = {
  chatEntities: Array<ChatEntityObj> | undefined;
  sendChat: (msg: string) => void;
  description: string;
};

export const Chat = (props: Props) => {
  const [curMsg, setCurMsg] = useState('');
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
        <div className="entities" ref={el}>
          {entities}
        </div>
        <Input
          placeholder="chat..."
          onKeyDown={onKeyDown}
          onChange={onChange}
          value={curMsg}
        />
      </Card>
    </div>
  );
};
