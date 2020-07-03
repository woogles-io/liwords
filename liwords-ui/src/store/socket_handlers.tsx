import { message, notification } from 'antd';
import { StoreData, ChatEntityType } from './store';
import {
  MessageType,
  SeekRequest,
  ErrorMessage,
  NewGameEvent,
  GameHistoryRefresher,
  MessageTypeMap,
  MatchRequest,
  GameAcceptedEvent,
  ClientGameplayEvent,
  ServerGameplayEvent,
  GameEndedEvent,
  ServerChallengeResultEvent,
  SeekRequests,
  JoinPath,
  UnjoinRealm,
  TimedOut,
  TokenSocketLogin,
  GameTurnsRefresher,
  RatingMode,
} from '../gen/api/proto/realtime/realtime_pb';
import { ActionType } from '../actions/actions';
import { endGameMessage } from './end_of_game';

const makemoveMP3 = require('../assets/makemove.mp3');

const makemoveSound = new Audio(makemoveMP3);

const parseMsg = (msg: Uint8Array) => {
  const msgType = msg[0] as MessageTypeMap[keyof MessageTypeMap];
  const msgBytes = msg.slice(1);

  const msgTypes = {
    [MessageType.SEEK_REQUEST]: SeekRequest,
    [MessageType.ERROR_MESSAGE]: ErrorMessage,
    [MessageType.NEW_GAME_EVENT]: NewGameEvent,
    [MessageType.GAME_HISTORY_REFRESHER]: GameHistoryRefresher,
    [MessageType.MATCH_REQUEST]: MatchRequest,
    [MessageType.GAME_ACCEPTED_EVENT]: GameAcceptedEvent,
    [MessageType.CLIENT_GAMEPLAY_EVENT]: ClientGameplayEvent,
    [MessageType.SERVER_GAMEPLAY_EVENT]: ServerGameplayEvent,
    [MessageType.GAME_ENDED_EVENT]: GameEndedEvent,
    [MessageType.SERVER_CHALLENGE_RESULT_EVENT]: ServerChallengeResultEvent,
    [MessageType.SEEK_REQUESTS]: SeekRequests,
    [MessageType.JOIN_PATH]: JoinPath,
    [MessageType.UNJOIN_REALM]: UnjoinRealm,
    [MessageType.TIMED_OUT]: TimedOut,
    [MessageType.TOKEN_SOCKET_LOGIN]: TokenSocketLogin,
    [MessageType.GAME_TURNS_REFRESHER]: GameTurnsRefresher,
  };

  const parsedMsg = msgTypes[msgType];
  return [msgType, parsedMsg.deserializeBinary(msgBytes)];
};

export const onSocketMsg = (storeData: StoreData) => {
  return (reader: FileReader) => {
    if (!reader.result) {
      return;
    }
    const msg = new Uint8Array(reader.result as ArrayBuffer);
    const [msgType, parsedMsg] = parseMsg(msg);

    switch (msgType) {
      case MessageType.SEEK_REQUEST: {
        const sr = parsedMsg as SeekRequest;
        const gameReq = sr.getGameRequest();
        const user = sr.getUser();
        if (!gameReq || !user) {
          return;
        }
        storeData.dispatchLobbyContext({
          actionType: ActionType.AddSoughtGame,
          payload: {
            seeker: user.getDisplayName(),
            userRating: user.getRelevantRating(),
            lexicon: gameReq.getLexicon(),
            initialTimeSecs: gameReq.getInitialTimeSeconds(),
            challengeRule: gameReq.getChallengeRule(),
            seekID: gameReq.getRequestId(),
            rated: gameReq.getRatingMode() === RatingMode.RATED,
          },
        });
        break;
      }

      case MessageType.SEEK_REQUESTS: {
        const sr = parsedMsg as SeekRequests;
        storeData.dispatchLobbyContext({
          actionType: ActionType.AddSoughtGames,
          payload: sr.getRequestsList().map((r) => {
            const gameReq = r.getGameRequest()!;
            const user = r.getUser()!;
            return {
              seeker: user.getDisplayName(),
              userRating: user.getRelevantRating(),
              lexicon: gameReq.getLexicon(),
              initialTimeSecs: gameReq.getInitialTimeSeconds(),
              challengeRule: gameReq.getChallengeRule(),
              seekID: gameReq.getRequestId(),
              rated: gameReq.getRatingMode() === RatingMode.RATED,
            };
          }),
        });

        break;
      }

      case MessageType.ERROR_MESSAGE: {
        console.log('got error msg');
        const err = parsedMsg as ErrorMessage;
        storeData.setErrorMessage(err.getMessage());
        notification.error({ message: 'Error', description: err.getMessage() });
        // storeData.addChat({
        //   entityType: ChatEntityType.ErrorMsg,
        //   sender: '',
        //   message: err.getMessage(),
        // });
        break;
      }

      case MessageType.GAME_ENDED_EVENT: {
        const gee = parsedMsg as GameEndedEvent;

        storeData.addChat({
          entityType: ChatEntityType.ServerMsg,
          sender: '',
          message: endGameMessage(gee),
        });
        storeData.stopClock();
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
        break;
      }

      case MessageType.GAME_HISTORY_REFRESHER: {
        const ghr = parsedMsg as GameHistoryRefresher;
        console.log('got refresher event', ghr);
        storeData.dispatchGameContext({
          actionType: ActionType.RefreshHistory,
          payload: ghr,
        });
        // If there is an Antd message about "waiting for game", destroy it.
        // XXX: This is a bit unideal.
        message.destroy();
        break;
      }

      case MessageType.GAME_TURNS_REFRESHER: {
        const gtr = parsedMsg as GameTurnsRefresher;
        console.log('got gameturns refrsher', gtr);
        storeData.dispatchGameContext({
          actionType: ActionType.RefreshTurns,
          payload: gtr,
        });
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
        makemoveSound.play();
        break;
      }

      case MessageType.SERVER_CHALLENGE_RESULT_EVENT: {
        const sge = parsedMsg as ServerChallengeResultEvent;
        console.log('got server challenge result event', sge);
        storeData.challengeResultEvent(sge);
        break;
      }

      case MessageType.GAME_ACCEPTED_EVENT: {
        const gae = parsedMsg as GameAcceptedEvent;
        console.log('got game accepted event', gae);
        storeData.dispatchLobbyContext({
          actionType: ActionType.RemoveSoughtGame,
          payload: gae.getRequestId(),
        });
        break;
      }
    }
  };
};
