import { MessageInstance } from "antd/lib/message/interface";
import { GameMetaEvent_EventType } from "../gen/api/proto/ipc/omgwords_pb";
import { GameMetaEvent } from "../gen/api/proto/ipc/omgwords_pb";
import { Millis } from "./timer_controller";

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

// A single add-time grant is 15 seconds; the toast shows for this long.
const ADD_TIME_SECONDS = 15;
const ADD_TIME_TOAST_DURATION = 3;

// Coalesce a burst of add-time grants (e.g. someone spamming the +time
// button) into a single, updating toast keyed per game, so the combined
// total is shown once instead of a stack of "15 seconds" messages covering
// the board. The running total resets once a burst settles.
const addTimeBursts: {
  [gameId: string]: { total: number; timer: ReturnType<typeof setTimeout> };
} = {};

export const metaStateFromMetaEvent = (
  message: MessageInstance,
  oldState: MetaEventState,
  metaEvent: GameMetaEvent,
  us: string,
) => {
  let metaState = MetaStates.NO_ACTIVE_REQUEST;
  let initialExpiry = 0;
  let evtId = "";
  let evtCreator = "";
  switch (metaEvent.type) {
    case GameMetaEvent_EventType.REQUEST_ABORT: {
      if (us === metaEvent.playerId) {
        metaState = MetaStates.REQUESTED_ABORT;
      } else {
        metaState = MetaStates.RECEIVER_ABORT_COUNTDOWN;
      }
      initialExpiry = metaEvent.expiry;
      evtId = metaEvent.origEventId;
      evtCreator = metaEvent.playerId;
      break;
    }

    case GameMetaEvent_EventType.REQUEST_ADJUDICATION: {
      if (us === metaEvent.playerId) {
        metaState = MetaStates.REQUESTED_ADJUDICATION;
      } else {
        metaState = MetaStates.RECEIVER_ADJUDICATION_COUNTDOWN;
      }
      initialExpiry = metaEvent.expiry;
      evtId = metaEvent.origEventId;
      evtCreator = metaEvent.playerId;
      break;
    }

    case GameMetaEvent_EventType.ABORT_DENIED: {
      evtCreator = metaEvent.playerId;
      let content = "Your opponent declined your request to cancel the game.";
      if (!evtCreator) {
        // if this isn't filled in, the abort request is auto cancelled, by
        // perhaps a move being made.
        content = "The cancel request expired.";
      } else if (evtCreator === oldState.evtCreator) {
        content = "The cancel request was withdrawn.";
      } else if (evtCreator === us) {
        content = "You declined your opponent's cancel request.";
      }

      message.info({
        content,
      });
      initialExpiry = 0;
      metaState = MetaStates.NO_ACTIVE_REQUEST;
      // the evtCreator is the one that denied the abort.
      evtId = "";
      break;
    }

    case GameMetaEvent_EventType.ABORT_ACCEPTED: {
      message.info({
        content: "The cancel request was accepted.",
      });
      initialExpiry = 0;
      metaState = MetaStates.NO_ACTIVE_REQUEST;
      // the evtCreator is the one that accepted the abort.
      evtCreator = metaEvent.playerId;
      evtId = "";
      break;
    }

    case GameMetaEvent_EventType.ADJUDICATION_ACCEPTED: {
      message.info({
        content: "The game was adjudicated.",
      });
      initialExpiry = 0;
      metaState = MetaStates.NO_ACTIVE_REQUEST;
      // the evtCreator is the one that accepted the adjudication.
      evtCreator = metaEvent.playerId;
      evtId = "";
      break;
    }

    case GameMetaEvent_EventType.ADJUDICATION_DENIED: {
      message.info({
        content: "The game will continue.",
      });
      initialExpiry = 0;
      metaState = MetaStates.NO_ACTIVE_REQUEST;
      // the evtCreator is the one that denied the adjudication.
      evtCreator = metaEvent.playerId;
      evtId = "";
      break;
    }

    case GameMetaEvent_EventType.ADD_TIME: {
      const gameId = metaEvent.gameId;
      const key = `add-time-${gameId}`;
      const existing = addTimeBursts[gameId];
      if (existing) {
        clearTimeout(existing.timer);
      }
      const total = (existing?.total ?? 0) + ADD_TIME_SECONDS;
      addTimeBursts[gameId] = {
        total,
        timer: setTimeout(() => {
          delete addTimeBursts[gameId];
        }, ADD_TIME_TOAST_DURATION * 1000),
      };
      const content =
        us === metaEvent.playerId
          ? `You added ${total} seconds to your opponent's clock.`
          : `Your opponent added ${total} seconds to your clock.`;
      // Stable key so repeated grants update one toast instead of stacking.
      message.info({ key, content, duration: ADD_TIME_TOAST_DURATION });
      // No state change needed - timer update comes via GameHistoryRefresher
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
