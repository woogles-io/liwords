import React from 'react';
import Rack from './rack';
import { useStoreContext } from '../store/store';
import Pool from './pool';
import { PoolFormatType } from '../constants/pool_formats';
import { Button, Modal } from 'antd';

// Render an exchange widget.

type Props = {
  rack: string;
  exchangedRack: Set<number>;
  onCancel: () => void;
  onOk: () => void;
  modalVisible: boolean;
  selectTile: (idx: number) => void;
};

export const ExchangeTiles = (props: Props) => {
  // convert exchangedRack to a letter string
  const { gameContext, setPoolFormat } = useStoreContext();
  return (
    <Modal
      className="exchange"
      title="Exchange tiles"
      visible={props.modalVisible}
      onOk={props.onOk}
      onCancel={props.onCancel}
      width={360}
      footer={[
        <>
          {props.exchangedRack.size > 0 ? (
            <p className="label">
              {`${props.exchangedRack.size} tiles selected`}
            </p>
          ) : null}
          <Button
            key="submit"
            type="primary"
            onClick={props.onOk}
            disabled={props.exchangedRack.size < 1}
          >
            Exchange
          </Button>
          ,
        </>,
      ]}
    >
      <Rack
        letters={props.rack}
        grabbable={false}
        onTileClick={props.selectTile}
        moveRackTile={() => {}}
        selected={props.exchangedRack}
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
