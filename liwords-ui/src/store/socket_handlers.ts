import { message, notification } from 'antd';
import {
  ChatEntityType,
  randomID,
  ChatEntityObj,
  PresenceEntity,
} from './store';
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
  ReadyForGame,
  LagMeasurement,
} from '../gen/api/proto/realtime/realtime_pb';
import { ActionType } from '../actions/actions';
import { endGameMessage } from './end_of_game';
import {
  SeekRequestToSoughtGame,
  GameMetaToActiveGame,
  SoughtGame,
} from './reducers/lobby_reducer';
import { BoopSounds } from '../sound/boop';
import {
  useChallengeResultEventStoreContext,
  useChatStoreContext,
  useExcludedPlayersStoreContext,
  useGameContextStoreContext,
  useGameEndMessageStoreContext,
  useLagStoreContext,
  useLobbyStoreContext,
  useLoginStateStoreContext,
  usePresenceStoreContext,
  useRedirGameStoreContext,
  useRematchRequestStoreContext,
  useTimerStoreContext,
} from '../store/store';

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
      [MessageType.READY_FOR_GAME]: ReadyForGame,
      [MessageType.LAG_MEASUREMENT]: LagMeasurement,
    };

    const parsedMsg = msgTypes[msgType];
    const topush = {
      msgType,
      parsedMsg: parsedMsg?.deserializeBinary(msgBytes),
    };
    msgs.push(topush);
    // eslint-disable-next-line no-param-reassign
    msg = msg.slice(3 + (msgLength - 1));
  }
  return msgs;
};

export const useOnSocketMsg = () => {
  const { challengeResultEvent } = useChallengeResultEventStoreContext();
  const { addChat, addChats } = useChatStoreContext();
  const { excludedPlayers } = useExcludedPlayersStoreContext();
  const { dispatchGameContext } = useGameContextStoreContext();
  const { setGameEndMessage } = useGameEndMessageStoreContext();
  const { setCurrentLagMs } = useLagStoreContext();
  const { dispatchLobbyContext } = useLobbyStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { setPresence, addPresences } = usePresenceStoreContext();
  const { setRedirGame } = useRedirGameStoreContext();
  const { setRematchRequest } = useRematchRequestStoreContext();
  const { stopClock } = useTimerStoreContext();

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
          console.log('Got a seek request', sr);

          const userID = sr.getUser()?.getUserId();
          if (!userID || excludedPlayers.has(userID)) {
            break;
          }

          const soughtGame = SeekRequestToSoughtGame(sr);
          if (soughtGame === null) {
            break;
          }

          dispatchLobbyContext({
            actionType: ActionType.AddSoughtGame,
            payload: soughtGame,
          });

          break;
        }

        case MessageType.SEEK_REQUESTS: {
          const sr = parsedMsg as SeekRequests;

          const soughtGames = new Array<SoughtGame>();

          sr.getRequestsList().forEach((r) => {
            const userID = r.getUser()?.getUserId();
            if (!userID || excludedPlayers.has(userID)) {
              return;
            }
            const sg = SeekRequestToSoughtGame(r);
            if (sg) {
              soughtGames.push(sg);
            }
          });

          dispatchLobbyContext({
            actionType: ActionType.AddSoughtGames,
            payload: soughtGames,
          });

          break;
        }

        case MessageType.MATCH_REQUEST: {
          const mr = parsedMsg as MatchRequest;

          const userID = mr.getUser()?.getUserId();
          if (!userID || excludedPlayers.has(userID)) {
            break;
          }

          const receiver = mr.getReceivingUser()?.getDisplayName();
          const soughtGame = SeekRequestToSoughtGame(mr);
          if (soughtGame === null) {
            break;
          }
          if (receiver === loginState.username) {
            BoopSounds.matchReqSound.play();
            if (mr.getRematchFor() !== '') {
              // Only display the rematch modal if we are the recipient
              // of the rematch request.
              setRematchRequest(mr);
            }
          }

          dispatchLobbyContext({
            actionType: ActionType.AddMatchRequest,
            payload: soughtGame,
          });
          break;
        }

        case MessageType.MATCH_REQUESTS: {
          const mr = parsedMsg as MatchRequests;

          const soughtGames = new Array<SoughtGame>();

          mr.getRequestsList().forEach((r) => {
            const userID = r.getUser()?.getUserId();
            if (!userID || excludedPlayers.has(userID)) {
              return;
            }
            const sg = SeekRequestToSoughtGame(r);
            if (sg) {
              soughtGames.push(sg);
            }
          });

          dispatchLobbyContext({
            actionType: ActionType.AddMatchRequests,
            payload: soughtGames,
          });
          break;
        }

        case MessageType.SERVER_MESSAGE: {
          const sm = parsedMsg as ServerMessage;
          message.warning(sm.getMessage(), 2);
          break;
        }

        case MessageType.LAG_MEASUREMENT: {
          const lag = parsedMsg as LagMeasurement;
          setCurrentLagMs(lag.getLagMs());
          break;
        }

        case MessageType.ERROR_MESSAGE: {
          console.log('got error msg');
          const err = parsedMsg as ErrorMessage;
          notification.error({
            message: 'Error',
            description: err.getMessage(),
          });
          addChat({
            entityType: ChatEntityType.ErrorMsg,
            sender: 'Woogles',
            message: err.getMessage(),
          });
          break;
        }

        case MessageType.CHAT_MESSAGE: {
          const cm = parsedMsg as ChatMessage;
          if (excludedPlayers.has(cm.getUserId())) {
            break;
          }

          // XXX: We should ignore this chat message if it's not for the right
          // channel.

          addChat({
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

          const entities = new Array<ChatEntityObj>();

          cms.getMessagesList().forEach((cm) => {
            if (!excludedPlayers.has(cm.getUserId())) {
              entities.push({
                entityType: ChatEntityType.UserChat,
                sender: cm.getUsername(),
                message: cm.getMessage(),
                timestamp: cm.getTimestamp(),
                id: randomID(),
              });
            }
          });

          addChats(entities);
          break;
        }

        case MessageType.USER_PRESENCE: {
          console.log('userpresence', parsedMsg);

          const up = parsedMsg as UserPresence;
          if (excludedPlayers.has(up.getUserId())) {
            break;
          }
          setPresence({
            uuid: up.getUserId(),
            username: up.getUsername(),
            channel: up.getChannel(),
            anon: up.getIsAnonymous(),
          });
          break;
        }

        case MessageType.USER_PRESENCES: {
          const ups = parsedMsg as UserPresences;

          const toAdd = new Array<PresenceEntity>();

          ups.getPresencesList().forEach((p) => {
            if (!excludedPlayers.has(p.getUserId())) {
              toAdd.push({
                uuid: p.getUserId(),
                username: p.getUsername(),
                channel: p.getChannel(),
                anon: p.getIsAnonymous(),
              });
            }
          });

          addPresences(toAdd);
          break;
        }

        case MessageType.GAME_ENDED_EVENT: {
          console.log('got game end evt');
          const gee = parsedMsg as GameEndedEvent;
          setGameEndMessage(endGameMessage(gee));
          stopClock();
          BoopSounds.endgameSound.play();
          break;
        }

        case MessageType.NEW_GAME_EVENT: {
          const nge = parsedMsg as NewGameEvent;
          console.log('got new game event', nge);

          // Determine if this is the tab that should accept the game.
          if (
            nge.getAccepterCid() !== loginState.connID &&
            nge.getRequesterCid() !== loginState.connID
          ) {
            console.log(
              'ignoring on this tab...',
              nge.getAccepterCid(),
              '-',
              nge.getRequesterCid(),
              '-',
              loginState.connID
            );
            break;
          }

          dispatchGameContext({
            actionType: ActionType.ClearHistory,
            payload: '',
          });
          const gid = nge.getGameId();
          setRedirGame(gid);
          setGameEndMessage('');
          break;
        }

        case MessageType.GAME_HISTORY_REFRESHER: {
          const ghr = parsedMsg as GameHistoryRefresher;
          console.log('got refresher event', ghr);
          dispatchGameContext({
            actionType: ActionType.RefreshHistory,
            payload: ghr,
          });
          setGameEndMessage('');

          // If there is an Antd message about "waiting for game", destroy it.
          // XXX: This is a bit unideal.
          message.destroy();
          break;
        }

        case MessageType.SERVER_GAMEPLAY_EVENT: {
          const sge = parsedMsg as ServerGameplayEvent;
          console.log('got server event', sge);
          dispatchGameContext({
            actionType: ActionType.AddGameEvent,
            payload: sge,
          });
          // play sound
          if (loginState.username === sge.getEvent()?.getNickname()) {
            BoopSounds.makeMoveSound.play();
          } else {
            BoopSounds.oppMoveSound.play();
          }
          break;
        }

        case MessageType.SERVER_CHALLENGE_RESULT_EVENT: {
          const sge = parsedMsg as ServerChallengeResultEvent;
          console.log('got server challenge result event', sge);
          challengeResultEvent(sge);
          if (!sge.getValid()) {
            BoopSounds.woofSound.play();
          }
          break;
        }

        case MessageType.SOUGHT_GAME_PROCESS_EVENT: {
          const gae = parsedMsg as SoughtGameProcessEvent;
          console.log('got game accepted event', gae);
          dispatchLobbyContext({
            actionType: ActionType.RemoveSoughtGame,
            payload: gae.getRequestId(),
          });

          break;
        }

        case MessageType.DECLINE_MATCH_REQUEST: {
          const dec = parsedMsg as DeclineMatchRequest;
          console.log('got decline match request', dec);
          dispatchLobbyContext({
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
          dispatchLobbyContext({
            actionType: ActionType.AddActiveGames,
            payload: age.getGamesList().map((g) => GameMetaToActiveGame(g)),
          });
          break;
        }

        case MessageType.GAME_DELETION: {
          // lobby context, remove active game
          const gde = parsedMsg as GameDeletion;
          console.log('delete active game', gde);
          dispatchLobbyContext({
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
          dispatchLobbyContext({
            actionType: ActionType.AddActiveGame,
            payload: activeGame,
          });
          break;
        }
      }
    });
  };
};
