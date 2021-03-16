import { message } from 'antd';
import { GameMetaEvent } from '../gen/api/proto/realtime/realtime_pb';
import { PlayerOrder } from './constants';
import { ClockController, Millis } from './timer_controller';

export enum MetaStates {
  NO_ACTIVE_REQUEST,
  REQUESTED_ABORT,
  REQUESTED_ADJUDICATION,
  RECEIVER_ABORT_COUNTDOWN,
  RECEIVER_ADJUDICATION_COUNTDOWN,
}

export type MetaEventState = {
  curEvt: MetaStates;
  // clockController: React.MutableRefObject<ClockController | null>;
  // onClockTick: (p: PlayerOrder, t: Millis) => void;
  // onClockTimeout: (p: PlayerOrder) => void;
};

// export const metaStateInitializer = (
//   clockController: React.MutableRefObject<ClockController | null>,
//   onClockTick: (p: PlayerOrder, t: Millis) => void,
//   onClockTimeout: (p: PlayerOrder) => void
// ): MetaEventState => ({
//   curEvt: MetaStates.NO_ACTIVE_REQUEST,
//   clockController,
//   onClockTick,
//   onClockTimeout,
// });

export const metaStateFromMetaEvent = (
  oldState: MetaEventState,
  metaEvent: GameMetaEvent,
  us: string
) => {
  let metaState = MetaStates.NO_ACTIVE_REQUEST;
  switch (metaEvent.getType()) {
    case GameMetaEvent.EventType.REQUEST_ABORT: {
      if (us === metaEvent.getPlayerId()) {
        metaState = MetaStates.REQUESTED_ABORT;
      } else {
        metaState = MetaStates.RECEIVER_ABORT_COUNTDOWN;
      }
      break;
    }

    case GameMetaEvent.EventType.REQUEST_ADJUDICATION: {
      if (us === metaEvent.getPlayerId()) {
        metaState = MetaStates.REQUESTED_ADJUDICATION;
      } else {
        metaState = MetaStates.RECEIVER_ADJUDICATION_COUNTDOWN;
      }
      break;
    }

    case GameMetaEvent.EventType.ABORT_DENIED: {
      message.info({
        content: 'The abort request was denied',
      });
      break;
    }
  }

  return {
    ...oldState,
    curEvt: metaState,
  };
};
