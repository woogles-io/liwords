import { Button, notification } from 'antd';
import React, { useCallback, useEffect, useRef } from 'react';
import {
  GameMetaEvent,
  MessageType,
} from '../gen/api/proto/realtime/realtime_pb';
import { MetaStates } from '../store/meta_game_events';
import { useGameMetaEventContext } from '../store/store';
import { useMountedState } from '../utils/mounted';
import { encodeToSocketFmt } from '../utils/protobuf';
import { SimpleTimer } from './simple_timer';

/*
    case ActionType.ProcessGameMetaEvent: {
      const p = action.payload as {
        gme: GameMetaEvent;
        us: string;
      };
      const newState = newGameStateFromMetaEvent(state, p.gme, p.us);
      return newState;
    }*/

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  gameID: string;
};

export const MetaEventDisplay = (props: Props) => {
  const { gameMetaEventContext } = useGameMetaEventContext();
  const { sendSocketMsg, gameID } = props;
  const { useState } = useMountedState();
  const [timerParams, setTimerParams] = useState({
    millisAtLastRefresh: 0,
    lastRefreshedPerformanceNow: performance.now(),
    isRunning: false,
  });

  const denyAbort = useCallback(
    (evtid: string) => {
      const deny = new GameMetaEvent();
      deny.setType(GameMetaEvent.EventType.ABORT_DENIED);
      deny.setOrigEventId(evtid);
      deny.setGameId(gameID);
      sendSocketMsg(
        encodeToSocketFmt(MessageType.GAME_META_EVENT, deny.serializeBinary())
      );
    },
    [sendSocketMsg, gameID]
  );

  const acceptAbort = useCallback(
    (evtid: string) => {
      const accept = new GameMetaEvent();
      accept.setType(GameMetaEvent.EventType.ABORT_ACCEPTED);
      accept.setOrigEventId(evtid);
      accept.setGameId(gameID);

      sendSocketMsg(
        encodeToSocketFmt(MessageType.GAME_META_EVENT, accept.serializeBinary())
      );
    },
    [sendSocketMsg, gameID]
  );

  // const [renderStartTime, setRenderStartTime] = useState(performance.now());
  // const [modalVisible, setModalVisible] = useState(false);
  useEffect(() => {
    if (gameMetaEventContext.curEvt === MetaStates.NO_ACTIVE_REQUEST) {
      return;
    }
    // setModalVisible(true);

    switch (gameMetaEventContext.curEvt) {
      case MetaStates.REQUESTED_ABORT:
        const startTime = performance.now();
        // setTimerParams((tp) => ({
        //   millisAtLastRefresh: tp.isRunning
        //     ? tp.millisAtLastRefresh -
        //       (startTime - tp.lastRefreshedPerformanceNow)
        //     : tp.millisAtLastRefresh,
        //   lastRefreshedPerformanceNow: startTime,
        //   isRunning: true,
        // }));
        setTimerParams((tp) => ({
          lastRefreshedPerformanceNow: startTime,
          millisAtLastRefresh: gameMetaEventContext.initialExpirySecs * 1000,
          isRunning: true,
        }));
        notification.info({
          message: '',
          description: (
            <>
              <p>
                Waiting for your opponent to respond to your cancel request.
              </p>
              <SimpleTimer {...timerParams} />
            </>
          ),
          placement: 'bottomRight',
          key: 'request-abort',
          duration: 0,
        });
        break;

      case MetaStates.RECEIVER_ABORT_COUNTDOWN:
        notification.info({
          message: '',
          description: (
            <>
              <p>
                Your opponent wants to cancel the game. Ratings won't change.
              </p>
              <Button
                type="text"
                onClick={() => {
                  denyAbort(gameMetaEventContext.evtId);
                }}
              >
                Keep playing
              </Button>
              <Button
                onClick={() => {
                  acceptAbort(gameMetaEventContext.evtId);
                }}
              >
                Yes, cancel
              </Button>
            </>
          ),
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
  }, [gameMetaEventContext, acceptAbort, denyAbort]);

  return null;
};
