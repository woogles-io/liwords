import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Card, message, Popconfirm } from "antd";
import { HomeOutlined, RightOutlined } from "@ant-design/icons";

import { Link, useSearchParams, useParams, useNavigate } from "react-router";
import { useFirefoxPatch } from "../utils/hooks/firefox";
import { useDefinitionAndPhonyChecker } from "../utils/hooks/definitions";
import { BoardPanel } from "./board_panel";
import { TopBar } from "../navigation/topbar";
import { Chat } from "../chat/chat";
import {
  useChatStoreContext,
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useGameEndMessageStoreContext,
  useLoginStateStoreContext,
  usePoolFormatStoreContext,
  useRematchRequestStoreContext,
  useTimerStoreContext,
  useTournamentStoreContext,
} from "../store/store";
import { PlayerCards } from "./player_cards";
import Pool from "./pool";
import { encodeToSocketFmt } from "../utils/protobuf";
import "./scss/gameroom.scss";
import { ScoreCard } from "./scorecard";
import { CommentsDrawer } from "./CommentsDrawer";
import { defaultGameInfo, GameInfo } from "./game_info";
import { useComments } from "../utils/hooks/comments";
import { GameCommentService } from "../gen/api/proto/comments_service/comments_service_pb";
import { Turn, gameEventsToTurns } from "../store/reducers/turns";
import { BoopSounds } from "../sound/boop";
import { StreakWidget } from "./streak_widget";
import { ChallengeRule, PlayState } from "../gen/api/vendor/macondo/macondo_pb";
import { endGameMessageFromGameInfo } from "../store/end_of_game";
import { Notepad, NotepadContextProvider } from "./notepad";
import { Analyzer, AnalyzerContextProvider } from "./analyzer";
import { isClubType, isPairedMode, sortTiles } from "../store/constants";
import { readyForTournamentGame } from "../tournament/ready";
import { isMobile } from "../utils/cwgame/common";
import { CompetitorStatus } from "../tournament/competitor_status";
import { MonitoringWidget } from "../tournament/monitoring/monitoring_widget";
import { MonitoringModal } from "../tournament/monitoring/monitoring_modal";
import { MetaEventControl } from "./meta_event_control";
import { useTourneyMetadata } from "../tournament/utils";
import { Disclaimer } from "./disclaimer";
import { alphabetFromName } from "../constants/alphabets";
import {
  ClientGameplayEventSchema,
  GameEndReason,
  GameInfoResponse,
  GameMode,
  GameType,
  ReadyForGameSchema,
  TimedOutSchema,
} from "../gen/api/proto/ipc/omgwords_pb";
import { MessageType } from "../gen/api/proto/ipc/ipc_pb";
import {
  DeclineSeekRequestSchema,
  SeekRequestSchema,
  SoughtGameProcessEventSchema,
} from "../gen/api/proto/ipc/omgseeks_pb";
import {
  StreakInfoResponse,
  StreakInfoResponseSchema,
} from "../gen/api/proto/game_service/game_service_pb";
import { flashError, useClient } from "../utils/hooks/connect";
import { GameMetadataService } from "../gen/api/proto/game_service/game_service_pb";
import { GameEventService } from "../gen/api/proto/omgwords_service/omgwords_pb";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { MonitoringData } from "../tournament/monitoring/types";
import { ActionType } from "../actions/actions";
import { syntheticGameInfo } from "../boardwizard/synthetic_game_info";
import { MachineLetter, MachineWord } from "../utils/cwgame/common";
import { create, toBinary } from "@bufbuild/protobuf";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { showTurnNotification } from "../utils/notifications";
import { timeCtrlToDisplayName } from "../store/constants";

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
  annotated?: boolean;
};

const StreakFetchDelay = 2000;

const DEFAULT_TITLE = "Woogles.io";

const ManageWindowTitleAndTurnSound = (props: {
  gameInfo: GameInfoResponse;
}) => {
  const { gameContext } = useGameContextStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { userID } = loginState;

  const userIDToNick = useMemo(() => {
    const ret: { [key: string]: string } = {};
    for (const userID in gameContext.uidToPlayerOrder) {
      const playerOrder = gameContext.uidToPlayerOrder[userID];
      for (const nick in gameContext.nickToPlayerOrder) {
        if (playerOrder === gameContext.nickToPlayerOrder[nick]) {
          ret[userID] = nick;
          break;
        }
      }
    }
    return ret;
  }, [gameContext.uidToPlayerOrder, gameContext.nickToPlayerOrder]);

  const playerNicks = useMemo(() => {
    return gameContext.players.map((player) => userIDToNick[player.userID]);
  }, [gameContext.players, userIDToNick]);

  const myId = useMemo(() => {
    const myPlayerOrder = gameContext.uidToPlayerOrder[userID];
    return myPlayerOrder === "p0" ? 0 : myPlayerOrder === "p1" ? 1 : null;
  }, [gameContext.uidToPlayerOrder, userID]);

  const gameDone =
    gameContext.playState === PlayState.GAME_OVER && !!gameContext.gameID;

  // do not play sound when game ends (e.g. resign) or has not loaded
  const canPlaySound = !gameDone && gameContext.gameID;
  const soundUnlocked = useRef(false);
  const notificationShown = useRef(false);

  useEffect(() => {
    if (canPlaySound) {
      if (!soundUnlocked.current) {
        // ignore first sound
        soundUnlocked.current = true;
        notificationShown.current = false;
        return;
      }

      if (myId === gameContext.onturn) {
        BoopSounds.playSound("oppMoveSound");

        // Show notification on your turn
        if (!notificationShown.current) {
          const opponentName =
            playerNicks.find((nick, idx) => idx !== myId) ?? "Opponent";

          // Get time control info
          const isCorrespondence =
            props.gameInfo.gameRequest?.gameMode === GameMode.CORRESPONDENCE;
          let timeCtrl: string;

          if (isCorrespondence) {
            // For correspondence games, show time per turn
            const timePerTurn =
              props.gameInfo.gameRequest?.initialTimeSeconds || 0;
            const days = Math.floor(timePerTurn / 86400);
            timeCtrl =
              days > 0
                ? `${days} day${days > 1 ? "s" : ""}/turn`
                : "Correspondence";
          } else {
            // For real-time games, use the standard format
            const [tc] = timeCtrlToDisplayName(
              props.gameInfo.gameRequest?.initialTimeSeconds || 0,
              props.gameInfo.gameRequest?.incrementSeconds || 0,
              props.gameInfo.gameRequest?.maxOvertimeMinutes || 0,
            );
            timeCtrl = tc;
          }

          showTurnNotification({
            opponentName,
            timeControl: timeCtrl,
            gameId: gameContext.gameID,
          });
          notificationShown.current = true;
        }
      } else {
        BoopSounds.playSound("makeMoveSound");
        // Reset notification flag when it's opponent's turn
        notificationShown.current = false;
      }
    } else {
      soundUnlocked.current = false;
      notificationShown.current = false;
    }
  }, [
    canPlaySound,
    myId,
    gameContext.onturn,
    playerNicks,
    gameContext.gameID,
    props.gameInfo.gameRequest?.gameMode,
    props.gameInfo.gameRequest?.initialTimeSeconds,
    props.gameInfo.gameRequest?.incrementSeconds,
    props.gameInfo.gameRequest?.maxOvertimeMinutes,
  ]);

  const desiredTitle = useMemo(() => {
    let title = "";
    if (!gameDone && myId === gameContext.onturn) {
      title += "*";
    }
    let first = true;
    for (let i = 0; i < gameContext.players.length; ++i) {
      if (gameContext.players[i].userID === userID) continue;
      if (first) {
        first = false;
      } else {
        title += " vs ";
      }
      title += playerNicks[i] ?? "?";
      if (!gameDone && myId == null && i === gameContext.onturn) {
        title += "*";
      }
    }
    if (title.length > 0) title += " - ";
    title += DEFAULT_TITLE;
    return title;
  }, [
    gameContext.onturn,
    gameContext.players,
    gameDone,
    myId,
    playerNicks,
    userID,
  ]);

  useEffect(() => {
    document.title = desiredTitle;
  }, [desiredTitle]);

  useEffect(() => {
    return () => {
      document.title = DEFAULT_TITLE;
    };
  }, []);

  return null;
};

const getChatTitle = (
  playerNames: Array<string> | undefined,
  username: string,
  isObserver: boolean,
): string => {
  if (!playerNames) {
    return "";
  }
  if (isObserver) {
    return playerNames.join(" versus ");
  }
  return playerNames.filter((n) => n !== username).shift() || "";
};

export const Table = React.memo((props: Props) => {
  const { gameID } = useParams();
  const navigate = useNavigate();
  const { addChat } = useChatStoreContext();

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { isExamining, handleExamineStart, handleExamineGoTo } =
    useExamineStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { gameEndMessage, setGameEndMessage } = useGameEndMessageStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();
  const { rematchRequest, setRematchRequest } = useRematchRequestStoreContext();
  const { pTimedOut, setPTimedOut } = useTimerStoreContext();
  const { username, userID, loggedIn } = loginState;
  const { tournamentContext, dispatchTournamentContext } =
    useTournamentStoreContext();
  const competitorState = useTournamentCompetitorState();
  const isRegistered = competitorState.isRegistered;
  const [playerNames, setPlayerNames] = useState(new Array<string>());
  const { sendSocketMsg } = props;
  const [gameInfo, setGameInfo] = useState<GameInfoResponse>(defaultGameInfo);
  const [streakGameInfo, setStreakGameInfo] = useState<StreakInfoResponse>(
    create(StreakInfoResponseSchema, {
      streak: [],
      playersInfo: [],
    }),
  );
  const [localCorresGames, setLocalCorresGames] = useState<
    Array<GameInfoResponse>
  >([]);
  const [isObserver, setIsObserver] = useState(false);
  const prevGameIDRef = useRef<string | undefined>(undefined);

  // Comments functionality
  const commentsClient = useClient(GameCommentService);
  const { comments, editComment, addNewComment, deleteComment } = useComments(
    commentsClient,
    props.annotated ?? false,
  );

  // Tournament client for monitoring
  const tClient = useClient(TournamentService);

  // Comments drawer state
  const [searchParams, setSearchParams] = useSearchParams();
  const [commentsDrawerVisible, setCommentsDrawerVisible] = useState(false);
  const [commentsDrawerEventNumber, setCommentsDrawerEventNumber] =
    useState<number>(0);

  // Monitoring modal state
  const monitoringModalVisible = searchParams.get("monitoring") === "true";

  // Mobile detection state
  const [isInMobileView, setIsInMobileView] = useState(isMobile());

  const closeMonitoringModal = useCallback(() => {
    const newParams = new URLSearchParams(searchParams);
    newParams.delete("monitoring");
    setSearchParams(newParams);
  }, [searchParams, setSearchParams]);

  const handleOpenCommentsDrawerForEvent = useCallback(
    (eventNumber: number) => {
      setCommentsDrawerEventNumber(eventNumber);
      setCommentsDrawerVisible(true);

      // Update URL with only comments parameter - don't activate analyzer from bubble clicks
      const newParams = new URLSearchParams(searchParams);
      const commentEventNumber = eventNumber + 1;
      newParams.set("comments", commentEventNumber.toString());
      setSearchParams(newParams);
    },
    [searchParams, setSearchParams],
  );

  const handleCloseCommentsDrawer = useCallback(() => {
    setCommentsDrawerVisible(false);

    // Update URL - remove comments parameter
    const newParams = new URLSearchParams(searchParams);
    newParams.delete("comments");
    setSearchParams(newParams);
  }, [searchParams, setSearchParams]);

  // Navigation removed - comments are now event-specific

  // Convert GameEvents to Turn objects
  const turns = useMemo(
    () => gameEventsToTurns(examinableGameContext.turns),
    [examinableGameContext.turns],
  );

  // Wrapper for scorecard to convert turnIndex to eventNumber
  const handleOpenCommentsDrawerForTurn = useCallback(
    (turnIndex: number) => {
      if (turnIndex >= 0 && turnIndex < turns.length) {
        const turn = turns[turnIndex];
        // Use the last event of the turn as the representative event (the actual move)
        const representativeEventNumber =
          turn.firstEvtIdx + turn.events.length - 1;
        handleOpenCommentsDrawerForEvent(representativeEventNumber);
      }
    },
    [turns, handleOpenCommentsDrawerForEvent],
  );

  // Handle URL parameters for comments drawer - run when turns are loaded
  useEffect(() => {
    if (!props.annotated || turns.length === 0) return;

    const commentsParam = searchParams.get("comments");

    if (commentsParam && commentsParam !== "true") {
      // Parse comments={eventNumber} (1-based)
      const eventNumber = parseInt(commentsParam) - 1; // Convert to 0-based event number

      // Validate that this event number exists in the turns
      let eventExists = false;
      for (const turn of turns) {
        if (
          eventNumber >= turn.firstEvtIdx &&
          eventNumber < turn.firstEvtIdx + turn.events.length
        ) {
          eventExists = true;
          break;
        }
      }

      if (eventExists) {
        setCommentsDrawerEventNumber(eventNumber);
        setCommentsDrawerVisible(true);
      }
    }
  }, [turns, props.annotated, searchParams]);

  // Comments are now filtered inside CommentsDrawer by eventNumber

  // Comment handlers for drawer
  const handleAddCommentInDrawer = useCallback(
    (comment: string) => {
      addNewComment(gameID || "", commentsDrawerEventNumber, comment);
    },
    [commentsDrawerEventNumber, addNewComment, gameID],
  );

  const handleEditCommentInDrawer = useCallback(
    (commentId: string, comment: string) => {
      editComment(commentId, comment);
    },
    [editComment],
  );

  const handleDeleteCommentInDrawer = useCallback(
    (commentId: string) => {
      deleteComment(commentId);
    },
    [deleteComment],
  );

  const tournamentNonDirectorObserver = useMemo(() => {
    // HACK: Check for both exact match and :readonly suffix
    // TODO: Replace with proper permissions field when backend schema is updated
    const isDirector = tournamentContext.directors?.some(
      (director) =>
        director === username || director === `${username}:readonly`,
    );
    return isObserver && !isDirector && !loginState.perms.includes("adm");
  }, [isObserver, loginState.perms, username, tournamentContext.directors]);
  useFirefoxPatch();
  const gmClient = useClient(GameMetadataService);
  const omgClient = useClient(GameEventService);
  const gameDone =
    gameContext.playState === PlayState.GAME_OVER && !!gameContext.gameID;

  useEffect(() => {
    const isCorrespondence =
      gameInfo.gameRequest?.gameMode === GameMode.CORRESPONDENCE;

    // Don't add beforeunload for correspondence games, finished games, or observers
    if (gameDone || isObserver || isCorrespondence) {
      return () => {};
    }

    const evtHandler = (evt: BeforeUnloadEvent) => {
      if (!gameDone && !isObserver && !isCorrespondence) {
        const msg = "You are currently in a game!";
        evt.returnValue = msg;
        return msg;
      }
      return true;
    };
    window.addEventListener("beforeunload", evtHandler);
    return () => {
      window.removeEventListener("beforeunload", evtHandler);
    };
  }, [gameDone, isObserver, gameInfo.gameRequest?.gameMode]);

  // Track window resize for mobile detection
  useEffect(() => {
    const handleResize = () => {
      setIsInMobileView(isMobile());
    };
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  useEffect(() => {
    // Request game API to get info about the game at the beginning.
    // Only reset state when gameID actually changes, not when other deps change
    const gameIDChanged = prevGameIDRef.current !== gameID;
    if (gameIDChanged) {
      setGameInfo(defaultGameInfo);
      setLocalCorresGames([]);
      message.destroy("board-messages");
      prevGameIDRef.current = gameID;
    }

    const fetchGameMetadata = async () => {
      try {
        const resp = await gmClient.getMetadata({ gameId: gameID });

        if (localStorage?.getItem("poolFormat")) {
          setPoolFormat(
            parseInt(localStorage.getItem("poolFormat") || "0", 10),
          );
        }

        if (resp.type === GameType.ANNOTATED) {
          // If this is an annotated game, leave early. We will use
          // a synthetic GameInfo constructed from the annotated game's
          // GameDocument.
          return;
        }
        setGameInfo(resp);

        if (resp.gameEndReason !== GameEndReason.NONE) {
          // Basically if we are here, we've reloaded the page after the game
          // ended. We want to synthesize a new GameEnd message
          setGameEndMessage(endGameMessageFromGameInfo(resp));
        }

        // If this is a correspondence game, fetch active correspondence games
        // to populate the next game button (single source of truth from API)
        if (resp.gameRequest?.gameMode === GameMode.CORRESPONDENCE) {
          try {
            const activeCorresGames =
              await gmClient.getActiveCorrespondenceGames({});

            setLocalCorresGames(activeCorresGames.gameInfo);
          } catch (e) {
            console.error("Failed to fetch active correspondence games:", e);
          }
        }
      } catch (e) {
        message.error({
          content: `Failed to fetch game information; please refresh. (Error: ${e})`,
          duration: 10,
        });
      }
    };

    fetchGameMetadata();

    return () => {
      // Cleanup messages
      message.destroy("board-messages");
    };
  }, [gameID, gmClient, setGameEndMessage, setPoolFormat]);

  useEffect(() => {
    // If we are in annotated mode, we must explicitly fetch the GameDocument
    // from the backend. This is a temporary thing that we will eventually
    // undo when we unite GameDocuments across the app.
    if (!props.annotated) {
      return;
    }
    const fetchGameDocument = async () => {
      console.log("fetching game document");

      try {
        const resp = await omgClient.getGameDocument({ gameId: gameID });
        dispatchGameContext({
          actionType: ActionType.InitFromDocument,
          payload: resp,
        });
      } catch (e) {
        message.error({
          content: `Failed to fetch initial game information; please refresh. (Error: ${e})`,
          duration: 10,
        });
      }
    };
    fetchGameDocument();
  }, [gameID, omgClient, dispatchGameContext, props.annotated]);

  useEffect(() => {
    if (gameContext.gameDocument.uid) {
      const gi = syntheticGameInfo(gameContext.gameDocument);
      setGameInfo(gi);
    }
  }, [gameContext.gameDocument]);

  useTourneyMetadata(
    "",
    gameInfo.tournamentId,
    dispatchTournamentContext,
    loginState,
    undefined,
  );

  // Fetch monitoring data on initial load if tournament requires monitoring
  useEffect(() => {
    if (
      !tournamentContext.metadata.id ||
      !loginState.loggedIn ||
      !tournamentContext.metadata.monitored
    ) {
      return;
    }

    const fetchMonitoringData = async () => {
      try {
        const response = await tClient.getTournamentMonitoring({
          tournamentId: tournamentContext.metadata.id,
        });

        // Convert to frontend format
        const data: MonitoringData[] = response.participants.map((p) => ({
          userId: p.userId,
          username: p.username,
          cameraKey: p.cameraKey,
          screenshotKey: p.screenshotKey,
          cameraStatus: p.cameraStatus,
          cameraTimestamp: p.cameraTimestamp
            ? new Date(
                Number(p.cameraTimestamp.seconds) * 1000 +
                  Number(p.cameraTimestamp.nanos) / 1000000,
              )
            : null,
          screenshotStatus: p.screenshotStatus,
          screenshotTimestamp: p.screenshotTimestamp
            ? new Date(
                Number(p.screenshotTimestamp.seconds) * 1000 +
                  Number(p.screenshotTimestamp.nanos) / 1000000,
              )
            : null,
        }));

        // Update tournament context with monitoring data
        dispatchTournamentContext({
          actionType: ActionType.SetMonitoringData,
          payload: data.reduce(
            (acc, d) => {
              acc[d.userId] = d;
              return acc;
            },
            {} as { [userId: string]: MonitoringData },
          ),
        });
      } catch (e) {
        flashError(e);
      }
    };

    fetchMonitoringData();
  }, [
    tournamentContext.metadata.id,
    tournamentContext.metadata.monitored,
    loginState.loggedIn,
    tClient,
    dispatchTournamentContext,
  ]);

  useEffect(() => {
    // Request streak info only if a few conditions are true.
    // We want to request it as soon as the original request ID comes in,
    // but only if this is an ongoing game. Also, we want to request it
    // as soon as the game ends (so the streak updates without having to go
    // to a new game).

    if (!gameInfo.gameRequest?.originalRequestId) {
      return;
    }
    if (gameDone && !gameEndMessage) {
      // if the game has long been over don't request this. Only request it
      // when we are going to play a game (or observe), or when the game just ended.
      return;
    }
    setTimeout(async () => {
      const resp = await gmClient.getRematchStreak({
        originalRequestId: gameInfo.gameRequest?.originalRequestId,
      });

      setStreakGameInfo(resp);

      // Put this on a delay. Otherwise the game might not be saved to the
      // db as having finished before the gameEndMessage comes in.
    }, StreakFetchDelay);

    // Call this when a gameEndMessage comes in, so the streak updates
    // at the end of the game.
  }, [
    gameInfo.gameRequest?.originalRequestId,
    gameEndMessage,
    gameDone,
    gmClient,
  ]);

  useEffect(() => {
    if (pTimedOut === undefined) return;
    // Otherwise, player timed out. This will only send once.
    // Send the time out if we're either of both players that are in the game.
    if (isObserver) return;
    if (!gameID) return;

    let timedout = "";

    gameInfo.players.forEach((p) => {
      if (gameContext.uidToPlayerOrder[p.userId] === pTimedOut) {
        timedout = p.userId;
      }
    });

    const to = create(TimedOutSchema);
    to.gameId = gameID;
    to.userId = timedout;
    sendSocketMsg(
      encodeToSocketFmt(MessageType.TIMED_OUT, toBinary(TimedOutSchema, to)),
    );
    setPTimedOut(undefined);
  }, [
    gameContext.nickToPlayerOrder,
    gameContext.uidToPlayerOrder,
    gameID,
    gameInfo.players,
    isObserver,
    pTimedOut,
    sendSocketMsg,
    setPTimedOut,
  ]);

  useEffect(() => {
    if (!gameID) return;
    let observer = true;
    gameInfo.players.forEach((p) => {
      if (userID === p.userId) {
        observer = false;
      }
    });
    setIsObserver(observer);
    setPlayerNames(gameInfo.players.map((p) => p.nickname));
    // If we are not the observer, tell the server we're ready for the game to start.
    if (gameInfo.gameEndReason === GameEndReason.NONE && !observer) {
      const evt = create(ReadyForGameSchema);
      evt.gameId = gameID;
      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.READY_FOR_GAME,
          toBinary(ReadyForGameSchema, evt),
        ),
      );
    }
  }, [userID, gameInfo, gameID, sendSocketMsg]);

  const enableHoverDefine = gameDone || isObserver;
  const { handleSetHover, hideDefinitionHover, definitionPopover } =
    useDefinitionAndPhonyChecker({
      addChat,
      enableHoverDefine,
      gameContext,
      gameDone,
      gameID,
      lexicon: gameInfo.gameRequest?.lexicon ?? "",
      variant: gameInfo.gameRequest?.rules?.variantName,
    });

  const acceptRematch = useCallback(
    (reqID: string) => {
      const evt = create(SoughtGameProcessEventSchema);
      evt.requestId = reqID;
      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.SOUGHT_GAME_PROCESS_EVENT,
          toBinary(SoughtGameProcessEventSchema, evt),
        ),
      );
    },
    [sendSocketMsg],
  );

  const handleAcceptRematch = useCallback(() => {
    const gr = rematchRequest.gameRequest;
    if (gr) {
      acceptRematch(gr.requestId);
      setRematchRequest(create(SeekRequestSchema, {}));
    }
  }, [acceptRematch, rematchRequest, setRematchRequest]);

  const declineRematch = useCallback(
    (reqID: string) => {
      const evt = create(DeclineSeekRequestSchema, { requestId: reqID });
      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.DECLINE_SEEK_REQUEST,
          toBinary(DeclineSeekRequestSchema, evt),
        ),
      );
    },
    [sendSocketMsg],
  );

  const handleDeclineRematch = useCallback(() => {
    const gr = rematchRequest.gameRequest;
    if (gr) {
      declineRematch(gr.requestId);
      setRematchRequest(create(SeekRequestSchema, {}));
    }
  }, [declineRematch, rematchRequest, setRematchRequest]);

  // Figure out what rack we should display.
  // If we are one of the players, display our rack.
  // If we are NOT one of the players (so an observer), display the rack of
  // the player on turn.
  let rack: MachineWord;
  const us = useMemo(
    () => gameInfo.players.find((p) => p.userId === userID),
    [gameInfo.players, userID],
  );
  if (us && !(gameDone && isExamining)) {
    rack =
      examinableGameContext.players.find((p) => p.userID === us.userId)
        ?.currentRack ?? new Array<MachineLetter>();
  } else {
    rack =
      examinableGameContext.players.find((p) => p.onturn)?.currentRack ??
      new Array<MachineLetter>();
  }
  const sortedRack = useMemo(
    () => sortTiles(rack, gameContext.alphabet),
    [rack, gameContext.alphabet],
  );

  // The game "starts" when the GameHistoryRefresher object comes in via the socket.
  // At that point gameID will be filled in.

  useEffect(() => {
    // Only play the start sound when:
    // 1. The game is not done
    // 2. We are a player (not an observer)
    // 3. The game is just starting (no turns yet)
    // 4. The game hasn't ended already
    // 5. This is not an annotated game
    // 6. The game context has been initialized (gameContext.gameID matches the route gameID)
    if (
      !gameDone &&
      !isObserver &&
      !props.annotated &&
      gameContext.turns.length === 0 &&
      gameInfo.gameEndReason === GameEndReason.NONE &&
      gameContext.gameID === gameID
    ) {
      BoopSounds.playSound("startgameSound");
    }
  }, [
    gameID,
    gameDone,
    isObserver,
    gameContext.turns.length,
    gameInfo.gameEndReason,
    props.annotated,
    gameContext.gameID,
  ]);

  const searchedTurn = useMemo(() => searchParams.get("turn"), [searchParams]);
  const turnAsStr = us && !gameDone ? "" : (searchedTurn ?? ""); // Do not examine our current games.
  const hasActivatedExamineRef = useRef(false);
  const [autocorrectURL, setAutocorrectURL] = useState(false);
  useEffect(() => {
    if (gameContext.gameID) {
      if (!hasActivatedExamineRef.current) {
        hasActivatedExamineRef.current = true;
        const turnAsInt = parseInt(turnAsStr, 10);
        if (isFinite(turnAsInt) && turnAsStr === String(turnAsInt)) {
          handleExamineStart();
          handleExamineGoTo(turnAsInt - 1); // ?turn= should start from one.

          // Autoscroll removed - comments now use drawer
        }
        setAutocorrectURL(true); // Trigger rerender.
      }
    }
  }, [
    gameContext.gameID,
    turnAsStr,
    handleExamineStart,
    handleExamineGoTo,
    props.annotated,
  ]);

  // Autocorrect the turn on the URL.
  // Do not autocorrect when NEW_GAME_EVENT redirects to a rematch.
  const canAutocorrectURL = autocorrectURL && gameID === gameContext.gameID;
  useEffect(() => {
    if (!canAutocorrectURL) return; // Too early if examining has not started.
    const turnParamShouldBe = isExamining
      ? String(examinableGameContext.turns.length + 1)
      : null;
    if (turnParamShouldBe !== searchedTurn) {
      if (turnParamShouldBe == null) {
        // Remove turn parameter while preserving other parameters
        const newParams = new URLSearchParams(searchParams);
        newParams.delete("turn");
        setSearchParams(newParams, { replace: true });
      } else {
        // Update turn parameter while preserving other parameters
        const newParams = new URLSearchParams(searchParams);
        newParams.set("turn", turnParamShouldBe);
        setSearchParams(newParams, { replace: true });
      }
    }
  }, [
    canAutocorrectURL,
    examinableGameContext.turns.length,
    isExamining,
    searchParams,
    searchedTurn,
    setSearchParams,
  ]);
  const boardTheme = "board--" + tournamentContext.metadata.boardStyle || "";
  const tileTheme = "tile--" + tournamentContext.metadata.tileStyle || "";
  const alphabet = useMemo(
    () => alphabetFromName(gameInfo.gameRequest?.rules?.letterDistributionName),
    [gameInfo],
  );
  const showingFinalTurn =
    gameContext.turns.length === examinableGameContext.turns.length;

  const feRackInfo = useMemo(() => {
    // Enable rack info to be available to all widgets all the time,
    // except in some private situations.
    if (gameDone) {
      // If the game is done, it's fine to always allow rack info
      return true;
    }
    // If we are not a director, but are observing, and private analysis is off:
    // if (
    //   tournamentNonDirectorObserver &&
    //   tournamentContext.metadata?.getPrivateAnalysis()
    // ) {
    //   return false;
    // }
    // If we are an anonymous observer, and this is a tournament, don't
    // allow rack info.
    if (!loggedIn && gameInfo.tournamentId) {
      return false;
    }
    return true;
  }, [gameDone, gameInfo.tournamentId, loggedIn]);

  // Calculate next correspondence game where it's user's turn
  // Uses localCorresGames as single source of truth (fetched via API)
  const { nextCorresGame, corresGamesWaiting } = useMemo(() => {
    const isCorrespondence =
      gameInfo.gameRequest?.gameMode === GameMode.CORRESPONDENCE;

    if (!isCorrespondence || !userID) {
      return { nextCorresGame: null, corresGamesWaiting: 0 };
    }

    // Convert localCorresGames to ActiveGame format
    const corresGames = localCorresGames.map((g) => ({
      gameID: g.gameId,
      players: g.players.map((p) => ({
        uuid: p.userId,
        nickname: p.nickname,
      })),
      playerOnTurn: g.playerOnTurn,
      lastUpdate: g.lastUpdate
        ? Number(g.lastUpdate.seconds) * 1000
        : Date.now(),
      incrementSecs: g.gameRequest?.incrementSeconds || 86400,
      lexicon: g.gameRequest?.lexicon || "",
      variant: g.gameRequest?.rules?.variantName || "",
      initialTimeSecs: g.gameRequest?.initialTimeSeconds || 0,
      challengeRule: g.gameRequest?.challengeRule || 0,
      rated: g.gameRequest?.ratingMode === 0, // RatingMode.RATED = 0
      maxOvertimeMinutes: g.gameRequest?.maxOvertimeMinutes || 0,
      tournamentID: g.tournamentId,
      gameMode: g.gameRequest?.gameMode || 0,
    }));

    if (corresGames.length === 0) {
      return { nextCorresGame: null, corresGamesWaiting: 0 };
    }

    // Get all correspondence games where it's user's turn, including current game
    const now = Date.now(); // Use the same reference time during iteration
    // ("now" does not have to be Date.now(), it can be any const picked to minimize overflow)
    const gamesOnMyTurn = corresGames
      .filter((ag) => {
        // Check if it's user's turn
        const playerIndex = ag.players.findIndex((p) => p.uuid === userID);
        if (playerIndex === -1) {
          return false;
        }

        return playerIndex === ag.playerOnTurn;
      })
      .map((ag) => {
        // TODO: This cannot consider time banks unless backend sends them.
        // Calculate time remaining for sorting
        const timeElapsedSecs = (now - (ag.lastUpdate || 0)) / 1000;
        const timeRemainingSecs = ag.incrementSecs - timeElapsedSecs;

        return {
          game: ag,
          timeRemaining: timeRemainingSecs,
        };
      })
      .sort((a, b) => {
        // Do not use a-b even if it should not overflow
        if (a.timeRemaining < b.timeRemaining) return -1;
        if (a.timeRemaining > b.timeRemaining) return 1;
        // Tiebreak to stabilize order (this may not be the same as the backend)
        if (a.game.gameID < b.game.gameID) return -1;
        if (a.game.gameID > b.game.gameID) return 1;
        return 0;
      }); // Sort by most urgent first

    // This would exist if current game is on my turn.
    const currentGameIndex = gamesOnMyTurn.findIndex(
      (gomt) => gomt.game.gameID === gameID,
    );

    // If not on my turn (opponent's turn, completed game, others' game),
    // next game is the first one that is on my turn.
    return {
      nextCorresGame:
        currentGameIndex >= 0
          ? gamesOnMyTurn.length > 1
            ? gamesOnMyTurn[(currentGameIndex + 1) % gamesOnMyTurn.length].game
            : null
          : gamesOnMyTurn.length > 0
            ? gamesOnMyTurn[0].game
            : null,
      corresGamesWaiting:
        gamesOnMyTurn.length + (currentGameIndex >= 0 ? -1 : 0),
    };
  }, [gameInfo.gameRequest?.gameMode, userID, gameID, localCorresGames]);

  const handleNextCorresGame = useCallback(() => {
    if (nextCorresGame) {
      // Use full page navigation to ensure clean component mount
      window.location.href = `/game/${encodeURIComponent(nextCorresGame.gameID)}`;
    }
  }, [nextCorresGame]);

  const gameEpilog = useMemo(() => {
    // XXX: this doesn't get updated when game ends, only when refresh?

    return (
      <React.Fragment>
        {showingFinalTurn && (
          <React.Fragment>
            {gameInfo.gameEndReason === GameEndReason.FORCE_FORFEIT && (
              <React.Fragment>
                Game ended in forfeit.{/* XXX: How to get winners? */}
              </React.Fragment>
            )}
            {gameInfo.gameEndReason === GameEndReason.ADJUDICATED && (
              <React.Fragment>
                Game was adjudicated based on the score at the league deadline.
              </React.Fragment>
            )}
            {gameInfo.gameEndReason === GameEndReason.ABORTED && (
              <React.Fragment>
                The game was cancelled. Rating and statistics were not affected.
              </React.Fragment>
            )}
          </React.Fragment>
        )}
      </React.Fragment>
    );
  }, [gameInfo.gameEndReason, showingFinalTurn]);

  // Navigation card (Back to League/NEXT button) - rendered in different locations for mobile/desktop
  const navigationCard = useMemo(
    () => (
      <Card className="left-menu">
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          {gameInfo.leagueId && gameInfo.leagueSlug ? (
            <Link to={`/leagues/${gameInfo.leagueSlug}`}>
              <HomeOutlined />
              Back to League
            </Link>
          ) : gameInfo.tournamentId ? (
            <Link to={tournamentContext.metadata?.slug}>
              <HomeOutlined />
              Back to
              {isClubType(tournamentContext.metadata?.type)
                ? " Club"
                : " Tournament"}
            </Link>
          ) : (
            <Link to="/">
              <HomeOutlined />
              Back to lobby
            </Link>
          )}
          {nextCorresGame && (
            <div
              className="next-corres-game"
              onClick={handleNextCorresGame}
              style={{
                cursor: "pointer",
                marginLeft: "12px",
                whiteSpace: "nowrap",
              }}
            >
              <RightOutlined /> Next
              {corresGamesWaiting > 1 && (
                <span style={{ marginLeft: "4px" }}>
                  ({corresGamesWaiting})
                </span>
              )}
            </div>
          )}
        </div>
      </Card>
    ),
    [
      gameInfo.leagueId,
      gameInfo.leagueSlug,
      gameInfo.tournamentId,
      tournamentContext.metadata,
      nextCorresGame,
      corresGamesWaiting,
      handleNextCorresGame,
    ],
  );

  if (!gameID) {
    return (
      <div className="game-container">
        These are not the games you are looking for.
      </div>
    );
  }
  let ret = (
    <div
      className={`game-container${isRegistered ? " competitor" : ""}${
        commentsDrawerVisible ? " comments-drawer-open" : ""
      }`}
    >
      <ManageWindowTitleAndTurnSound gameInfo={gameInfo} />
      <TopBar
        tournamentID={gameInfo.tournamentId}
        leagueSlug={gameInfo.leagueSlug}
        nextCorresGameID={nextCorresGame?.gameID}
        corresGamesWaiting={corresGamesWaiting}
      />
      <div className={`game-table ${boardTheme} ${tileTheme}`}>
        <div
          className={`chat-area ${
            !isExamining && tournamentContext.metadata.disclaimer
              ? "has-disclaimer"
              : ""
          }`}
          id="left-sidebar"
        >
          {/* Navigation card (Back/NEXT) - only in chat-area for desktop */}
          {!isInMobileView && navigationCard}
          {playerNames.length > 1 ? (
            <Chat
              sendChat={props.sendChat}
              highlight={tournamentContext.directors}
              highlightText="Director"
              defaultChannel={`chat.${
                isObserver ? "gametv" : "game"
              }${props.annotated ? ".anno" : ""}.${gameID}`}
              defaultDescription={getChatTitle(
                playerNames,
                username,
                isObserver,
              )}
              tournamentID={gameInfo.tournamentId}
            />
          ) : null}
          {isExamining ? (
            <Analyzer includeCard />
          ) : (
            <React.Fragment key="not-examining">
              <Notepad includeCard />
              {tournamentContext.metadata.disclaimer && (
                <Disclaimer
                  disclaimer={tournamentContext.metadata.disclaimer}
                  logoUrl={tournamentContext.metadata.logo}
                />
              )}
            </React.Fragment>
          )}
          {isRegistered && (
            <CompetitorStatus
              sendReady={() =>
                readyForTournamentGame(
                  sendSocketMsg,
                  tournamentContext.metadata?.id,
                  competitorState,
                )
              }
            />
          )}
        </div>
        {/* There are two player cards, css hides one of them. */}
        <div className="sticky-player-card-container">
          <PlayerCards
            horizontal
            gameMeta={gameInfo}
            playerMeta={gameInfo.players}
            hideProfileLink={gameInfo.type === GameType.ANNOTATED}
          />
        </div>
        <div className="play-area">
          <BoardPanel
            anonymousViewer={!loggedIn}
            username={username}
            board={examinableGameContext.board}
            currentRack={sortedRack}
            events={examinableGameContext.turns}
            gameID={gameID}
            sendSocketMsg={props.sendSocketMsg}
            sendGameplayEvent={(evt) =>
              props.sendSocketMsg(
                encodeToSocketFmt(
                  MessageType.CLIENT_GAMEPLAY_EVENT,
                  toBinary(ClientGameplayEventSchema, evt),
                ),
              )
            }
            gameDone={gameDone}
            playerMeta={gameInfo.players}
            tournamentID={gameInfo.tournamentId}
            leagueID={gameInfo.leagueId}
            leagueSlug={gameInfo.leagueSlug}
            vsBot={gameInfo.gameRequest?.playerVsBot ?? false}
            gameMode={gameInfo.gameRequest?.gameMode}
            tournamentSlug={tournamentContext.metadata?.slug}
            tournamentPairedMode={isPairedMode(
              tournamentContext.metadata?.type,
            )}
            tournamentNonDirectorObserver={tournamentNonDirectorObserver}
            tournamentPrivateAnalysis={
              tournamentContext.metadata?.privateAnalysis
            }
            lexicon={gameInfo.gameRequest?.lexicon ?? ""}
            alphabet={alphabet}
            challengeRule={
              gameInfo.gameRequest?.challengeRule ?? ChallengeRule.VOID
            }
            handleAcceptRematch={
              rematchRequest.rematchFor === gameID ? handleAcceptRematch : null
            }
            handleAcceptAbort={() => {}}
            handleSetHover={handleSetHover}
            handleUnsetHover={hideDefinitionHover}
            definitionPopover={definitionPopover}
          />
          {!gameDone && (
            <MetaEventControl
              sendSocketMsg={props.sendSocketMsg}
              gameID={gameID}
            />
          )}
          {/* StreakWidget in play-area for desktop view */}
          {!isInMobileView && <StreakWidget streakInfo={streakGameInfo} />}
        </div>
        <div className="data-area" id="right-sidebar">
          {/* There are two competitor cards, css hides one of them. */}
          {isRegistered && (
            <CompetitorStatus
              sendReady={() =>
                readyForTournamentGame(
                  sendSocketMsg,
                  tournamentContext.metadata?.id,
                  competitorState,
                )
              }
            />
          )}
          {/* There are two player cards, css hides one of them. */}
          <PlayerCards
            gameMeta={gameInfo}
            playerMeta={gameInfo.players}
            hideProfileLink={gameInfo.type === GameType.ANNOTATED}
          />
          <GameInfo
            meta={gameInfo}
            tournamentName={tournamentContext.metadata?.name}
            colorOverride={tournamentContext.metadata?.color}
            logoUrl={tournamentContext.metadata?.logo}
            description={gameContext.gameDocument?.description}
            gameDocument={gameContext.gameDocument}
            currentUserId={userID}
          />
          <Pool
            pool={examinableGameContext?.pool}
            currentRack={sortedRack}
            poolFormat={poolFormat}
            setPoolFormat={setPoolFormat}
            alphabet={alphabet}
          />
          {/* Navigation card (Back/NEXT) in data-area for mobile view (after Pool) */}
          {isInMobileView && navigationCard}
          {/* StreakWidget in data-area for mobile view (after navigation) */}
          {isInMobileView && <StreakWidget streakInfo={streakGameInfo} />}
          <Popconfirm
            title={`${rematchRequest.user?.displayName} sent you a rematch request`}
            open={rematchRequest.rematchFor !== ""}
            onConfirm={handleAcceptRematch}
            onCancel={handleDeclineRematch}
            okText="Accept"
            cancelText="Decline"
          />
          <ScoreCard
            isExamining={isExamining}
            events={examinableGameContext.turns}
            board={examinableGameContext.board}
            playerMeta={gameInfo.players}
            poolFormat={poolFormat}
            gameEpilog={gameEpilog}
            showComments={props.annotated ?? false}
            onOpenCommentsDrawer={
              props.annotated ? handleOpenCommentsDrawerForTurn : undefined
            }
            comments={comments || []}
            editComment={editComment}
            addNewComment={addNewComment}
            deleteComment={deleteComment}
            isCorrespondence={
              gameInfo.gameRequest?.gameMode === GameMode.CORRESPONDENCE
            }
            timeBankP0={
              gameInfo.gameRequest?.timeBankMinutes
                ? gameInfo.gameRequest.timeBankMinutes * 60000
                : undefined
            }
            timeBankP1={
              gameInfo.gameRequest?.timeBankMinutes
                ? gameInfo.gameRequest.timeBankMinutes * 60000
                : undefined
            }
          />
        </div>
      </div>
    </div>
  );

  // Add the CommentsDrawer and Monitoring components
  ret = (
    <>
      {ret}
      {/* Monitoring widget - only show if tournament requires monitoring and user is not observer */}
      {gameInfo.tournamentId && !isObserver && <MonitoringWidget />}
      {/* Monitoring modal */}
      <MonitoringModal
        visible={monitoringModalVisible}
        onClose={closeMonitoringModal}
      />
      {props.annotated && (
        <CommentsDrawer
          visible={commentsDrawerVisible}
          onClose={handleCloseCommentsDrawer}
          eventNumber={commentsDrawerEventNumber}
          comments={comments || []}
          turns={turns}
          board={examinableGameContext.board}
          alphabet={gameContext.alphabet}
          players={gameInfo.players}
          onAddComment={handleAddCommentInDrawer}
          onEditComment={handleEditCommentInDrawer}
          onDeleteComment={handleDeleteCommentInDrawer}
          gameId={gameID || ""}
        />
      )}
    </>
  );

  ret = (
    <NotepadContextProvider
      children={ret}
      feRackInfo={feRackInfo}
      gameID={gameID}
    />
  );
  ret = (
    <AnalyzerContextProvider
      children={ret}
      lexicon={gameInfo.gameRequest?.lexicon ?? ""}
      variant={gameInfo.gameRequest?.rules?.variantName}
    />
  );
  return ret;
});
