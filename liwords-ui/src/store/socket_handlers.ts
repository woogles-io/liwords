import { useCallback, useRef } from 'react';
import { useHistory } from 'react-router-dom';
import { message, notification } from 'antd';
import {
  ChatEntityType,
  randomID,
  ChatEntityObj,
  PresenceEntity,
  useChallengeResultEventStoreContext,
  useChatStoreContext,
  useExamineStoreContext,
  useExcludedPlayersStoreContext,
  useGameContextStoreContext,
  useGameEndMessageStoreContext,
  useLagStoreContext,
  useLobbyStoreContext,
  useLoginStateStoreContext,
  usePresenceStoreContext,
  useRematchRequestStoreContext,
  useTimerStoreContext,
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
  MatchRequestCancellation,
  TournamentGameEndedEvent,
  RematchStartedEvent,
} from '../gen/api/proto/realtime/realtime_pb';
import { ActionType } from '../actions/actions';
import { endGameMessage } from './end_of_game';
import {
  SeekRequestToSoughtGame,
  SoughtGame,
  GameInfoResponseToActiveGame,
} from './reducers/lobby_reducer';
import { BoopSounds } from '../sound/boop';
import { ActiveChatChannels } from '../gen/api/proto/user_service/user_service_pb';
import {
  GameInfoResponse,
  GameInfoResponses,
} from '../gen/api/proto/game_service/game_service_pb';

// Feature flag.
export const enableShowSocket =
  localStorage?.getItem('enableShowSocket') === 'true';

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
      // XXX: delete these next two.
      [MessageType.GAME_META_EVENT]: GameMeta,
      [MessageType.ACTIVE_GAMES]: ActiveGames,
      // ...
      [MessageType.ONGOING_GAME_EVENT]: GameInfoResponse,
      [MessageType.ONGOING_GAMES]: GameInfoResponses,
      [MessageType.GAME_DELETION]: GameDeletion,
      [MessageType.MATCH_REQUESTS]: MatchRequests,
      [MessageType.DECLINE_MATCH_REQUEST]: DeclineMatchRequest,
      [MessageType.CHAT_MESSAGE]: ChatMessage,
      [MessageType.CHAT_MESSAGES]: ChatMessages,
      [MessageType.USER_PRESENCE]: UserPresence,
      [MessageType.USER_PRESENCES]: UserPresences,
      [MessageType.READY_FOR_GAME]: ReadyForGame,
      [MessageType.LAG_MEASUREMENT]: LagMeasurement,
      [MessageType.MATCH_REQUEST_CANCELLATION]: MatchRequestCancellation,
      [MessageType.TOURNAMENT_GAME_ENDED_EVENT]: TournamentGameEndedEvent,
      [MessageType.REMATCH_STARTED]: RematchStartedEvent,
      [MessageType.CHAT_CHANNELS]: ActiveChatChannels,
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

export const ReverseMessageType = (() => {
  const ret = [];
  for (const k in MessageType) {
    ret[(MessageType as { [key: string]: any })[k]] = k;
  }
  return ret;
})();

export const useOnSocketMsg = () => {
  const { challengeResultEvent } = useChallengeResultEventStoreContext();
  const { addChat, addChats } = useChatStoreContext();
  const { excludedPlayers } = useExcludedPlayersStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { setGameEndMessage } = useGameEndMessageStoreContext();
  const { setCurrentLagMs } = useLagStoreContext();
  const { dispatchLobbyContext } = useLobbyStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { setPresence, addPresences } = usePresenceStoreContext();
  const { setRematchRequest } = useRematchRequestStoreContext();
  const { stopClock } = useTimerStoreContext();
  const { isExamining } = useExamineStoreContext();

  const history = useHistory();
  const historyRef = useRef(history);
  historyRef.current = history;

  return useCallback(
    (reader: FileReader) => {
      if (!reader.result) {
        return;
      }
      const msgs = parseMsgs(new Uint8Array(reader.result as ArrayBuffer));

      msgs.forEach((msg) => {
        const { msgType, parsedMsg } = msg;

        if (enableShowSocket) {
          console.log(
            '%crcvd',
            'background: pink',
            ReverseMessageType[msgType] ?? msgType,
            parsedMsg.toObject(),
            performance.now()
          );
        }

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

          case MessageType.CHAT_CHANNELS: {
            const cc = parsedMsg as ActiveChatChannels;
            console.log('got chat channels', cc);

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
            console.log('gameContext', gameContext);
            let inReceiverGameList = false;
            if (receiver === loginState.username) {
              BoopSounds.playSound('matchReqSound');
              const rematchFor = mr.getRematchFor();
              const { path } = loginState;
              console.log(
                'sg',
                soughtGame.tournamentID,
                'gc',
                gameContext.gameID
              );
              if (soughtGame.tournamentID) {
                // This is a match game attached to a tourney.
                // XXX: When we have a tourney reducer we should refer to said reducer's
                //  state instead of looking at the path
                if (
                  path ===
                  `/tournament/${encodeURIComponent(soughtGame.tournamentID)}`
                ) {
                  dispatchLobbyContext({
                    actionType: ActionType.AddMatchRequest,
                    payload: soughtGame,
                  });
                  inReceiverGameList = true;
                } else if (rematchFor === gameContext.gameID) {
                  setRematchRequest(mr);
                } else {
                  notification.info({
                    message: 'Tournament Match Request',
                    description: `You have a tournament match request from ${soughtGame.seeker}. Please return to your tournament at your convenience.`,
                  });
                }
              } else if (gameContext.gameID) {
                if (rematchFor === gameContext.gameID) {
                  // Only display the rematch modal if we are the recipient
                  // of the rematch request.
                  setRematchRequest(mr);
                } else {
                  notification.info({
                    message: 'Match Request',
                    description: `You have a match request from ${soughtGame.seeker}, in the lobby.`,
                  });
                  inReceiverGameList = true;
                }
              } // else, we're in the lobby. Handle it below.
            }

            if (!inReceiverGameList) {
              dispatchLobbyContext({
                actionType: ActionType.AddMatchRequest,
                payload: soughtGame,
              });
            }
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
            message.warning({
              content: sm.getMessage(),
              duration: 3,
              key: 'server-message',
            });
            break;
          }

          case MessageType.REMATCH_STARTED: {
            const rs = parsedMsg as RematchStartedEvent;
            const gid = rs.getRematchGameId();
            const url = `/game/${encodeURIComponent(gid)}`;
            if (isExamining) {
              notification.info({
                message: 'A rematch has started',
                description: 'Click this notification to watch',
                key: 'rematch-notification',
                duration: 10, // 10 seconds,
                onClick: () => {
                  historyRef.current.replace(url);
                  notification.close('rematch-notification');
                },
              });
            } else {
              historyRef.current.replace(url);
              setGameEndMessage('');
            }
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
              channel: 'server',
            });
            break;
          }

          case MessageType.CHAT_MESSAGE: {
            const cm = parsedMsg as ChatMessage;
            if (excludedPlayers.has(cm.getUserId())) {
              break;
            }

            // XXX: This is a temporary fix while we can only display one
            // channel's chat at once.
            const { path } = loginState;
            if (
              path.startsWith('/game/') &&
              cm.getChannel().startsWith('chat.tournament')
            ) {
              break;
            }

            addChat({
              entityType: ChatEntityType.UserChat,
              sender: cm.getUsername(),
              message: cm.getMessage(),
              timestamp: cm.getTimestamp(),
              senderId: cm.getUserId(),
              channel: cm.getChannel(),
            });
            if (cm.getUsername() !== loginState.username) {
              // BoopSounds.playSound('receiveMsgSound');
              // Not yet, until we figure out how to just play it for private
              // msgs.
            }
            break;
          }

          case MessageType.CHAT_MESSAGES: {
            // These replace all existing messages.
            const cms = parsedMsg as ChatMessages;

            // XXX: This is a temporary fix while we can only display one
            // channel's chat at once.
            const { path } = loginState;
            if (
              path.startsWith('/game/') &&
              (cms.getMessagesList().length === 0 ||
                cms
                  .getMessagesList()[0]
                  ?.getChannel()
                  .startsWith('chat.tournament'))
            ) {
              break;
            }

            const entities = new Array<ChatEntityObj>();

            cms.getMessagesList().forEach((cm) => {
              if (!excludedPlayers.has(cm.getUserId())) {
                entities.push({
                  entityType: ChatEntityType.UserChat,
                  sender: cm.getUsername(),
                  message: cm.getMessage(),
                  timestamp: cm.getTimestamp(),
                  senderId: cm.getUserId(),
                  id: randomID(),
                  channel: cm.getChannel(),
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
            // XXX: This is a temporary fix while we can only display one
            // channel's presence at once.
            const { path } = loginState;
            if (
              path.startsWith('/game/') &&
              up.getChannel().startsWith('chat.tournament')
            ) {
              break;
            }
            setPresence({
              uuid: up.getUserId(),
              username: up.getUsername(),
              channel: up.getChannel(),
              anon: up.getIsAnonymous(),
              deleting: up.getDeleting(),
            });
            break;
          }

          case MessageType.USER_PRESENCES: {
            const ups = parsedMsg as UserPresences;

            const toAdd = new Array<PresenceEntity>();

            // XXX: This is a temporary fix while we can only display one
            // channel's presence at once.
            const { path } = loginState;
            if (
              path.startsWith('/game/') &&
              (ups.getPresencesList().length === 0 ||
                ups
                  .getPresencesList()[0]
                  ?.getChannel()
                  .startsWith('chat.tournament'))
            ) {
              break;
            }

            ups.getPresencesList().forEach((p) => {
              if (!excludedPlayers.has(p.getUserId())) {
                toAdd.push({
                  uuid: p.getUserId(),
                  username: p.getUsername(),
                  channel: p.getChannel(),
                  anon: p.getIsAnonymous(),
                  deleting: p.getDeleting(),
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

            dispatchGameContext({
              actionType: ActionType.EndGame,
              payload: gee,
            });

            BoopSounds.playSound('endgameSound');
            break;
          }

          case MessageType.TOURNAMENT_GAME_ENDED_EVENT: {
            const gee = parsedMsg as TournamentGameEndedEvent;
            dispatchLobbyContext({
              actionType: ActionType.AddTourneyGame,
              payload: gee,
            });

            dispatchLobbyContext({
              actionType: ActionType.RemoveActiveGame,
              payload: gee.getGameId(),
            });

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
            historyRef.current.replace(`/game/${encodeURIComponent(gid)}`);
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

            // If there is an Antd message about "waiting for game", destroy it.
            // XXX: This is a bit unideal.
            message.destroy('server-message');
            break;
          }

          case MessageType.SERVER_GAMEPLAY_EVENT: {
            const sge = parsedMsg as ServerGameplayEvent;
            console.log('got server event', sge.toObject());
            dispatchGameContext({
              actionType: ActionType.AddGameEvent,
              payload: sge,
            });
            // play sound
            if (loginState.username === sge.getEvent()?.getNickname()) {
              BoopSounds.playSound('makeMoveSound');
            } else {
              BoopSounds.playSound('oppMoveSound');
            }
            break;
          }

          case MessageType.SERVER_CHALLENGE_RESULT_EVENT: {
            const sge = parsedMsg as ServerChallengeResultEvent;
            console.log('got server challenge result event', sge);
            challengeResultEvent(sge);
            if (!sge.getValid()) {
              BoopSounds.playSound('woofSound');
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

          case MessageType.MATCH_REQUEST_CANCELLATION: {
            const mrc = parsedMsg as MatchRequestCancellation;
            dispatchLobbyContext({
              actionType: ActionType.RemoveSoughtGame,
              payload: mrc.getRequestId(),
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

          // XXX: Delete me - obsolete
          case MessageType.ACTIVE_GAMES: {
            // lobby context, set list of active games
            const age = parsedMsg as ActiveGames;
            console.log('OBSOLETE, REFRESH APP', age);

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

          // XXX: Delete me - obsolete
          case MessageType.GAME_META_EVENT: {
            // lobby context, add active game
            const gme = parsedMsg as GameMeta;
            console.log('OBSOLETE, REFRESH APP', gme);

            break;
          }

          case MessageType.ONGOING_GAME_EVENT: {
            // lobby context, add active game
            const gme = parsedMsg as GameInfoResponse;
            console.log('add active game', gme);
            const activeGame = GameInfoResponseToActiveGame(gme);
            if (!activeGame) {
              return;
            }
            dispatchLobbyContext({
              actionType: ActionType.AddActiveGame,
              payload: activeGame,
            });
            break;
          }

          case MessageType.ONGOING_GAMES: {
            const age = parsedMsg as GameInfoResponses;
            console.log('got active games', age);
            dispatchLobbyContext({
              actionType: ActionType.AddActiveGames,
              payload: age
                .getGameInfoList()
                .map((g) => GameInfoResponseToActiveGame(g)),
            });
            break;
          }
        }
      });
    },
    [
      addChat,
      addChats,
      addPresences,
      challengeResultEvent,
      dispatchGameContext,
      dispatchLobbyContext,
      excludedPlayers,
      gameContext,
      loginState,
      setCurrentLagMs,
      setGameEndMessage,
      setPresence,
      setRematchRequest,
      stopClock,
      isExamining,
    ]
  );
};
