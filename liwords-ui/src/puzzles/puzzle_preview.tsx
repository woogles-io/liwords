import React, { useCallback, useEffect } from 'react';
import { Card, message } from 'antd';
import { BoardPreview } from '../settings/board_preview';
import './puzzle_preview.scss';
import { useMountedState } from '../utils/mounted';
import {
  NextClosestRatingPuzzleIdRequest,
  NextClosestRatingPuzzleIdResponse,
  PuzzleRequest,
  PuzzleResponse,
} from '../gen/api/proto/puzzle_service/puzzle_service_pb';
import { LiwordsAPIError, postProto } from '../api/api';
import { ActionType } from '../actions/actions';
import { useGameContextStoreContext } from '../store/store';

const previewTilesLayout = [
  '               ',
  '               ',
  '               ',
  '               ',
  '               ',
  '               ',
  '               ',
  ' PUZzLE        ',
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
  const [puzzleID, setPuzzleID] = useState<string | null>(null);

  const loadNewPuzzle = useCallback(
    async (firstLoad?: boolean) => {
      if (!userLexicon) {
        return;
      }
      const req = new NextClosestRatingPuzzleIdRequest();
      const respType = NextClosestRatingPuzzleIdResponse;
      const method = 'GetNextClosestRatingPuzzleId';
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
    },
    [userLexicon]
  );

  useEffect(() => {
    if (userLexicon) {
      loadNewPuzzle();
    }
  }, [userLexicon]);

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
  }, [puzzleID]);

  return (
    <Card title="Next puzzle" className="puzzle-preview">
      <BoardPreview
        tilesLayout={previewTilesLayout}
        board={puzzleID ? gameContext.board : undefined}
      />
    </Card>
  );
});
