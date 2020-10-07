import React, { useEffect, useState } from 'react';
import Rack from './rack';
import { useStoreContext } from '../store/store';
import Pool from './pool';
import { PoolFormatType } from '../constants/pool_formats';
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

export const ExchangeTiles = (props: Props) => {
  const [exchangedRackIndices, setExchangedRackIndices] = useState(
    new Set<number>()
  );
  const [exchangedRack, setExchangedRack] = useState('');

  const [delayInput, setDelayInput] = useState(true);

  useEffect(() => {
    const keydown = (e: KeyboardEvent) => {
      if (delayInput || !props.modalVisible) {
        return;
      }
      if (e.key === 'Enter' && exchangedRack.length) {
        props.onOk(exchangedRack);
      }
      const key = e.key.toLocaleUpperCase();
      const tempToExchange = new Set<number>(exchangedRackIndices);
      if (!exchangedRack.includes(key)) {
        const temporaryRack = props.rack.split('');
        // Add all instances of the key the first time it is picked
        while (temporaryRack.includes(key)) {
          tempToExchange.add(temporaryRack.lastIndexOf(key));
          temporaryRack.splice(temporaryRack.lastIndexOf(key), 1);
        }
      } else {
        // Find the last one that's currently selected and deselect
        let searchPoint = props.rack.length;
        while (searchPoint > 0) {
          const candidate = props.rack.lastIndexOf(key, searchPoint - 1);
          if (tempToExchange.has(candidate)) {
            tempToExchange.delete(candidate);
            searchPoint = 0;
          } else {
            searchPoint = candidate;
          }
        }
      }
      setExchangedRackIndices(tempToExchange);
    };
    window.addEventListener('keydown', keydown);
    return () => {
      window.removeEventListener('keydown', keydown);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  });
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
  const { gameContext, setPoolFormat } = useStoreContext();
  const selectTileForExchange = (idx: number) => {
    const newExchangedRackIndices = new Set(exchangedRackIndices);
    if (newExchangedRackIndices.has(idx)) {
      newExchangedRackIndices.delete(idx);
    } else {
      newExchangedRackIndices.add(idx);
    }
    setExchangedRackIndices(newExchangedRackIndices);
  };
  return (
    <Modal
      className="exchange"
      title="Exchange tiles"
      visible={props.modalVisible}
      onOk={() => {
        props.onOk(exchangedRack);
      }}
      onCancel={props.onCancel}
      width={360}
      footer={[
        <>
          {exchangedRackIndices.size > 0 ? (
            <p className="label">{`${exchangedRackIndices.size} tiles selected`}</p>
          ) : null}
          <Button
            key="submit"
            type="primary"
            onClick={() => {
              props.onOk(exchangedRack);
            }}
            disabled={exchangedRackIndices.size < 1}
          >
            Exchange
          </Button>
        </>,
      ]}
    >
      <Rack
        letters={props.rack}
        grabbable={false}
        onTileClick={selectTileForExchange}
        moveRackTile={() => {}}
        selected={exchangedRackIndices}
      />

      <Pool
        omitCard={true}
        pool={gameContext?.pool}
        currentRack={props.rack}
        poolFormat={PoolFormatType.Alphabet}
        setPoolFormat={setPoolFormat}
      />
    </Modal>
  );
};
