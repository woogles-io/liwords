import {
  useExcludedPlayersStoreContext,
  useLoginStateStoreContext,
} from '../store/store';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import React from 'react';
import { notification } from 'antd';

type BlockerProps = {
  className?: string;
  target: string;
  tagName?: string;
  blockCallback?: () => void;
  userName?: string;
};

export const TheBlocker = (props: BlockerProps) => {
  const { excludedPlayers, setPendingBlockRefresh } =
    useExcludedPlayersStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { userID } = loginState;

  // Don't block yourself. It makes chat annoying.
  if (userID === props.target) {
    return null;
  }

  let apiFunc: string;
  let blockText: string;

  if (excludedPlayers.has(props.target)) {
    apiFunc = 'Remove';
    blockText = props.userName
      ? `Unblock ${props.userName}`
      : 'Unblock this user';
  } else {
    apiFunc = 'Add';
    blockText = props.userName ? `Block ${props.userName}` : 'Block this user';
    // Add some confirmation.
  }

  const blockAction = () => {
    axios
      .post(
        toAPIUrl('user_service.SocializeService', `${apiFunc}Block`),
        {
          uuid: props.target,
        },
        { withCredentials: true }
      )
      .then(() => {
        setPendingBlockRefresh(true);
        if (props.blockCallback) {
          props.blockCallback();
        }
      })
      .catch((e) => {
        if (e.response) {
          notification.error({
            message: 'Error',
            description: e.response.data.msg,
            duration: 4,
          });
        } else {
          console.log(e);
        }
      });
  };

  const DynamicTagName = (props.tagName ||
    'span') as keyof JSX.IntrinsicElements;
  return (
    <DynamicTagName onClick={blockAction} className={props.className || ''}>
      {blockText}
    </DynamicTagName>
  );
};
