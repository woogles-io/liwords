import React from 'react';

import {
  CrosswordGameTileValues,
  runeToValues,
} from '../constants/tile_values';
import Tile from './tile';

const TileSpacing = 6;

type Props = {
  letters: string;
  tileDim: number;
  grabbable: boolean;
  onTileClick?: (idx: number) => void;
};

class Rack extends React.Component<Props> {
  renderTiles() {
    const tiles = [];
    if (!this.props.letters || this.props.letters.length === 0) {
      return null;
    }

    for (let n = 0; n < this.props.letters.length; n += 1) {
      const rune = this.props.letters[n];
      tiles.push(
        <Tile
          rune={rune}
          value={runeToValues(rune, CrosswordGameTileValues)}
          lastPlayed={false}
          key={`tile_${n}`}
          scale={false}
          grabbable={this.props.grabbable}
          onClick={() => {
            if (this.props.onTileClick) {
              this.props.onTileClick(n);
            }
          }}
        />
      );
    }
    return <>{tiles}</>;
  }

  render() {
    return (
      <div className="rack">
        {this.renderTiles()}
      </div>
    );
  }
}

export default Rack;
