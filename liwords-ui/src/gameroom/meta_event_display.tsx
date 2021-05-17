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

/*
    case ActionType.ProcessGameMetaEvent: {
      const p = action.payload as {
        gme: GameMetaEvent;
        us: string;
      };
      const newState = newGameStateFromMetaEvent(state, p.gme, p.us);
      return newState;
    }*/

// This magical timer was written by Andy. I am not sure how it works.
const ShowTimer = ({
  lastRefreshedPerformanceNow,
  millisAtLastRefresh,
  isRunning,
}: {
  lastRefreshedPerformanceNow: number;
  millisAtLastRefresh: number;
  isRunning: boolean;
}) => {
  const { useState } = useMountedState();
  const [rerender, setRerender] = useState([]);
  void rerender;
  const lastRaf = useRef(0);

  const cb = useCallback(() => {
    setRerender([]);
    lastRaf.current = requestAnimationFrame(cb);
  }, []);

  useEffect(() => {
    cb();
    return () => cancelAnimationFrame(lastRaf.current);
  }, [cb]);

  const currentMillis = isRunning
    ? millisAtLastRefresh - (performance.now() - lastRefreshedPerformanceNow)
    : millisAtLastRefresh;
  const currentSec = Math.ceil(currentMillis / 1000);
  return <>{`${currentSec} second${currentSec === 1 ? '' : 's'}`}</>;
};

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  gameID: string;
};

export const MetaEventDisplay = (props: Props) => {
  const { gameMetaEventContext } = useGameMetaEventContext();
  const { sendSocketMsg, gameID } = props;
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
        console.log(
          'now',
          startTime,
          'expiry',
          gameMetaEventContext.initialExpirySecs
        );
        notification.info({
          message: '',
          description: (
            <>
              <p>
                Waiting for your opponent to respond to your cancel request.
              </p>
              <ShowTimer
                lastRefreshedPerformanceNow={startTime}
                millisAtLastRefresh={
                  gameMetaEventContext.initialExpirySecs * 1000
                }
                isRunning
              />
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
