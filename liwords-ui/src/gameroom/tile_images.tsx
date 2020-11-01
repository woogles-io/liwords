import React from 'react';
import { DndProvider } from 'react-dnd';
import { TouchBackend } from 'react-dnd-touch-backend';
import BoardSpace from './board_space';
import Tile from './tile';
import { BonusType } from '../constants/board_layout';
import {
  CrosswordGameTileValues,
  runeToValues,
} from '../constants/tile_values';

const TileImages = React.memo((props: {}) => {
  // 1. Go to this page (see App.tsx for url).
  // 2. Screen-capture normally. Any white border can be cropped later.
  //    On Mac: cmd+shift+4.
  // 3. Crop the white border and assign transparency.
  //    On Mac: brew install netpbm
  // pngtopnm img.png | pnmcrop | pnmtopng -transparent '=#000000' > tiles.png

  return (
    <DndProvider backend={TouchBackend}>
      <div
        style={{
          alignItems: 'center',
          background: '#ffffff',
          display: 'flex',
          height: '100vh',
          justifyContent: 'center',
        }}
      >
        <div
          style={{
            background: '#000000',
            display: 'flex',
            flexWrap: 'wrap',
            width: '340px',
          }}
        >
          {Array.from(
            'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz ',
            (ch) => (
              <div>
                <Tile
                  rune={ch}
                  value={runeToValues(ch, CrosswordGameTileValues)}
                  lastPlayed={false}
                  key={ch}
                  grabbable={false}
                />
              </div>
            )
          )}
          {[
            BonusType.DoubleWord,
            BonusType.TripleWord,
            //BonusType.QuadrupleWord,
            BonusType.DoubleLetter,
            BonusType.TripleLetter,
            //BonusType.QuadrupleLetter,
            BonusType.StartingSquare,
            BonusType.NoBonus,
          ].map((bonusType) => (
            <div style={{ minWidth: '34px' }}>
              <BoardSpace
                bonusType={
                  bonusType === BonusType.StartingSquare
                    ? BonusType.DoubleWord
                    : bonusType
                }
                key={bonusType}
                arrow={false}
                arrowHoriz={false}
                startingSquare={bonusType === BonusType.StartingSquare}
                clicked={() => {}}
                handleTileDrop={(e: any) => {}}
              />
            </div>
          ))}
        </div>
      </div>
    </DndProvider>
  );
});

export default TileImages;
