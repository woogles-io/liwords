// boardwizard is our board editor

import { HomeOutlined } from "@ant-design/icons";
import { App, Card } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router";
import { ActionType } from "../actions/actions";
import { alphabetFromName } from "../constants/alphabets";
import { Analyzer, AnalyzerContextProvider } from "../gameroom/analyzer";
import { BoardPanel } from "../gameroom/board_panel";
import { PlayerCards } from "../gameroom/player_cards";
import Pool from "../gameroom/pool";
import { ScoreCard } from "../gameroom/scorecard";
import { CommentsDrawer } from "../gameroom/CommentsDrawer";
import { GameInfo } from "../gameroom/game_info";

import {
  ClientGameplayEvent,
  PlayerInfoSchema as OMGPlayerInfoSchema,
  ChallengeRule as OMGChallengeRule,
  GameDocumentSchema,
  GameDocument_MinimalPlayerInfoSchema,
  GameRulesSchema,
} from "../gen/api/proto/ipc/omgwords_pb";
import { GameEventService } from "../gen/api/proto/omgwords_service/omgwords_pb";
import { defaultLetterDistribution } from "../lobby/sought_game_interactions";
import { TopBar } from "../navigation/topbar";
import { sortTiles } from "../store/constants";
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  usePoolFormatStoreContext,
} from "../store/store";
import { useClient, flashError } from "../utils/hooks/connect";
import { useDefinitionAndPhonyChecker } from "../utils/hooks/definitions";
import { EditorControl } from "./editor_control";
import { PlayState } from "../gen/api/proto/ipc/omgwords_pb";
import { syntheticGameInfo } from "./synthetic_game_info";
import { EditorLandingPage } from "./new_game";
import { MachineLetter, MachineWord } from "../utils/cwgame/common";
import { create } from "@bufbuild/protobuf";
import { useComments } from "../utils/hooks/comments";
import { GameCommentService } from "../gen/api/proto/comments_service/comments_service_pb";
import { gameEventsToTurns } from "../store/reducers/turns";

const doNothing = () => {};

const blankGamePayload = create(GameDocumentSchema, {
  players: [
    create(GameDocument_MinimalPlayerInfoSchema, {
      nickname: "player1",
      userId: "player1",
    }),
    create(GameDocument_MinimalPlayerInfoSchema, {
      nickname: "player2",
      userId: "player2",
    }),
  ],
});

export const BoardEditor = () => {
  const { gameID } = useParams();
  const navigate = useNavigate();

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();

  const {
    handleExamineStart,
    handleExamineLast,
    handleExamineDisableShortcuts,
  } = useExamineStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();
  const { notification } = App.useApp();
  const { handleSetHover, hideDefinitionHover, definitionPopover } =
    useDefinitionAndPhonyChecker({
      addChat: doNothing,
      enableHoverDefine: true,
      gameContext,
      gameDone: true,
      gameID: gameID,
      lexicon: gameContext.gameDocument.lexicon,
      variant: gameContext.gameDocument.variant,
    });

  const eventClient = useClient(GameEventService);
  const commentsClient = useClient(GameCommentService);

  // Comments system
  const { comments, editComment, addNewComment, deleteComment } = useComments(
    commentsClient,
    true, // Editor always shows comments
  );

  // Comments drawer state
  const [searchParams, setSearchParams] = useSearchParams();
  const [commentsDrawerVisible, setCommentsDrawerVisible] = useState(false);
  const [commentsDrawerEventNumber, setCommentsDrawerEventNumber] =
    useState<number>(0);

  useEffect(() => {
    if (gameContext.gameID) {
      handleExamineStart();
      handleExamineLast();
      handleExamineDisableShortcuts();
    }
  }, [
    handleExamineStart,
    handleExamineLast,
    handleExamineDisableShortcuts,
    gameContext.gameID,
  ]);

  // Convert GameEvents to Turn objects
  const turns = useMemo(
    () => gameEventsToTurns(examinableGameContext.turns),
    [examinableGameContext.turns],
  );

  // Comments drawer handlers
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

  const handleCloseCommentsDrawer = useCallback(() => {
    setCommentsDrawerVisible(false);

    // Update URL - remove comments parameter
    const newParams = new URLSearchParams(searchParams);
    newParams.delete("comments");
    setSearchParams(newParams);
  }, [searchParams, setSearchParams]);

  // Handle URL parameters for comments drawer - run when turns are loaded
  useEffect(() => {
    if (turns.length === 0) return;

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
  }, [turns, searchParams]);

  // Comment handlers for drawer
  const handleAddCommentInDrawer = useCallback(
    (comment: string) => {
      addNewComment(gameID || "", commentsDrawerEventNumber, comment);
    },
    [commentsDrawerEventNumber, addNewComment, gameID],
  );

  const fetchAndDispatchDocument = useCallback(
    async (gid: string, redirect: boolean) => {
      try {
        const resp = await eventClient.getGameDocument({
          gameId: gid,
        });
        console.log("got a game document, dispatching, redirect is", redirect);
        dispatchGameContext({
          actionType: ActionType.InitFromDocument,
          payload: resp,
        });
        if (redirect) {
          // Also, redirect the URL so we can subscribe to the right channel
          // on the socket.
          navigate(`/editor/${encodeURIComponent(gid)}`, { replace: true });
        }
      } catch (e) {
        flashError(e);
      }
    },
    [dispatchGameContext, eventClient, navigate],
  );

  // Initialize on mount with unfinished game, new game, or existing game:
  useEffect(() => {
    if (gameID) {
      fetchAndDispatchDocument(gameID, false);
      return;
    }
    const initFromDoc = async () => {
      let continuedGame;

      try {
        const resp = await eventClient.getMyUnfinishedGames({});
        if (resp.games.length > 0) {
          continuedGame = resp.games[resp.games.length - 1];
        }
      } catch (e) {
        flashError(e);
      }

      if (!continuedGame) {
        // Just dispatch a blank game.
        dispatchGameContext({
          actionType: ActionType.InitFromDocument,
          payload: blankGamePayload,
        });
        return;
      }
      // Otherwise, fetch the game from the server and try to continue it.
      fetchAndDispatchDocument(continuedGame.gameId, true);
    };

    initFromDoc();
  }, [gameID, eventClient, fetchAndDispatchDocument, dispatchGameContext]);

  useEffect(() => {
    if (gameContext.playState === PlayState.WAITING_FOR_FINAL_PASS) {
      notification.info({
        message: "Pass or challenge",
        description:
          "The bag is now empty; please Pass or Challenge to end the game.",
      });
    }
  }, [gameContext.playState, notification]);

  const sortedRack = useMemo(() => {
    const rack =
      examinableGameContext.players.find((p) => p.onturn)?.currentRack ??
      new Array<MachineLetter>();
    return sortTiles(rack, examinableGameContext.alphabet);
  }, [examinableGameContext.alphabet, examinableGameContext.players]);

  const alphabet = useMemo(
    () => alphabetFromName(gameContext.gameDocument.letterDistribution),
    [gameContext.gameDocument.letterDistribution],
  );

  const changeCurrentRack = async (rack: MachineWord, evtIdx: number) => {
    let onturn = gameContext.onturn;
    let amendment = false;
    const racks: [Uint8Array, Uint8Array] = [
      new Uint8Array(),
      new Uint8Array(),
    ];

    if (evtIdx !== gameContext.turns.length) {
      // We're trying to edit an old event's rack.
      // not onturn here
      onturn = gameContext.turns[evtIdx].playerIndex;
      amendment = true;
    }

    racks[onturn] = Uint8Array.from(rack);

    try {
      await eventClient.setRacks({
        gameId: gameContext.gameID,
        racks,
        eventNumber: evtIdx,
        amendment,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const sendGameplayEvent = async (evt: ClientGameplayEvent) => {
    let amendment = false;
    let evtIdx = gameContext.turns.length;

    if (examinableGameContext.turns.length !== gameContext.turns.length) {
      // We're trying to edit an old event.
      amendment = true;
      evtIdx = examinableGameContext.turns.length;
    }

    try {
      await eventClient.sendGameEvent({
        event: evt,
        userId:
          examinableGameContext.players[examinableGameContext.onturn].userID,
        amendment,
        eventNumber: evtIdx,
      });
      // Skip to the end after sending the event
      // Note: This causes a brief flicker for amendments (showing end state before truncation)
      // TODO: Fix the flicker by properly managing examiner position during document updates
      handleExamineLast();
    } catch (e) {
      flashError(e);
    }
  };

  const omgPlayerInfo = (pname: string, idx: number) => {
    const collapsed = pname.replaceAll(" ", "");
    return create(OMGPlayerInfoSchema, {
      nickname: collapsed,
      fullName: pname,
      userId: collapsed,
      first: idx === 0,
    });
  };

  const createNewGame = async (
    p1name: string,
    p2name: string,
    lex: string,
    chrule: OMGChallengeRule,
  ) => {
    // the lexicon and letter distribution are tied together.
    const ld = defaultLetterDistribution(lex);
    try {
      const resp = await eventClient.createBroadcastGame({
        playersInfo: [p1name, p2name].map(omgPlayerInfo),
        lexicon: lex,
        rules: create(GameRulesSchema, {
          boardLayoutName: "CrosswordGame", // for now
          letterDistributionName: ld,
          variantName: "classic", // for now
        }),
        challengeRule: chrule,
        public: false,
      });
      fetchAndDispatchDocument(resp.gameId, true);
    } catch (e) {
      flashError(e);
    }
  };

  const editGame = async (
    p1name: string,
    p2name: string,
    description: string,
  ) => {
    try {
      await eventClient.patchGameDocument({
        document: create(GameDocumentSchema, {
          players: [p1name, p2name].map((p, idx) => {
            const collapsed = p.replaceAll(" ", "");
            return create(GameDocument_MinimalPlayerInfoSchema, {
              nickname: collapsed,
              realName: p,
              userId: collapsed,
              quit: gameContext.gameDocument.players[idx].quit,
            });
          }),
          description: description,
          uid: gameContext.gameID,
        }),
      });
      notification.success({
        message: "Saved game information",
      });
    } catch (e) {
      flashError(e);
    }
  };

  const deleteGame = async (gid: string) => {
    try {
      await eventClient.deleteBroadcastGame({ gameId: gid });
      dispatchGameContext({
        actionType: ActionType.InitFromDocument,
        payload: blankGamePayload,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const macChallengeRule = useMemo(
    () => gameContext.gameDocument.challengeRule.valueOf(),
    [gameContext.gameDocument.challengeRule],
  );
  // Create a GameInfoResponse for the purposes of rendering a few of our widgets.
  const gameInfo = useMemo(() => {
    return syntheticGameInfo(gameContext.gameDocument);
  }, [gameContext.gameDocument]);

  if (!gameContext.gameID) {
    return <EditorLandingPage createNewGame={createNewGame} />;
  }

  let ret = (
    <div className="game-container editor-subpage">
      <TopBar />
      <div className="game-table">
        <div className="chat-area" id="left-sidebar">
          <Card className="left-menu">
            <Link to="/">
              <HomeOutlined />
              Back to lobby
            </Link>
          </Card>
          {/* <Chat
            sendChat={props.sendChat}
            defaultChannel={`chat.${isObserver ? 'gametv' : 'game'}.${gameID}`}
            defaultDescription={getChatTitle(playerNames, username, isObserver)}
            tournamentID={gameInfo.tournamentId}
          /> */}
          <Card></Card>
          <Analyzer includeCard />

          <Card
            title="Editor controls"
            className="editor-control"
            style={{ marginTop: 12 }}
          >
            <EditorControl
              createNewGame={() => {}}
              gameID={gameContext.gameID}
              deleteGame={deleteGame}
              editGame={editGame}
            />
          </Card>
        </div>
        <div className="sticky-player-card-container">
          <PlayerCards
            horizontal
            gameMeta={gameInfo}
            playerMeta={gameInfo.players}
            hideProfileLink
          />
        </div>

        <div className="play-area">
          <BoardPanel
            boardEditingMode={true}
            anonymousViewer={false} // tbd
            username={""} // shouldn't matter, but it might have to be some large random string
            board={examinableGameContext.board}
            currentRack={sortedRack}
            events={examinableGameContext.turns}
            gameID={gameContext.gameID}
            sendSocketMsg={doNothing}
            sendGameplayEvent={(evt) => sendGameplayEvent(evt)}
            gameDone={false} // tbd
            playerMeta={gameInfo.players}
            tournamentID={""}
            vsBot={false}
            tournamentPairedMode={false}
            lexicon={gameContext.gameDocument.lexicon}
            alphabet={alphabet}
            challengeRule={macChallengeRule}
            handleAcceptRematch={doNothing}
            handleAcceptAbort={doNothing}
            handleSetHover={handleSetHover}
            handleUnsetHover={hideDefinitionHover}
            definitionPopover={definitionPopover}
            changeCurrentRack={changeCurrentRack}
            exitableExaminer={false}
          />
        </div>

        <div className="data-area" id="right-sidebar">
          {/* There are two player cards, css hides one of them. */}
          <PlayerCards
            gameMeta={gameInfo}
            playerMeta={gameInfo.players}
            hideProfileLink
          />
          <GameInfo meta={gameInfo} tournamentName={""} />
          <Pool
            pool={examinableGameContext?.pool}
            currentRack={sortedRack}
            poolFormat={poolFormat}
            setPoolFormat={setPoolFormat}
            alphabet={alphabet}
          />
          <ScoreCard
            isExamining={true}
            events={examinableGameContext.turns}
            board={examinableGameContext.board}
            playerMeta={gameInfo.players}
            poolFormat={poolFormat}
            showComments={true}
            onOpenCommentsDrawer={handleOpenCommentsDrawerForTurn}
            comments={comments}
            editComment={editComment}
          />
        </div>
      </div>
      <CommentsDrawer
        visible={commentsDrawerVisible}
        onClose={handleCloseCommentsDrawer}
        eventNumber={commentsDrawerEventNumber}
        comments={comments || []}
        turns={turns}
        board={examinableGameContext.board}
        alphabet={alphabet}
        players={gameInfo.players}
        onAddComment={handleAddCommentInDrawer}
        onEditComment={editComment}
        onDeleteComment={deleteComment}
        gameId={gameID || ""}
        baseUrl={`${window.location.origin}/anno/${encodeURIComponent(gameID || "")}`}
      />
    </div>
  );
  ret = (
    <AnalyzerContextProvider
      children={ret}
      lexicon={gameContext.gameDocument.lexicon}
      variant={gameContext.gameDocument.variant}
    />
  );
  return ret;
};
