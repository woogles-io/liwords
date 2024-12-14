import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Card } from "antd";
import { BoardPreview } from "../settings/board_preview";
import "./puzzle_preview.scss";
import { ActionType } from "../actions/actions";
import {
  useGameContextStoreContext,
  useLoginStateStoreContext,
} from "../store/store";
import { sortTiles } from "../store/constants";
import Tile from "../gameroom/tile";
import {
  Alphabet,
  scoreFor,
  StandardEnglishAlphabet,
} from "../constants/alphabets";
import { TouchBackend } from "react-dnd-touch-backend";
import { DndProvider } from "react-dnd";
import { PlayerAvatar } from "../shared/player_avatar";
import { RatingBadge } from "../lobby/rating_badge";
import { flashError, useClient } from "../utils/hooks/connect";
import {
  PuzzleRequestSchema,
  PuzzleService,
  StartPuzzleIdRequestSchema,
} from "../gen/api/proto/puzzle_service/puzzle_service_pb";
import { MachineLetter } from "../utils/cwgame/common";
import { create } from "@bufbuild/protobuf";

export const PuzzlePreview = React.memo(() => {
  const userLexicon = localStorage?.getItem("puzzleLexicon");
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { username, loggedIn } = loginState;
  const [rack, setRack] = useState(new Array<MachineLetter>());
  const [alphabet, setAlphabet] = useState<Alphabet>(StandardEnglishAlphabet);
  const [puzzleID, setPuzzleID] = useState<string | null>(null);
  const [userRating, setUserRating] = useState<number | undefined>(undefined);
  const [puzzleRating, setPuzzleRating] = useState<number | undefined>(
    undefined,
  );
  const puzzleClient = useClient(PuzzleService);

  const loadNewPuzzle = useCallback(async () => {
    if (!userLexicon) {
      // A hard coded id for a simple puzzle to display. Clicking in will

      setPuzzleID("E3kGXKyzhYirNzsMfCW3QV");
      return;
    }
    const req = create(StartPuzzleIdRequestSchema, {});
    req.lexicon = userLexicon;
    try {
      const resp = await puzzleClient.getStartPuzzleId(req);
      setPuzzleID(resp.puzzleId);
    } catch (err) {
      flashError(err);
    }
  }, [userLexicon, puzzleClient]);

  useEffect(() => {
    loadNewPuzzle();
  }, [loadNewPuzzle]);

  useEffect(() => {
    async function fetchPuzzleData(id: string) {
      const req = create(PuzzleRequestSchema, { puzzleId: id });

      try {
        const resp = await puzzleClient.getPuzzle(req);
        const gh = resp.history;
        setUserRating(resp.answer?.newUserRating);
        setPuzzleRating(resp.answer?.newPuzzleRating);
        dispatchGameContext({
          actionType: ActionType.SetupStaticPosition,
          payload: gh,
        });
      } catch (err) {
        const typescriptErr = err as Error;
        if (
          typescriptErr.name === "ConnectError" &&
          typescriptErr.message ===
            "[invalid_argument] cannot get id from uuid E3kGXKyzhYirNzsMfCW3QV: no rows for table puzzles"
        ) {
          // The hard coded puzzle only exists in production.
          setPuzzleID(null);
        } else {
          flashError(err);
        }
      }
    }
    if (puzzleID) {
      fetchPuzzleData(puzzleID);
    }
  }, [puzzleID, puzzleRating, puzzleClient, dispatchGameContext]);

  useEffect(() => {
    const rack =
      gameContext.players.find((p) => p.onturn)?.currentRack ??
      new Array<MachineLetter>();
    setRack(sortTiles(rack, gameContext.alphabet));
    setAlphabet(gameContext.alphabet);
  }, [gameContext]);

  const renderTiles = useMemo(() => {
    const tiles = [];
    if (!rack || rack.length === 0) {
      return null;
    }
    const noop = () => {};
    for (let n = 0; n < rack.length; n += 1) {
      const letter = rack[n];
      tiles.push(
        <Tile
          letter={letter}
          alphabet={alphabet}
          value={scoreFor(alphabet, letter)}
          lastPlayed={false}
          playerOfTile={0}
          key={`tile_${n}`}
          selected={false}
          grabbable={false}
          rackIndex={n}
          returnToRack={noop}
          moveRackTile={noop}
          onClick={noop}
        />,
      );
    }
    return <>{tiles}</>;
  }, [alphabet, rack]);

  const title = useMemo(() => {
    return userLexicon ? "Next puzzle" : "Try a puzzle";
  }, [userLexicon]);

  return (
    <Card
      title={title}
      className={`puzzle-preview ${!puzzleID ? "tease" : ""}`}
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
              <RatingBadge rating={userRating?.toString() || "1500?"} />
            </div>
          </div>
          <div className="player-rating">
            <PlayerAvatar icon={<i className="fa-solid fa-puzzle-piece" />} />
            <div className="player-details">
              <p>Equity Puzzle</p>
              <RatingBadge rating={puzzleRating?.toString() || "1500?"} />
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
