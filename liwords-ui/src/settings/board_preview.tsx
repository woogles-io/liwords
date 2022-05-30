import React from 'react';
import { DndProvider } from 'react-dnd';
import { TouchBackend } from 'react-dnd-touch-backend';
import { CrosswordGameGridLayout } from '../constants/board_layout';
import BoardSpaces from '../gameroom/board_spaces';
import Tiles from '../gameroom/tiles';
import { Alphabet, StandardEnglishAlphabet } from '../constants/alphabets';
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
  let previewBoard: Board;
  if (props.board) {
    previewBoard = props.board;
  } else {
    previewBoard = new Board();
    if (props.tilesLayout) {
      previewBoard.setTileLayout(props.tilesLayout);
    }
  }
  console.log(previewBoard);
  return (
    <div className="board board-preview">
      <div className="board-spaces-container">
        <BoardSpaces
          gridDim={CrosswordGameGridLayout.length}
          gridLayout={CrosswordGameGridLayout}
          placementArrow={{
            row: 0,
            col: 0,
            horizontal: true,
            show: false,
          }}
          squareClicked={() => {}}
        />
        {previewBoard.letters ? (
          <Tiles
            tileColorId={1}
            gridDim={15}
            tilesLayout={previewBoard.letters}
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
  );
});
