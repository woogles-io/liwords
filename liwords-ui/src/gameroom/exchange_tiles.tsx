import React from 'react';
import Rack from './rack';

// Render an exchange widget.

type Props = {
  rack: string;
  exchangedRack: string;
  selectTile: (idx: number) => void;
};

export const ExchangeTiles = (props: Props) => {
  // convert exchangedRack to a letter string

  return (
    <div>
      <h3>
        Select the tiles you wish to <em>exchange</em>:
      </h3>
      <Rack
        letters={props.rack}
        grabbable={false}
        onTileClick={props.selectTile}
        swapRackTiles={() => {}}
      />
      <h3>Exchanging:</h3>
      <h4>{props.exchangedRack}</h4>
    </div>
  );
};
