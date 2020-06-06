import React from 'react';
import { Card } from 'antd';
import { ChatEntity } from './chat_entity';
import { ChatEntityObj } from '../store/store';

type Props = {
  chatEntities: Array<ChatEntityObj>;
};

export const Chat = (props: Props) => {
  const entities = props.chatEntities.map((ent) => {
    return (
      <ChatEntity
        entityType={ent.entityType}
        key={ent.id}
        sender={ent.sender}
        message={ent.message}
      />
    );
  });

  return <Card style={{ textAlign: 'left' }}>{entities}</Card>;
};
