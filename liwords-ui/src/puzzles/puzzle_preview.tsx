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
import {
  useGameContextStoreContext,
  useLoginStateStoreContext,
} from '../store/store';
import { sortTiles } from '../store/constants';
import Tile from '../gameroom/tile';
import {
  Alphabet,
  runeToValues,
  StandardEnglishAlphabet,
} from '../constants/alphabets';
import { TouchBackend } from 'react-dnd-touch-backend';
import { DndProvider } from 'react-dnd';
import { PlayerAvatar } from '../shared/player_avatar';
import { RatingBadge } from '../lobby/rating_badge';

export const PuzzlePreview = React.memo(() => {
  const userLexicon = localStorage?.getItem('puzzleLexicon');
  const { useState } = useMountedState();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { username, loggedIn } = loginState;
  const [rack, setRack] = useState('');
  const [alphabet, setAlphabet] = useState<Alphabet>(StandardEnglishAlphabet);
  const [puzzleID, setPuzzleID] = useState<string | null>(null);
  const [userRating, setUserRating] = useState<number | undefined>(undefined);
  const [puzzleRating, setPuzzleRating] = useState<number | undefined>(
    undefined
  );

  const loadNewPuzzle = useCallback(async () => {
    if (!userLexicon) {
      // A hard coded id for a simple puzzle to display. Clicking in will

      setPuzzleID('E3kGXKyzhYirNzsMfCW3QV');
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
    loadNewPuzzle();
  }, [loadNewPuzzle]);

  useEffect(() => {
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
        setUserRating(resp.getAnswer()?.getNewUserRating());
        setPuzzleRating(resp.getAnswer()?.getNewPuzzleRating());
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
      console.log('fetching puzzle info');
      fetchPuzzleData(puzzleID);
    }
  }, [puzzleID, puzzleRating, dispatchGameContext]);

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

  const title = useMemo(() => {
    return userLexicon ? 'Next puzzle' : 'Try a puzzle';
  }, [userLexicon]);

  return (
    <Card
      title={title}
      className={`puzzle-preview ${!puzzleID ? 'tease' : ''}`}
    >
      <div className="puzzle-container">
        <DndProvider backend={TouchBackend}>
          <a href="/puzzle">
            <BoardPreview board={gameContext.board} alphabet={alphabet} />
            <div className="puzzle-rack">{renderTiles}</div>
          </a>
        </DndProvider>
      </div>
      {loggedIn && !!userRating ? (
        <div className="rating-info">
          <div className="player-rating">
            <PlayerAvatar username={username} />
            <div className="player-details">
              <p>{username}</p>
              <RatingBadge rating={userRating?.toString() || '1500?'} />
            </div>
          </div>
          <div className="player-rating">
            <PlayerAvatar icon={<i className="fa-solid fa-puzzle-piece" />} />
            <div className="player-details">
              <p>Equity Puzzle</p>
              <RatingBadge rating={puzzleRating?.toString() || '1500?'} />
            </div>
          </div>
        </div>
      ) : (
        <div className="new-puzzler-cta">
          <p>
            We have thousands of puzzles created from real games to help you
            practice play finding. Ready to try one?
          </p>
        </div>
      )}
    </Card>
  );
});
