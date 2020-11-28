import React from 'react';
import moment from 'moment';
import { ChatEntityType } from '../store/store';
import { UsernameWithContext } from '../shared/usernameWithContext';
import { Wooglinkify } from '../shared/wooglinkify';

type EntityProps = {
  entityType: ChatEntityType;
  sender: string;
  senderId?: string;
  channel: string;
  sendChannel: string;
  message: string;
  timestamp?: number;
  anonymous?: boolean;
  highlight: boolean;
  sendMessage: (msg: string, receiver: string) => void;
};

export const ChatEntity = (props: EntityProps) => {
  let ts = '';
  if (props.timestamp) {
    ts = moment(props.timestamp).format('MMM Do - LT');
  }
  let el;
  let senderClass = 'sender';
  let channel = '';
  if (props.highlight) {
    senderClass = 'special-sender';
  }
  switch (props.entityType) {
    case ChatEntityType.ServerMsg:
      el = (
        <div>
          <span className="server-message">{props.message}</span>
        </div>
      );
      break;
    case ChatEntityType.ErrorMsg:
      el = (
        <div>
          <span className="server-error">{props.message}</span>
        </div>
      );
      break;
    case ChatEntityType.UserChat:
      if (props.channel && props.sendChannel !== props.channel) {
        // This is temporary code; once we have separate chats per receiver/room/etc
        // we can probably get rid of this (and props.sendChannel as well)
        if (props.channel.startsWith('chat.tournament')) {
          channel = ' (tournament chat)';
        } else if (props.channel.startsWith('chat.pm')) {
          channel = ' (private message)';
        }
      }
      el = (
        <div className="chat-entity">
          <p className="timestamp">
            {ts}
            {channel}
          </p>
          <p className="message-body">
            <span className={senderClass}>
              <UsernameWithContext
                username={props.sender}
                userID={props.senderId}
                sendMessage={props.sendMessage}
              />
            </span>
            <span className="message">
              <Wooglinkify message={props.message} />
            </span>
          </p>
        </div>
      );
      break;
    default:
      el = null;
  }
  return el;
};
