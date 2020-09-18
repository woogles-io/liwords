import React, { useState } from 'react';
import { Button, Popconfirm } from 'antd';
import { ExchangeTiles } from './exchange_tiles';

type Props = {
  observer?: boolean;
  onExchange: (rack: string) => void;
  onPass: () => void;
  onResign: () => void;
  onRecall: () => void;
  onChallenge: () => void;
  onCommit: () => void;
  onExamine: () => void;
  onRematch: () => void;
  gameEndControls: boolean;
  showRematch: boolean;
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

const GameControls = React.memo((props: Props) => {
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

  if (props.gameEndControls) {
    return (
      <EndGameControls
        onRematch={props.onRematch}
        onExamine={props.onExamine}
        showRematch={props.showRematch}
      />
    );
  }

  if (props.observer) {
    return null;
  }

  return (
    <div className="game-controls">
      <Popconfirm
        title="Are you sure you wish to resign?"
        onConfirm={props.onResign}
        okText="Yes"
        cancelText="No"
      >
        <Button danger>Resign</Button>
      </Popconfirm>

      <Button onClick={props.onPass} danger>
        Pass
      </Button>

      <Button onClick={props.onChallenge}>Challenge</Button>

      <Button type="primary" onClick={showChallengeModal}>
        Exchange
      </Button>

      <ExchangeTiles
        rack={props.currentRack}
        exchangedRack={exchangedRack}
        selectTile={(idx) => selectTileForExchange(idx)}
        modalVisible={modalVisible}
        onOk={handleModalOk}
        onCancel={handleModalCancel}
      />

      <Button type="primary" onClick={props.onCommit}>
        Play
      </Button>
    </div>
  );
});

type EGCProps = {
  onRematch: () => void;
  showRematch: boolean;
  onExamine: () => void;
};

const EndGameControls = (props: EGCProps) => (
  <div className="game-controls">
    <Button>Options</Button>
    <Button onClick={props.onExamine}>Export GCG</Button>
    <Button onClick={() => window.location.replace('/')}>Exit</Button>
    {props.showRematch && (
      <Button type="primary" onClick={props.onRematch}>
        Rematch
      </Button>
    )}
  </div>
);

export default GameControls;
