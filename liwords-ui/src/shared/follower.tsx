import {
  useFriendsStoreContext,
  useLoginStateStoreContext,
} from '../store/store';
import React, { forwardRef, useImperativeHandle } from 'react';
import { flashError, useClient } from '../utils/hooks/connect';
import { SocializeService } from '../gen/api/proto/user_service/user_service_connectweb';

type FollowerProps = {
  className?: string;
  target: string;
  tagName?: string;
  friendCallback?: () => void;
};

export type FollowerHandle = {
  friendAction: () => void;
};

export const TheFollower = forwardRef((props: FollowerProps, ref) => {
  const { friends, setPendingFriendsRefresh } = useFriendsStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { userID } = loginState;
  const socializeClient = useClient(SocializeService);

  const friendAction = async () => {
    try {
      await socializeClient[apiFunc]({ uuid: props.target });
      if (props.friendCallback) {
        props.friendCallback();
      }
    } catch (e) {
      flashError(e);
    } finally {
      setPendingFriendsRefresh(true);
    }
  };

  useImperativeHandle(ref, () => ({
    friendAction,
  }));

  if (userID === props.target) {
    return null;
  }

  let apiFunc: 'addFollow' | 'removeFollow';
  let friendText: string;
  if (friends[props.target]) {
    apiFunc = 'removeFollow';
    friendText = 'Remove from friends';
  } else {
    apiFunc = 'addFollow';
    friendText = 'Add friend';
    // Add some confirmation.
  }

  const DynamicTagName = (props.tagName ||
    'span') as keyof JSX.IntrinsicElements;
  return (
    <DynamicTagName className={props.className || ''}>
      {friendText}
    </DynamicTagName>
  );
});
