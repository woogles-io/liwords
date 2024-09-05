import {
  useExcludedPlayersStoreContext,
  useLoginStateStoreContext,
} from '../store/store';
import React, { forwardRef, useImperativeHandle } from 'react';
import { flashError, useClient } from '../utils/hooks/connect';
import { SocializeService } from '../gen/api/proto/user_service/user_service_connect';

type BlockerProps = {
  className?: string;
  target: string;
  tagName?: string;
  blockCallback?: () => void;
  userName?: string;
};

export type BlockerHandle = {
  blockAction: () => void;
};

export const TheBlocker = forwardRef((props: BlockerProps, ref) => {
  const { excludedPlayers, setPendingBlockRefresh } =
    useExcludedPlayersStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { userID } = loginState;
  const socializeClient = useClient(SocializeService);

  let apiFunc: 'addBlock' | 'removeBlock';
  let blockText: string;

  if (excludedPlayers.has(props.target)) {
    apiFunc = 'removeBlock';
    blockText = props.userName
      ? `Unblock ${props.userName}`
      : 'Unblock this user';
  } else {
    apiFunc = 'addBlock';
    blockText = props.userName ? `Block ${props.userName}` : 'Block this user';
    // Add some confirmation.
  }
  const blockAction = async () => {
    try {
      await socializeClient[apiFunc]({ uuid: props.target });
      setPendingBlockRefresh(true);
      if (props.blockCallback) {
        props.blockCallback();
      }
    } catch (e) {
      flashError(e);
    }
  };

  useImperativeHandle(ref, () => ({
    blockAction,
  }));

  // Don't block yourself. It makes chat annoying.
  if (userID === props.target) {
    return null;
  }

  const DynamicTagName = (props.tagName ||
    'span') as keyof JSX.IntrinsicElements;
  return (
    <DynamicTagName className={props.className || ''}>
      {blockText}
    </DynamicTagName>
  );
});
