import React from 'react';
import { Alert } from 'antd';
import { ChatEntityType } from '../store/store';

const ServerMsgColor = '#237804';
const ChatSenderColor = '#ad4e00';
const ServerErrColor = '#a8071a';

type EntityProps = {
  entityType: ChatEntityType;
  sender: string;
  message: string;
};

export const ChatEntity = (props: EntityProps) => {
  let el;
  switch (props.entityType) {
    case ChatEntityType.ServerMsg:
      el = (
        <div>
          <span style={{ color: ServerMsgColor }}>{props.message}</span>
        </div>
      );
      break;
    case ChatEntityType.ErrorMsg:
      el = (
        <div>
          <span style={{ color: ServerErrColor }}>{props.message}</span>
        </div>
      );
      break;
    case ChatEntityType.UserChat:
      el = (
        <div>
          <span style={{ color: ChatSenderColor }}>{props.sender}</span>:{' '}
          <span style={{ color: 'black' }}>{props.message}</span>
        </div>
      );
      break;
    default:
      el = null;
  }
  return el;
};
