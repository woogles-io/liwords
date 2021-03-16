import { notification } from 'antd';
import Modal from 'antd/lib/modal/Modal';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { PlayerOrder } from '../store/constants';
import { MetaStates } from '../store/meta_game_events';
import { useGameMetaEventContext } from '../store/store';
import { ClockController, Millis, Times } from '../store/timer_controller';
import { useTimer } from '../store/use_timer';

/**
 *
 */

/*
    case ActionType.ProcessGameMetaEvent: {
      const p = action.payload as {
        gme: GameMetaEvent;
        us: string;
      };
      const newState = newGameStateFromMetaEvent(state, p.gme, p.us);
      return newState;
    }*/

export type MetaDisplayState = {
  clockController: React.MutableRefObject<ClockController | null> | null;
  onClockTick: (p: PlayerOrder, t: Millis) => void;
  onClockTimeout: (p: PlayerOrder) => void;
  metaState: MetaStates;
};

const countdownTimer = () => {
  const {
    clockController,
    stopClock,
    timerContext,
    pTimedOut,
    setPTimedOut,
    onClockTick,
    onClockTimeout,
  } = useTimer();
};

type Props = {};

export const MetaEventDisplay = (props: Props) => {
  const { gameMetaEventContext } = useGameMetaEventContext();

  // const [modalVisible, setModalVisible] = useState(false);
  useEffect(() => {
    if (gameMetaEventContext.curEvt === MetaStates.NO_ACTIVE_REQUEST) {
      return;
    }
    // setModalVisible(true);
    switch (gameMetaEventContext.curEvt) {
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
  }, [gameMetaEventContext.curEvt]);

  return null;
};
