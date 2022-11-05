import {
  useExcludedPlayersStoreContext,
  useLoginStateStoreContext,
} from '../store/store';
import React from 'react';
import { flashError, useClient } from '../utils/hooks/connect';
import { SocializeService } from '../gen/api/proto/user_service/user_service_connectweb';

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
  const socializeClient = useClient(SocializeService);
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

  const DynamicTagName = (props.tagName ||
    'span') as keyof JSX.IntrinsicElements;
  return (
    <DynamicTagName onClick={blockAction} className={props.className || ''}>
      {blockText}
    </DynamicTagName>
  );
};
