import React, { useRef, useState, useCallback, useEffect } from 'react';
import { Button } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import {
  useGameContextStoreContext,
  useTentativeTileContext,
} from '../store/store';
import { sortBlanksLast } from '../store/constants';
import {
  contiguousTilesFromTileSet,
  simpletile,
} from '../utils/cwgame/scoring';
import { Direction, isMobile } from '../utils/cwgame/common';

type NotepadProps = {
  style?: React.CSSProperties;
};

const humanReadablePosition = (
  direction: Direction,
  firstLetter: simpletile
): string => {
  const readableCol = String.fromCodePoint(firstLetter.col + 65);
  const readableRow = (firstLetter.row + 1).toString();
  if (direction === 1) {
    return readableRow + readableCol;
  }
  return readableCol + readableRow;
};

export const Notepad = React.memo((props: NotepadProps) => {
  const notepadEl = useRef<HTMLTextAreaElement>(null);
  const [curNotepad, setCurNotepad] = useState('');
  const {
    displayedRack,
    placedTiles,
    placedTilesTempScore,
  } = useTentativeTileContext();
  const { gameContext } = useGameContextStoreContext();
  const board = gameContext.board;
  const addPlay = useCallback(() => {
    const contiguousTiles = contiguousTilesFromTileSet(placedTiles, board);
    let play = '';
    let position = '';
    const leave = sortBlanksLast(displayedRack.split('').sort().join(''));
    if (contiguousTiles?.length === 2) {
      position = humanReadablePosition(
        contiguousTiles[1],
        contiguousTiles[0][0]
      );
      play = contiguousTiles[0]
        .map((tile) =>
          tile.fresh ? tile.letter : `(${tile.letter.toLowerCase()})`
        )
        .join('');
    }
    setCurNotepad(
      `${curNotepad ? curNotepad + '\n' : ''}${
        play ? position + ' ' + play + ' ' : ''
      }${placedTilesTempScore ? placedTilesTempScore + ' ' : ''}${leave}`
    );
    // Return focus to board on all but mobile so the key commands can be used immediately
    if (!isMobile()) {
      document.getElementById('board-container')?.focus();
    }
  }, [displayedRack, placedTiles, placedTilesTempScore, curNotepad, board]);
  useEffect(() => {
    if (notepadEl.current && !(notepadEl.current === document.activeElement)) {
      notepadEl.current.scrollTop = notepadEl.current.scrollHeight || 0;
    }
  }, [curNotepad]);
  return (
    <div className="notepad-container" style={props.style}>
      <textarea
        className="notepad"
        value={curNotepad}
        ref={notepadEl}
        spellCheck={false}
        style={props.style}
        onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => {
          setCurNotepad(e.target.value);
        }}
      />
      <Button
        shape="circle"
        icon={<PlusOutlined />}
        type="primary"
        onClick={addPlay}
      />
    </div>
  );
});
