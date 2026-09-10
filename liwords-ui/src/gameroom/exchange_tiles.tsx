import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import Rack from "./rack";
import {
  useGameContextStoreContext,
  usePoolFormatStoreContext,
} from "../store/store";
import Pool from "./pool";
import { singularCount } from "../utils/plural";
import { Button } from "antd";
import { Modal } from "../utils/focus_modal";
import { Alphabet, getMachineLetterForKey } from "../constants/alphabets";
import { MachineWord } from "../utils/cwgame/common";

const doNothing = () => {};

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
// two in step: a tile that is drawn flipped up is always a tile we will send.
const tilesForIndices = (
  indices: Set<number>,
  rack: MachineWord,
): MachineWord => {
  const sorted = Array.from(indices.keys()).sort((a, b) => a - b);
  return sorted.map((idx) => rack[idx]);
};

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
    // to wipe out any tile clicked during that first 100ms.
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
  const selectTileForExchange = useCallback((idx: number) => {
    setExchangedRackIndices((prev) => {
      const newExchangedRackIndices = new Set(prev);
      if (newExchangedRackIndices.has(idx)) {
        newExchangedRackIndices.delete(idx);
      } else {
        newExchangedRackIndices.add(idx);
      }
      return newExchangedRackIndices;
    });
  }, []);
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
            <p className="label">{`${singularCount(
              exchangedRackIndices.size,
              "tile",
              "tiles",
            )} selected`}</p>
          ) : null}
          <Button
            key="submit"
            type="primary"
            onClick={handleOnOk}
            disabled={exchangedRackIndices.size < 1}
          >
            Exchange
          </Button>
        </>
      }
    >
      <Rack
        tileColorId={props.tileColorId}
        letters={props.rack}
        grabbable={false}
        onTileClick={selectTileForExchange}
        moveRackTile={doNothing}
        selected={exchangedRackIndices}
        alphabet={props.alphabet}
      />

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
