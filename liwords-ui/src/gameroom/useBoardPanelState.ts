import { useState, useRef, useCallback, useMemo, useContext } from "react";
import { DrawingHandlersSetterContext } from "./drawing";
import {
  useExaminableGameContextStoreContext,
  useExaminableGameEndMessageStoreContext,
  useExaminableTimerStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useTentativeTileContext,
  useTimerStoreContext,
} from "../store/store";
import { MachineLetter } from "../utils/cwgame/common";
import { Board } from "../utils/cwgame/board";
import { PlayerInfo } from "../gen/api/proto/ipc/omgwords_pb";

type UseBoardPanelStateProps = {
  playerMeta: Array<PlayerInfo>;
  username: string;
  board: Board;
  currentRack: Array<MachineLetter>;
  puzzleMode?: boolean;
  boardEditingMode?: boolean;
};

export function useBoardPanelState(props: UseBoardPanelStateProps) {
  // Modes
  const [currentMode, setCurrentMode] = useState<
    | "BLANK_MODAL"
    | "DRAWING_HOTKEY"
    | "EXCHANGE_MODAL"
    | "NORMAL"
    | "BLIND"
    | "EDITING_RACK"
    | "WAITING_FOR_RACK_EDIT"
  >("NORMAL");

  // Drawing context
  const { drawingCanBeEnabled, handleKeyDown: handleDrawingKeyDown } =
    useContext(DrawingHandlersSetterContext);

  // Arrow state
  const [arrowProperties, setArrowProperties] = useState({
    row: 0,
    col: 0,
    horizontal: false,
    show: false,
  });

  // Store/context hooks
  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { gameEndMessage: examinableGameEndMessage } =
    useExaminableGameEndMessageStoreContext();
  const { timerContext: examinableTimerContext } =
    useExaminableTimerStoreContext();

  const { isExamining, handleExamineStart } = useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();
  const { stopClock } = useTimerStoreContext();

  // Exchange allowed
  const [exchangeAllowed, setExchangeAllowed] = useState(true);

  // Shortcuts
  const handlePassShortcut = useRef<(() => void) | null>(null);
  const setHandlePassShortcut = useCallback(
    (
      makeNewValue:
        | ((oldValue: (() => void) | null) => (() => void) | null)
        | null,
    ): void => {
      handlePassShortcut.current =
        typeof makeNewValue === "function"
          ? makeNewValue(handlePassShortcut.current)
          : makeNewValue;
    },
    [],
  );
  const handleChallengeShortcut = useRef<(() => void) | null>(null);
  const setHandleChallengeShortcut = useCallback(
    (
      makeNewValue:
        | ((oldValue: (() => void) | null) => (() => void) | null)
        | null,
    ): void => {
      handleChallengeShortcut.current =
        typeof makeNewValue === "function"
          ? makeNewValue(handleChallengeShortcut.current)
          : makeNewValue;
    },
    [],
  );
  const handleNeitherShortcut = useRef<(() => void) | null>(null);
  const setHandleNeitherShortcut = useCallback(
    (
      makeNewValue:
        | ((oldValue: (() => void) | null) => (() => void) | null)
        | null,
    ): void => {
      handleNeitherShortcut.current =
        typeof makeNewValue === "function"
          ? makeNewValue(handleNeitherShortcut.current)
          : makeNewValue;
    },
    [],
  );
  const boardContainer = useRef<HTMLDivElement>(null);

  // Tentative tile context
  const {
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
  } = useTentativeTileContext();

  // Observer and turn logic
  const observer = !props.playerMeta.some((p) => p.nickname === props.username);
  const isMyTurn = useMemo(() => {
    if (props.puzzleMode) {
      return true;
    }
    if (props.boardEditingMode) {
      return true;
    }
    const iam = gameContext.nickToPlayerOrder[props.username];
    return iam && iam === `p${examinableGameContext.onturn}`;
  }, [
    gameContext.nickToPlayerOrder,
    props.username,
    props.boardEditingMode,
    examinableGameContext.onturn,
    props.puzzleMode,
  ]);

  return {
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
  };
}
