import React, { useCallback, useEffect, useMemo } from 'react';
import { Card, message } from 'antd';
import { BoardPreview } from '../settings/board_preview';
import './puzzle_preview.scss';
import { useMountedState } from '../utils/mounted';
import {
  StartPuzzleIdRequest,
  StartPuzzleIdResponse,
  PuzzleRequest,
  PuzzleResponse,
} from '../gen/api/proto/puzzle_service/puzzle_service_pb';
import { LiwordsAPIError, postProto } from '../api/api';
import { ActionType } from '../actions/actions';
import { useGameContextStoreContext } from '../store/store';
import { sortTiles } from '../store/constants';
import Tile from '../gameroom/tile';
import {
  Alphabet,
  runeToValues,
  StandardEnglishAlphabet,
} from '../constants/alphabets';
import { TouchBackend } from 'react-dnd-touch-backend';
import { DndProvider } from 'react-dnd';

const previewTilesLayout = [
  '               ',
  '               ',
  '               ',
  '               ',
  '               ',
  '               ',
  '               ',
  ' PUZzLEs       ',
  '               ',
  '               ',
  '               ',
  '               ',
  '               ',
  '               ',
  '               ',
];

export const PuzzlePreview = React.memo(() => {
  const userLexicon = localStorage?.getItem('puzzleLexicon');
  const { useState } = useMountedState();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const [rack, setRack] = useState('');
  const [alphabet, setAlphabet] = useState<Alphabet>(StandardEnglishAlphabet);
  const [puzzleID, setPuzzleID] = useState<string | null>(null);

  const loadNewPuzzle = useCallback(async () => {
    if (!userLexicon) {
      return;
    }
    const req = new StartPuzzleIdRequest();
    const respType = StartPuzzleIdResponse;
    const method = 'GetStartPuzzleId';
    req.setLexicon(userLexicon);
    try {
      const resp = await postProto(
        respType,
        'puzzle_service.PuzzleService',
        method,
        req
      );
      console.log('got resp', resp.toObject());
      setPuzzleID(resp.getPuzzleId());
    } catch (err) {
      message.error({
        content: `Puzzle: ${(err as LiwordsAPIError).message}`,
        duration: 5,
      });
    }
  }, [userLexicon]);

  useEffect(() => {
    if (userLexicon) {
      loadNewPuzzle();
    }
  }, [loadNewPuzzle, userLexicon]);

  useEffect(() => {
    console.log('fetching puzzle info');
    async function fetchPuzzleData(id: string) {
      const req = new PuzzleRequest();
      req.setPuzzleId(id);

      try {
        const resp = await postProto(
          PuzzleResponse,
          'puzzle_service.PuzzleService',
          'GetPuzzle',
          req
        );
        console.log('got puzzle', resp.toObject());

        const gh = resp.getHistory();
        dispatchGameContext({
          actionType: ActionType.SetupStaticPosition,
          payload: gh,
        });
      } catch (err) {
        message.error({
          content: (err as LiwordsAPIError).message,
          duration: 5,
        });
      }
    }
    if (puzzleID) {
      fetchPuzzleData(puzzleID);
    }
  }, [dispatchGameContext, puzzleID]);

  useEffect(() => {
    const rack = gameContext.players.find((p) => p.onturn)?.currentRack ?? '';
    setRack(sortTiles(rack));
    setAlphabet(gameContext.alphabet);
  }, [gameContext]);

  const renderTiles = useMemo(() => {
    const tiles = [];
    if (!rack || rack.length === 0) {
      return null;
    }
    const noop = () => {};
    for (let n = 0; n < rack.length; n += 1) {
      const rune = rack[n];
      tiles.push(
        <Tile
          rune={rune}
          value={runeToValues(alphabet, rune)}
          lastPlayed={false}
          playerOfTile={0}
          key={`tile_${n}`}
          scale={false}
          selected={false}
          grabbable={false}
          rackIndex={n}
          returnToRack={noop}
          moveRackTile={noop}
          onClick={noop}
        />
      );
    }
    return <>{tiles}</>;
  }, [alphabet, rack]);

  return (
    <Card title="Next puzzle" className="puzzle-preview">
      <div className="puzzle-container">
        <DndProvider backend={TouchBackend}>
          <BoardPreview
            tilesLayout={!userLexicon ? previewTilesLayout : undefined}
            board={puzzleID ? gameContext.board : undefined}
            alphabet={alphabet}
          />
          <div className="puzzle-rack">{renderTiles}</div>
        </DndProvider>
      </div>
    </Card>
  );
});
