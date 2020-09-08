import { message, notification } from 'antd';
import { StoreData, ChatEntityType, randomID } from './store';
import {
  MessageType,
  SeekRequest,
  ErrorMessage,
  ServerMessage,
  NewGameEvent,
  GameHistoryRefresher,
  MessageTypeMap,
  MatchRequest,
  SoughtGameProcessEvent,
  ClientGameplayEvent,
  ServerGameplayEvent,
  GameEndedEvent,
  ServerChallengeResultEvent,
  SeekRequests,
  TimedOut,
  GameMeta,
  ActiveGames,
  GameDeletion,
  MatchRequests,
  DeclineMatchRequest,
  ChatMessage,
  ChatMessages,
  UserPresence,
  UserPresences,
} from '../gen/api/proto/realtime/realtime_pb';
import { ActionType } from '../actions/actions';
import { endGameMessage } from './end_of_game';
import {
  SeekRequestToSoughtGame,
  GameMetaToActiveGame,
} from './reducers/lobby_reducer';
import { BoopSounds } from '../sound/boop';

export const parseMsgs = (msg: Uint8Array) => {
  // Multiple msgs can come in the same packet.
  const msgs = [];

  while (msg.length > 0) {
    const msgLength = msg[0] * 256 + msg[1];
    const msgType = msg[2] as MessageTypeMap[keyof MessageTypeMap];
    const msgBytes = msg.slice(3, 3 + (msgLength - 1));

    const msgTypes = {
      [MessageType.SEEK_REQUEST]: SeekRequest,
      [MessageType.ERROR_MESSAGE]: ErrorMessage,
      [MessageType.SERVER_MESSAGE]: ServerMessage,
      [MessageType.NEW_GAME_EVENT]: NewGameEvent,
      [MessageType.GAME_HISTORY_REFRESHER]: GameHistoryRefresher,
      [MessageType.MATCH_REQUEST]: MatchRequest,
      [MessageType.SOUGHT_GAME_PROCESS_EVENT]: SoughtGameProcessEvent,
      [MessageType.CLIENT_GAMEPLAY_EVENT]: ClientGameplayEvent,
      [MessageType.SERVER_GAMEPLAY_EVENT]: ServerGameplayEvent,
      [MessageType.GAME_ENDED_EVENT]: GameEndedEvent,
      [MessageType.SERVER_CHALLENGE_RESULT_EVENT]: ServerChallengeResultEvent,
      [MessageType.SEEK_REQUESTS]: SeekRequests,
      [MessageType.TIMED_OUT]: TimedOut,
      [MessageType.GAME_META_EVENT]: GameMeta,
      [MessageType.ACTIVE_GAMES]: ActiveGames,
      [MessageType.GAME_DELETION]: GameDeletion,
      [MessageType.MATCH_REQUESTS]: MatchRequests,
      [MessageType.DECLINE_MATCH_REQUEST]: DeclineMatchRequest,
      [MessageType.CHAT_MESSAGE]: ChatMessage,
      [MessageType.CHAT_MESSAGES]: ChatMessages,
      [MessageType.USER_PRESENCE]: UserPresence,
      [MessageType.USER_PRESENCES]: UserPresences,
    };

    const parsedMsg = msgTypes[msgType];
    const topush = {
      msgType,
      parsedMsg: parsedMsg.deserializeBinary(msgBytes),
    };
    msgs.push(topush);
    // eslint-disable-next-line no-param-reassign
    msg = msg.slice(3 + (msgLength - 1));
  }
  return msgs;
};

export const onSocketMsg = (username: string, storeData: StoreData) => {
  return (reader: FileReader) => {
    if (!reader.result) {
      return;
    }
    const msgs = parseMsgs(new Uint8Array(reader.result as ArrayBuffer));

    msgs.forEach((msg) => {
      const { msgType, parsedMsg } = msg;

      switch (msgType) {
        case MessageType.SEEK_REQUEST: {
          const sr = parsedMsg as SeekRequest;

          const soughtGame = SeekRequestToSoughtGame(sr);
          if (soughtGame === null) {
            return;
          }

          storeData.dispatchLobbyContext({
            actionType: ActionType.AddSoughtGame,
            payload: soughtGame,
          });
          break;
        }

        case MessageType.SEEK_REQUESTS: {
          const sr = parsedMsg as SeekRequests;
          storeData.dispatchLobbyContext({
            actionType: ActionType.AddSoughtGames,
            payload: sr
              .getRequestsList()
              .map((r) => SeekRequestToSoughtGame(r)),
          });

          break;
        }

        case MessageType.MATCH_REQUEST: {
          const mr = parsedMsg as MatchRequest;
          const receiver = mr.getReceivingUser()?.getDisplayName();
          const soughtGame = SeekRequestToSoughtGame(mr);
          if (soughtGame === null) {
            return;
          }
          if (receiver === username) {
            BoopSounds.matchReqSound.play();
            if (mr.getRematchFor() !== '') {
              // Only display the rematch modal if we are the recipient
              // of the rematch request.
              storeData.setRematchRequest(mr);
            }
          }

          storeData.dispatchLobbyContext({
            actionType: ActionType.AddMatchRequest,
            payload: soughtGame,
          });
          break;
        }

        case MessageType.MATCH_REQUESTS: {
          const mr = parsedMsg as MatchRequests;
          storeData.dispatchLobbyContext({
            actionType: ActionType.AddMatchRequests,
            payload: mr
              .getRequestsList()
              .map((r) => SeekRequestToSoughtGame(r)),
          });
          break;
        }

        case MessageType.SERVER_MESSAGE: {
          const sm = parsedMsg as ServerMessage;
          message.warning(sm.getMessage(), 2);
          break;
        }

        case MessageType.ERROR_MESSAGE: {
          console.log('got error msg');
          const err = parsedMsg as ErrorMessage;
          notification.error({
            message: 'Error',
            description: err.getMessage(),
          });
          storeData.addChat({
            entityType: ChatEntityType.ErrorMsg,
            sender: 'Woogles',
            message: err.getMessage(),
          });
          break;
        }

        case MessageType.CHAT_MESSAGE: {
          const cm = parsedMsg as ChatMessage;
          storeData.addChat({
            entityType: ChatEntityType.UserChat,
            sender: cm.getUsername(),
            message: cm.getMessage(),
            timestamp: cm.getTimestamp(),
          });
          break;
        }

        case MessageType.CHAT_MESSAGES: {
          // These replace all existing messages.
          const cms = parsedMsg as ChatMessages;

          const entities = cms.getMessagesList().map((cm) => ({
            entityType: ChatEntityType.UserChat,
            sender: cm.getUsername(),
            message: cm.getMessage(),
            timestamp: cm.getTimestamp(),
            id: randomID(),
          }));

          storeData.addChats(entities);
          break;
        }

        case MessageType.USER_PRESENCE: {
          console.log('userpresence', parsedMsg);

          const up = parsedMsg as UserPresence;
          storeData.setPresence({
            uuid: up.getUserId(),
            username: up.getUsername(),
            channel: up.getChannel(),
            // authenticated: true,
          });
          break;
        }

        case MessageType.USER_PRESENCES: {
          const ups = parsedMsg as UserPresences;
          const toAdd = ups.getPresencesList().map((p) => ({
            uuid: p.getUserId(),
            username: p.getUsername(),
            channel: p.getChannel(),
          }));
          console.log('userpresences', toAdd);

          storeData.addPresences(toAdd);
          break;
        }

        case MessageType.GAME_ENDED_EVENT: {
          console.log('got game end evt');
          const gee = parsedMsg as GameEndedEvent;
          storeData.setGameEndMessage(endGameMessage(gee));
          storeData.stopClock();
          BoopSounds.endgameSound.play();
          break;
        }

        case MessageType.NEW_GAME_EVENT: {
          const nge = parsedMsg as NewGameEvent;
          console.log('got new game event', nge);
          storeData.dispatchGameContext({
            actionType: ActionType.ClearHistory,
            payload: '',
          });
          const gid = nge.getGameId();
          storeData.setRedirGame(gid);
          storeData.setGameEndMessage('');
          break;
        }

        case MessageType.GAME_HISTORY_REFRESHER: {
          const ghr = parsedMsg as GameHistoryRefresher;
          console.log('got refresher event', ghr);
          storeData.dispatchGameContext({
            actionType: ActionType.RefreshHistory,
            payload: ghr,
          });
          storeData.setGameEndMessage('');

          // If there is an Antd message about "waiting for game", destroy it.
          // XXX: This is a bit unideal.
          message.destroy();
          break;
        }

        case MessageType.SERVER_GAMEPLAY_EVENT: {
          const sge = parsedMsg as ServerGameplayEvent;
          console.log('got server event', sge);
          storeData.dispatchGameContext({
            actionType: ActionType.AddGameEvent,
            payload: sge,
          });
          // play sound
          if (username === sge.getEvent()?.getNickname()) {
            BoopSounds.makeMoveSound.play();
          } else {
            BoopSounds.oppMoveSound.play();
          }
          break;
        }

        case MessageType.SERVER_CHALLENGE_RESULT_EVENT: {
          const sge = parsedMsg as ServerChallengeResultEvent;
          console.log('got server challenge result event', sge);
          storeData.challengeResultEvent(sge);
          if (!sge.getValid()) {
            BoopSounds.woofSound.play();
          }
          break;
        }

        case MessageType.SOUGHT_GAME_PROCESS_EVENT: {
          const gae = parsedMsg as SoughtGameProcessEvent;
          console.log('got game accepted event', gae);
          storeData.dispatchLobbyContext({
            actionType: ActionType.RemoveSoughtGame,
            payload: gae.getRequestId(),
          });
          break;
        }

        case MessageType.DECLINE_MATCH_REQUEST: {
          const dec = parsedMsg as DeclineMatchRequest;
          console.log('got decline match request', dec);
          storeData.dispatchLobbyContext({
            actionType: ActionType.RemoveSoughtGame,
            payload: dec.getRequestId(),
          });
          notification.info({
            message: 'Declined',
            description: 'Your match request was declined.',
          });
          break;
        }

        case MessageType.ACTIVE_GAMES: {
          // lobby context, set list of active games
          const age = parsedMsg as ActiveGames;
          console.log('got active games', age);
          storeData.dispatchLobbyContext({
            actionType: ActionType.AddActiveGames,
            payload: age.getGamesList().map((g) => GameMetaToActiveGame(g)),
          });
          break;
        }

        case MessageType.GAME_DELETION: {
          // lobby context, remove active game
          const gde = parsedMsg as GameDeletion;
          console.log('delete active game', gde);
          storeData.dispatchLobbyContext({
            actionType: ActionType.RemoveActiveGame,
            payload: gde.getId(),
          });
          break;
        }

        case MessageType.GAME_META_EVENT: {
          // lobby context, add active game
          const gme = parsedMsg as GameMeta;
          console.log('add active game', gme);
          const activeGame = GameMetaToActiveGame(gme);
          if (!activeGame) {
            return;
          }
          storeData.dispatchLobbyContext({
            actionType: ActionType.AddActiveGame,
            payload: activeGame,
          });
          break;
        }
      }
    });
  };
};
