import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Button, Tooltip, Affix, App } from "antd";
import { Modal } from "../utils/focus_modal";
import { DndProvider } from "react-dnd";
import MultiBackend, { MultiBackendOptions } from "./dnd-backend";
import {
  ArrowDownOutlined,
  EditOutlined,
  SyncOutlined,
} from "@ant-design/icons";
import {
  uniqueTileIdx,
  EphemeralTile,
  EmptyRackSpaceMachineLetter,
  MachineWord,
  MachineLetter,
  EmptyBoardSpaceMachineLetter,
  BlankMachineLetter,
} from "../utils/cwgame/common";

import GameBoard from "./board";
import GameControls from "./game_controls";
import { Rack } from "./rack";
import { ExchangeTiles } from "./exchange_tiles";
import { ChallengeWordsModal } from "./challenge_words_modal";
import {
  nextArrowPropertyState,
  handleKeyPress,
  handleDroppedTile,
  handleTileDeletion,
  returnTileToRack,
  designateBlank,
  nextArrowStateAfterTilePlacement,
} from "../utils/cwgame/tile_placement";

import {
  tilesetToMoveEvent,
  exchangeMoveEvent,
  passMoveEvent,
  resignMoveEvent,
  challengeMoveEvent,
  nicknameFromEvt,
} from "../utils/cwgame/game_event";
import { Board } from "../utils/cwgame/board";
import { encodeToSocketFmt } from "../utils/protobuf";
import { isSpanish, sharedEnableAutoShuffle } from "../store/constants";
import { BlankSelector } from "./blank_selector";
import { GameMetaMessage } from "./game_meta_message";
import {
  ChallengeRule,
  GameEvent,
  GameEvent_Type,
  PlayState,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { TilePreview } from "./tile";
import { Alphabet } from "../constants/alphabets";
import { MessageType } from "../gen/api/proto/ipc/ipc_pb";
import {
  MatchUserSchema,
  SeekRequestSchema,
  SeekState,
} from "../gen/api/proto/ipc/omgseeks_pb";
import {
  ClientGameplayEvent,
  GameMetaEvent_EventType,
  GameMetaEventSchema,
  GameMode,
  PlayerInfo,
} from "../gen/api/proto/ipc/omgwords_pb";
import { PuzzleStatus } from "../gen/api/proto/puzzle_service/puzzle_service_pb";
import { useClient } from "../utils/hooks/connect";
import { GameMetadataService } from "../gen/api/proto/game_service/game_service_pb";

import { RackEditor } from "./rack_editor";
import { shuffleLetters, gcgExport, backupKey } from "./board_panel_utils";
import { handleBlindfoldKeydown } from "./blindfold_mode";
import { useBoardPanelState } from "./useBoardPanelState";
import { useTilePlacement } from "./useTilePlacement";

// The frame atop is 24 height
// The frames on the sides are 24 in width, surrounded by a 14 pix gutter
const EnterKey = "Enter";
import variables from "../base.module.scss";
import { create, toBinary } from "@bufbuild/protobuf";
import { consoleInstructions as drawingConsoleInstructions } from "./drawing";
const { colorPrimary } = variables;

type Props = {
  anonymousViewer: boolean;
  username: string;
  currentRack: MachineWord;
  events: Array<GameEvent>;
  gameID: string;
  challengeRule: ChallengeRule;
  board: Board;
  sendSocketMsg: (msg: Uint8Array) => void;
  sendGameplayEvent: (evt: ClientGameplayEvent) => void;
  gameDone: boolean;
  playerMeta: Array<PlayerInfo>;
  puzzleMode?: boolean;
  puzzleSolved?: number;
  boardEditingMode?: boolean;
  tournamentSlug?: string;
  tournamentID?: string;
  tournamentPairedMode?: boolean;
  tournamentNonDirectorObserver?: boolean;
  tournamentPrivateAnalysis?: boolean;
  leagueID?: string;
  leagueSlug?: string;
  lexicon: string;
  alphabet: Alphabet;
  handleAcceptRematch: (() => void) | null;
  handleAcceptAbort: (() => void) | null;
  handleSetHover?: (
    x: number,
    y: number,
    words: Array<string> | undefined,
  ) => void;
  handleUnsetHover?: () => void;
  definitionPopover?:
    | { x: number; y: number; content: React.ReactNode }
    | undefined;
  vsBot: boolean;
  exitableExaminer?: boolean;
  changeCurrentRack?: (rack: MachineWord, evtIdx: number) => void;
  gameMode?: number;
};

export const BoardPanel = React.memo((props: Props) => {
  const {
    currentMode,
    setCurrentMode,
    drawingCanBeEnabled,
    handleDrawingKeyDown,
    arrowProperties,
    setArrowProperties,
    examinableGameContext,
    examinableGameEndMessage,
    examinableTimerContext,
    isExamining,
    handleExamineStart,
    gameContext,
    stopClock,
    exchangeAllowed,
    setExchangeAllowed,
    handlePassShortcut,
    setHandlePassShortcut,
    handleChallengeShortcut,
    setHandleChallengeShortcut,
    handleNeitherShortcut,
    setHandleNeitherShortcut,
    boardContainer,
    displayedRack,
    setDisplayedRack,
    placedTiles,
    setPlacedTiles,
    placedTilesTempScore,
    setPlacedTilesTempScore,
    blindfoldCommand,
    setBlindfoldCommand,
    blindfoldUseNPA,
    setBlindfoldUseNPA,
    pendingExchangeTiles,
    setPendingExchangeTiles,
    observer,
    isMyTurn,
  } = useBoardPanelState({
    playerMeta: props.playerMeta,
    username: props.username,
    board: props.board,
    currentRack: props.currentRack,
    puzzleMode: props.puzzleMode,
    boardEditingMode: props.boardEditingMode,
  });

  const { recallTiles, shuffleTiles, moveRackTile } = useTilePlacement({
    arrowProperties,
    setArrowProperties,
    placedTiles,
    setPlacedTiles,
    setPlacedTilesTempScore,
    displayedRack,
    setDisplayedRack,
    board: props.board,
    currentRack: props.currentRack,
    setPendingExchangeTiles,
  });

  const {
    board,
    gameID,
    playerMeta,
    sendSocketMsg,
    sendGameplayEvent,
    handleUnsetHover,
    username,
    boardEditingMode,
  } = props;

  const { message, notification } = App.useApp();

  // Challenge modal state
  const [challengeModalVisible, setChallengeModalVisible] = useState(false);

  // Get words from the last play for challenge modal
  const lastWordsFormed = useMemo(() => {
    if (examinableGameContext.turns.length > 0) {
      const lastTurn =
        examinableGameContext.turns[examinableGameContext.turns.length - 1];
      // wordsFormed comes from backend via GameEvent, so indices match backend
      return lastTurn?.wordsFormed ?? [];
    }
    return [];
  }, [examinableGameContext.turns]);

  const makeMove = useCallback(
    (move: string, addl?: Array<MachineLetter>) => {
      if (isExamining && !boardEditingMode) return;
      let moveEvt;
      if (move !== "resign" && !isMyTurn) {
        console.log(
          "off turn move attempts",
          gameContext.nickToPlayerOrder,
          username,
          examinableGameContext.onturn,
        );
        // It is not my turn. Ignore this event.
        message.warning({
          content: "It is not your turn.",
          className: "board-hud-message",
          key: "board-messages",
          duration: 1.5,
        });
        return;
      }

      switch (move) {
        case "exchange":
          if (addl) {
            moveEvt = exchangeMoveEvent(addl, gameID, gameContext.alphabet);
          }
          break;
        case "pass":
          moveEvt = passMoveEvent(gameID);
          break;
        case "resign":
          moveEvt = resignMoveEvent(gameID);
          break;
        case "challenge":
          // Show word selection modal ONLY for:
          // 1. 5-point challenge rule AND
          // 2. Editor mode (boardEditingMode prop)
          if (
            boardEditingMode &&
            props.challengeRule === ChallengeRule.FIVE_POINT &&
            lastWordsFormed.length > 0
          ) {
            setChallengeModalVisible(true);
            return; // Don't send event yet, wait for modal confirmation
          }
          // For live games or non-5-point, challenge all words (current behavior)
          moveEvt = challengeMoveEvent(gameID);
          break;
        case "commit":
          // Check if we have pending exchange tiles from the analyzer
          if (pendingExchangeTiles) {
            moveEvt = exchangeMoveEvent(
              pendingExchangeTiles,
              gameID,
              gameContext.alphabet,
            );
            // Clear the pending exchange tiles
            setPendingExchangeTiles(null);
          } else {
            moveEvt = tilesetToMoveEvent(placedTiles, board, gameID);
            if (!moveEvt) {
              // this is an invalid play
              return;
            }
          }
          clearBackupRef.current = true;
          break;
      }
      if (!moveEvt) {
        return;
      }
      sendGameplayEvent(moveEvt);

      // Don't stop the clock; the next user event to come in will change the
      // clock over.
      // stopClock();
      if (boardContainer.current) {
        // Reenable keyboard shortcut after passing with 22.
        boardContainer.current.focus();
      }
    },
    [
      gameContext.alphabet,
      gameContext.nickToPlayerOrder,
      boardContainer,
      boardEditingMode,
      examinableGameContext.onturn,
      isExamining,
      isMyTurn,
      placedTiles,
      board,
      gameID,
      sendGameplayEvent,
      username,
      message,
      pendingExchangeTiles,
      setPendingExchangeTiles,
      props.challengeRule,
      lastWordsFormed,
    ],
  );

  const sendMetaEvent = useCallback(
    (evtType: GameMetaEvent_EventType) => {
      const metaEvt = create(GameMetaEventSchema);
      metaEvt.type = evtType;
      metaEvt.gameId = gameID;

      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.GAME_META_EVENT,
          toBinary(GameMetaEventSchema, metaEvt),
        ),
      );
    },
    [sendSocketMsg, gameID],
  );

  // For reinstating a premove if an invalid move that invalidates it is successfully challenged off.
  const backupStatesRef = useRef(
    new Map<
      string,
      {
        displayedRack: Array<MachineLetter>;
        placedTiles: Set<EphemeralTile>;
        placedTilesTempScore: number | undefined;
        arrowProperties: {
          row: number;
          col: number;
          horizontal: boolean;
          show: boolean;
        };
      }
    >(),
  );

  // for use with right-click
  const recallOneTile = useCallback(
    (row: number, col: number) => {
      const handlerReturn = handleTileDeletion(
        {
          row,
          col,
          horizontal: true,
          show: true,
        },
        displayedRack,
        placedTiles,
        props.board,
        props.alphabet,
      );
      setDisplayedRack(handlerReturn.newDisplayedRack);
      // ignore the newArrow
      setPlacedTiles(handlerReturn.newPlacedTiles);
      setPlacedTilesTempScore(handlerReturn.playScore);
    },
    [
      displayedRack,
      placedTiles,
      props.alphabet,
      props.board,
      setDisplayedRack,
      setPlacedTiles,
      setPlacedTilesTempScore,
    ],
  );

  const clearBackupRef = useRef<boolean>(false);
  const lastLettersRef = useRef<Array<MachineLetter>>(undefined);
  const lastRackRef = useRef<Array<MachineLetter>>(undefined);
  const lastIsExaminingRef = useRef<boolean>(undefined);
  const readOnlyEffectDependenciesRef = useRef<{
    displayedRack: Array<MachineLetter>;
    isMyTurn: boolean;
    placedTiles: Set<EphemeralTile>;
    dim: number;
    arrowProperties: {
      row: number;
      col: number;
      horizontal: boolean;
      show: boolean;
    };
    placedTilesTempScore: number | undefined;
  }>(undefined);
  readOnlyEffectDependenciesRef.current = {
    displayedRack,
    isMyTurn,
    placedTiles,
    dim: props.board.dim,
    arrowProperties,
    placedTilesTempScore,
  };

  // Need to sync state to props here whenever the board changes.
  useEffect(() => {
    if (lastIsExaminingRef.current !== isExamining) {
      // Throw away all backups when toggling examiner.
      lastIsExaminingRef.current = isExamining;
      backupStatesRef.current.clear();
    }
    let fullReset = false;
    const lastLetters = lastLettersRef.current;
    const dep = readOnlyEffectDependenciesRef.current!;
    if (lastLetters === undefined) {
      // First load.
      fullReset = true;
    } else if (props.puzzleMode) {
      // XXX: Without this, when exiting from examining an earlier turn on
      // puzzle mode, it does not reset the rack to the latest rack. Why?
      fullReset = true;
    } else if (props.boardEditingMode) {
      // See comment above. I don't know why it doesn't show the latest rack
      // when we edit more than once.
      fullReset = true;
    } else if (
      JSON.stringify(
        [
          ...[...dep.placedTiles].map((ephemeralTile) => {
            const ml = ephemeralTile.letter;
            return (ml & 0x80) !== 0 ? 0 : ml;
          }),
          ...dep.displayedRack.filter((ml) => ml !== 0x80),
        ].sort(),
      ) !== JSON.stringify([...props.currentRack].sort())
    ) {
      // First load after receiving rack.
      // Or other cases where the tiles don't match up.
      fullReset = true;
    } else if (isExamining) {
      // Prevent stuck tiles.
      fullReset = true;
    } else if (!dep.isMyTurn) {
      // Opponent's turn means we have just made a move. (Assumption: there are only two players.)
      fullReset = true;
    } else {
      // Opponent just did something. Check if it affects any premove.
      // TODO: revisit when supporting non-square boards.
      const letterAt = (
        row: number,
        col: number,
        letters: Array<MachineLetter>,
      ) =>
        row < 0 || row >= dep.dim || col < 0 || col >= dep.dim
          ? null
          : letters[row * dep.dim + col];
      const letterChanged = (row: number, col: number) =>
        letterAt(row, col, lastLetters) !==
        letterAt(row, col, props.board.letters);
      const hookChanged = (
        row: number,
        col: number,
        drow: number,
        dcol: number,
      ) => {
        while (true) {
          row += drow;
          col += dcol;
          if (letterChanged(row, col)) return true;
          const letter = letterAt(row, col, props.board.letters);
          if (letter === null || letter === EmptyBoardSpaceMachineLetter) {
            return false;
          }
        }
      };
      const placedTileAffected = (row: number, col: number) =>
        letterChanged(row, col) ||
        hookChanged(row, col, -1, 0) ||
        hookChanged(row, col, +1, 0) ||
        hookChanged(row, col, 0, -1) ||
        hookChanged(row, col, 0, +1);
      // If no tiles have been placed, but placement arrow is shown,
      // reset based on if that position is affected.
      // This avoids having the placement arrow behind a tile.
      if (
        (dep.placedTiles.size === 0 && dep.arrowProperties.show
          ? [dep.arrowProperties as { row: number; col: number }]
          : Array.from(dep.placedTiles)
        ).some(({ row, col }) => placedTileAffected(row, col))
      ) {
        fullReset = true;
      }
    }
    const bak = backupStatesRef.current.get(
      backupKey(props.board.letters, props.currentRack),
    );
    // Do not reset if considering a new placement move when challenging.
    if (fullReset || (bak && dep.placedTiles.size === 0)) {
      backupStatesRef.current.clear();
      if (!clearBackupRef.current) {
        const lastRack = lastRackRef.current;
        if (lastLetters && lastRack && dep.displayedRack.length) {
          backupStatesRef.current.set(backupKey(lastLetters, lastRack), {
            displayedRack: dep.displayedRack,
            placedTiles: dep.placedTiles,
            placedTilesTempScore: dep.placedTilesTempScore,
            arrowProperties: dep.arrowProperties,
          });
        }
      }
      clearBackupRef.current = false;
      if (bak) {
        setDisplayedRack(bak.displayedRack);
        setPlacedTiles(bak.placedTiles);
        setPlacedTilesTempScore(bak.placedTilesTempScore);
        setArrowProperties(bak.arrowProperties);
      } else {
        let rack = props.currentRack;
        if (sharedEnableAutoShuffle) {
          rack = shuffleLetters(rack);
        }
        setDisplayedRack(rack);
        setPlacedTiles(new Set<EphemeralTile>());
        setPlacedTilesTempScore(0);
        setArrowProperties({
          row: 0,
          col: 0,
          horizontal: false,
          show: false,
        });
      }
    }
    lastLettersRef.current = props.board.letters;
    lastRackRef.current = props.currentRack;
  }, [
    isExamining,
    props.board.letters,
    props.boardEditingMode,
    props.currentRack,
    props.puzzleMode,
    setArrowProperties,
    setDisplayedRack,
    setPlacedTiles,
    setPlacedTilesTempScore,
  ]);

  useEffect(() => {
    // Stop the clock if we unload the board panel.
    return () => {
      stopClock();
    };
  }, [stopClock]);

  useEffect(() => {
    const bag = { ...gameContext.pool };
    for (let i = 0; i < props.currentRack.length; i += 1) {
      bag[props.currentRack[i]] -= 1;
    }
    const tilesRemaining =
      Object.values(bag).reduce((acc, cur) => {
        return acc + cur;
      }, 0) - 7;
    // Subtract 7 for opponent rack, won't matter when the
    // rack is smaller than that because past the threshold by then
    if (isSpanish(props.lexicon)) {
      setExchangeAllowed(
        tilesRemaining >= 1 || props.boardEditingMode === true,
      );
    } else {
      setExchangeAllowed(
        tilesRemaining >= 7 || props.boardEditingMode === true,
      );
    }
  }, [
    gameContext.pool,
    props.currentRack,
    props.boardEditingMode,
    props.lexicon,
    setExchangeAllowed,
  ]);

  useEffect(() => {
    if (
      examinableGameContext.playState === PlayState.WAITING_FOR_FINAL_PASS &&
      isMyTurn
    ) {
      const finalAction = (
        <>
          Your opponent has played their final tiles. You must{" "}
          <span
            className="message-action"
            onClick={() => makeMove("pass")}
            role="button"
          >
            pass
          </span>{" "}
          or{" "}
          <span
            className="message-action"
            role="button"
            onClick={() => makeMove("challenge")}
          >
            challenge
          </span>
          .
        </>
      );

      message.info(
        {
          content: finalAction,
          className: "board-hud-message",
          key: "board-messages",
        },
        15,
      );
    }
  }, [examinableGameContext.playState, isMyTurn, makeMove, message]);

  useEffect(() => {
    if (!props.events.length) {
      return;
    }
    const evt = props.events[props.events.length - 1];
    const evtNickname = nicknameFromEvt(evt, props.playerMeta);
    if (evtNickname === props.username) {
      return;
    }
    let boardMessage = null;
    switch (evt.type) {
      case GameEvent_Type.PASS:
        boardMessage = `${evtNickname} passed`;
        break;
      case GameEvent_Type.EXCHANGE:
        boardMessage = `${evtNickname} exchanged ${
          evt.exchanged || evt.numTilesFromRack
        }`;
        break;
    }
    if (boardMessage && !props.puzzleMode) {
      message.info(
        {
          content: boardMessage,
          className: "board-hud-message",
          key: "board-messages",
        },
        3,
        undefined,
      );
    }
  }, [
    props.events,
    props.playerMeta,
    props.username,
    props.puzzleMode,
    message,
  ]);

  const numTurns = examinableGameContext.turns.length;

  useEffect(() => {
    // Set the current mode to "NORMAL" if we are editing the board,
    // and the user is moving around the analyzer. This prevents keeping
    // the rack editor or other modals open.
    if (props.boardEditingMode) {
      setCurrentMode("NORMAL");
    }
  }, [numTurns, props.boardEditingMode, setCurrentMode]);

  const squareClicked = useCallback(
    (row: number, col: number) => {
      if (board.letterAt(row, col) !== EmptyBoardSpaceMachineLetter) {
        // If there is a tile on this square, ignore the click.
        return;
      }
      setArrowProperties(nextArrowPropertyState(arrowProperties, row, col));
      handleUnsetHover?.();
    },
    [arrowProperties, board, handleUnsetHover, setArrowProperties],
  );

  const keydown = useCallback(
    (evt: React.KeyboardEvent) => {
      if (evt.ctrlKey || evt.altKey || evt.metaKey) {
        // Alt+3 should not challenge. Ignore Ctrl, Alt/Opt, and Win/Cmd.
        return;
      }
      let { key } = evt;
      // Neutralize caps lock to prevent accidental blank usage.
      if (key.length === 1) {
        if (!evt.shiftKey && key >= "A" && key <= "Z") {
          // Without shift, can only type lowercase.
          key = key.toLowerCase();
        } else if (evt.shiftKey && key >= "a" && key <= "z") {
          // With shift, can only type uppercase.
          key = key.toUpperCase();
        }
      }

      if (currentMode === "BLIND") {
        handleBlindfoldKeydown(evt, {
          key,
          blindfoldCommand,
          setBlindfoldCommand,
          blindfoldUseNPA,
          setBlindfoldUseNPA,
          isMyTurn,
          pool: gameContext.pool,
          players: gameContext.players,
          board: gameContext.board,
          turns: gameContext.turns,
          playState: examinableGameContext.playState,
          p0Time: examinableTimerContext.p0,
          p1Time: examinableTimerContext.p1,
          playerMeta,
          username,
          exchangeAllowed,
          setCurrentMode,
          makeMove,
          gameDone: props.gameDone,
          handleNeitherShortcut,
          setArrowProperties,
          nicknameFromEvt,
          currentRack: props.currentRack,
          alphabet: props.alphabet,
        });
      } else if (currentMode === "NORMAL") {
        if (
          key.toUpperCase() === ";" &&
          localStorage?.getItem("enableBlindfoldMode") === "true"
        ) {
          evt.preventDefault();
          if (handleNeitherShortcut.current) handleNeitherShortcut.current();
          setCurrentMode("BLIND");
          return;
        }
        if (isMyTurn && !props.gameDone) {
          if (key === "2") {
            evt.preventDefault();
            if (handlePassShortcut.current) handlePassShortcut.current();
            return;
          }
          if (key === "3") {
            evt.preventDefault();
            if (handleChallengeShortcut.current)
              handleChallengeShortcut.current();
            return;
          }
          if (key === "4" && exchangeAllowed) {
            evt.preventDefault();
            if (handleNeitherShortcut.current) handleNeitherShortcut.current();
            setCurrentMode("EXCHANGE_MODAL");
            return;
          }
          if (key === "$" && exchangeAllowed) {
            evt.preventDefault();
            makeMove("exchange", props.currentRack);
            return;
          }
        }
        if (key === "ArrowLeft" || key === "ArrowRight") {
          evt.preventDefault();
          setArrowProperties({
            ...arrowProperties,
            horizontal: !arrowProperties.horizontal,
          });
          return;
        }
        if (key === "ArrowDown") {
          evt.preventDefault();
          recallTiles();
          return;
        }
        if (key === "ArrowUp") {
          evt.preventDefault();
          shuffleTiles();
          return;
        }
        if (key === EnterKey) {
          evt.preventDefault();
          makeMove("commit");
          return;
        }
        if (key === "?") {
          return;
        }
        // This should return a new set of arrow properties, and also set
        // some state further up (the tiles layout with a "just played" type
        // marker)
        const handlerReturn = handleKeyPress(
          arrowProperties,
          props.board,
          key,
          displayedRack,
          placedTiles,
          props.alphabet,
          props.boardEditingMode,
          examinableGameContext.pool,
        );

        if (handlerReturn === null) {
          return;
        }
        evt.preventDefault();

        setDisplayedRack(handlerReturn.newDisplayedRack);
        setArrowProperties(handlerReturn.newArrow);
        setPlacedTiles(handlerReturn.newPlacedTiles);
        setPlacedTilesTempScore(handlerReturn.playScore);
      }
    },
    [
      arrowProperties,
      blindfoldCommand,
      blindfoldUseNPA,
      examinableGameContext.playState,
      examinableTimerContext.p0,
      examinableTimerContext.p1,
      examinableGameContext.pool,
      gameContext.pool,
      gameContext.board,
      gameContext.players,
      gameContext.turns,
      props.alphabet,
      props.boardEditingMode,
      playerMeta,
      username,
      currentMode,
      displayedRack,
      exchangeAllowed,
      setDisplayedRack,
      setPlacedTiles,
      setPlacedTilesTempScore,
      setBlindfoldCommand,
      setBlindfoldUseNPA,
      isMyTurn,
      makeMove,
      placedTiles,
      props.board,
      props.currentRack,
      props.gameDone,
      recallTiles,
      shuffleTiles,
      setCurrentMode,
      setArrowProperties,
      handlePassShortcut,
      handleNeitherShortcut,
      handleChallengeShortcut,
    ],
  );

  const handleTileDrop = useCallback(
    (row: number, col: number, rackIndex = -1, tileIndex = -1) => {
      const handlerReturn = handleDroppedTile(
        row,
        col,
        props.board,
        displayedRack,
        placedTiles,
        rackIndex,
        tileIndex,
        props.alphabet,
      );
      if (handlerReturn === null) {
        return;
      }
      setDisplayedRack(handlerReturn.newDisplayedRack);
      setPlacedTiles(handlerReturn.newPlacedTiles);
      setPlacedTilesTempScore(handlerReturn.playScore);
      setArrowProperties({ row: 0, col: 0, horizontal: false, show: false });
      if (handlerReturn.isUndesignated) {
        setCurrentMode("BLANK_MODAL");
      }
    },
    [
      displayedRack,
      placedTiles,
      props.alphabet,
      props.board,
      setDisplayedRack,
      setPlacedTilesTempScore,
      setPlacedTiles,
      setArrowProperties,
      setCurrentMode,
    ],
  );

  const clickToBoard = useCallback(
    (rackIndex: number) => {
      if (
        !arrowProperties.show ||
        arrowProperties.row >= props.board.dim ||
        arrowProperties.col >= props.board.dim
      ) {
        return null;
      }
      const handlerReturn = handleDroppedTile(
        arrowProperties.row,
        arrowProperties.col,
        props.board,
        displayedRack,
        placedTiles,
        rackIndex,
        uniqueTileIdx(arrowProperties.row, arrowProperties.col),
        props.alphabet,
      );
      if (handlerReturn === null) {
        return;
      }
      setDisplayedRack(handlerReturn.newDisplayedRack);
      setPlacedTiles(handlerReturn.newPlacedTiles);
      setPlacedTilesTempScore(handlerReturn.playScore);
      if (handlerReturn.isUndesignated) {
        setCurrentMode("BLANK_MODAL");
      }
      // Create an ephemeral tile map with unique keys.
      const ephTileMap: { [tileIdx: number]: EphemeralTile } = {};
      handlerReturn.newPlacedTiles.forEach((t) => {
        ephTileMap[uniqueTileIdx(t.row, t.col)] = t;
      });

      setArrowProperties(
        nextArrowStateAfterTilePlacement(
          arrowProperties,
          ephTileMap,
          1,
          props.board,
        ),
      );
    },
    [
      arrowProperties,
      displayedRack,
      placedTiles,
      props.alphabet,
      setDisplayedRack,
      setPlacedTiles,
      setPlacedTilesTempScore,
      props.board,
      setArrowProperties,
      setCurrentMode,
    ],
  );

  const handleBoardTileClick = useCallback(
    (ml: MachineLetter) => {
      if (ml === BlankMachineLetter) {
        setCurrentMode("BLANK_MODAL");
      }
    },
    [setCurrentMode],
  );

  const handleBlankSelection = useCallback(
    (letter: MachineLetter) => {
      const handlerReturn = designateBlank(
        props.board,
        placedTiles,
        displayedRack,
        letter,
        props.alphabet,
      );
      if (handlerReturn === null) {
        return;
      }
      setCurrentMode("NORMAL");
      setPlacedTiles(handlerReturn.newPlacedTiles);
      setPlacedTilesTempScore(handlerReturn.playScore);
    },
    [
      displayedRack,
      placedTiles,
      props.alphabet,
      props.board,
      setPlacedTiles,
      setPlacedTilesTempScore,
      setCurrentMode,
    ],
  );

  const handleBlankModalCancel = useCallback(() => {
    setCurrentMode("NORMAL");
  }, [setCurrentMode]);

  const returnToRack = useCallback(
    (rackIndex: number | undefined, tileIndex: number | undefined) => {
      const handlerReturn = returnTileToRack(
        props.board,
        displayedRack,
        placedTiles,
        props.alphabet,
        rackIndex,
        tileIndex,
      );
      if (handlerReturn === null) {
        return;
      }
      setDisplayedRack(handlerReturn.newDisplayedRack);
      setPlacedTiles(handlerReturn.newPlacedTiles);
      setPlacedTilesTempScore(handlerReturn.playScore);
      setArrowProperties({ row: 0, col: 0, horizontal: false, show: false });
    },
    [
      displayedRack,
      placedTiles,
      setPlacedTilesTempScore,
      setDisplayedRack,
      setPlacedTiles,
      props.alphabet,
      props.board,
      setArrowProperties,
    ],
  );

  const showExchangeModal = useCallback(() => {
    setCurrentMode("EXCHANGE_MODAL");
  }, [setCurrentMode]);

  const handleExchangeModalOk = useCallback(
    (exchangedTiles: Array<MachineLetter>) => {
      setCurrentMode("NORMAL");
      makeMove("exchange", exchangedTiles);
    },
    [makeMove, setCurrentMode],
  );

  const rematch = useCallback(() => {
    const evt = create(SeekRequestSchema);
    const receiver = create(MatchUserSchema);

    let opp = "";
    playerMeta.forEach((p) => {
      if (!(p.nickname === username)) {
        opp = p.nickname;
      }
    });

    if (observer) {
      return;
    }

    receiver.displayName = opp;
    evt.receivingUser = receiver;
    evt.receiverIsPermanent = true;
    evt.userState = SeekState.READY;

    evt.rematchFor = gameID;
    if (props.tournamentID) {
      evt.tournamentId = props.tournamentID;
    }
    sendSocketMsg(
      encodeToSocketFmt(
        MessageType.SEEK_REQUEST,
        toBinary(SeekRequestSchema, evt),
      ),
    );

    notification.info({
      message: "Rematch",
      description: `Sent rematch request to ${opp}`,
    });
  }, [
    observer,
    gameID,
    playerMeta,
    sendSocketMsg,
    username,
    props.tournamentID,
    notification,
  ]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<Element>) => {
      if (drawingCanBeEnabled) {
        // To activate a drawing hotkey, type 0, then the hotkey.
        if (currentMode === "NORMAL" || currentMode === "DRAWING_HOTKEY") {
          if (e.ctrlKey || e.altKey || e.metaKey) {
            // Do not prevent Ctrl+0/Cmd+0.
          } else {
            if (currentMode === "DRAWING_HOTKEY") {
              e.preventDefault();
              setCurrentMode("NORMAL");
              handleDrawingKeyDown(e);
              return;
            }
            if (e.key === "0") {
              e.preventDefault();
              setCurrentMode("DRAWING_HOTKEY");
              console.log(drawingConsoleInstructions);
              return;
            }
          }
        }
      }
      if (e.ctrlKey || e.altKey || e.metaKey) {
        // If a modifier key is held, never mind.
      } else {
        // prevent page from scrolling
        if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === " ") {
          e.preventDefault();
        }
      }
      keydown(e);
    },
    [
      currentMode,
      drawingCanBeEnabled,
      handleDrawingKeyDown,
      keydown,
      setCurrentMode,
    ],
  );

  // Removed auto-open rack editor - users can now type directly on the board
  // without setting a rack first in editor mode (rack inference handles it)
  useEffect(() => {
    // This used to auto-open the rack editor when the rack was empty,
    // but now we allow typing directly on the board in editor mode
    // The backend will infer the rack from the placed tiles
  }, [currentMode, props.boardEditingMode, props.currentRack, setCurrentMode]);

  useEffect(() => {
    if (
      currentMode === "WAITING_FOR_RACK_EDIT" &&
      props.currentRack.filter((v) => v !== EmptyRackSpaceMachineLetter)
        .length > 0
    ) {
      setCurrentMode("NORMAL");
    }
  }, [currentMode, props.currentRack, setCurrentMode]);

  // Just put this in onKeyPress to block all typeable keys so that typos from
  // placing a tile not on rack also do not trigger type-to-find on firefox.
  const preventFirefoxTypeToSearch = useCallback(
    (e: { preventDefault: () => void }) => {
      if (currentMode !== "EDITING_RACK") {
        e.preventDefault();
      }
    },
    [currentMode],
  );

  const metadataClient = useClient(GameMetadataService);

  const handlePass = useCallback(() => makeMove("pass"), [makeMove]);
  const handleResign = useCallback(() => makeMove("resign"), [makeMove]);
  const handleChallenge = useCallback(() => makeMove("challenge"), [makeMove]);
  const handleCommit = useCallback(() => makeMove("commit"), [makeMove]);
  const handleExportGCG = useCallback(
    () => gcgExport(props.gameID, props.playerMeta, metadataClient),
    [props.gameID, props.playerMeta, metadataClient],
  );
  const handleExchangeTilesCancel = useCallback(() => {
    setCurrentMode("NORMAL");
  }, [setCurrentMode]);

  const handleChallengeConfirm = useCallback(
    (selectedIndices: number[]) => {
      const moveEvt = challengeMoveEvent(gameID, selectedIndices);
      sendGameplayEvent(moveEvt);
      setChallengeModalVisible(false);
    },
    [gameID, sendGameplayEvent],
  );

  const handleChallengeCancel = useCallback(() => {
    setChallengeModalVisible(false);
  }, []);
  const handleRequestAbort = useCallback(() => {
    sendMetaEvent(GameMetaEvent_EventType.REQUEST_ABORT);
  }, [sendMetaEvent]);
  const handleNudge = useCallback(() => {
    sendMetaEvent(GameMetaEvent_EventType.REQUEST_ADJUDICATION);
  }, [sendMetaEvent]);
  const showAbort = useMemo(() => {
    // This hardcoded number is also on the backend.
    const isCorrespondence = props.gameMode === 1;
    if (isCorrespondence) return false;
    return !props.vsBot && gameContext.turns.length <= 7;
  }, [gameContext.turns, props.vsBot, props.gameMode]);
  const showNudge = useMemo(() => {
    // Only show nudge if this is not a tournament/club game and it's not our turn.
    const isCorrespondence = props.gameMode === GameMode.CORRESPONDENCE;
    if (isCorrespondence) return false;
    return !isMyTurn && !props.vsBot && props.tournamentID === "";
  }, [isMyTurn, props.tournamentID, props.vsBot, props.gameMode]);
  const isCorrespondenceGame = props.gameMode === GameMode.CORRESPONDENCE;
  const anonymousTourneyViewer =
    props.tournamentID && props.anonymousViewer && !props.gameDone;
  const correspondenceSpectator =
    isCorrespondenceGame && !props.gameDone && props.currentRack.length === 0;
  const nonDirectorAnalyzerDisallowed =
    props.tournamentNonDirectorObserver && props.tournamentPrivateAnalysis;
  const stillWaitingForGameToStart =
    props.currentRack.length === 0 &&
    !props.gameDone &&
    examinableGameContext.playState !== PlayState.WAITING_FOR_FINAL_PASS &&
    !props.boardEditingMode;
  let gameMetaMessage;
  if (examinableGameEndMessage) {
    gameMetaMessage = examinableGameEndMessage;
  } else if (correspondenceSpectator) {
    gameMetaMessage = "Tiles are hidden for correspondence game spectators";
  } else if (anonymousTourneyViewer) {
    gameMetaMessage = "Log in or register to see player tiles";
  } else if (stillWaitingForGameToStart) {
    gameMetaMessage = "Waiting for game to start...";
    if (gameContext.gameDocument?.uid) {
      gameMetaMessage = "Waiting for rack information...";
    }
  } else if (props.puzzleMode && props.anonymousViewer) {
    gameMetaMessage = "Log in or register to start solving puzzles";
  }

  // playerOrder enum seems to ensure we can only have two-player games :-(
  const myId = useMemo(() => {
    const myPlayerOrder = gameContext.nickToPlayerOrder[props.username];
    return myPlayerOrder === "p0" ? 0 : myPlayerOrder === "p1" ? 1 : null;
  }, [gameContext.nickToPlayerOrder, props.username]);

  const tileColorId =
    (props.gameDone ? null : myId) ?? examinableGameContext.onturn;

  const showControlsForGame = !anonymousTourneyViewer && !props.puzzleMode;
  const authedSolvingPuzzle = props.puzzleMode && !props.anonymousViewer;

  let gameControls = null;

  if (showControlsForGame || authedSolvingPuzzle || boardEditingMode) {
    gameControls = (
      <GameControls
        isExamining={isExamining}
        myTurn={authedSolvingPuzzle || boardEditingMode ? true : isMyTurn}
        finalPassOrChallenge={
          examinableGameContext.playState === PlayState.WAITING_FOR_FINAL_PASS
        }
        allowAnalysis={
          authedSolvingPuzzle
            ? props.puzzleSolved !== PuzzleStatus.UNANSWERED
            : nonDirectorAnalyzerDisallowed
              ? examinableGameContext.playState === PlayState.GAME_OVER
              : true
        }
        exchangeAllowed={exchangeAllowed}
        observer={authedSolvingPuzzle || boardEditingMode ? false : observer}
        onRecall={recallTiles}
        showExchangeModal={showExchangeModal}
        onPass={handlePass}
        onResign={handleResign}
        onRequestAbort={handleRequestAbort}
        onNudge={handleNudge}
        onChallenge={handleChallenge}
        onCommit={handleCommit}
        onRematch={props.handleAcceptRematch ?? rematch}
        onExamine={handleExamineStart}
        onExportGCG={handleExportGCG}
        showNudge={authedSolvingPuzzle ? false : showNudge}
        showAbort={authedSolvingPuzzle ? false : showAbort}
        showRematch={
          authedSolvingPuzzle ? false : examinableGameEndMessage !== ""
        }
        gameEndControls={
          authedSolvingPuzzle
            ? props.puzzleSolved !== PuzzleStatus.UNANSWERED
            : examinableGameEndMessage !== "" || props.gameDone
        }
        tournamentSlug={props.tournamentSlug}
        tournamentPairedMode={props.tournamentPairedMode}
        isLeagueGame={!!props.leagueID}
        lexicon={props.lexicon}
        challengeRule={props.challengeRule}
        setHandlePassShortcut={setHandlePassShortcut}
        setHandleChallengeShortcut={setHandleChallengeShortcut}
        setHandleNeitherShortcut={setHandleNeitherShortcut}
        exitableExaminer={props.exitableExaminer}
        puzzleMode={props.puzzleMode}
        boardEditingMode={props.boardEditingMode}
      />
    );
  }

  const gameBoard = (
    <div
      id="board-container"
      ref={boardContainer}
      className="board-container"
      onKeyDown={handleKeyDown}
      onKeyPress={preventFirefoxTypeToSearch}
      tabIndex={-1}
      role="textbox"
    >
      <GameBoard
        tileColorId={tileColorId}
        gridSize={props.board.dim}
        gridLayout={props.board.gridLayout}
        handleBoardTileClick={handleBoardTileClick}
        handleTileDrop={handleTileDrop}
        tilesLayout={props.board.letters}
        lastPlayedTiles={examinableGameContext.lastPlayedTiles}
        playerOfTileAt={examinableGameContext.playerOfTileAt}
        tentativeTiles={placedTiles}
        tentativeTileScore={placedTilesTempScore}
        squareClicked={squareClicked}
        placementArrowProperties={arrowProperties}
        handleSetHover={props.handleSetHover}
        handleUnsetHover={props.handleUnsetHover}
        definitionPopover={props.definitionPopover}
        alphabet={props.alphabet}
        recallOneTile={recallOneTile}
      />

      {gameMetaMessage && !props.puzzleMode ? (
        <GameMetaMessage message={gameMetaMessage} />
      ) : (
        <Affix offsetTop={126} className="rack-affix">
          <div className="rack-container">
            <Tooltip
              title="Reset Rack &darr;"
              placement="bottomRight"
              mouseEnterDelay={0.1}
              mouseLeaveDelay={0.01}
              color={colorPrimary}
            >
              <Button
                shape="circle"
                icon={<ArrowDownOutlined />}
                type="primary"
                onClick={recallTiles}
              />
            </Tooltip>
            {props.boardEditingMode && (
              <Tooltip
                title="Edit Rack"
                placement="bottomRight"
                mouseEnterDelay={0.1}
                mouseLeaveDelay={0.01}
                color={colorPrimary}
              >
                <Button
                  shape="circle"
                  icon={<EditOutlined />}
                  type="primary"
                  onClick={() => {
                    setCurrentMode("EDITING_RACK");
                  }}
                />
              </Tooltip>
            )}
            {currentMode === "EDITING_RACK" ? (
              <RackEditor
                currentRack={displayedRack}
                alphabet={props.alphabet}
                rackCallback={(rack: MachineWord) => {
                  if (props.changeCurrentRack) {
                    props.changeCurrentRack(
                      rack,
                      examinableGameContext.turns.length,
                    );
                  }
                  setCurrentMode("WAITING_FOR_RACK_EDIT");
                }}
                cancelCallback={() => setCurrentMode("NORMAL")}
              />
            ) : (
              <Rack
                tileColorId={tileColorId}
                letters={displayedRack}
                grabbable
                returnToRack={returnToRack}
                onTileClick={clickToBoard}
                moveRackTile={moveRackTile}
                alphabet={props.alphabet}
              />
            )}
            <Tooltip
              title="Shuffle &uarr;"
              placement="bottomLeft"
              mouseEnterDelay={0.1}
              mouseLeaveDelay={0.01}
              color={colorPrimary}
            >
              <Button
                shape="circle"
                icon={<SyncOutlined />}
                type="primary"
                onClick={shuffleTiles}
                autoFocus={true}
              />
            </Tooltip>
          </div>
        </Affix>
      )}
      {gameMetaMessage && props.puzzleMode && (
        <GameMetaMessage message={gameMetaMessage} />
      )}
      <TilePreview gridDim={props.board.dim} />
      {gameControls}
      <ExchangeTiles
        tileColorId={tileColorId}
        alphabet={props.alphabet}
        rack={props.currentRack}
        modalVisible={currentMode === "EXCHANGE_MODAL"}
        onOk={handleExchangeModalOk}
        onCancel={handleExchangeTilesCancel}
      />
      <ChallengeWordsModal
        wordsFormed={lastWordsFormed}
        onCancel={handleChallengeCancel}
        onConfirm={handleChallengeConfirm}
        modalVisible={challengeModalVisible}
        challengeRule={props.challengeRule}
      />
      <Modal
        className="blank-modal"
        title="Designate your blank"
        open={currentMode === "BLANK_MODAL"}
        onCancel={handleBlankModalCancel}
        width={360}
        footer={null}
      >
        <BlankSelector
          tileColorId={tileColorId}
          handleSelection={handleBlankSelection}
          alphabet={props.alphabet}
        />
      </Modal>
    </div>
  );
  return (
    <DndProvider backend={MultiBackend} options={MultiBackendOptions}>
      {gameBoard}
    </DndProvider>
  );
});
