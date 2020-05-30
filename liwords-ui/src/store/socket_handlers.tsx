import { StoreData } from './store';
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
} from '../gen/api/proto/game_service_pb';

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
        storeData.addSoughtGame({
          seeker: user.getUsername(),
          lexicon: gameReq.getLexicon(),
          initialTimeSecs: gameReq.getInitialTimeSeconds(),
          challengeRule: gameReq.getChallengeRule(),
          seekID: gameReq.getRequestId(),
        });
        break;
      }
      case MessageType.ERROR_MESSAGE: {
        const err = parsedMsg as ErrorMessage;
        // Show the error in some sort of pop-up in the future.
        console.error(err.getMessage());
        break;
      }

      case MessageType.NEW_GAME_EVENT: {
        const nge = parsedMsg as NewGameEvent;
        const gid = nge.getGameId();
        storeData.setRedirGame(gid);
        break;
      }

      case MessageType.GAME_HISTORY_REFRESHER: {
        const ghr = parsedMsg as GameHistoryRefresher;
        console.log('got refresher event', ghr);
        storeData.gameHistoryRefresher(ghr);
        break;
      }

      case MessageType.SERVER_GAMEPLAY_EVENT: {
        const sge = parsedMsg as ServerGameplayEvent;
        console.log('got server event', sge);
        storeData.processGameplayEvent(sge);
        break;
      }
    }
  };
};
