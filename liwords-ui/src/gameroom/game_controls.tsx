import React, { useState } from 'react';
import { Button, Modal } from 'antd';
import { ExchangeTiles } from './exchange_tiles';

type Props = {
  onExchange: (rack: string) => void;
  onPass: () => void;
  onChallenge: () => void;
  onCommit: () => void;
  currentRack: string;
};

const exchangeSetToString = (
  origRack: string,
  selectedTiles: Set<number>
): string => {
  const indices = Array.from(selectedTiles.keys());
  indices.sort();
  const e = indices.map((idx) => origRack[idx]);
  return e.join('');
};

const GameControls = (props: Props) => {
  const [modalVisible, setModalVisible] = useState(false);
  const [exchangedRack, setExchangedRack] = useState(new Set<number>());

  const showChallengeModal = () => {
    // reset exchange rack when opening modal.
    setExchangedRack(new Set<number>());
    setModalVisible(true);
  };

  const handleModalOk = () => {
    setModalVisible(false);
    props.onExchange(exchangeSetToString(props.currentRack, exchangedRack));
  };

  const handleModalCancel = () => {
    setModalVisible(false);
  };

  const selectTileForExchange = (idx: number) => {
    const newExchangedRack = new Set(exchangedRack);
    if (newExchangedRack.has(idx)) {
      newExchangedRack.delete(idx);
    } else {
      newExchangedRack.add(idx);
    }
    setExchangedRack(newExchangedRack);
  };

  return (
    <>
      <Button style={{ width: 100, height: 40, marginLeft: 25 }}>
        Options
      </Button>

      <Button
        onClick={props.onPass}
        danger
        style={{ width: 100, height: 40, marginLeft: 25 }}
      >
        Pass
      </Button>

      <Button
        type="primary"
        onClick={props.onChallenge}
        style={{ width: 100, height: 40, marginLeft: 25 }}
      >
        Challenge
      </Button>

      <Button
        type="primary"
        style={{ width: 100, height: 40, marginLeft: 25 }}
        onClick={showChallengeModal}
      >
        Exchange
      </Button>
      <Modal
        title="Exchange tiles"
        visible={modalVisible}
        onOk={handleModalOk}
        onCancel={handleModalCancel}
      >
        <ExchangeTiles
          rack={props.currentRack}
          exchangedRack={exchangeSetToString(props.currentRack, exchangedRack)}
          selectTile={(idx) => selectTileForExchange(idx)}
        />
      </Modal>

      <Button
        type="primary"
        onClick={props.onCommit}
        style={{ width: 100, height: 40, marginLeft: 25 }}
      >
        Play
      </Button>
    </>
  );
};

export default GameControls;
