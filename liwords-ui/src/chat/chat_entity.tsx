import React from 'react';
import moment from 'moment';
import {
  ChatEntityType,
  useExcludedPlayersStoreContext,
  useModeratorStoreContext,
} from '../store/store';
import { UsernameWithContext } from '../shared/usernameWithContext';
import { Wooglinkify } from '../shared/wooglinkify';
import { Modal, Tag } from 'antd';
import {
  CrownFilled,
  SafetyCertificateFilled,
  StarFilled,
} from '@ant-design/icons';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { moderateUser, deleteChatMessage } from '../mod/moderate';
import { PlayerAvatar } from '../shared/player_avatar';

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
    title: (
      <p className="readable-text-color">Do you want to delete this message</p>
    ),
    icon: <ExclamationCircleOutlined />,
    content: <p className="readable-text-color">{message}</p>,
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
  const { moderators, admins } = useModeratorStoreContext();
  if (props.timestamp) {
    if (
      moment(Date.now()).format('MMM Do') !==
      moment(props.timestamp).format('MMM Do')
    ) {
      ts = moment(props.timestamp).format('MMM Do - LT');
    } else {
      ts = moment(props.timestamp).format('LT');
    }
  }
  let el;
  let senderClass = 'sender';
  let fromMod = false;
  let fromAdmin = false;
  const channel = '';

  // Don't render until we know who's been blocked
  if (!excludedPlayersFetched) {
    return null;
  }

  if (props.senderId && excludedPlayers.has(props.senderId)) {
    return null;
  }
  if (props.senderId && moderators.has(props.senderId)) {
    fromMod = true;
  }
  if (props.senderId && admins.has(props.senderId)) {
    fromAdmin = true;
  }
  if (props.highlight || fromMod || fromAdmin) {
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
          <PlayerAvatar
            player={{
              user_id: props.senderId,
            }}
            username={props.sender}
          />
          <div className="message-details">
            <p className="sender-info">
              <span className={senderClass}>
                <UsernameWithContext
                  username={props.sender}
                  userID={props.senderId}
                  includeFlag
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
                  <Tag
                    className="director"
                    icon={<CrownFilled />}
                    color={'#d5cad6'}
                  >
                    {props.highlightText}
                  </Tag>
                )}
                {!props.highlight && fromAdmin && (
                  <Tag
                    className="admin"
                    icon={<StarFilled />}
                    color={'#F4B000'}
                  >
                    Admin
                  </Tag>
                )}
                {!props.highlight && !fromAdmin && fromMod && (
                  <Tag
                    className="mod"
                    icon={<SafetyCertificateFilled />}
                    color={'#E6FFDF'}
                  >
                    Moderator
                  </Tag>
                )}
                <span className="timestamp">
                  {ts} {channel}
                </span>
              </span>
            </p>
            <p>
              <span className="message">
                <Wooglinkify message={props.message} />
              </span>
            </p>
          </div>
        </div>
      );
      break;
    default:
      el = null;
  }
  return el;
};
