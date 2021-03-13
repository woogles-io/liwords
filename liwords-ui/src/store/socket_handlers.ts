import { useCallback, useRef } from 'react';
import { useHistory } from 'react-router-dom';
import { message, notification } from 'antd';
import {
  ChatEntityType,
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
  useTournamentStoreContext,
} from './store';
import {
  ChatMessage,
  ChatMessageDeleted,
  ClientGameplayEvent,
  DeclineMatchRequest,
  ErrorMessage,
  FullTournamentDivisions,
  GameDeletion,
  GameEndedEvent,
  GameHistoryRefresher,
  GameMetaEvent,
  LagMeasurement,
  MatchRequest,
  MatchRequestCancellation,
  MatchRequests,
  MessageType,
  MessageTypeMap,
  NewGameEvent,
  ReadyForGame,
  ReadyForTournamentGame,
  RematchStartedEvent,
  SeekRequest,
  SeekRequests,
  ServerChallengeResultEvent,
  ServerGameplayEvent,
  ServerMessage,
  SoughtGameProcessEvent,
  TimedOut,
  TournamentDataResponse,
  TournamentDivisionDataResponse,
  TournamentDivisionDeletedResponse,
  TournamentGameEndedEvent,
  TournamentRoundStarted,
  UserPresence,
  UserPresences,
} from '../gen/api/proto/realtime/realtime_pb';
import { ActionType } from '../actions/actions';
import { endGameMessage } from './end_of_game';
import {
  GameInfoResponseToActiveGame,
  SeekRequestToSoughtGame,
  SoughtGame,
} from './reducers/lobby_reducer';
import { BoopSounds } from '../sound/boop';
import {
  GameInfoResponse,
  GameInfoResponses,
} from '../gen/api/proto/game_service/game_service_pb';
import { TourneyStatus } from './reducers/tournament_reducer';

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
      [MessageType.ONGOING_GAME_EVENT]: GameInfoResponse,
      [MessageType.ONGOING_GAMES]: GameInfoResponses,
      [MessageType.GAME_DELETION]: GameDeletion,
      [MessageType.MATCH_REQUESTS]: MatchRequests,
      [MessageType.DECLINE_MATCH_REQUEST]: DeclineMatchRequest,
      [MessageType.CHAT_MESSAGE]: ChatMessage,
      [MessageType.USER_PRESENCE]: UserPresence,
      [MessageType.USER_PRESENCES]: UserPresences,
      [MessageType.READY_FOR_GAME]: ReadyForGame,
      [MessageType.READY_FOR_TOURNAMENT_GAME]: ReadyForTournamentGame,
      [MessageType.TOURNAMENT_ROUND_STARTED]: TournamentRoundStarted,
      [MessageType.LAG_MEASUREMENT]: LagMeasurement,
      [MessageType.MATCH_REQUEST_CANCELLATION]: MatchRequestCancellation,
      [MessageType.TOURNAMENT_GAME_ENDED_EVENT]: TournamentGameEndedEvent,
      [MessageType.REMATCH_STARTED]: RematchStartedEvent,
      [MessageType.GAME_META_EVENT]: GameMetaEvent,
      [MessageType.TOURNAMENT_MESSAGE]: TournamentDataResponse,
      [MessageType.TOURNAMENT_DIVISION_MESSAGE]: TournamentDivisionDataResponse,
      [MessageType.TOURNAMENT_DIVISION_DELETED_MESSAGE]: TournamentDivisionDeletedResponse,
      [MessageType.TOURNAMENT_FULL_DIVISIONS_MESSAGE]: FullTournamentDivisions,
      [MessageType.CHAT_MESSAGE_DELETED]: ChatMessageDeleted,
    };

    const parsedMsg = msgTypes[msgType];
    const topush = {
      msgType,
      parsedMsg: parsedMsg?.deserializeBinary(msgBytes),
      msgLength,
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
  const { addChat, deleteChat } = useChatStoreContext();
  const { excludedPlayers } = useExcludedPlayersStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { setGameEndMessage } = useGameEndMessageStoreContext();
  const { setCurrentLagMs } = useLagStoreContext();
  const { dispatchLobbyContext } = useLobbyStoreContext();
  const {
    tournamentContext,
    dispatchTournamentContext,
  } = useTournamentStoreContext();
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
        const { msgType, parsedMsg, msgLength } = msg;

        if (enableShowSocket) {
          console.log(
            '%crcvd',
            'background: pink',
            ReverseMessageType[msgType] ?? msgType,
            parsedMsg?.toObject(),
            performance.now(),
            'bytelength:',
            msgLength
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
              console.log(
                'sg',
                soughtGame.tournamentID,
                'gc',
                gameContext.gameID,
                'tc',
                tournamentContext
              );
              if (soughtGame.tournamentID) {
                // This is a match game attached to a tourney.
                console.log('match attached to tourney');
                if (
                  tournamentContext.metadata.id === soughtGame.tournamentID &&
                  !gameContext.gameID
                ) {
                  console.log('matches this tourney, and we are not in a game');

                  dispatchLobbyContext({
                    actionType: ActionType.AddMatchRequest,
                    payload: soughtGame,
                  });
                  inReceiverGameList = true;
                } else if (rematchFor && rematchFor === gameContext.gameID) {
                  console.log('it is a rematch');
                  setRematchRequest(mr);
                } else {
                  console.log('tourney match request elsewhere');
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
            const err = parsedMsg as ErrorMessage;
            notification.open({
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
            addChat({
              entityType: ChatEntityType.UserChat,
              sender: cm.getUsername(),
              message: cm.getMessage(),
              timestamp: cm.getTimestamp(),
              senderId: cm.getUserId(),
              channel: cm.getChannel(),
              id: cm.getId(),
            });
            if (cm.getUsername() !== loginState.username) {
              const tokenizedName = cm.getChannel().split('.');
              if (tokenizedName.length > 1 && tokenizedName[1] === 'pm') {
                BoopSounds.playSound('receiveMsgSound');
              }
            }
            break;
          }

          case MessageType.CHAT_MESSAGE_DELETED: {
            const cm = parsedMsg as ChatMessageDeleted;
            deleteChat(cm.getId(), cm.getChannel());

            break;
          }

          case MessageType.USER_PRESENCE: {
            const up = parsedMsg as UserPresence;
            if (excludedPlayers.has(up.getUserId())) {
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

          case MessageType.TOURNAMENT_ROUND_STARTED: {
            const trs = parsedMsg as TournamentRoundStarted;
            dispatchTournamentContext({
              actionType: ActionType.StartTourneyRound,
              payload: trs,
            });
            if (
              tournamentContext?.competitorState?.division === trs.getDivision()
            ) {
              BoopSounds.playSound('startTourneyRoundSound');
            }
            break;
          }

          case MessageType.TOURNAMENT_GAME_ENDED_EVENT: {
            const gee = parsedMsg as TournamentGameEndedEvent;
            dispatchTournamentContext({
              actionType: ActionType.AddTourneyGameResult,
              payload: gee,
            });

            dispatchTournamentContext({
              actionType: ActionType.RemoveActiveGame,
              payload: gee.getGameId(),
            });

            break;
          }

          case MessageType.TOURNAMENT_DIVISION_MESSAGE: {
            const tdm = parsedMsg as TournamentDivisionDataResponse;

            dispatchTournamentContext({
              actionType: ActionType.SetDivisionData,
              payload: {
                divisionMessage: tdm,
                loginState,
              },
            });

            break;
          }

          case MessageType.NEW_GAME_EVENT: {
            const nge = parsedMsg as NewGameEvent;

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
            dispatchGameContext({
              actionType: ActionType.RefreshHistory,
              payload: ghr,
            });

            // If there is an Antd message about "waiting for game", destroy it.
            message.destroy('server-message');
            break;
          }

          case MessageType.SERVER_GAMEPLAY_EVENT: {
            const sge = parsedMsg as ServerGameplayEvent;
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
            challengeResultEvent(sge);
            if (!sge.getValid()) {
              BoopSounds.playSound('woofSound');
            }
            break;
          }

          case MessageType.SOUGHT_GAME_PROCESS_EVENT: {
            const gae = parsedMsg as SoughtGameProcessEvent;
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

          case MessageType.GAME_META_EVENT: {
            const gme = parsedMsg as GameMetaEvent;

            dispatchGameContext({
              actionType: ActionType.ProcessGameMetaEvent,
              payload: {
                gme,
                us: loginState.userID,
              },
            });
            break;
          }

          case MessageType.GAME_DELETION: {
            // lobby context, remove active game
            const gde = parsedMsg as GameDeletion;
            dispatchLobbyContext({
              actionType: ActionType.RemoveActiveGame,
              payload: gde.getId(),
            });
            break;
          }

          case MessageType.ONGOING_GAME_EVENT: {
            // lobby context, add active game
            const gme = parsedMsg as GameInfoResponse;
            const activeGame = GameInfoResponseToActiveGame(gme);
            if (!activeGame) {
              return;
            }
            const dispatchFn = tournamentContext.metadata.id
              ? dispatchTournamentContext
              : dispatchLobbyContext;
            dispatchFn({
              actionType: ActionType.AddActiveGame,
              payload: activeGame,
            });
            break;
          }

          case MessageType.ONGOING_GAMES: {
            const age = parsedMsg as GameInfoResponses;
            const dispatchFn = tournamentContext.metadata.id
              ? dispatchTournamentContext
              : dispatchLobbyContext;

            dispatchFn({
              actionType: ActionType.AddActiveGames,
              payload: age
                .getGameInfoList()
                .map((g) => GameInfoResponseToActiveGame(g)),
            });
            break;
          }

          case MessageType.READY_FOR_TOURNAMENT_GAME: {
            const ready = parsedMsg as ReadyForTournamentGame;
            if (tournamentContext.metadata.id !== ready.getTournamentId()) {
              // Ignore this message (for now -- we may actually want to display
              // this in other contexts, like the lobby, an unrelated game, etc).
              break;
            }
            if (
              ready.getPlayerId() ===
              `${loginState.userID}:${loginState.username}`
            ) {
              dispatchTournamentContext({
                actionType: ActionType.SetTourneyStatus,
                payload: ready.getUnready()
                  ? TourneyStatus.ROUND_OPEN
                  : TourneyStatus.ROUND_READY,
              });
            } else {
              // The opponent sent this message.
              dispatchTournamentContext({
                actionType: ActionType.SetTourneyStatus,
                payload: ready.getUnready()
                  ? TourneyStatus.ROUND_OPEN
                  : TourneyStatus.ROUND_OPPONENT_WAITING,
              });
            }
            break;
          }

          case MessageType.TOURNAMENT_FULL_DIVISIONS_MESSAGE: {
            const tfdm = parsedMsg as FullTournamentDivisions;
            dispatchTournamentContext({
              actionType: ActionType.SetDivisionsData,
              payload: {
                fullDivisions: tfdm,
                loginState,
              },
            });

            break;
          }
        }
      });
    },
    [
      addChat,
      addPresences,
      challengeResultEvent,
      deleteChat,
      dispatchGameContext,
      dispatchLobbyContext,
      dispatchTournamentContext,
      excludedPlayers,
      gameContext,
      loginState,
      setCurrentLagMs,
      setGameEndMessage,
      setPresence,
      setRematchRequest,
      stopClock,
      tournamentContext,
      isExamining,
    ]
  );
};
