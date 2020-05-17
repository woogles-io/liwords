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
          width={this.props.tileDim}
          height={this.props.tileDim}
          x={n * (this.props.tileDim + TileSpacing)}
          y={0}
          lastPlayed={false}
          key={`tile_${n}`}
          scale={false}
        />
      );
    }
    return <>{tiles}</>;
  }

  render() {
    return (
      <svg
        width={this.props.tileDim * 7 + TileSpacing * 6}
        height={this.props.tileDim}
      >
        <g>{this.renderTiles()}</g>
      </svg>
    );
  }
}

export default Rack;
