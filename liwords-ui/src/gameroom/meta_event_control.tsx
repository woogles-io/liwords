import React, { useCallback, useEffect } from 'react';
import { MessageType } from '../gen/api/proto/ipc/ipc_pb';
import { GameMetaEvent } from '../gen/api/proto/ipc/omgwords_pb';
import { MetaStates } from '../store/meta_game_events';
import { useGameMetaEventContext } from '../store/store';
import { useMountedState } from '../utils/mounted';
import { encodeToSocketFmt } from '../utils/protobuf';
import { ShowNotif } from './show_notif';

type Props = {
    sendSocketMsg: (msg: Uint8Array) => void;
    gameID: string;
};

export const MetaEventControl = (props: Props) => {
    const { gameMetaEventContext } = useGameMetaEventContext();
    const { sendSocketMsg, gameID } = props;
    const { useState } = useMountedState();
    // can't get this to work with types:
    const [activeNotif, setActiveNotif] = useState<React.ReactElement | null>(
        null
    );

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

    const denyAdjudication = useCallback(
        (evtid: string) => {
            const deny = new GameMetaEvent();
            deny.setType(GameMetaEvent.EventType.ADJUDICATION_DENIED);
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

    const eventTimeout = useCallback(
        (evtid: string) => {
            const to = new GameMetaEvent();
            to.setType(GameMetaEvent.EventType.TIMER_EXPIRED);
            to.setOrigEventId(evtid);
            to.setGameId(gameID);

            sendSocketMsg(
                encodeToSocketFmt(MessageType.GAME_META_EVENT, to.serializeBinary())
            );
        },
        [sendSocketMsg, gameID]
    );

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
    ]);

    return activeNotif;
};
