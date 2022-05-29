import React from 'react';
import { DndProvider } from 'react-dnd';
import { TouchBackend } from 'react-dnd-touch-backend';
import { CrosswordGameGridLayout } from '../constants/board_layout';
import BoardSpaces from '../gameroom/board_spaces';
import Tiles from '../gameroom/tiles';
import {
  Alphabet,
  alphabetFromName,
  StandardEnglishAlphabet,
} from '../constants/alphabets';
import { EphemeralTile, PlayedTiles } from '../utils/cwgame/common';
import { Board } from '../utils/cwgame/board';

type Props = {
  tilesLayout?: string[];
  lastPlayedTiles?: PlayedTiles;
  alphabet?: Alphabet;
  board?: Board;
};

/**
 * Creates a static representation of a position from either a Board
 * or a static grid of runes. Board takes precedence;
 */

export const BoardPreview = React.memo((props: Props) => {
  const previewBoard: Board = new Board();
  if (props.tilesLayout) {
    previewBoard.setTileLayout(props.tilesLayout);
  }
  return (
    <DndProvider backend={TouchBackend}>
      <div className="board board-preview">
        <div className="board-spaces-container">
          <BoardSpaces
            gridDim={previewBoard.dim}
            gridLayout={CrosswordGameGridLayout}
            placementArrow={{
              row: 0,
              col: 0,
              horizontal: true,
              show: false,
            }}
            squareClicked={() => {}}
          />
          {props.tilesLayout ? (
            <Tiles
              tileColorId={1}
              gridDim={previewBoard.dim}
              tilesLayout={
                props.board ? props.board.letters : previewBoard.letters
              }
              alphabet={props.alphabet || StandardEnglishAlphabet}
              lastPlayedTiles={props.lastPlayedTiles || {}}
              playerOfTileAt={{}}
              onClick={() => {}}
              placementArrow={{
                col: 0,
                horizontal: false,
                row: 0,
                show: false,
              }}
              scaleTiles={true}
              tentativeTiles={new Set<EphemeralTile>()}
              tentativeTileScore={undefined}
            />
          ) : null}
        </div>
      </div>
    </DndProvider>
  );
});
