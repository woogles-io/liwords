import React from 'react';
import { DndProvider } from 'react-dnd';
import { TouchBackend } from 'react-dnd-touch-backend';
import { CrosswordGameGridLayout } from '../constants/board_layout';
import BoardSpaces from '../gameroom/board_spaces';

export const BoardPreview = React.memo(() => {
  return (
    <DndProvider backend={TouchBackend}>
      <div className="board board-preview">
        <div className="board-spaces-container">
          <BoardSpaces
            gridDim={15}
            gridLayout={CrosswordGameGridLayout}
            placementArrow={{
              row: 0,
              col: 0,
              horizontal: true,
              show: false,
            }}
            squareClicked={() => {}}
          />
        </div>
      </div>
    </DndProvider>
  );
});
