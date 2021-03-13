import { notification } from 'antd';
import Modal from 'antd/lib/modal/Modal';
import React, { useEffect, useState } from 'react';
import { MetaStates } from '../store/reducers/game_reducer';
import { useGameContextStoreContext } from '../store/store';

// XXX: Using this component only for its effect. Should rewrite
// as a hook only.
export const MetaEventDisplay = () => {
  const { gameContext } = useGameContextStoreContext();
  // const [modalVisible, setModalVisible] = useState(false);
  useEffect(() => {
    if (gameContext.metaState === MetaStates.NO_ACTIVE_REQUEST) {
      return;
    }
    // setModalVisible(true);
    switch (gameContext.metaState) {
      case MetaStates.REQUESTED_ABORT:
        notification.info({
          message: '',
          description:
            'Waiting for your opponent to respond to your cancel request.',
          placement: 'bottomRight',
          key: 'request-abort',
          duration: 0,
        });
        break;

      case MetaStates.RECEIVER_ABORT_COUNTDOWN:
        notification.info({
          message: '',
          description:
            "Your opponent wants to cancel the game. Ratings won't change.",
          key: 'received-abort',
          placement: 'bottomRight',
          duration: 0,
        });
        break;

      case MetaStates.REQUESTED_ADJUDICATION:
        notification.info({
          message: '',
          description: 'Waiting for your opponent to respond to your nudge.',
          key: 'request-adjudication',
          placement: 'bottomRight',
          duration: 0,
        });
        break;

      case MetaStates.RECEIVER_ADJUDICATION_COUNTDOWN:
        notification.info({
          message: '',
          description:
            'Your opponent nudged you! Hit "Keep playing" if you\'re still there.',
          key: 'received-adjudication',
          placement: 'bottomRight',
          duration: 0,
        });
        break;
    }
  }, [gameContext.metaState]);

  return null;
};
