import React, { useCallback, useEffect, useState } from 'react';
import { MessageType } from '../gen/api/proto/ipc/ipc_pb';
import {
  GameMetaEvent_EventType,
  GameMetaEventSchema,
} from '../gen/api/proto/ipc/omgwords_pb';
import { MetaStates } from '../store/meta_game_events';
import { useGameMetaEventContext } from '../store/store';
import { encodeToSocketFmt } from '../utils/protobuf';
import { ShowNotif } from './show_notif';
import { App } from 'antd';
import { create, toBinary } from '@bufbuild/protobuf';

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  gameID: string;
};

export const MetaEventControl = (props: Props) => {
  const { gameMetaEventContext } = useGameMetaEventContext();
  const { sendSocketMsg, gameID } = props;
  // can't get this to work with types:
  const [activeNotif, setActiveNotif] = useState<React.ReactElement | null>(
    null
  );

  const denyAbort = useCallback(
    (evtid: string) => {
      const deny = create(GameMetaEventSchema, {});
      deny.type = GameMetaEvent_EventType.ABORT_DENIED;
      deny.origEventId = evtid;
      deny.gameId = gameID;
      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.GAME_META_EVENT,
          toBinary(GameMetaEventSchema, deny)
        )
      );
    },
    [sendSocketMsg, gameID]
  );

  const denyAdjudication = useCallback(
    (evtid: string) => {
      const deny = create(GameMetaEventSchema, {});
      deny.type = GameMetaEvent_EventType.ADJUDICATION_DENIED;
      deny.origEventId = evtid;
      deny.gameId = gameID;
      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.GAME_META_EVENT,
          toBinary(GameMetaEventSchema, deny)
        )
      );
    },
    [sendSocketMsg, gameID]
  );

  const acceptAbort = useCallback(
    (evtid: string) => {
      const accept = create(GameMetaEventSchema);
      accept.type = GameMetaEvent_EventType.ABORT_ACCEPTED;
      accept.origEventId = evtid;
      accept.gameId = gameID;

      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.GAME_META_EVENT,
          toBinary(GameMetaEventSchema, accept)
        )
      );
    },
    [sendSocketMsg, gameID]
  );

  const eventTimeout = useCallback(
    (evtid: string) => {
      const to = create(GameMetaEventSchema);
      to.type = GameMetaEvent_EventType.TIMER_EXPIRED;
      to.origEventId = evtid;
      to.gameId = gameID;

      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.GAME_META_EVENT,
          toBinary(GameMetaEventSchema, to)
        )
      );
    },
    [sendSocketMsg, gameID]
  );

  const { notification } = App.useApp();

  // const [renderStartTime, setRenderStartTime] = useState(performance.now());
  // const [modalVisible, setModalVisible] = useState(false);
  useEffect(() => {
    if (gameMetaEventContext.curEvt === MetaStates.NO_ACTIVE_REQUEST) {
      setActiveNotif(null);
      return;
    }

    switch (gameMetaEventContext.curEvt) {
      case MetaStates.REQUESTED_ABORT:
        setActiveNotif(
          <ShowNotif
            notification={notification}
            maxDuration={gameMetaEventContext.initialExpiry}
            onExpire={() => {
              eventTimeout(gameMetaEventContext.evtId);
            }}
            onAccept={undefined}
            onDecline={() => {
              denyAbort(gameMetaEventContext.evtId);
            }}
            introText="Waiting for your opponent to respond to your cancel request."
            countdownText="Automatic cancellation in "
            acceptText=""
            declineText="Keep playing"
          />
        );
        break;

      case MetaStates.RECEIVER_ABORT_COUNTDOWN:
        setActiveNotif(
          <ShowNotif
            notification={notification}
            maxDuration={gameMetaEventContext.initialExpiry}
            onExpire={() => {
              eventTimeout(gameMetaEventContext.evtId);
            }}
            onAccept={() => {
              acceptAbort(gameMetaEventContext.evtId);
            }}
            onDecline={() => {
              denyAbort(gameMetaEventContext.evtId);
            }}
            introText="Your opponent wants to cancel the game. Ratings won't change."
            countdownText="Automatic cancellation in "
            acceptText="Yes, cancel"
            declineText="Keep playing"
          />
        );
        break;

      case MetaStates.REQUESTED_ADJUDICATION:
        setActiveNotif(
          <ShowNotif
            notification={notification}
            maxDuration={gameMetaEventContext.initialExpiry}
            onExpire={() => {
              eventTimeout(gameMetaEventContext.evtId);
            }}
            onAccept={undefined}
            onDecline={() => {
              denyAdjudication(gameMetaEventContext.evtId);
            }}
            introText="Waiting for your opponent to respond to your nudge."
            countdownText="Automatic forfeit in "
            acceptText=""
            declineText="Keep playing"
          />
        );

        break;

      case MetaStates.RECEIVER_ADJUDICATION_COUNTDOWN:
        setActiveNotif(
          <ShowNotif
            notification={notification}
            maxDuration={gameMetaEventContext.initialExpiry}
            onExpire={() => {
              eventTimeout(gameMetaEventContext.evtId);
            }}
            onDecline={() => {
              denyAdjudication(gameMetaEventContext.evtId);
            }}
            introText={
              'Your opponent nudged you! Hit "Keep playing" if you\'re still there.'
            }
            countdownText="Automatic forfeit in "
            acceptText=""
            declineText="Keep playing"
          />
        );

        break;
    }
  }, [
    gameMetaEventContext,
    acceptAbort,
    denyAbort,
    denyAdjudication,
    eventTimeout,
    notification,
  ]);

  return activeNotif;
};
