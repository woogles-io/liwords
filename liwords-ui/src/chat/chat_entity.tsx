import React from 'react';
import moment from 'moment';
import { ChatEntityType, useExcludedPlayersStoreContext } from '../store/store';
import { UsernameWithContext } from '../shared/usernameWithContext';
import { Wooglinkify } from '../shared/wooglinkify';
import { Modal, Tag } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { moderateUser, deleteChatMessage } from '../mod/moderate';

type EntityProps = {
  entityType: ChatEntityType;
  sender: string;
  senderId?: string;
  channel: string;
  msgID: string;
  message: string;
  timestamp?: number;
  anonymous?: boolean;
  highlight: boolean;
  highlightText?: string;
  sendMessage?: (uuid: string, username: string) => void;
};

const deleteMessage = (
  sender: string,
  msgid: string,
  message: string,
  channel: string
) => {
  Modal.confirm({
    title: 'Do you want to delete this message?',
    icon: <ExclamationCircleOutlined />,
    content: message,
    onOk() {
      deleteChatMessage(sender, msgid, channel);
    },
    onCancel() {
      console.log('no');
    },
  });
};

export const ChatEntity = (props: EntityProps) => {
  let ts = '';

  const {
    excludedPlayers,
    excludedPlayersFetched,
  } = useExcludedPlayersStoreContext();
  if (props.timestamp) {
    ts = moment(props.timestamp).format('MMM Do - LT');
  }
  let el;
  let senderClass = 'sender';
  const channel = '';

  // Don't render until we know who's been blocked
  if (!excludedPlayersFetched) {
    return null;
  }

  if (props.senderId && excludedPlayers.has(props.senderId)) {
    return null;
  }
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
                omitSendMessage={!props.sendMessage}
                sendMessage={props.sendMessage}
                showDeleteMessage
                showModTools
                deleteMessage={() => {
                  if (props.senderId) {
                    deleteMessage(
                      props.senderId,
                      props.msgID,
                      props.message,
                      props.channel
                    );
                  }
                }}
                moderate={moderateUser}
              />
              {props.highlightText && props.highlight && (
                <Tag color={'#d5cad6'}>{props.highlightText}</Tag>
              )}
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
