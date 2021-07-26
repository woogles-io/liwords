import {
  useFriendsStoreContext,
  useLoginStateStoreContext,
} from '../store/store';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import React from 'react';
import { notification } from 'antd';

type FollowerProps = {
  className?: string;
  target: string;
  tagName?: string;
  friendCallback?: () => void;
};

export const TheFollower = (props: FollowerProps) => {
  const { friends, setPendingFriendsRefresh } = useFriendsStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { userID } = loginState;

  if (userID === props.target) {
    return null;
  }

  let apiFunc: string;
  let friendText: string;
  if (friends[props.target]) {
    apiFunc = 'Remove';
    friendText = 'Remove from friends';
  } else {
    apiFunc = 'Add';
    friendText = 'Add friend';
    // Add some confirmation.
  }

  const friendAction = () => {
    axios
      .post(
        toAPIUrl('user_service.SocializeService', `${apiFunc}Follow`),
        {
          uuid: props.target,
        },
        { withCredentials: true }
      )
      .then(() => {
        if (props.friendCallback) {
          props.friendCallback();
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
      })
      .finally(() => {
        setPendingFriendsRefresh(true);
      });
  };

  const DynamicTagName = (props.tagName ||
    'span') as keyof JSX.IntrinsicElements;
  return (
    <DynamicTagName onClick={friendAction} className={props.className || ''}>
      {friendText}
    </DynamicTagName>
  );
};
