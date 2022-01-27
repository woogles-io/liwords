import { message } from 'antd';
import { GameMetaEvent } from '../gen/api/proto/ipc/omgwords_pb';
import { Millis } from './timer_controller';

export enum MetaStates {
    NO_ACTIVE_REQUEST,
    REQUESTED_ABORT,
    REQUESTED_ADJUDICATION,
    RECEIVER_ABORT_COUNTDOWN,
    RECEIVER_ADJUDICATION_COUNTDOWN,
}

export type MetaEventState = {
    curEvt: MetaStates;
    initialExpiry: Millis;
    evtId: string;
    evtCreator: string; // the user ID of the player that generated this event.
};

export const metaStateFromMetaEvent = (
    oldState: MetaEventState,
    metaEvent: GameMetaEvent,
    us: string
) => {
    let metaState = MetaStates.NO_ACTIVE_REQUEST;
    let initialExpiry = 0;
    let evtId = '';
    let evtCreator = '';
    switch (metaEvent.getType()) {
        case GameMetaEvent.EventType.REQUEST_ABORT: {
            if (us === metaEvent.getPlayerId()) {
                metaState = MetaStates.REQUESTED_ABORT;
            } else {
                metaState = MetaStates.RECEIVER_ABORT_COUNTDOWN;
            }
            initialExpiry = metaEvent.getExpiry();
            evtId = metaEvent.getOrigEventId();
            evtCreator = metaEvent.getPlayerId();
            break;
        }

        case GameMetaEvent.EventType.REQUEST_ADJUDICATION: {
            if (us === metaEvent.getPlayerId()) {
                metaState = MetaStates.REQUESTED_ADJUDICATION;
            } else {
                metaState = MetaStates.RECEIVER_ADJUDICATION_COUNTDOWN;
            }
            initialExpiry = metaEvent.getExpiry();
            evtId = metaEvent.getOrigEventId();
            evtCreator = metaEvent.getPlayerId();
            break;
        }

        case GameMetaEvent.EventType.ABORT_DENIED: {
            evtCreator = metaEvent.getPlayerId();
            let content = 'Your opponent declined your request to cancel the game.';
            if (!evtCreator) {
                // if this isn't filled in, the abort request is auto cancelled.
                content = 'The cancel request expired.';
            } else if (evtCreator === oldState.evtCreator) {
                content = 'The cancel request was withdrawn.';
            }

            message.info({
                content,
            });
            initialExpiry = 0;
            metaState = MetaStates.NO_ACTIVE_REQUEST;
            // the evtCreator is the one that denied the abort.
            evtId = '';
            break;
        }

        case GameMetaEvent.EventType.ABORT_ACCEPTED: {
            message.info({
                content: 'The cancel request was accepted.',
            });
            initialExpiry = 0;
            metaState = MetaStates.NO_ACTIVE_REQUEST;
            // the evtCreator is the one that accepted the abort.
            evtCreator = metaEvent.getPlayerId();
            evtId = '';
            break;
        }

        case GameMetaEvent.EventType.ADJUDICATION_ACCEPTED: {
            message.info({
                content: 'The game was adjudicated.',
            });
            initialExpiry = 0;
            metaState = MetaStates.NO_ACTIVE_REQUEST;
            // the evtCreator is the one that accepted the adjudication.
            evtCreator = metaEvent.getPlayerId();
            evtId = '';
            break;
        }

        case GameMetaEvent.EventType.ADJUDICATION_DENIED: {
            message.info({
                content: 'The game will continue.',
            });
            initialExpiry = 0;
            metaState = MetaStates.NO_ACTIVE_REQUEST;
            // the evtCreator is the one that denied the adjudication.
            evtCreator = metaEvent.getPlayerId();
            evtId = '';
            break;
        }
    }

    return {
        ...oldState,
        curEvt: metaState,
        initialExpiry,
        evtId,
        evtCreator,
    };
};
