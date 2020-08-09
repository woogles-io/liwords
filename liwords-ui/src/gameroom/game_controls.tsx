import React, { useState } from 'react';
import { Button } from 'antd';
import { Link } from 'react-router-dom';
import { ExchangeTiles } from './exchange_tiles';

type Props = {
  onExchange: (rack: string) => void;
  onPass: () => void;
  onRecall: () => void;
  onChallenge: () => void;
  onCommit: () => void;
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
        showRematch={props.showRematch}
      />
    );
  }

  return (
    <div className="game-controls">
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
};

const EndGameControls = (props: EGCProps) => (
  <div className="game-controls">
    <Button>Options</Button>
    <Button>Examine</Button>
    <Button>
      <Link to="/">Exit</Link>
    </Button>
    {props.showRematch && (
      <Button type="primary" onClick={props.onRematch}>
        Rematch
      </Button>
    )}
  </div>
);

export default GameControls;
