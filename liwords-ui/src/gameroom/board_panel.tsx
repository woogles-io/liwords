import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import { TouchBackend } from 'react-dnd-touch-backend';
import { Button, notification, message, Tooltip, Affix } from 'antd';
import { Modal } from '../utils/focus_modal';
import { DndProvider } from 'react-dnd';
import {
  ArrowDownOutlined,
  EditOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import {
  isTouchDevice,
  uniqueTileIdx,
  EphemeralTile,
  EmptyRackSpaceMachineLetter,
  MachineWord,
  MachineLetter,
  EmptyBoardSpaceMachineLetter,
  BlankMachineLetter,
} from '../utils/cwgame/common';
import { useMountedState } from '../utils/mounted';

import GameBoard from './board';
import { DrawingHandlersSetterContext } from './drawing';
import GameControls from './game_controls';
import { Rack } from './rack';
import { ExchangeTiles } from './exchange_tiles';
import {
  nextArrowPropertyState,
  handleKeyPress,
  handleDroppedTile,
  handleTileDeletion,
  returnTileToRack,
  designateBlank,
  stableInsertRack,
  nextArrowStateAfterTilePlacement,
} from '../utils/cwgame/tile_placement';

import { say, wordToSayString } from '../utils/cwgame/blindfold';
import { singularCount } from '../utils/plural';

import {
  tilesetToMoveEvent,
  exchangeMoveEvent,
  passMoveEvent,
  resignMoveEvent,
  challengeMoveEvent,
  nicknameFromEvt,
} from '../utils/cwgame/game_event';
import { Board, parseCoordinates } from '../utils/cwgame/board';
import { encodeToSocketFmt } from '../utils/protobuf';
import {
  useExaminableGameContextStoreContext,
  useExaminableGameEndMessageStoreContext,
  useExaminableTimerStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useTentativeTileContext,
  useTimerStoreContext,
} from '../store/store';
import { sharedEnableAutoShuffle } from '../store/constants';
import { BlankSelector } from './blank_selector';
import { GameMetaMessage } from './game_meta_message';
import {
  ChallengeRule,
  GameEvent,
  GameEvent_Type,
  PlayState,
} from '../gen/api/proto/macondo/macondo_pb';
import { TilePreview } from './tile';
import {
  Alphabet,
  machineLetterToRune,
  machineWordToRunes,
  runesToMachineWord,
} from '../constants/alphabets';
import { MessageType } from '../gen/api/proto/ipc/ipc_pb';
import {
  MatchUser,
  SeekRequest,
  SeekState,
} from '../gen/api/proto/ipc/omgseeks_pb';
import {
  ClientGameplayEvent,
  GameMetaEvent,
  GameMetaEvent_EventType,
  PlayerInfo,
} from '../gen/api/proto/ipc/omgwords_pb';
import { PuzzleStatus } from '../gen/api/proto/puzzle_service/puzzle_service_pb';
import { flashError, useClient } from '../utils/hooks/connect';
import { GameMetadataService } from '../gen/api/proto/game_service/game_service_connectweb';
import { PromiseClient } from '@domino14/connect-web';
import { RackEditor } from './rack_editor';

// The frame atop is 24 height
// The frames on the sides are 24 in width, surrounded by a 14 pix gutter
const EnterKey = 'Enter';
import * as colors from '../base.scss';

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
  lexicon: string;
  alphabet: Alphabet;
  handleAcceptRematch: (() => void) | null;
  handleAcceptAbort: (() => void) | null;
  handleSetHover?: (
    x: number,
    y: number,
    words: Array<string> | undefined
  ) => void;
  handleUnsetHover?: () => void;
  definitionPopover?:
    | { x: number; y: number; content: React.ReactNode }
    | undefined;
  vsBot: boolean;
  exitableExaminer?: boolean;
  changeCurrentRack?: (rack: MachineWord, evtIdx: number) => void;
};

const shuffleLetters = (a: Array<MachineLetter>): Array<MachineLetter> => {
  const alistWithGaps = [...a];
  const alist = alistWithGaps.filter((x) => x !== EmptyRackSpaceMachineLetter);
  const n = alist.length;

  let somethingChanged = false;
  for (let i = n - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    if (alist[i] !== alist[j]) {
      somethingChanged = true;
      const tmp = alist[i];
      alist[i] = alist[j];
      alist[j] = tmp;
    }
  }

  if (!somethingChanged) {
    // Let's change something if possible.
    const j = Math.floor(Math.random() * n);
    const x = [];
    for (let i = 0; i < n; ++i) {
      if (alist[i] !== alist[j]) {
        x.push(i);
      }
    }

    if (x.length > 0) {
      const i = x[Math.floor(Math.random() * x.length)];
      const tmp = alist[i];
      alist[i] = alist[j];
      alist[j] = tmp;
    }
  }

  // Preserve the gaps.
  let r = 0;
  return alistWithGaps.map((x) =>
    x === EmptyRackSpaceMachineLetter ? x : alist[r++]
  );
};

const gcgExport = async (
  gameID: string,
  playerMeta: Array<PlayerInfo>,
  gameMetadataClient: PromiseClient<typeof GameMetadataService>
) => {
  try {
    const resp = await gameMetadataClient.getGCG({ gameId: gameID });
    const url = window.URL.createObjectURL(new Blob([resp.gcg]));
    const link = document.createElement('a');
    link.href = url;
    let downloadFilename = `${gameID}.gcg`;
    // TODO: allow more characters as applicable
    // Note: does not actively prevent saving .dotfiles or nul.something
    if (playerMeta.every((x) => /^[-0-9A-Za-z_.]+$/.test(x.nickname))) {
      const byStarts: Array<Array<string>> = [[], []];
      for (const x of playerMeta) {
        byStarts[+!!x.first].push(x.nickname);
      }
      downloadFilename = `${[...byStarts[1], ...byStarts[0]].join(
        '-'
      )}-${gameID}.gcg`;
    }
    link.setAttribute('download', downloadFilename);
    document.body.appendChild(link);
    link.onclick = () => {
      link.remove();
      setTimeout(() => {
        window.URL.revokeObjectURL(url);
      }, 1000);
    };
    link.click();
  } catch (e) {
    flashError(e);
  }
};

const backupKey = (letters: Array<MachineLetter>, rack: Array<MachineLetter>) =>
  JSON.stringify({ letters, rack });

export const BoardPanel = React.memo((props: Props) => {
  const { useState } = useMountedState();

  // Poka-yoke against accidentally having multiple modes active.
  const [currentMode, setCurrentMode] = useState<
    | 'BLANK_MODAL'
    | 'DRAWING_HOTKEY'
    | 'EXCHANGE_MODAL'
    | 'NORMAL'
    | 'BLIND'
    | 'EDITING_RACK'
    | 'WAITING_FOR_RACK_EDIT'
  >('NORMAL');

  const { drawingCanBeEnabled, handleKeyDown: handleDrawingKeyDown } =
    React.useContext(DrawingHandlersSetterContext);
  const [arrowProperties, setArrowProperties] = useState({
    row: 0,
    col: 0,
    horizontal: false,
    show: false,
  });

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { gameEndMessage: examinableGameEndMessage } =
    useExaminableGameEndMessageStoreContext();
  const { timerContext: examinableTimerContext } =
    useExaminableTimerStoreContext();

  const { isExamining, handleExamineStart } = useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();
  const { stopClock } = useTimerStoreContext();
  const [exchangeAllowed, setexchangeAllowed] = useState(true);
  const handlePassShortcut = useRef<(() => void) | null>(null);
  const setHandlePassShortcut = useCallback((x) => {
    handlePassShortcut.current =
      typeof x === 'function' ? x(handlePassShortcut.current) : x;
  }, []);
  const handleChallengeShortcut = useRef<(() => void) | null>(null);
  const setHandleChallengeShortcut = useCallback((x) => {
    handleChallengeShortcut.current =
      typeof x === 'function' ? x(handleChallengeShortcut.current) : x;
  }, []);
  const handleNeitherShortcut = useRef<(() => void) | null>(null);
  const setHandleNeitherShortcut = useCallback((x) => {
    handleNeitherShortcut.current =
      typeof x === 'function' ? x(handleNeitherShortcut.current) : x;
  }, []);
  const boardContainer = useRef<HTMLDivElement>(null);

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
  } = useTentativeTileContext();

  const observer = !props.playerMeta.some((p) => p.nickname === props.username);
  const isMyTurn = useMemo(() => {
    if (props.puzzleMode) {
      // it is always my turn in puzzle mode.
      return true;
    }
    if (props.boardEditingMode) {
      // it is also always "my" turn in board editing mode.
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

  const makeMove = useCallback(
    (move: string, addl?: Array<MachineLetter>) => {
      if (isExamining && !boardEditingMode) return;
      let moveEvt;
      if (move !== 'resign' && !isMyTurn) {
        console.log(
          'off turn move attempts',
          gameContext.nickToPlayerOrder,
          username,
          examinableGameContext.onturn
        );
        // It is not my turn. Ignore this event.
        message.warn({
          content: 'It is not your turn.',
          className: 'board-hud-message',
          key: 'board-messages',
          duration: 1.5,
        });
        return;
      }

      switch (move) {
        case 'exchange':
          if (addl) {
            moveEvt = exchangeMoveEvent(addl, gameID, gameContext.alphabet);
          }
          break;
        case 'pass':
          moveEvt = passMoveEvent(gameID);
          break;
        case 'resign':
          moveEvt = resignMoveEvent(gameID);
          break;
        case 'challenge':
          moveEvt = challengeMoveEvent(gameID);
          break;
        case 'commit':
          moveEvt = tilesetToMoveEvent(placedTiles, board, gameID);
          if (!moveEvt) {
            // this is an invalid play
            return;
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
      boardEditingMode,
      examinableGameContext.onturn,
      isExamining,
      isMyTurn,
      placedTiles,
      board,
      gameID,
      sendGameplayEvent,
      username,
    ]
  );

  const sendMetaEvent = useCallback(
    (evtType: GameMetaEvent_EventType) => {
      const metaEvt = new GameMetaEvent();
      metaEvt.type = evtType;
      metaEvt.gameId = gameID;

      sendSocketMsg(
        encodeToSocketFmt(MessageType.GAME_META_EVENT, metaEvt.toBinary())
      );
    },
    [sendSocketMsg, gameID]
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
    >()
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
        props.alphabet
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
    ]
  );

  const recallTiles = useCallback(() => {
    if (arrowProperties.show) {
      let { row, col } = arrowProperties;
      const { horizontal } = arrowProperties;
      const matchesLocation = ({
        row: tentativeRow,
        col: tentativeCol,
      }: {
        row: number;
        col: number;
      }) => row === tentativeRow && col === tentativeCol;
      if (
        horizontal &&
        row >= 0 &&
        row < props.board.dim &&
        col > 0 &&
        col <= props.board.dim
      ) {
        // Inefficient way to get around TypeScript restriction.
        const placedTilesArray = Array.from(placedTiles);
        let best = col;
        while (col > 0) {
          --col;
          if (
            props.board.letters[row * props.board.dim + col] !==
            EmptyBoardSpaceMachineLetter
          ) {
            // continue
          } else if (placedTilesArray.some(matchesLocation)) {
            best = col;
          } else {
            break;
          }
        }
        if (best !== arrowProperties.col) {
          setArrowProperties({ ...arrowProperties, col: best });
        }
      } else if (
        !horizontal &&
        col >= 0 &&
        col < props.board.dim &&
        row > 0 &&
        row <= props.board.dim
      ) {
        // Inefficient way to get around TypeScript restriction.
        const placedTilesArray = Array.from(placedTiles);
        let best = row;
        while (row > 0) {
          --row;
          if (
            props.board.letters[row * props.board.dim + col] !==
            EmptyBoardSpaceMachineLetter
          ) {
            // continue
          } else if (placedTilesArray.some(matchesLocation)) {
            best = row;
          } else {
            break;
          }
        }
        if (best !== arrowProperties.row) {
          setArrowProperties({ ...arrowProperties, row: best });
        }
      }
    }

    setPlacedTilesTempScore(0);
    setPlacedTiles(new Set<EphemeralTile>());
    setDisplayedRack(props.currentRack);
  }, [
    arrowProperties,
    placedTiles,
    props.board.dim,
    props.board.letters,
    props.currentRack,
    setPlacedTilesTempScore,
    setPlacedTiles,
    setDisplayedRack,
  ]);

  const shuffleTiles = useCallback(() => {
    setDisplayedRack(shuffleLetters(displayedRack));
  }, [setDisplayedRack, displayedRack]);

  const clearBackupRef = useRef<boolean>(false);
  const lastLettersRef = useRef<Array<MachineLetter>>();
  const lastRackRef = useRef<Array<MachineLetter>>();
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
  }>();
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
    let fullReset = false;
    const lastLetters = lastLettersRef.current;
    // XXX: please fix me:
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
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
      props.currentRack.length > 0 &&
      dep.displayedRack.length === 0 &&
      !dep.placedTiles.size
    ) {
      // First load after receiving rack.
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
        letters: Array<MachineLetter>
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
        dcol: number
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
      backupKey(props.board.letters, props.currentRack)
    );
    // Do not reset if considering a new placement move when challenging.
    if (fullReset || (bak && dep.placedTiles.size === 0)) {
      backupStatesRef.current.clear();
      if (!clearBackupRef.current) {
        const lastRack = lastRackRef.current;
        if (lastLetters && lastRack) {
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
    setexchangeAllowed(tilesRemaining >= 7 || props.boardEditingMode === true);
  }, [gameContext.pool, props.currentRack, props.boardEditingMode]);

  useEffect(() => {
    if (
      examinableGameContext.playState === PlayState.WAITING_FOR_FINAL_PASS &&
      isMyTurn
    ) {
      const finalAction = (
        <>
          Your opponent has played their final tiles. You must{' '}
          <span
            className="message-action"
            onClick={() => makeMove('pass')}
            role="button"
          >
            pass
          </span>{' '}
          or{' '}
          <span
            className="message-action"
            role="button"
            onClick={() => makeMove('challenge')}
          >
            challenge
          </span>
          .
        </>
      );

      message.info(
        {
          content: finalAction,
          className: 'board-hud-message',
          key: 'board-messages',
        },
        15
      );
    }
  }, [examinableGameContext.playState, isMyTurn, makeMove]);

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
          className: 'board-hud-message',
          key: 'board-messages',
        },
        3,
        undefined
      );
    }
  }, [props.events, props.playerMeta, props.username, props.puzzleMode]);

  const numTurns = examinableGameContext.turns.length;

  useEffect(() => {
    // Set the current mode to "NORMAL" if we are editing the board,
    // and the user is moving around the analyzer. This prevents keeping
    // the rack editor or other modals open.
    if (props.boardEditingMode) {
      setCurrentMode('NORMAL');
    }
  }, [numTurns, props.boardEditingMode]);

  const squareClicked = useCallback(
    (row: number, col: number) => {
      if (board.letterAt(row, col) !== EmptyBoardSpaceMachineLetter) {
        // If there is a tile on this square, ignore the click.
        return;
      }
      setArrowProperties(nextArrowPropertyState(arrowProperties, row, col));
      handleUnsetHover?.();
    },
    [arrowProperties, board, handleUnsetHover]
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
        if (!evt.shiftKey && key >= 'A' && key <= 'Z') {
          // Without shift, can only type lowercase.
          key = key.toLowerCase();
        } else if (evt.shiftKey && key >= 'a' && key <= 'z') {
          // With shift, can only type uppercase.
          key = key.toUpperCase();
        }
      }

      if (currentMode === 'BLIND') {
        const PlayerScoresAndTimes = (): [
          string,
          number,
          string,
          string,
          number,
          string
        ] => {
          const timepenalty = (time: number) => {
            // Calculate a timepenalty for speech purposes only. The backend will
            // also properly calculate this.

            if (time >= 0) {
              return 0;
            }

            const minsOvertime = Math.ceil(Math.abs(time) / 60000);
            return minsOvertime * 10;
          };

          let p0 = gameContext.players[0];
          let p1 = gameContext.players[1];

          let p0Time = examinableTimerContext.p0;
          let p1Time = examinableTimerContext.p1;

          if (props.playerMeta[0].userId === p1.userID) {
            [p0, p1] = [p1, p0];
            [p0Time, p1Time] = [p1Time, p0Time];
          }

          const playing =
            examinableGameContext.playState !== PlayState.GAME_OVER;
          const applyTimePenalty = !isExamining && playing;
          let p0Score = p0?.score ?? 0;
          if (applyTimePenalty) p0Score -= timepenalty(p0Time);
          let p1Score = p1?.score ?? 0;
          if (applyTimePenalty) p1Score -= timepenalty(p1Time);

          // Always list the player scores and times first
          if (props.playerMeta[1].nickname === props.username) {
            return [
              'you',
              p1Score,
              playerTimeToText(p1Time),
              'opponent',
              p0Score,
              playerTimeToText(p0Time),
            ];
          }
          return [
            'you',
            p0Score,
            playerTimeToText(p0Time),
            'opponent',
            p1Score,
            playerTimeToText(p1Time),
          ];
        };

        const sayGameEvent = (ge: GameEvent) => {
          const type = ge.type;
          let nickname = 'opponent.';
          const evtNickname = nicknameFromEvt(ge, props.playerMeta);
          if (evtNickname === props.username) {
            nickname = 'you.';
          }
          const playedTiles = ge.playedTiles;
          const mainWord = ge.wordsFormed[0];
          let blankAwareWord = '';
          for (let i = 0; i < playedTiles.length; i++) {
            const tile = playedTiles[i];
            if (tile >= 'a' && tile <= 'z') {
              blankAwareWord += tile;
            } else {
              blankAwareWord += mainWord[i];
            }
          }
          if (type === GameEvent_Type.TILE_PLACEMENT_MOVE) {
            say(
              nickname + ' ' + wordToSayString(ge.position, blindfoldUseNPA),
              wordToSayString(blankAwareWord, blindfoldUseNPA) +
                ' ' +
                ge.score.toString()
            );
          } else if (type === GameEvent_Type.PHONY_TILES_RETURNED) {
            say(nickname + ' lost challenge', '');
          } else if (type === GameEvent_Type.EXCHANGE) {
            say(nickname + ' exchanged ' + ge.exchanged, '');
          } else if (type === GameEvent_Type.PASS) {
            say(nickname + ' passed', '');
          } else if (type === GameEvent_Type.CHALLENGE) {
            say(nickname + ' challenged', '');
          } else if (type === GameEvent_Type.CHALLENGE_BONUS) {
            say(nickname + ' challenge bonus', '');
          } else {
            // This is a bum way to deal with all other events
            // but I am holding out for a better solution to saying events altogether
            say(nickname + ' 5 point challenge or outplay', '');
          }
        };

        const playerTimeToText = (ms: number): string => {
          const neg = ms < 0;
          // eslint-disable-next-line no-param-reassign
          const absms = Math.abs(ms);
          // const mins = Math.floor(ms / 60000);
          let totalSecs;
          if (!neg) {
            totalSecs = Math.ceil(absms / 1000);
          } else {
            totalSecs = Math.floor(absms / 1000);
          }
          const secs = totalSecs % 60;
          const mins = Math.floor(totalSecs / 60);

          let negative = '';
          if (neg) {
            negative = 'negative ';
          }
          let minutes = '';
          if (mins) {
            minutes = singularCount(mins, 'minute', 'minutes') + ' and ';
          }
          return negative + minutes + singularCount(secs, 'second', 'seconds');
        };

        let newBlindfoldCommand = blindfoldCommand;
        if (key === EnterKey) {
          // There is a better way to do this
          // This should be done like the Scorecards
          // are. It should access the Scorecard info somehow
          // but I is of the not knowing.
          if (blindfoldCommand.toUpperCase() === 'P') {
            if (gameContext.turns.length < 2) {
              say('no previous play', '');
            } else {
              sayGameEvent(gameContext.turns[gameContext.turns.length - 2]);
            }
          } else if (blindfoldCommand.toUpperCase() === 'C') {
            if (gameContext.turns.length < 1) {
              say('no current play', '');
            } else {
              sayGameEvent(gameContext.turns[gameContext.turns.length - 1]);
            }
          } else if (blindfoldCommand.toUpperCase() === 'S') {
            const [, p0Score, , , p1Score] = PlayerScoresAndTimes();
            const scoresay = `${p0Score} to ${p1Score}`;
            say(scoresay, '');
          } else if (
            blindfoldCommand.toUpperCase() === 'E' &&
            exchangeAllowed &&
            !props.gameDone
          ) {
            evt.preventDefault();
            if (handleNeitherShortcut.current) handleNeitherShortcut.current();
            setCurrentMode('EXCHANGE_MODAL');
            setBlindfoldCommand('');
            say('exchange modal opened', '');
            return;
          } else if (
            blindfoldCommand.toUpperCase() === 'PASS' &&
            !props.gameDone
          ) {
            makeMove('pass');
            setCurrentMode('NORMAL');
          } else if (
            blindfoldCommand.toUpperCase() === 'CHAL' &&
            !props.gameDone
          ) {
            makeMove('challenge');
            setCurrentMode('NORMAL');
            return;
          } else if (blindfoldCommand.toUpperCase() === 'T') {
            const [, , p0Time, , , p1Time] = PlayerScoresAndTimes();
            const timesay = `${p0Time} to ${p1Time}.`;
            say(timesay, '');
          } else if (blindfoldCommand.toUpperCase() === 'R') {
            say(
              wordToSayString(
                machineWordToRunes(props.currentRack, props.alphabet),
                blindfoldUseNPA
              ),
              ''
            );
          } else if (blindfoldCommand.toUpperCase() === 'B') {
            const bag = { ...gameContext.pool };
            for (let i = 0; i < props.currentRack.length; i += 1) {
              bag[props.currentRack[i]] -= 1;
            }
            let numTilesRemaining = 0;
            let tilesRemaining = '';
            let blankString = ' ';
            for (const [key, value] of Object.entries(bag)) {
              const letter =
                machineLetterToRune(parseInt(key, 10), props.alphabet) + '. ';
              if (value > 0) {
                numTilesRemaining += value;
                if (key === '0') {
                  blankString = `${value}, blank`;
                } else {
                  tilesRemaining += `${value}, ${letter}`;
                }
              }
            }
            say(
              `${numTilesRemaining} tiles unseen, ` +
                wordToSayString(tilesRemaining, blindfoldUseNPA) +
                blankString,
              ''
            );
          } else if (
            blindfoldCommand.charAt(0).toUpperCase() === 'B' &&
            blindfoldCommand.length === 2 &&
            blindfoldCommand.charAt(1).match(/[a-z.]/i)
          ) {
            const bag = { ...gameContext.pool };
            for (let i = 0; i < props.currentRack.length; i += 1) {
              bag[props.currentRack[i]] -= 1;
            }
            let tile = blindfoldCommand.charAt(1).toUpperCase();
            try {
              const letter = runesToMachineWord(tile, props.alphabet)[0];
              let numTiles = bag[letter];
              if (tile === '.') {
                tile = '?';
                numTiles = bag[letter];
                say(`${numTiles}, blank`, '');
              } else {
                say(
                  wordToSayString(`${numTiles}, ${tile}`, blindfoldUseNPA),
                  ''
                );
              }
            } catch (e) {
              // do nothing.
            }
          } else if (blindfoldCommand.toUpperCase() === 'N') {
            setBlindfoldUseNPA(!blindfoldUseNPA);
            say(
              'NATO Phonetic Alphabet is ' +
                (!blindfoldUseNPA ? ' enabled.' : ' disabled.'),
              ''
            );
          } else if (blindfoldCommand.toUpperCase() === 'W') {
            if (isMyTurn) {
              say('It is your turn.', '');
            } else {
              say("It is your opponent's turn", '');
            }
          } else if (blindfoldCommand.toUpperCase() === 'L') {
            say(
              'B for bag. C for current play. ' +
                'E for exchange. N for NATO pronunciations. ' +
                'P for the previous play. R for rack. ' +
                'S for score. T for time. W for turn. ' +
                'P, A, S, S, for pass. C, H, A, L, for challenge.',
              ''
            );
          } else {
            const blindfoldCoordinates = parseCoordinates(blindfoldCommand);
            if (blindfoldCoordinates !== undefined) {
              // Valid coordinates, place the arrow
              say(wordToSayString(blindfoldCommand, blindfoldUseNPA), '');
              const board = gameContext.board;
              const existingTile = board.letterAt(
                blindfoldCoordinates.row,
                blindfoldCoordinates.col
              );
              if (existingTile === EmptyBoardSpaceMachineLetter) {
                setArrowProperties({
                  row: blindfoldCoordinates.row,
                  col: blindfoldCoordinates.col,
                  horizontal: blindfoldCoordinates.horizontal,
                  show: true,
                });
              }
            } else {
              console.log('invalid command: ', blindfoldCommand);
              say('invalid command', '');
            }
          }

          newBlindfoldCommand = '';
          setCurrentMode('NORMAL');
        } else {
          newBlindfoldCommand = blindfoldCommand + key.toUpperCase();
        }
        setBlindfoldCommand(newBlindfoldCommand);
      } else if (currentMode === 'NORMAL') {
        if (
          key.toUpperCase() === ';' &&
          localStorage?.getItem('enableBlindfoldMode') === 'true'
        ) {
          evt.preventDefault();
          if (handleNeitherShortcut.current) handleNeitherShortcut.current();
          setCurrentMode('BLIND');
          return;
        }
        if (isMyTurn && !props.gameDone) {
          if (key === '2') {
            evt.preventDefault();
            if (handlePassShortcut.current) handlePassShortcut.current();
            return;
          }
          if (key === '3') {
            evt.preventDefault();
            if (handleChallengeShortcut.current)
              handleChallengeShortcut.current();
            return;
          }
          if (key === '4' && exchangeAllowed) {
            evt.preventDefault();
            if (handleNeitherShortcut.current) handleNeitherShortcut.current();
            setCurrentMode('EXCHANGE_MODAL');
            return;
          }
          if (key === '$' && exchangeAllowed) {
            evt.preventDefault();
            makeMove('exchange', props.currentRack);
            return;
          }
        }
        if (key === 'ArrowLeft' || key === 'ArrowRight') {
          evt.preventDefault();
          setArrowProperties({
            ...arrowProperties,
            horizontal: !arrowProperties.horizontal,
          });
          return;
        }
        if (key === 'ArrowDown') {
          evt.preventDefault();
          recallTiles();
          return;
        }
        if (key === 'ArrowUp') {
          evt.preventDefault();
          shuffleTiles();
          return;
        }
        if (key === EnterKey) {
          evt.preventDefault();
          makeMove('commit');
          return;
        }
        if (key === '?') {
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
          props.alphabet
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
      gameContext.pool,
      gameContext.board,
      examinableGameContext.playState,
      examinableTimerContext.p0,
      examinableTimerContext.p1,
      gameContext.players,
      gameContext.turns,
      isExamining,
      props.alphabet,
      props.playerMeta,
      props.username,
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
    ]
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
        props.alphabet
      );
      if (handlerReturn === null) {
        return;
      }
      setDisplayedRack(handlerReturn.newDisplayedRack);
      setPlacedTiles(handlerReturn.newPlacedTiles);
      setPlacedTilesTempScore(handlerReturn.playScore);
      setArrowProperties({ row: 0, col: 0, horizontal: false, show: false });
      if (handlerReturn.isUndesignated) {
        setCurrentMode('BLANK_MODAL');
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
    ]
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
        props.alphabet
      );
      if (handlerReturn === null) {
        return;
      }
      setDisplayedRack(handlerReturn.newDisplayedRack);
      setPlacedTiles(handlerReturn.newPlacedTiles);
      setPlacedTilesTempScore(handlerReturn.playScore);
      if (handlerReturn.isUndesignated) {
        setCurrentMode('BLANK_MODAL');
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
          props.board
        )
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
    ]
  );

  const handleBoardTileClick = useCallback((ml: MachineLetter) => {
    if (ml === BlankMachineLetter) {
      setCurrentMode('BLANK_MODAL');
    }
  }, []);

  const handleBlankSelection = useCallback(
    (letter: MachineLetter) => {
      const handlerReturn = designateBlank(
        props.board,
        placedTiles,
        displayedRack,
        letter,
        props.alphabet
      );
      if (handlerReturn === null) {
        return;
      }
      setCurrentMode('NORMAL');
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
    ]
  );

  const handleBlankModalCancel = useCallback(() => {
    setCurrentMode('NORMAL');
  }, []);

  const returnToRack = useCallback(
    (rackIndex: number | undefined, tileIndex: number | undefined) => {
      const handlerReturn = returnTileToRack(
        props.board,
        displayedRack,
        placedTiles,
        props.alphabet,
        rackIndex,
        tileIndex
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
    ]
  );

  const moveRackTile = useCallback(
    (newIndex: number | undefined, oldIndex: number | undefined) => {
      if (typeof newIndex === 'number' && typeof oldIndex === 'number') {
        const leftIndex = Math.min(oldIndex, newIndex);
        const rightIndex = Math.max(oldIndex, newIndex) + 1;
        // Within only the affected area, replace oldIndex with empty,
        // and then insert that removed tile at the desired place.
        setDisplayedRack(
          displayedRack
            .slice(0, leftIndex)
            .concat(
              stableInsertRack(
                displayedRack
                  .slice(leftIndex, oldIndex)
                  .concat(EmptyRackSpaceMachineLetter)
                  .concat(displayedRack.slice(oldIndex + 1, rightIndex)),
                newIndex - leftIndex,
                displayedRack[oldIndex]
              )
            )
            .concat(displayedRack.slice(rightIndex))
        );
      }
    },
    [displayedRack, setDisplayedRack]
  );

  const showExchangeModal = useCallback(() => {
    setCurrentMode('EXCHANGE_MODAL');
  }, []);

  const handleExchangeModalOk = useCallback(
    (exchangedTiles: Array<MachineLetter>) => {
      setCurrentMode('NORMAL');
      makeMove('exchange', exchangedTiles);
    },
    [makeMove]
  );

  const rematch = useCallback(() => {
    const evt = new SeekRequest();
    const receiver = new MatchUser();

    let opp = '';
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
    sendSocketMsg(encodeToSocketFmt(MessageType.SEEK_REQUEST, evt.toBinary()));

    notification.info({
      message: 'Rematch',
      description: `Sent rematch request to ${opp}`,
    });
  }, [
    observer,
    gameID,
    playerMeta,
    sendSocketMsg,
    username,
    props.tournamentID,
  ]);

  const handleKeyDown = useCallback(
    (e) => {
      if (drawingCanBeEnabled) {
        // To activate a drawing hotkey, type 0, then the hotkey.
        if (currentMode === 'NORMAL' || currentMode === 'DRAWING_HOTKEY') {
          if (e.ctrlKey || e.altKey || e.metaKey) {
            // Do not prevent Ctrl+0/Cmd+0.
          } else {
            if (currentMode === 'DRAWING_HOTKEY') {
              e.preventDefault();
              setCurrentMode('NORMAL');
              handleDrawingKeyDown(e);
              return;
            }
            if (e.key === '0') {
              e.preventDefault();
              setCurrentMode('DRAWING_HOTKEY');
              console.log(
                'You pressed 0. Now press one of these keys:' +
                  '\n0 = Toggle drawing' +
                  '\nU = Undo' +
                  '\nW = Wipe' +
                  '\nF = Freehand mode' +
                  '\nL = Line mode' +
                  '\nA = Arrow mode' +
                  '\nQ = Quadrangle mode' +
                  '\nC = Circle mode' +
                  '\nS = Snap (does not affect freehand)' +
                  '\nD = Do not snap' +
                  '\nR = Red pen' +
                  '\nG = Green pen' +
                  '\nB = Blue pen' +
                  '\nY = Yellow pen' +
                  '\nE = Eraser'
              );
              return;
            }
          }
        }
      }
      if (e.ctrlKey || e.altKey || e.metaKey) {
        // If a modifier key is held, never mind.
      } else {
        // prevent page from scrolling
        if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === ' ') {
          e.preventDefault();
        }
      }
      keydown(e);
    },
    [currentMode, drawingCanBeEnabled, handleDrawingKeyDown, keydown]
  );

  useEffect(() => {
    if (
      currentMode !== 'EDITING_RACK' &&
      currentMode !== 'WAITING_FOR_RACK_EDIT' &&
      props.boardEditingMode &&
      props.currentRack.filter((v) => v !== EmptyRackSpaceMachineLetter)
        .length === 0
    ) {
      setCurrentMode('EDITING_RACK');
    }
  }, [currentMode, props.boardEditingMode, props.currentRack]);

  useEffect(() => {
    if (
      currentMode === 'WAITING_FOR_RACK_EDIT' &&
      props.currentRack.filter((v) => v !== EmptyRackSpaceMachineLetter)
        .length > 0
    ) {
      setCurrentMode('NORMAL');
    }
  }, [currentMode, props.currentRack]);

  // Just put this in onKeyPress to block all typeable keys so that typos from
  // placing a tile not on rack also do not trigger type-to-find on firefox.
  const preventFirefoxTypeToSearch = useCallback(
    (e) => {
      if (currentMode !== 'EDITING_RACK') {
        e.preventDefault();
      }
    },
    [currentMode]
  );

  const metadataClient = useClient(GameMetadataService);

  const handlePass = useCallback(() => makeMove('pass'), [makeMove]);
  const handleResign = useCallback(() => makeMove('resign'), [makeMove]);
  const handleChallenge = useCallback(() => makeMove('challenge'), [makeMove]);
  const handleCommit = useCallback(() => makeMove('commit'), [makeMove]);
  const handleExportGCG = useCallback(
    () => gcgExport(props.gameID, props.playerMeta, metadataClient),
    [props.gameID, props.playerMeta, metadataClient]
  );
  const handleExchangeTilesCancel = useCallback(() => {
    setCurrentMode('NORMAL');
  }, []);
  const handleRequestAbort = useCallback(() => {
    sendMetaEvent(GameMetaEvent_EventType.REQUEST_ABORT);
  }, [sendMetaEvent]);
  const handleNudge = useCallback(() => {
    sendMetaEvent(GameMetaEvent_EventType.REQUEST_ADJUDICATION);
  }, [sendMetaEvent]);
  const showAbort = useMemo(() => {
    // This hardcoded number is also on the backend.
    return !props.vsBot && gameContext.turns.length <= 7;
  }, [gameContext.turns, props.vsBot]);
  const showNudge = useMemo(() => {
    // Only show nudge if this is not a tournament/club game and it's not our turn.
    return !isMyTurn && !props.vsBot && props.tournamentID === '';
  }, [isMyTurn, props.tournamentID, props.vsBot]);
  const anonymousTourneyViewer =
    props.tournamentID && props.anonymousViewer && !props.gameDone;
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
  } else if (anonymousTourneyViewer) {
    gameMetaMessage = 'Log in or register to see player tiles';
  } else if (stillWaitingForGameToStart) {
    gameMetaMessage = 'Waiting for game to start...';
    if (gameContext.gameDocument?.uid) {
      gameMetaMessage = 'Waiting for rack information...';
    }
  } else if (props.puzzleMode && props.anonymousViewer) {
    gameMetaMessage = 'Log in or register to start solving puzzles';
  }

  // playerOrder enum seems to ensure we can only have two-player games :-(
  const myId = useMemo(() => {
    const myPlayerOrder = gameContext.nickToPlayerOrder[props.username];
    // eslint-disable-next-line no-nested-ternary
    return myPlayerOrder === 'p0' ? 0 : myPlayerOrder === 'p1' ? 1 : null;
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
          authedSolvingPuzzle ? false : examinableGameEndMessage !== ''
        }
        gameEndControls={
          authedSolvingPuzzle
            ? props.puzzleSolved !== PuzzleStatus.UNANSWERED
            : examinableGameEndMessage !== '' || props.gameDone
        }
        tournamentSlug={props.tournamentSlug}
        tournamentPairedMode={props.tournamentPairedMode}
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
  if (authedSolvingPuzzle) {
    gameControls = (
      <Affix offsetTop={126} className="rack-affix">
        {gameControls}
      </Affix>
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
              color={colors.default.colorPrimary}
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
                color={colors.default.colorPrimary}
              >
                <Button
                  shape="circle"
                  icon={<EditOutlined />}
                  type="primary"
                  onClick={() => {
                    setCurrentMode('EDITING_RACK');
                  }}
                />
              </Tooltip>
            )}
            {currentMode === 'EDITING_RACK' ? (
              <RackEditor
                currentRack={displayedRack}
                alphabet={props.alphabet}
                rackCallback={(rack: MachineWord) => {
                  if (props.changeCurrentRack) {
                    props.changeCurrentRack(
                      rack,
                      examinableGameContext.turns.length
                    );
                  }
                  setCurrentMode('WAITING_FOR_RACK_EDIT');
                }}
                cancelCallback={() => setCurrentMode('NORMAL')}
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
              color={colors.default.colorPrimary}
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
      {isTouchDevice() ? <TilePreview gridDim={props.board.dim} /> : null}
      {gameControls}
      <ExchangeTiles
        tileColorId={tileColorId}
        alphabet={props.alphabet}
        rack={props.currentRack}
        modalVisible={currentMode === 'EXCHANGE_MODAL'}
        onOk={handleExchangeModalOk}
        onCancel={handleExchangeTilesCancel}
      />
      <Modal
        className="blank-modal"
        title="Designate your blank"
        open={currentMode === 'BLANK_MODAL'}
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
  return <DndProvider backend={TouchBackend}>{gameBoard}</DndProvider>;
});
