import React, { useCallback, useEffect, useLayoutEffect, useRef } from "react";

import BoardSpaces from "./board_spaces";
import { useDrawing } from "./drawing";
import { useExamineStoreContext } from "../store/store";
import { useLocalStorageBool } from "../utils/use_local_storage";
import { PlacementArrow } from "../utils/cwgame/tile_placement";
import BoardCoordLabels from "./board_coord_labels";
import Tiles from "./tiles";
import {
  EphemeralTile,
  MachineLetter,
  PlayedTiles,
  PlayerOfTiles,
} from "../utils/cwgame/common";
import { Alphabet } from "../constants/alphabets";
import { LearnOverlay } from "../learn/learn_overlay";
import { SuperCrosswordGameGridLayout } from "../constants/board_layout";

type Props = {
  tileColorId: number;
  gridLayout: Array<string>;
  gridSize: number;
  handleBoardTileClick: (letter: MachineLetter) => void;
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined,
  ) => void;
  tilesLayout: Array<MachineLetter>;
  alphabet: Alphabet;
  lastPlayedTiles: PlayedTiles;
  playerOfTileAt: PlayerOfTiles;
  squareClicked: (row: number, col: number) => void;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
  placementArrowProperties: PlacementArrow;
  handleSetHover?: (
    x: number,
    y: number,
    words: Array<string> | undefined,
  ) => void;
  handleUnsetHover?: () => void;
  definitionPopover?:
    | { x: number; y: number; content: React.ReactNode }
    | undefined;
  recallOneTile?: (row: number, col: number) => void;
  // Rotate the board 180 degrees, as if turned to face the opponent.
  flipped?: boolean;
  // Spin into a new orientation rather than jump to it.
  animateFlip?: boolean;
  // Turn the board around by hand; enables dragging the coordinate labels.
  onFlipBoard?: () => void;
};

// The board's angle right now, in degrees from 0 up to 360, as drawn
// (mid-spin included).
const currentAngle = (el: HTMLElement) => {
  const t = getComputedStyle(el).transform;
  if (!t || t === "none") return 0;
  // matrix(a, b, ...) or matrix3d(a, b, ...): a = cos, b = sin.
  const [a, b] = t
    .slice(t.indexOf("(") + 1)
    .split(",")
    .map(parseFloat);
  const deg = (Math.atan2(b, a) * 180) / Math.PI;
  return (deg + 360) % 360;
};

const pointerAngle = (e: React.PointerEvent, cx: number, cy: number) =>
  (Math.atan2(e.clientY - cy, e.clientX - cx) * 180) / Math.PI;

// The same turn, as the shortest way round: more than -180 up to 180.
const shortestTurn = (deg: number) => {
  const d = ((deg % 360) + 360) % 360;
  return d > 180 ? d - 360 : d;
};

const TURNING_Z = "10";

// Spin from an angle to the resting angle of a side: clockwise unless
// settling after a drag, which takes the shorter way. Speed is that of
// a 0.25s half-turn, so a short settle is quick.
const spinBoard = (
  el: HTMLElement,
  spinRef: React.MutableRefObject<Animation | null>,
  from: number,
  flipped: boolean,
  settle: boolean,
) => {
  spinRef.current?.cancel();
  spinRef.current = null;
  const rest = flipped ? 180 : 0;
  const offset = (((rest - from) % 360) + 360) % 360;
  const turn = settle ? shortestTurn(offset) : offset;
  if (
    turn === 0 ||
    typeof el.animate !== "function" ||
    window.matchMedia?.("(prefers-reduced-motion: reduce)").matches
  ) {
    return;
  }
  // While turning, the corners reach past the board's square; keep them
  // above what is around it (but under popups and menus).
  spinRef.current = el.animate(
    [
      { transform: `rotate(${from}deg)`, zIndex: TURNING_Z },
      { transform: `rotate(${from + turn}deg)`, zIndex: TURNING_Z },
    ],
    {
      duration: (250 * Math.abs(turn)) / 180,
      easing: "ease-out",
    },
  );
};

const Board = React.memo((props: Props) => {
  // Keep frames the same size, and shrink or grow the
  // board squares as necessary.

  const { outerDivProps, svgDrawing, isDrawingEnabled } = useDrawing(
    props.gridSize,
  );
  const { isExamining } = useExamineStoreContext();

  // Secret setting: when enabled, premium-square labels (2x letter, 3x word,
  // etc.) are always shown instead of only on hover. The gameroom SCSS keys
  // off the "show-bonus-labels" body class.
  const [showBonusLabels] = useLocalStorageBool("showBonusLabels");
  useEffect(() => {
    if (!showBonusLabels) return;
    document.body.classList.add("show-bonus-labels");
    return () => {
      document.body.classList.remove("show-bonus-labels");
    };
  }, [showBonusLabels]);

  let zomgClass = "";
  if (props.gridSize === SuperCrosswordGameGridLayout.length) {
    zomgClass = " zomgboard";
  }
  const flipped = !!props.flipped;
  const flippedClass = flipped ? " board-flipped" : "";

  // Each flip spins clockwise from wherever the board is now: from rest
  // that is one half-turn, and a flip that arrives mid-spin carries on
  // round to the new side instead of jumping back to a starting angle.
  // A layout effect, so the new resting angle is never painted first.
  const boardRef = useRef<HTMLDivElement>(null);
  const spinRef = useRef<Animation | null>(null);
  // Set when a drag lets go past the halfway point: that spin starts from
  // where the drag left the board and takes the shorter way.
  const settleFromRef = useRef<number | null>(null);
  const lastFlippedRef = useRef(flipped);
  // Only the squares turn; the coordinate labels stay where they are,
  // upright, and are numbered the other way round instead.
  const squares = () =>
    boardRef.current?.querySelector<HTMLElement>(".board-spaces-container") ??
    null;
  useLayoutEffect(() => {
    if (lastFlippedRef.current === flipped) return;
    lastFlippedRef.current = flipped;
    const el = squares();
    const settleFrom = settleFromRef.current;
    settleFromRef.current = null;
    if (!el) return;
    if (settleFrom !== null) {
      spinBoard(el, spinRef, settleFrom, flipped, true);
      return;
    }
    const spinning = spinRef.current?.playState === "running";
    if (!props.animateFlip) {
      spinRef.current?.cancel();
      return;
    }
    const from = spinning ? currentAngle(el) : flipped ? 0 : 180;
    spinBoard(el, spinRef, from, flipped, false);
  }, [flipped, props.animateFlip]);

  // Dragging the coordinate labels turns the board by hand; letting go
  // settles it upright or upside down, whichever is nearer.
  const { onFlipBoard } = props;
  const turnable = !!onFlipBoard && !isDrawingEnabled;
  const dragRef = useRef<{
    pointerId: number;
    centerX: number;
    centerY: number;
    lastPointer: number;
    angle: number;
    startX: number;
    startY: number;
    startTime: number;
  } | null>(null);
  // A double tap on the labels flips the board, like the 6 key.
  const lastTapRef = useRef(0);
  const onPointerDown = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      const board = boardRef.current;
      const el = squares();
      if (!turnable || !board || !el || e.button !== 0) return;
      if (!(e.target as Element).closest(".coord-labels")) return;
      e.preventDefault();
      const spinning = spinRef.current?.playState === "running";
      const angle = spinning ? currentAngle(el) : flipped ? 180 : 0;
      spinRef.current?.cancel();
      // The squares turn about their own middle, so it stays put mid-spin.
      const rect = el.getBoundingClientRect();
      const centerX = rect.left + rect.width / 2;
      const centerY = rect.top + rect.height / 2;
      dragRef.current = {
        pointerId: e.pointerId,
        centerX,
        centerY,
        lastPointer: pointerAngle(e, centerX, centerY),
        angle,
        startX: e.clientX,
        startY: e.clientY,
        startTime: e.timeStamp,
      };
      board.setPointerCapture?.(e.pointerId);
      el.style.transform = `rotate(${angle}deg)`;
      el.style.zIndex = TURNING_Z;
    },
    [turnable, flipped],
  );
  const onPointerMove = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current;
    const el = squares();
    if (!drag || !el || e.pointerId !== drag.pointerId) return;
    const p = pointerAngle(e, drag.centerX, drag.centerY);
    drag.angle += shortestTurn(p - drag.lastPointer);
    drag.lastPointer = p;
    el.style.transform = `rotate(${drag.angle}deg)`;
  }, []);
  const onPointerUp = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      const drag = dragRef.current;
      const el = squares();
      if (!drag || !el || e.pointerId !== drag.pointerId) return;
      dragRef.current = null;
      el.style.transform = "";
      el.style.zIndex = "";
      const tap =
        Math.hypot(e.clientX - drag.startX, e.clientY - drag.startY) < 8 &&
        e.timeStamp - drag.startTime < 300;
      if (tap) {
        spinBoard(el, spinRef, drag.angle, flipped, true);
        if (e.timeStamp - lastTapRef.current < 350 && onFlipBoard) {
          lastTapRef.current = 0;
          onFlipBoard();
        } else {
          lastTapRef.current = e.timeStamp;
        }
        return;
      }
      lastTapRef.current = 0;
      const rest = ((drag.angle % 360) + 360) % 360;
      const wantFlipped = rest > 90 && rest < 270;
      if (wantFlipped !== flipped && onFlipBoard) {
        settleFromRef.current = drag.angle;
        onFlipBoard();
      } else {
        spinBoard(el, spinRef, drag.angle, flipped, true);
      }
    },
    [flipped, onFlipBoard],
  );

  return (
    <div
      ref={boardRef}
      className={`board${zomgClass}${flippedClass}${
        turnable ? " board-turnable" : ""
      }`}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerUp}
    >
      <BoardCoordLabels gridDim={props.gridSize} flipped={flipped} />
      <div
        className="board-spaces-container"
        id="board"
        onMouseLeave={props.handleUnsetHover}
        {...outerDivProps}
      >
        <BoardSpaces
          gridDim={props.gridSize}
          gridLayout={props.gridLayout}
          handleTileDrop={props.handleTileDrop}
          placementArrow={props.placementArrowProperties}
          squareClicked={props.squareClicked}
        />
        {!isExamining && <LearnOverlay gridDim={props.gridSize} />}
        <Tiles
          tileColorId={props.tileColorId}
          gridDim={props.gridSize}
          onClick={props.handleBoardTileClick}
          tilesLayout={props.tilesLayout}
          lastPlayedTiles={props.lastPlayedTiles}
          playerOfTileAt={props.playerOfTileAt}
          tentativeTiles={props.tentativeTiles}
          placementArrow={props.placementArrowProperties}
          tentativeTileScore={props.tentativeTileScore}
          handleSetHover={props.handleSetHover}
          handleUnsetHover={props.handleUnsetHover}
          definitionPopover={props.definitionPopover}
          handleTileDrop={props.handleTileDrop}
          alphabet={props.alphabet}
          recallOneTile={props.recallOneTile}
        />
        {svgDrawing}
      </div>
    </div>
  );
});

export default Board;
