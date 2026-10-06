import React, { useCallback, useEffect, useRef, useState } from "react";
import { useDragDropManager, useDrop, XYCoord } from "react-dnd";
import Tile, { TILE_TYPE } from "./tile";
import { MachineLetter, MachineWord } from "../utils/cwgame/common";
import { Alphabet, scoreFor } from "../constants/alphabets";
import {
  RackDragState,
  RackGeometry,
  applyRackDrag,
  initialRackDragState,
  nextRackDragState,
  previewSlots,
  sameRackDragState,
} from "./rack_drag";

// const TileSpacing = 6;

const calculatePosition = (
  position: XYCoord,
  rackElement: HTMLElement,
  rackEmptyLeftElement: HTMLElement,
  rackSize: number,
) => {
  const rackLeft = rackElement.getBoundingClientRect().left;
  const rackWidth = rackElement.clientWidth;
  const rackEmptyWidth = rackEmptyLeftElement.clientWidth;
  const tileSize = (rackWidth - 2 * rackEmptyWidth) / rackSize;
  const relativePosition = position.x - rackLeft;
  if (relativePosition < rackEmptyWidth) return 0;
  if (relativePosition > rackWidth - rackEmptyWidth) return rackSize;
  return Math.floor((relativePosition - rackEmptyWidth) / tileSize);
};

type Props = {
  tileColorId: number;
  letters: MachineWord;
  grabbable: boolean;
  alphabet: Alphabet;
  onTileClick?: (idx: number) => void;
  selected?: Set<number>;
  rearrangeRack?: (newRack: Array<MachineLetter>) => void;
  returnToRack?: (
    rackIndex: number | undefined,
    tileIndex: number | undefined,
  ) => void;
};

const RACK_DROP_RESULT = { rack: true };

const prefersReducedMotion = () =>
  window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;

const measureRack = (rackElement: HTMLElement): RackGeometry | null => {
  const slots = Array.from(
    rackElement.querySelectorAll<HTMLElement>("[data-rack-slot]"),
  ).map((el) => el.getBoundingClientRect());
  if (slots.length === 0) {
    return null;
  }
  const rackRect = rackElement.getBoundingClientRect();
  const slotCenters = slots.map((r) => r.left + r.width / 2);
  return {
    top: rackRect.top,
    bottom: rackRect.bottom,
    slotCenters,
    slotWidth:
      slots.length > 1
        ? (slotCenters[slots.length - 1] - slotCenters[0]) / (slots.length - 1)
        : slots[0].width,
    tileHeight: slots[0].height,
  };
};

// Tracks a drag that started from one of this rack's tiles and computes a
// preview of where the tiles would land. The rack's DOM order and the
// displayed rack are left untouched until the drag ends: rack tiles are keyed
// by index, and re-rendering the dragged node mid-drag (or detaching it, on
// iOS) would break the drag.
const useRackDrag = (
  rackRef: React.RefObject<HTMLDivElement | null>,
  enabled: boolean,
) => {
  const manager = useDragDropManager();
  const [dragState, setDragState] = useState<RackDragState | null>(null);
  // Latest state, read when the drag ends. Only reset when a new drag
  // begins, so it survives the end-of-drag state change regardless of
  // callback order.
  const lastStateRef = useRef<RackDragState | null>(null);
  const geometryRef = useRef<RackGeometry | null>(null);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    const monitor = manager.getMonitor();
    const update = (next: RackDragState | null) => {
      if (!sameRackDragState(lastStateRef.current, next)) {
        lastStateRef.current = next;
        setDragState(next);
      }
    };
    const unsubState = monitor.subscribeToStateChange(() => {
      const dragging =
        monitor.isDragging() && monitor.getItemType() === TILE_TYPE;
      const item = dragging ? monitor.getItem() : null;
      if (item?.rackIndex == null) {
        if (geometryRef.current) {
          geometryRef.current = null;
          setDragState(null);
        }
        return;
      }
      if (geometryRef.current || !rackRef.current) {
        return;
      }
      geometryRef.current = measureRack(rackRef.current);
      if (geometryRef.current) {
        lastStateRef.current = null;
        update(initialRackDragState(parseInt(item.rackIndex, 10)));
      }
    });
    const unsubOffset = monitor.subscribeToOffsetChange(() => {
      const geo = geometryRef.current;
      const prev = lastStateRef.current;
      const offset = monitor.getClientOffset();
      if (!geo || !prev || !offset) {
        return;
      }
      update(nextRackDragState(prev, offset, geo));
    });
    return () => {
      unsubState();
      unsubOffset();
    };
  }, [manager, rackRef, enabled]);

  const takeFinalState = useCallback(() => {
    const s = lastStateRef.current;
    lastStateRef.current = null;
    return s;
  }, []);

  return {
    dragState,
    slotWidth: geometryRef.current?.slotWidth,
    takeFinalState,
  };
};

export const Rack = React.memo((props: Props) => {
  const rackRef = useRef<HTMLDivElement>(null);
  const { dragState, slotWidth, takeFinalState } = useRackDrag(
    rackRef,
    props.grabbable,
  );

  const lettersRef = useRef(props.letters);
  lettersRef.current = props.letters;
  const rearrangeRef = useRef(props.rearrangeRack);
  rearrangeRef.current = props.rearrangeRack;

  // Commit a rack-to-rack drag. This runs from the drag source's end(), so it
  // also catches drops just outside the rack element (within the lift
  // tolerance), where no drop target receives them.
  const onTileDragEnd = useCallback(
    (didDrop: boolean, dropResult: unknown) => {
      const finalState = takeFinalState();
      // dnd-core copies drop results, so check the marker, not identity.
      if (didDrop && !(dropResult as { rack?: boolean } | null)?.rack) {
        // Dropped on the board, which has already handled it.
        return;
      }
      if (
        !finalState ||
        finalState.target === null ||
        finalState.target === finalState.from
      ) {
        return;
      }
      rearrangeRef.current?.(applyRackDrag(lettersRef.current, finalState));
    },
    [takeFinalState],
  );

  const [, drop] = useDrop({
    accept: TILE_TYPE,
    drop: (item: { rackIndex?: string; tileIndex?: string }, monitor) => {
      if (item.rackIndex != null) {
        // Committed in onTileDragEnd from the previewed arrangement.
        return RACK_DROP_RESULT;
      }
      const clientOffset = monitor.getClientOffset();
      const rackElement = document.getElementById("rack");
      const rackEmptyElement = document.getElementById("left-empty");
      let rackPosition = 0;
      if (clientOffset && rackElement && rackEmptyElement) {
        rackPosition = calculatePosition(
          clientOffset,
          rackElement,
          rackEmptyElement,
          props.letters.length,
        );
      }
      if (props.returnToRack && item.tileIndex) {
        props.returnToRack(rackPosition, parseInt(item.tileIndex, 10));
      }
      return undefined;
    },
    collect: (monitor) => ({
      isOver: !!monitor.isOver(),
      canDrop: !!monitor.canDrop(),
    }),
  });
  // Always apply drop zone, multi-backend will handle device detection
  useEffect(() => {
    drop(rackRef);
  }, [drop]);

  const renderTiles = () => {
    const tiles = [];
    if (props.letters.length === 0) {
      return null;
    }

    const slots = dragState
      ? previewSlots(props.letters.length, dragState)
      : null;
    const transition = prefersReducedMotion() ? "none" : "transform 150ms ease";

    for (let n = 0; n < props.letters.length; n += 1) {
      const letter = props.letters[n];
      const shift = slots && slotWidth ? (slots[n] - n) * slotWidth : 0;
      tiles.push(
        <div
          key={`tile_${n}`}
          data-rack-slot={n}
          style={
            dragState
              ? {
                  transform: `translateX(${shift}px)`,
                  transition,
                  // The dragged tile follows the pointer as a preview; its
                  // slot shows where it would land.
                  visibility: n === dragState.from ? "hidden" : undefined,
                }
              : undefined
          }
        >
          <Tile
            letter={letter}
            alphabet={props.alphabet}
            value={scoreFor(props.alphabet, letter)}
            lastPlayed={false}
            playerOfTile={props.tileColorId}
            selected={props.selected && props.selected.has(n)}
            grabbable={props.grabbable}
            rackIndex={n}
            onDragEnd={onTileDragEnd}
            onClick={() => {
              if (props.onTileClick) {
                props.onTileClick(n);
              }
            }}
          />
        </div>,
      );
    }
    return <>{tiles}</>;
  };

  return (
    <div className="rack" ref={rackRef} id="rack">
      <div className="empty-rack droppable" id="left-empty" />
      {renderTiles()}
      <div className="empty-rack droppable" />
    </div>
  );
});

export default Rack;
