import { useCallback } from "react";
import { useNavigate } from "react-router";
import {
  useChallengeResultEventStoreContext,
  useChatStoreContext,
  useExamineStoreContext,
  useExcludedPlayersStoreContext,
  useGameContextStoreContext,
  useGameEndMessageStoreContext,
  useGameMetaEventContext,
  useFriendsStoreContext,
  useLagStoreContext,
  useLobbyStoreContext,
  useLoginStateStoreContext,
  usePresenceStoreContext,
  useRematchRequestStoreContext,
  useTimerStoreContext,
  useTournamentStoreContext,
} from "./store";

import { ActionType } from "../actions/actions";
import { endGameMessage } from "./end_of_game";
import {
  GameInfoResponseToActiveGame,
  SeekRequestToSoughtGame,
  SoughtGame,
} from "./reducers/lobby_reducer";
import { BoopSounds } from "../sound/boop";
import { metaStateFromMetaEvent } from "./meta_game_events";
import { parseWooglesError } from "../utils/parse_woogles_error";
import {
  LagMeasurement,
  LagMeasurementSchema,
  MessageType,
  ServerMessage,
  ServerMessageSchema,
} from "../gen/api/proto/ipc/ipc_pb";
import {
  DeclineSeekRequest,
  DeclineSeekRequestSchema,
  SeekRequest,
  SeekRequestSchema,
  SeekRequests,
  SeekRequestsSchema,
  SoughtGameProcessEvent,
  SoughtGameProcessEventSchema,
} from "../gen/api/proto/ipc/omgseeks_pb";
import {
  ErrorMessage,
  ErrorMessageSchema,
} from "../gen/api/proto/ipc/errors_pb";
import {
  ChatMessage,
  ChatMessageDeleted,
  ChatMessageDeletedSchema,
  ChatMessageSchema,
} from "../gen/api/proto/ipc/chat_pb";
import {
  NewGameEvent,
  GameHistoryRefresher,
  ServerGameplayEvent,
  GameEndedEvent,
  ServerChallengeResultEvent,
  GameDeletion,
  RematchStartedEvent,
  GameMetaEvent,
  GameInfoResponse,
  GameInfoResponses,
  NewGameEventSchema,
  GameHistoryRefresherSchema,
  ClientGameplayEventSchema,
  ServerGameplayEventSchema,
  GameEndedEventSchema,
  ServerChallengeResultEventSchema,
  TimedOutSchema,
  GameInfoResponseSchema,
  GameInfoResponsesSchema,
  GameDeletionSchema,
  ReadyForGameSchema,
  RematchStartedEventSchema,
  GameMetaEventSchema,
  ActiveGameEntrySchema,
  ServerOMGWordsEventSchema,
  GameDocumentEventSchema,
} from "../gen/api/proto/ipc/omgwords_pb";
import {
  UserPresence,
  UserPresences,
  PresenceEntry,
  UserPresenceSchema,
  UserPresencesSchema,
  PresenceEntrySchema,
} from "../gen/api/proto/ipc/presence_pb";
import {
  ReadyForTournamentGame,
  TournamentRoundStarted,
  TournamentGameEndedEvent,
  DivisionRoundControls,
  DivisionPairingsResponse,
  DivisionControlsResponse,
  PlayersAddedOrRemovedResponse,
  TournamentFinishedResponse,
  TournamentDivisionDeletedResponse,
  DivisionPairingsDeletedResponse,
  TournamentDataResponse,
  TournamentDivisionDataResponse,
  ReadyForTournamentGameSchema,
  TournamentRoundStartedSchema,
  TournamentGameEndedEventSchema,
  FullTournamentDivisionsSchema,
  DivisionRoundControlsSchema,
  DivisionPairingsResponseSchema,
  DivisionControlsResponseSchema,
  PlayersAddedOrRemovedResponseSchema,
  TournamentFinishedResponseSchema,
  TournamentDivisionDeletedResponseSchema,
  DivisionPairingsDeletedResponseSchema,
  TournamentDataResponseSchema,
  TournamentDivisionDataResponseSchema,
  PlayerCheckinResponseSchema,
  PlayerCheckinResponse,
} from "../gen/api/proto/ipc/tournament_pb";
import {
  ProfileUpdate,
  ProfileUpdateSchema,
} from "../gen/api/proto/ipc/users_pb";
import { ChatEntityType, PresenceEntity } from "./constants";
import { ServerOMGWordsEvent } from "../gen/api/proto/ipc/omgwords_pb";
import { GameDocumentEvent } from "../gen/api/proto/ipc/omgwords_pb";
import { App } from "antd";
import { fromBinary } from "@bufbuild/protobuf";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
// Feature flag.
export const enableShowSocket =
  localStorage?.getItem("enableShowSocket") === "true";

const MsgTypesMap = {
  [MessageType.SEEK_REQUEST]: SeekRequestSchema,
  [MessageType.ERROR_MESSAGE]: ErrorMessageSchema,
  [MessageType.SERVER_MESSAGE]: ServerMessageSchema,
  [MessageType.NEW_GAME_EVENT]: NewGameEventSchema,
  [MessageType.GAME_HISTORY_REFRESHER]: GameHistoryRefresherSchema,
  [MessageType.MATCH_REQUEST]: SeekRequestSchema,
  [MessageType.SOUGHT_GAME_PROCESS_EVENT]: SoughtGameProcessEventSchema,
  [MessageType.CLIENT_GAMEPLAY_EVENT]: ClientGameplayEventSchema,
  [MessageType.SERVER_GAMEPLAY_EVENT]: ServerGameplayEventSchema,
  [MessageType.GAME_ENDED_EVENT]: GameEndedEventSchema,
  [MessageType.SERVER_CHALLENGE_RESULT_EVENT]: ServerChallengeResultEventSchema,
  [MessageType.SEEK_REQUESTS]: SeekRequestsSchema,
  [MessageType.TIMED_OUT]: TimedOutSchema,
  [MessageType.ONGOING_GAME_EVENT]: GameInfoResponseSchema,
  [MessageType.ONGOING_GAMES]: GameInfoResponsesSchema,
  [MessageType.OUR_CORRESPONDENCE_GAMES]: GameInfoResponsesSchema,
  [MessageType.OUR_CORRESPONDENCE_SEEKS]: SeekRequestsSchema,
  [MessageType.GAME_DELETION]: GameDeletionSchema,
  [MessageType.MATCH_REQUESTS]: SeekRequestsSchema,
  [MessageType.DECLINE_SEEK_REQUEST]: DeclineSeekRequestSchema,
  [MessageType.CHAT_MESSAGE]: ChatMessageSchema,
  [MessageType.USER_PRESENCE]: UserPresenceSchema,
  [MessageType.USER_PRESENCES]: UserPresencesSchema,
  [MessageType.READY_FOR_GAME]: ReadyForGameSchema,
  [MessageType.READY_FOR_TOURNAMENT_GAME]: ReadyForTournamentGameSchema,
  [MessageType.TOURNAMENT_ROUND_STARTED]: TournamentRoundStartedSchema,
  [MessageType.LAG_MEASUREMENT]: LagMeasurementSchema,
  [MessageType.TOURNAMENT_GAME_ENDED_EVENT]: TournamentGameEndedEventSchema,
  [MessageType.REMATCH_STARTED]: RematchStartedEventSchema,
  [MessageType.GAME_META_EVENT]: GameMetaEventSchema,
  [MessageType.TOURNAMENT_FULL_DIVISIONS_MESSAGE]:
    FullTournamentDivisionsSchema,
  [MessageType.TOURNAMENT_DIVISION_ROUND_CONTROLS_MESSAGE]:
    DivisionRoundControlsSchema,
  [MessageType.TOURNAMENT_DIVISION_PAIRINGS_MESSAGE]:
    DivisionPairingsResponseSchema,
  [MessageType.TOURNAMENT_DIVISION_CONTROLS_MESSAGE]:
    DivisionControlsResponseSchema,
  [MessageType.TOURNAMENT_DIVISION_PLAYER_CHANGE_MESSAGE]:
    PlayersAddedOrRemovedResponseSchema,
  [MessageType.TOURNAMENT_FINISHED_MESSAGE]: TournamentFinishedResponseSchema,
  [MessageType.TOURNAMENT_DIVISION_DELETED_MESSAGE]:
    TournamentDivisionDeletedResponseSchema,
  [MessageType.TOURNAMENT_DIVISION_PAIRINGS_DELETED_MESSAGE]:
    DivisionPairingsDeletedResponseSchema,
  [MessageType.CHAT_MESSAGE_DELETED]: ChatMessageDeletedSchema,
  [MessageType.TOURNAMENT_MESSAGE]: TournamentDataResponseSchema,
  [MessageType.TOURNAMENT_DIVISION_MESSAGE]:
    TournamentDivisionDataResponseSchema,
  [MessageType.PRESENCE_ENTRY]: PresenceEntrySchema,
  [MessageType.ACTIVE_GAME_ENTRY]: ActiveGameEntrySchema,
  [MessageType.PROFILE_UPDATE_EVENT]: ProfileUpdateSchema,
  [MessageType.OMGWORDS_GAMEPLAY_EVENT]: ServerOMGWordsEventSchema,
  [MessageType.OMGWORDS_GAMEDOCUMENT]: GameDocumentEventSchema,
  [MessageType.TOURNAMENT_PLAYER_CHECKIN]: PlayerCheckinResponseSchema,
};

export const parseMsgs = (
  msg: Uint8Array,
): Array<{
  msgType: MessageType;
  parsedMsg: ReturnType<typeof fromBinary>;
  msgLength: number;
}> => {
  // Multiple msgs can come in the same packet.

  const msgs = [];

  while (msg.length > 0) {
    // Extract message length from the first two bytes
    const msgLength = msg[0] * 256 + msg[1];

    // Extract message type from the third byte
    const msgType = msg[2] as MessageType;

    // Extract the payload for the message
    const msgBytes = msg.slice(3, 3 + (msgLength - 1));

    // Parse the message using the appropriate schema
    const schema = MsgTypesMap[msgType];
    if (!schema) {
      throw new Error(`No schema found for message type: ${msgType} (${msg})`);
    }

    const parsedMsg = fromBinary(schema, msgBytes);

    // Add the parsed message to the result array
    msgs.push({
      msgType,
      parsedMsg,
      msgLength,
    });

    // Remove the processed message from the input buffer
    msg = msg.slice(3 + (msgLength - 1));
  }
  return msgs;
};

export const ReverseMessageType = (() => {
  const ret = [];
  for (const k in MessageType) {
    ret[(MessageType as unknown as { [key: string]: number })[k]] = k;
  }
  return ret;
})();

export const useOnSocketMsg = () => {
  const { challengeResultEvent } = useChallengeResultEventStoreContext();
  const { addChat, deleteChat } = useChatStoreContext();
  const { excludedPlayers } = useExcludedPlayersStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { gameMetaEventContext, setGameMetaEventContext } =
    useGameMetaEventContext();
  const { setGameEndMessage } = useGameEndMessageStoreContext();
  const { setCurrentLagMs } = useLagStoreContext();
  const { dispatchLobbyContext } = useLobbyStoreContext();
  const { tournamentContext, dispatchTournamentContext } =
    useTournamentStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { setPresence, addPresences } = usePresenceStoreContext();
  const { friends, setFriends } = useFriendsStoreContext();
  const { setRematchRequest } = useRematchRequestStoreContext();
  const { stopClock } = useTimerStoreContext();
  const { isExamining } = useExamineStoreContext();

  const navigate = useNavigate();
  const { message, notification } = App.useApp();
  const tourneyCompetitorState = useTournamentCompetitorState();

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
            "%crcvd",
            "background: pink",
            ReverseMessageType[msgType] ?? msgType,
            parsedMsg,
            performance.now(),
            "bytelength:",
            msgLength,
          );
        }

        switch (msgType) {
          case MessageType.SEEK_REQUEST: {
            const sr = parsedMsg as SeekRequest;

            if (!sr.receiverIsPermanent) {
              console.log("Got a seek request", sr);

              const userID = sr.user?.userId;
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
            } else {
              const userID = sr.user?.userId;
              if (!userID || excludedPlayers.has(userID)) {
                break;
              }

              const receiver = sr.receivingUser?.displayName;
              const soughtGame = SeekRequestToSoughtGame(sr);
              if (soughtGame === null) {
                break;
              }
              console.log("gameContext", gameContext);
              let inReceiverGameList = false;
              if (receiver === loginState.username) {
                BoopSounds.playSound("matchReqSound");
                const rematchFor = sr.rematchFor;
                console.log(
                  "sg",
                  soughtGame.tournamentID,
                  "gc",
                  gameContext.gameID,
                  "tc",
                  tournamentContext,
                );
                if (soughtGame.tournamentID) {
                  // This is a match game attached to a tourney.
                  console.log("match attached to tourney");
                  if (
                    tournamentContext.metadata?.id ===
                      soughtGame.tournamentID &&
                    !gameContext.gameID
                  ) {
                    console.log(
                      "matches this tourney, and we are not in a game",
                    );

                    dispatchLobbyContext({
                      actionType: ActionType.AddSoughtGame,
                      payload: soughtGame,
                    });
                    inReceiverGameList = true;
                  } else if (rematchFor && rematchFor === gameContext.gameID) {
                    console.log("it is a rematch");
                    setRematchRequest(sr);
                  } else {
                    console.log("tourney match request elsewhere");
                    notification.info({
                      message: "Tournament Match Request",
                      description: `You have a tournament match request from ${soughtGame.seeker}. Please return to your tournament at your convenience.`,
                    });
                  }
                } else if (gameContext.gameID) {
                  if (rematchFor === gameContext.gameID) {
                    // Only display the rematch modal if we are the recipient
                    // of the rematch request.
                    setRematchRequest(sr);
                  } else {
                    notification.info({
                      message: "Match Request",
                      description: `You have a match request from ${soughtGame.seeker}, in the lobby.`,
                    });
                    inReceiverGameList = true;
                  }
                } // else, we're in the lobby. Handle it below.
              }

              if (!inReceiverGameList) {
                dispatchLobbyContext({
                  actionType: ActionType.AddSoughtGame,
                  payload: soughtGame,
                });
              }
              break;
            }
          }

          case MessageType.SEEK_REQUESTS: {
            const sr = parsedMsg as SeekRequests;

            const soughtGames = new Array<SoughtGame>();

            sr.requests.forEach((r) => {
              const userID = r.user?.userId;
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

          case MessageType.SERVER_MESSAGE: {
            const sm = parsedMsg as ServerMessage;
            message.warning({
              content: sm.message,
              duration: 3,
              key: "server-message",
            });
            break;
          }

          case MessageType.REMATCH_STARTED: {
            const rs = parsedMsg as RematchStartedEvent;
            const gid = rs.rematchGameId;
            const url = `/game/${encodeURIComponent(gid)}`;
            if (isExamining) {
              notification.info({
                message: "A rematch has started",
                description: "Click this notification to watch",
                key: "rematch-notification",
                duration: 10, // 10 seconds,
                onClick: () => {
                  navigate(url);
                  notification.destroy("rematch-notification");
                },
              });
            } else {
              navigate(url, { replace: true });
              setGameEndMessage("");
            }
            break;
          }

          case MessageType.LAG_MEASUREMENT: {
            const lag = parsedMsg as LagMeasurement;
            setCurrentLagMs(lag.lagMs);
            break;
          }

          case MessageType.ERROR_MESSAGE: {
            const err = parsedMsg as ErrorMessage;
            const errorMessage = parseWooglesError(err.message);
            notification.open({
              message: "Error",
              description: errorMessage,
            });
            addChat({
              entityType: ChatEntityType.ErrorMsg,
              sender: "Woogles",
              message: errorMessage,
              channel: "server",
            });
            break;
          }

          case MessageType.CHAT_MESSAGE: {
            const cm = parsedMsg as ChatMessage;
            if (excludedPlayers.has(cm.userId)) {
              break;
            }
            addChat({
              entityType: ChatEntityType.UserChat,
              sender: cm.username,
              message: cm.message,
              timestamp: cm.timestamp,
              senderId: cm.userId,
              channel: cm.channel,
              id: cm.id,
            });
            if (cm.username !== loginState.username) {
              const tokenizedName = cm.channel.split(".");
              if (tokenizedName.length > 1 && tokenizedName[1] === "pm") {
                BoopSounds.playSound("receiveMsgSound");
              }
            }
            break;
          }

          case MessageType.CHAT_MESSAGE_DELETED: {
            const cm = parsedMsg as ChatMessageDeleted;
            deleteChat(cm.id, cm.channel);

            break;
          }

          case MessageType.USER_PRESENCE: {
            // Note: UserPresence is for chats. Not for follows.
            const up = parsedMsg as UserPresence;
            if (excludedPlayers.has(up.userId)) {
              break;
            }

            setPresence({
              uuid: up.userId,
              username: up.username,
              channel: up.channel,
              anon: up.isAnonymous,
              deleting: up.deleting,
            });

            // Show chat messages for game channel presence changes (opponent only)
            if (
              up.channel.startsWith("chat.game.") &&
              up.userId !== loginState.userID
            ) {
              if (up.deleting) {
                addChat({
                  entityType: ChatEntityType.ErrorMsg,
                  sender: "",
                  message: "Opponent is no longer in this room.",
                  channel: "server",
                });
              } else {
                addChat({
                  entityType: ChatEntityType.ServerMsg,
                  sender: "",
                  message: "Opponent has returned to this room.",
                  channel: "server",
                });
              }
            }
            break;
          }

          case MessageType.USER_PRESENCES: {
            // Note: UserPresences is for chats. Not for follows.
            const ups = parsedMsg as UserPresences;

            const toAdd = new Array<PresenceEntity>();

            ups.presences.forEach((p) => {
              if (!excludedPlayers.has(p.userId)) {
                toAdd.push({
                  uuid: p.userId,
                  username: p.username,
                  channel: p.channel,
                  anon: p.isAnonymous,
                  deleting: p.deleting,
                });
              }
            });

            addPresences(toAdd);
            break;
          }

          case MessageType.PRESENCE_ENTRY: {
            // Note: PresenceEntry is for follows. Not for chats.
            const pe = parsedMsg as PresenceEntry;

            setFriends({
              ...friends,
              [pe.userId]: {
                uuid: pe.userId,
                username: pe.username,
                channel: pe.channel,
              },
            });

            // Interpretation notes (check Go code if these are still valid):
            // - The channel list is always sorted and unique.
            // - [] may also mean we have just unfollowed.
            // - No initial state is sent on connect.
            // - When doing GetFollows after socket (re)connects, assume the
            //   response arrived before the socket reconnected (any
            //   PresenceEntry received after the socket reconnects overrides
            //   the GetFollows response). So if we received a PresenceEntry
            //   with [] before GetFollows returns some channels, the [] wins.
            // - Latest sent entry wins, generally.
            // - Duplicate PresenceEntry may be sent. Ignore them.
            // - Multiple PresenceEntry may be sent quickly. It is recommended
            //   not to immediately display a toast notification on receipt.
            // - It may be good to have a function that takes a set of channels
            //   and picks at most one primary channel.

            break;
          }

          case MessageType.ACTIVE_GAME_ENTRY: {
            // Note: This is actually never sent out to the frontend.
            // const age = parsedMsg as ActiveGameEntry;

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

            BoopSounds.playSound("endgameSound");
            break;
          }

          case MessageType.TOURNAMENT_FULL_DIVISIONS_MESSAGE: {
            // socket doesn't send this anymore (At the moment)
            // we may make it start doing so again.
            break;
          }

          case MessageType.TOURNAMENT_ROUND_STARTED: {
            const trs = parsedMsg as TournamentRoundStarted;
            dispatchTournamentContext({
              actionType: ActionType.StartTourneyRound,
              payload: {
                trs,
                loginState,
              },
            });
            if (tourneyCompetitorState.division === trs.division) {
              BoopSounds.playSound("startTourneyRoundSound");
            }
            break;
          }

          case MessageType.TOURNAMENT_GAME_ENDED_EVENT: {
            // Clubhouse mode tournament game ended event.
            const gee = parsedMsg as TournamentGameEndedEvent;
            dispatchTournamentContext({
              actionType: ActionType.AddTourneyGameResult,
              payload: gee,
            });

            dispatchTournamentContext({
              actionType: ActionType.RemoveActiveGame,
              payload: gee.gameId,
            });

            break;
          }

          case MessageType.TOURNAMENT_DIVISION_ROUND_CONTROLS_MESSAGE: {
            const tdrcm = parsedMsg as DivisionRoundControls;

            dispatchTournamentContext({
              actionType: ActionType.SetDivisionRoundControls,
              payload: {
                roundControls: tdrcm,
                loginState,
              },
            });

            break;
          }

          case MessageType.TOURNAMENT_DIVISION_PAIRINGS_MESSAGE: {
            const tdpm = parsedMsg as DivisionPairingsResponse;

            dispatchTournamentContext({
              actionType: ActionType.SetDivisionPairings,
              payload: {
                dpr: tdpm,
                loginState,
              },
            });

            break;
          }

          case MessageType.TOURNAMENT_DIVISION_PAIRINGS_DELETED_MESSAGE: {
            const tdpm = parsedMsg as DivisionPairingsDeletedResponse;

            dispatchTournamentContext({
              actionType: ActionType.DeleteDivisionPairings,
              payload: {
                dpdr: tdpm,
                loginState,
              },
            });

            break;
          }

          case MessageType.TOURNAMENT_DIVISION_CONTROLS_MESSAGE: {
            const tdcm = parsedMsg as DivisionControlsResponse;

            dispatchTournamentContext({
              actionType: ActionType.SetDivisionControls,
              payload: {
                divisionControlsResponse: tdcm,
                loginState,
              },
            });

            break;
          }

          case MessageType.TOURNAMENT_DIVISION_PLAYER_CHANGE_MESSAGE: {
            const parr = parsedMsg as PlayersAddedOrRemovedResponse;

            dispatchTournamentContext({
              actionType: ActionType.SetDivisionPlayers,
              payload: {
                parr,
                loginState,
              },
            });

            break;
          }

          case MessageType.TOURNAMENT_FINISHED_MESSAGE: {
            const tfm = parsedMsg as TournamentFinishedResponse;

            dispatchTournamentContext({
              actionType: ActionType.SetTournamentFinished,
              payload: {
                divisionMessage: tfm,
                loginState,
              },
            });

            break;
          }

          case MessageType.NEW_GAME_EVENT: {
            const nge = parsedMsg as NewGameEvent;

            // Determine if this is the tab that should accept the game.
            if (
              nge.accepterCid !== loginState.connID &&
              nge.requesterCid !== loginState.connID
            ) {
              console.log(
                "ignoring on this tab...",
                nge.accepterCid,
                "-",
                nge.requesterCid,
                "-",
                loginState.connID,
              );
              break;
            }

            dispatchGameContext({
              actionType: ActionType.ClearHistory,
              payload: "",
            });
            const gid = nge.gameId;
            navigate(`/game/${encodeURIComponent(gid)}`, { replace: true });
            setGameEndMessage("");
            break;
          }

          case MessageType.GAME_HISTORY_REFRESHER: {
            const ghr = parsedMsg as GameHistoryRefresher;
            dispatchGameContext({
              actionType: ActionType.RefreshHistory,
              payload: ghr,
            });

            // If the history refresher contains a meta event,
            // set it properly.
            const gme = ghr.outstandingEvent;
            if (gme) {
              setGameMetaEventContext(
                metaStateFromMetaEvent(
                  message,
                  gameMetaEventContext,
                  gme,
                  loginState.userID,
                ),
              );
            }

            // If there is an Antd message about "waiting for game", destroy it.
            message.destroy("server-message");
            break;
          }

          case MessageType.OMGWORDS_GAMEDOCUMENT: {
            const d = parsedMsg as GameDocumentEvent;
            dispatchGameContext({
              actionType: ActionType.InitFromDocument,
              payload: d.doc,
            });
            break;
          }

          case MessageType.SERVER_GAMEPLAY_EVENT: {
            const sge = parsedMsg as ServerGameplayEvent;
            dispatchGameContext({
              actionType: ActionType.AddGameEvent,
              payload: sge,
            });
            // play sound
            // (moved to ../gameroom/table.tsx to avoid challenge sounds)
            break;
          }

          case MessageType.OMGWORDS_GAMEPLAY_EVENT: {
            const sge = parsedMsg as ServerOMGWordsEvent;
            dispatchGameContext({
              actionType: ActionType.AddOMGWordsEvent,
              payload: sge,
            });
            break;
          }

          case MessageType.SERVER_CHALLENGE_RESULT_EVENT: {
            const sge = parsedMsg as ServerChallengeResultEvent;
            challengeResultEvent(sge);
            if (!sge.valid) {
              BoopSounds.playSound("woofSound");
            } else {
              BoopSounds.playSound("meowSound");
            }
            break;
          }

          case MessageType.SOUGHT_GAME_PROCESS_EVENT: {
            const gae = parsedMsg as SoughtGameProcessEvent;
            dispatchLobbyContext({
              actionType: ActionType.RemoveSoughtGame,
              payload: gae.requestId,
            });

            break;
          }

          case MessageType.DECLINE_SEEK_REQUEST: {
            const dec = parsedMsg as DeclineSeekRequest;
            dispatchLobbyContext({
              actionType: ActionType.RemoveSoughtGame,
              payload: dec.requestId,
            });
            notification.info({
              message: "Declined",
              description: "Your match request was declined.",
            });
            break;
          }

          case MessageType.GAME_META_EVENT: {
            const gme = parsedMsg as GameMetaEvent;
            setGameMetaEventContext(
              metaStateFromMetaEvent(
                message,
                gameMetaEventContext,
                gme,
                loginState.userID,
              ),
            );
            BoopSounds.playSound("abortnudgeSound");

            break;
          }

          case MessageType.GAME_DELETION: {
            // lobby context, remove active game
            const gde = parsedMsg as GameDeletion;
            dispatchLobbyContext({
              actionType: ActionType.RemoveActiveGame,
              payload: gde.id,
            });
            if (!!tournamentContext.metadata?.id) {
              dispatchTournamentContext({
                actionType: ActionType.RemoveActiveGame,
                payload: gde.id,
              });
            }

            break;
          }

          case MessageType.ONGOING_GAME_EVENT: {
            // lobby context, add or update active/correspondence game
            const gme = parsedMsg as GameInfoResponse;
            const activeGame = GameInfoResponseToActiveGame(gme);
            if (!activeGame) {
              return;
            }

            // For correspondence games, use UpdateCorrespondenceGame to handle updates
            if (activeGame.gameMode === 1) {
              dispatchLobbyContext({
                actionType: ActionType.UpdateCorrespondenceGame,
                payload: {
                  correspondenceGame: activeGame,
                },
              });
            } else {
              // For regular games, use the existing AddActiveGame logic
              const dispatchFn = tournamentContext.metadata?.id
                ? dispatchTournamentContext
                : dispatchLobbyContext;
              dispatchFn({
                actionType: ActionType.AddActiveGame,
                payload: {
                  activeGame,
                  loginState,
                },
              });
            }
            break;
          }

          case MessageType.ONGOING_GAMES: {
            const age = parsedMsg as GameInfoResponses;
            console.log("got active games", age, "tc", tournamentContext);

            let inTourney = !!tournamentContext.metadata?.id;
            if (!inTourney) {
              const gil = age.gameInfo;
              if (
                gil.length &&
                gil[0].tournamentId &&
                gil.every((g) => g.tournamentId === gil[0].tournamentId)
              ) {
                console.log("in a tourney");
                inTourney = true;
              }
            }
            // XXX: This method of determining whether we're in a tourney
            // is a temporary one until we close
            // https://github.com/woogles-io/liwords/issues/614
            const dispatchFn = inTourney
              ? dispatchTournamentContext
              : dispatchLobbyContext;

            dispatchFn({
              actionType: ActionType.AddActiveGames,
              payload: {
                activeGames: age.gameInfo.map((g) =>
                  GameInfoResponseToActiveGame(g),
                ),
                loginState,
              },
            });
            break;
          }

          case MessageType.OUR_CORRESPONDENCE_GAMES: {
            const corresGames = parsedMsg as GameInfoResponses;
            console.log("got correspondence games", corresGames);

            dispatchLobbyContext({
              actionType: ActionType.AddCorrespondenceGames,
              payload: {
                correspondenceGames: corresGames.gameInfo.map((g) =>
                  GameInfoResponseToActiveGame(g),
                ),
              },
            });
            break;
          }

          case MessageType.OUR_CORRESPONDENCE_SEEKS: {
            const corresSeeks = parsedMsg as SeekRequests;
            console.log("got correspondence seeks", corresSeeks);

            const soughtGames = corresSeeks.requests
              .map((req) => SeekRequestToSoughtGame(req))
              .filter((sg) => sg !== null);

            dispatchLobbyContext({
              actionType: ActionType.SetCorrespondenceSeeks,
              payload: {
                correspondenceSeeks: soughtGames,
              },
            });
            break;
          }

          case MessageType.READY_FOR_TOURNAMENT_GAME: {
            const ready = parsedMsg as ReadyForTournamentGame;
            if (tournamentContext.metadata?.id !== ready.tournamentId) {
              // Ignore this message (for now -- we may actually want to display
              // this in other contexts, like the lobby, an unrelated game, etc).
              break;
            }
            dispatchTournamentContext({
              actionType: ActionType.SetReadyForGame,
              payload: {
                ready,
                loginState,
              },
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

          case MessageType.TOURNAMENT_DIVISION_DELETED_MESSAGE: {
            const tdd = parsedMsg as TournamentDivisionDeletedResponse;

            dispatchTournamentContext({
              actionType: ActionType.DeleteDivision,
              payload: tdd,
            });
            break;
          }

          case MessageType.TOURNAMENT_PLAYER_CHECKIN: {
            const tpc = parsedMsg as PlayerCheckinResponse;
            dispatchTournamentContext({
              actionType: ActionType.SetTourneyPlayerCheckin,
              payload: tpc,
            });
            break;
          }

          case MessageType.PROFILE_UPDATE_EVENT: {
            const pue = parsedMsg as ProfileUpdate;
            dispatchLobbyContext({
              actionType: ActionType.UpdateProfile,
              payload: pue,
            });
            break;
          }

          case MessageType.TOURNAMENT_MESSAGE: {
            const tm = parsedMsg as TournamentDataResponse;
            dispatchTournamentContext({
              actionType: ActionType.SetTourneyReducedMetadata,
              payload: tm,
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
      gameMetaEventContext,
      loginState,
      friends,
      navigate,
      setFriends,
      setCurrentLagMs,
      setGameEndMessage,
      setGameMetaEventContext,
      setPresence,
      setRematchRequest,
      stopClock,
      tournamentContext,
      isExamining,
      message,
      notification,
      tourneyCompetitorState.division,
    ],
  );
};
