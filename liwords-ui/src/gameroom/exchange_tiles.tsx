import React from 'react';
import Rack from './rack';
import { useStoreContext } from '../store/store';
import Pool from './pool';
import { PoolFormatType } from '../constants/pool_formats';

// Render an exchange widget.

type Props = {
  rack: string;
  exchangedRack: string;
  selectTile: (idx: number) => void;
};

export const ExchangeTiles = (props: Props) => {
  // convert exchangedRack to a letter string
  const { gameContext, poolFormat, setPoolFormat } = useStoreContext();
  return (
    <div className="exchange">
      <h4>{props.exchangedRack}</h4>
      <Rack
        letters={props.rack}
        grabbable={false}
        onTileClick={props.selectTile}
        swapRackTiles={() => {}}
      />

      <Pool
        omitCard={true}
        pool={gameContext?.pool}
        currentRack={props.rack}
        poolFormat={PoolFormatType.Alphabet}
        setPoolFormat={setPoolFormat}
      />
    </div>
  );
};
