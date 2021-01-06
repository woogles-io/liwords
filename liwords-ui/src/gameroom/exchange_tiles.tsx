import React, { useCallback, useEffect } from 'react';
import { useMountedState } from '../utils/mounted';
import Rack from './rack';
import {
  useGameContextStoreContext,
  usePoolFormatStoreContext,
} from '../store/store';
import Pool from './pool';
import { singularCount } from '../utils/plural';
import { Button, Modal } from 'antd';
// Render an exchange widget.

type Props = {
  rack: string;
  onCancel: () => void;
  onOk: (tilesToExchange: string) => void;
  modalVisible: boolean;
};

type SelectedTile = {
  rune: string;
  index: number;
};

export const ExchangeTiles = React.memo((props: Props) => {
  const { useState } = useMountedState();

  const [exchangedRackIndices, setExchangedRackIndices] = useState(
    new Set<number>()
  );
  const [exchangedRack, setExchangedRack] = useState('');

  const [delayInput, setDelayInput] = useState(true);

  const propsOnOk = props.onOk;

  // Temporary message until UI shows it.
  useEffect(() => {
    if (props.modalVisible) {
      console.log(
        'When exchanging, press - to toggle the tiles selected. For example, type 4 E - Enter to exchange 6 and keep E.'
      );
    }
  }, [props.modalVisible]);

  const keydown = useCallback(
    (e: KeyboardEvent) => {
      if (delayInput || !props.modalVisible) {
        return;
      }
      if (e.key === 'Enter') {
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
      if (key === '-') {
        if (props.rack.length > 0) {
          const tempToExchange = new Set<number>();
          for (let i = 0; i < props.rack.length; ++i) {
            if (!exchangedRackIndices.has(i)) {
              tempToExchange.add(i);
            }
          }
          setExchangedRackIndices(tempToExchange);
        }
        return;
      }

      // Select one more instance if any.
      let canDeselect = false;
      for (let i = 0; i < props.rack.length; ++i) {
        if (props.rack[i] === key) {
          if (!exchangedRackIndices.has(i)) {
            setExchangedRackIndices(new Set(exchangedRackIndices).add(i));
            return;
          }
          canDeselect = true;
        }
      }

      if (canDeselect) {
        // Deselect all instances at once.
        const tempToExchange = new Set(exchangedRackIndices);
        for (let i = 0; i < props.rack.length; ++i) {
          if (props.rack[i] === key) {
            tempToExchange.delete(i);
          }
        }
        setExchangedRackIndices(tempToExchange);
      }
    },
    [
      delayInput,
      exchangedRack,
      exchangedRackIndices,
      props.modalVisible,
      props.rack,
      propsOnOk,
    ]
  );
  useEffect(() => {
    window.addEventListener('keydown', keydown);
    return () => {
      window.removeEventListener('keydown', keydown);
    };
  }, [keydown]);
  useEffect(() => {
    // Wait to start taking keys so we don't "preselect" whatever key they
    // hit to open the exchange modal.
    // reset exchange rack when opening modal.

    window.setTimeout(() => {
      setDelayInput(false);
      setExchangedRackIndices(new Set<number>());
    }, 100);
  }, [props.modalVisible]);
  useEffect(() => {
    const indices = Array.from(exchangedRackIndices.keys());
    indices.sort();
    const e = indices.map((idx) => props.rack[idx]);
    setExchangedRack(e.join(''));
  }, [exchangedRackIndices, props.rack]);
  const { gameContext } = useGameContextStoreContext();
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();
  const selectTileForExchange = useCallback(
    (idx: number) => {
      const newExchangedRackIndices = new Set(exchangedRackIndices);
      if (newExchangedRackIndices.has(idx)) {
        newExchangedRackIndices.delete(idx);
      } else {
        newExchangedRackIndices.add(idx);
      }
      setExchangedRackIndices(newExchangedRackIndices);
    },
    [exchangedRackIndices]
  );
  const doNothing = useCallback(() => {}, []);
  const handleOnOk = useCallback(() => {
    propsOnOk(exchangedRack);
  }, [propsOnOk, exchangedRack]);

  return (
    <Modal
      className="exchange"
      title="Exchange tiles"
      visible={props.modalVisible}
      onOk={handleOnOk}
      onCancel={props.onCancel}
      width={360}
      footer={
        <>
          {exchangedRackIndices.size > 0 ? (
            <p className="label">{`${singularCount(
              exchangedRackIndices.size,
              'tile',
              'tiles'
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
        letters={props.rack}
        grabbable={false}
        onTileClick={selectTileForExchange}
        moveRackTile={doNothing}
        selected={exchangedRackIndices}
      />

      <Pool
        omitCard={true}
        pool={gameContext?.pool}
        currentRack={props.rack}
        poolFormat={poolFormat}
        setPoolFormat={setPoolFormat}
      />
    </Modal>
  );
});
