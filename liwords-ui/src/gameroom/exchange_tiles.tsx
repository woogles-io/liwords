import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { prefersReducedMotion } from "./rack";
import Tile from "./tile";
import { sortTiles } from "../store/constants";
import {
  useGameContextStoreContext,
  usePoolFormatStoreContext,
} from "../store/store";
import Pool from "./pool";
import { Button } from "antd";
import { Modal } from "../utils/focus_modal";
import {
  Alphabet,
  getMachineLetterForKey,
  machineWordToRunes,
  scoreFor,
} from "../constants/alphabets";
import { MachineWord } from "../utils/cwgame/common";

// Render an exchange widget.

type Props = {
  tileColorId: number;
  rack: MachineWord;
  alphabet: Alphabet;
  onCancel: () => void;
  onOk: (tilesToExchange: MachineWord) => void;
  modalVisible: boolean;
};

// The set of selected rack positions is the single source of truth for both
// what the modal draws and what we submit. Deriving the tiles during render
// (rather than in an effect, into a second piece of state) is what keeps those
// two in step: a tile that is drawn in the row to exchange is always a tile we
// will send.
const tilesForIndices = (
  indices: Set<number>,
  rack: MachineWord,
): MachineWord => {
  const sorted = Array.from(indices.keys()).sort((a, b) => a - b);
  return sorted.map((idx) => rack[idx]);
};

type TileRowProps = {
  label: string;
  row: string;
  // whether the rack's tile at a place is in this row
  has: (idx: number) => boolean;
  rack: MachineWord;
  alphabet: Alphabet;
  tileColorId: number;
  onTileClick: (idx: number) => void;
};

// One of the modal's two rows, the tiles to exchange or the tiles kept. Each
// tile keeps its place in the rack, which the row leaves empty while the tile
// is in the other one: a tile clicked goes straight to the other row, and no
// other tile moves.
const TileRow = (props: TileRowProps) => (
  <div className="exchange-row" data-row={props.row}>
    <p className="label">{props.label}</p>
    <div className="rack">
      {props.rack.map((letter, idx) =>
        props.has(idx) ? (
          <div key={idx} data-rack-index={idx}>
            <Tile
              letter={letter}
              alphabet={props.alphabet}
              value={scoreFor(props.alphabet, letter)}
              lastPlayed={false}
              playerOfTile={props.tileColorId}
              grabbable={false}
              clickable
              onClick={() => props.onTileClick(idx)}
            />
          </div>
        ) : (
          <div key={idx} className="exchange-place" />
        ),
      )}
    </div>
  </div>
);

export const ExchangeTiles = React.memo((props: Props) => {
  const [exchangedRackIndices, setExchangedRackIndices] = useState(
    new Set<number>(),
  );

  const [delayInput, setDelayInput] = useState(true);

  const propsOnOk = props.onOk;

  const exchangedRack = useMemo(
    () => tilesForIndices(exchangedRackIndices, props.rack),
    [exchangedRackIndices, props.rack],
  );

  // the rack's places kept, in the order the rack has
  const keptIndices = useMemo(
    () =>
      props.rack.flatMap((_, idx) =>
        exchangedRackIndices.has(idx) ? [] : [idx],
      ),
    [exchangedRackIndices, props.rack],
  );
  // the tiles named in the footer and on the button, in the user's tile order,
  // as the analyzer names an exchange
  const keptRunes = machineWordToRunes(
    sortTiles(
      keptIndices.map((idx) => props.rack[idx]),
      props.alphabet,
    ),
    props.alphabet,
    false,
    true,
  );
  const exchangedRunes = machineWordToRunes(
    sortTiles(exchangedRack, props.alphabet),
    props.alphabet,
    false,
    true,
  );

  // A tile that changes rows slides there: the tops of the rack's tiles are
  // noted just before a change, and each one then elsewhere is animated from
  // its old top to its new one, unless the user asks for less motion.
  const rowsRef = useRef<HTMLDivElement>(null);
  const topsBefore = useRef<Map<number, number> | null>(null);
  const noteTops = useCallback(() => {
    const tops = new Map<number, number>();
    rowsRef.current
      ?.querySelectorAll<HTMLElement>("[data-rack-index]")
      .forEach((el) =>
        tops.set(Number(el.dataset.rackIndex), el.getBoundingClientRect().top),
      );
    topsBefore.current = tops;
  }, []);
  useLayoutEffect(() => {
    const before = topsBefore.current;
    topsBefore.current = null;
    if (!before || prefersReducedMotion()) {
      return;
    }
    rowsRef.current
      ?.querySelectorAll<HTMLElement>("[data-rack-index]")
      .forEach((el) => {
        const was = before.get(Number(el.dataset.rackIndex));
        const moved =
          was === undefined ? 0 : was - el.getBoundingClientRect().top;
        // the Web Animations API, where there is one
        if (moved && typeof el.animate === "function") {
          el.animate(
            [{ transform: `translateY(${moved}px)` }, { transform: "none" }],
            { duration: 150, easing: "ease" },
          );
        }
      });
  }, [exchangedRackIndices]);

  // Temporary message until UI shows it.
  useEffect(() => {
    if (props.modalVisible) {
      console.log(
        "When exchanging, press - to toggle the tiles selected. For example, type 4 E - Enter to exchange 6 and keep E.",
      );
    }
  }, [props.modalVisible]);

  const keydown = useCallback(
    (e: KeyboardEvent) => {
      if (delayInput || !props.modalVisible) {
        return;
      }
      if (e.key === "Enter") {
        // Prevent also activating the focused button.
        // Previously, if the Exchange button was clicked,
        // pressing Enter would reactivate the exchange modal.
        // This did not happen when using the shortcut.
        e.preventDefault();
        if (exchangedRack.length) {
          propsOnOk(exchangedRack);
        }
        return;
      }
      const key = e.key.toLocaleUpperCase();

      // Toggle all. To keep selected tiles, toggle just before exchanging.
      if (key === "-") {
        if (props.rack.length > 0) {
          noteTops();
          setExchangedRackIndices((prev) => {
            const tempToExchange = new Set<number>();
            for (let i = 0; i < props.rack.length; ++i) {
              if (!prev.has(i)) {
                tempToExchange.add(i);
              }
            }
            return tempToExchange;
          });
        }
        return;
      }

      const ml = getMachineLetterForKey(key, props.alphabet);

      noteTops();
      setExchangedRackIndices((prev) => {
        // Select one more instance if any.
        let canDeselect = false;
        for (let i = 0; i < props.rack.length; ++i) {
          if (props.rack[i] === ml) {
            if (!prev.has(i)) {
              return new Set(prev).add(i);
            }
            canDeselect = true;
          }
        }
        if (!canDeselect) {
          return prev;
        }
        // Deselect all instances at once.
        const tempToExchange = new Set(prev);
        for (let i = 0; i < props.rack.length; ++i) {
          if (props.rack[i] === ml) {
            tempToExchange.delete(i);
          }
        }
        return tempToExchange;
      });
    },
    [
      delayInput,
      exchangedRack,
      props.modalVisible,
      props.rack,
      props.alphabet,
      propsOnOk,
      noteTops,
    ],
  );
  // Subscribe once and dispatch through a ref, so the listener that is actually
  // attached to the window can never be an older closure than the last render.
  // (Re-subscribing from an effect leaves a window, after a keystroke is
  // committed but before effects flush, where the attached handler still holds
  // the previous selection.)
  const keydownRef = useRef(keydown);
  keydownRef.current = keydown;
  useEffect(() => {
    const listener = (e: KeyboardEvent) => keydownRef.current(e);
    window.addEventListener("keydown", listener);
    return () => {
      window.removeEventListener("keydown", listener);
    };
  }, []);
  useEffect(() => {
    if (!props.modalVisible) {
      return;
    }
    // Start from a clean selection every time the modal opens, then wait a beat
    // before taking keys so we don't "preselect" whatever key they hit to open
    // it. Only `delayInput` is deferred: clearing the selection on a timer used
    // to wipe out any tile clicked during that first 100ms. Nothing slides as
    // it opens.
    topsBefore.current = null;
    setExchangedRackIndices(new Set<number>());
    setDelayInput(true);
    const timeout = window.setTimeout(() => {
      setDelayInput(false);
    }, 100);
    return () => {
      window.clearTimeout(timeout);
    };
  }, [props.modalVisible]);
  const { gameContext } = useGameContextStoreContext();
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();
  const selectTileForExchange = useCallback(
    (idx: number) => {
      noteTops();
      setExchangedRackIndices((prev) => {
        const newExchangedRackIndices = new Set(prev);
        if (newExchangedRackIndices.has(idx)) {
          newExchangedRackIndices.delete(idx);
        } else {
          newExchangedRackIndices.add(idx);
        }
        return newExchangedRackIndices;
      });
    },
    [noteTops],
  );
  const handleOnOk = useCallback(() => {
    propsOnOk(exchangedRack);
  }, [propsOnOk, exchangedRack]);

  return (
    <Modal
      className="exchange"
      title="Exchange tiles"
      open={props.modalVisible}
      onOk={handleOnOk}
      onCancel={props.onCancel}
      width={360}
      footer={
        <>
          {exchangedRackIndices.size > 0 ? (
            <p className="label">
              {keptIndices.length > 0 ? `Keep ${keptRunes}` : "Keep none"}
            </p>
          ) : null}
          <Button
            key="submit"
            type="primary"
            onClick={handleOnOk}
            disabled={exchangedRackIndices.size < 1}
          >
            {exchangedRackIndices.size > 0
              ? `Exchange ${exchangedRunes}`
              : "Exchange"}
          </Button>
        </>
      }
    >
      {/* the tiles to exchange above, lifted out of the rack; the tiles kept
          below them */}
      <div ref={rowsRef}>
        <TileRow
          label="Exchange"
          row="exchange"
          has={(idx) => exchangedRackIndices.has(idx)}
          rack={props.rack}
          alphabet={props.alphabet}
          tileColorId={props.tileColorId}
          onTileClick={selectTileForExchange}
        />
        <TileRow
          label="Keep"
          row="keep"
          has={(idx) => !exchangedRackIndices.has(idx)}
          rack={props.rack}
          alphabet={props.alphabet}
          tileColorId={props.tileColorId}
          onTileClick={selectTileForExchange}
        />
      </div>

      <Pool
        omitCard={true}
        pool={gameContext?.pool}
        currentRack={props.rack}
        poolFormat={poolFormat}
        setPoolFormat={setPoolFormat}
        alphabet={props.alphabet}
      />
    </Modal>
  );
});
