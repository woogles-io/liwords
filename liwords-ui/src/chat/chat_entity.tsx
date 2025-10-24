import React from "react";
import moment from "moment";
import {
  useExcludedPlayersStoreContext,
  useModeratorStoreContext,
} from "../store/store";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { Wooglinkify } from "../shared/wooglinkify";
import { App, Tag } from "antd";
import {
  CrownFilled,
  EyeFilled,
  SafetyCertificateFilled,
  StarFilled,
} from "@ant-design/icons";
import { ExclamationCircleOutlined } from "@ant-design/icons";
import { moderateUser, deleteChatMessage } from "../mod/moderate";
import { PettableAvatar, PlayerAvatar } from "../shared/player_avatar";
import { ChatEntityType } from "../store/constants";
import { useClient } from "../utils/hooks/connect";
import { ModService } from "../gen/api/proto/mod_service/mod_service_pb";
import { HookAPI } from "antd/lib/modal/useModal";
import { Client } from "@connectrpc/connect";

type EntityProps = {
  entityType: ChatEntityType;
  sender: string;
  senderId?: string;
  channel: string;
  msgID: string;
  message: string;
  timestamp?: bigint;
  anonymous?: boolean;
  highlight: boolean;
  highlightText?: string;
  sendMessage?: (uuid: string, username: string) => void;
};

const deleteMessage = (
  modal: HookAPI,
  sender: string,
  msgid: string,
  message: string,
  channel: string,
  modClient: Client<typeof ModService>,
) => {
  modal.confirm({
    title: (
      <p className="readable-text-color">Do you want to delete this message</p>
    ),
    icon: <ExclamationCircleOutlined />,
    content: <p className="readable-text-color">{message}</p>,
    onOk() {
      deleteChatMessage(sender, msgid, channel, modClient);
    },
    onCancel() {
      console.log("no");
    },
  });
};

export const ChatEntity = (props: EntityProps) => {
  let ts = "";

  const { excludedPlayers, excludedPlayersFetched } =
    useExcludedPlayersStoreContext();
  const { moderators, admins } = useModeratorStoreContext();
  if (props.timestamp) {
    if (
      moment(Date.now()).format("MMM Do") !==
      moment(Number(props.timestamp)).format("MMM Do")
    ) {
      ts = moment(Number(props.timestamp)).format("MMM Do - LT");
    } else {
      ts = moment(Number(props.timestamp)).format("LT");
    }
  }
  let el;
  let senderClass = "sender";
  let fromMod = false;
  let fromAdmin = false;
  const channel = "";
  const modClient = useClient(ModService);
  const { modal } = App.useApp();

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
    senderClass = "special-sender";
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
          <PettableAvatar>
            <PlayerAvatar
              player={{
                userId: props.senderId,
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
                          modal,
                          props.senderId,
                          props.msgID,
                          props.message,
                          props.channel,
                          modClient,
                        );
                      }
                    }}
                    moderate={moderateUser}
                  />
                  {props.highlightText && props.highlight && (
                    <Tag
                      className={
                        props.highlightText === "Invigilator"
                          ? "invigilator"
                          : "director"
                      }
                      icon={
                        props.highlightText === "Invigilator" ? (
                          <EyeFilled />
                        ) : (
                          <CrownFilled />
                        )
                      }
                      color={
                        props.highlightText === "Invigilator"
                          ? "#7eb3d6"
                          : "#d5cad6"
                      }
                    >
                      {props.highlightText}
                    </Tag>
                  )}
                  {!props.highlight && fromAdmin && (
                    <Tag
                      className="admin"
                      icon={<StarFilled />}
                      color={"#F4B000"}
                    >
                      Admin
                    </Tag>
                  )}
                  {!props.highlight && !fromAdmin && fromMod && (
                    <Tag
                      className="mod"
                      icon={<SafetyCertificateFilled />}
                      color={"#E6FFDF"}
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
          </PettableAvatar>
        </div>
      );
      break;
    default:
      el = null;
  }
  return el;
};
