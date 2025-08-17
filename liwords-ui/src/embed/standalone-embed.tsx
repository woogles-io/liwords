import React, { useState, useMemo, useCallback } from "react";
import { DndProvider } from "react-dnd";
import { TouchBackend } from "react-dnd-touch-backend";
import Board from "../gameroom/board";
import { StaticPlayerCards } from "../puzzles/static_player_cards";
import { Alphabet, alphabetFromName } from "../constants/alphabets";
import {
  CrosswordGameGridLayout,
  SuperCrosswordGameGridLayout,
} from "../constants/board_layout";
import { Board as BoardClass } from "../utils/cwgame/board";
import { MachineLetter, EphemeralTile } from "../utils/cwgame/common";
import {
  GameDocument,
  GameEvent as OMGWordsGameEvent,
  GameEvent_Type as OMGWordsGameEventType,
} from "../gen/api/proto/ipc/omgwords_pb";
import "./standalone-embed.scss";
import "../gameroom/scss/gameroom.scss";

export interface StandaloneEmbedProps {
  gameDocument: GameDocument;
  options?: {
    width?: number;
    height?: number;
    showControls?: boolean;
    showScores?: boolean;
    showMoveList?: boolean;
    theme?: "light" | "dark";
  };
}

// Simple event structure for board operations
type SimpleGameEvent = {
  type: number;
  row: number;
  column: number;
  direction: number;
  playedTiles: Uint8Array;
  score: number;
  playerIndex: number;
  isBingo: boolean;
  exchanged: Uint8Array;
  cumulative: number;
};

// Convert OMGWords GameEvent to simple format for board operations
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const convertToSimpleEvt = (oevt: any): SimpleGameEvent => {
  // The JSON uses lowercase field names
  // Determine event type based on fields present
  let type = OMGWordsGameEventType.TILE_PLACEMENT_MOVE; // default
  if (oevt.exchanged) {
    type = OMGWordsGameEventType.EXCHANGE;
  } else if (oevt.pass !== undefined) {
    type = OMGWordsGameEventType.PASS;
  } else if (oevt.challengedPhony !== undefined) {
    type = OMGWordsGameEventType.PHONY_TILES_RETURNED;
  }

  // Parse playedTiles from base64 if it's a string
  let playedTiles = oevt.playedTiles;
  if (typeof playedTiles === "string") {
    // Decode base64 to Uint8Array
    const binaryString = atob(playedTiles);
    playedTiles = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      playedTiles[i] = binaryString.charCodeAt(i);
    }
  }

  // Parse direction
  const direction = oevt.direction === "VERTICAL" ? 1 : 0;

  // console.log("Converting event - type:", type, "player:", oevt.playerIndex ?? 0, "cumulative:", oevt.cumulative, "tiles:", playedTiles?.length);

  return {
    type,
    row: oevt.row ?? 0,
    column: oevt.column ?? 0,
    direction,
    playedTiles: playedTiles ?? new Uint8Array(),
    score: oevt.score ?? 0,
    playerIndex: oevt.playerIndex ?? 0, // Default to player 0 if not specified
    isBingo: oevt.isBingo ?? false,
    exchanged: oevt.exchanged ?? new Uint8Array(),
    cumulative: oevt.cumulative ?? 0,
  };
};

// Place tiles on board (adapted from game_reducer.ts)
const placeOnBoard = (
  board: BoardClass,
  pool: Uint8Array,
  evt: SimpleGameEvent,
  alphabet: Alphabet,
): [{ [key: string]: boolean }, Uint8Array] => {
  const lastPlayedTiles: { [key: string]: boolean } = {};
  const newPool = [...pool];

  if (evt.type === OMGWordsGameEventType.TILE_PLACEMENT_MOVE) {
    const row = evt.row;
    const col = evt.column;
    const dir = evt.direction;
    const playedTiles = evt.playedTiles;

    for (let i = 0; i < playedTiles.length; i++) {
      const ml = playedTiles[i] as MachineLetter;
      if (dir === 0) {
        // HORIZONTAL
        board.addTile({ row, col: col + i, ml });
        lastPlayedTiles[`${row},${col + i}`] = true;
      } else {
        // VERTICAL
        board.addTile({ row: row + i, col, ml });
        lastPlayedTiles[`${row + i},${col}`] = true;
      }

      // Update pool
      const tileIdx = newPool.indexOf(ml);
      if (tileIdx !== -1) {
        newPool.splice(tileIdx, 1);
      }
    }
  }

  return [lastPlayedTiles, Uint8Array.from(newPool)];
};

// Unplace tiles from board (for handling challenges)
const unplaceOnBoard = (
  board: BoardClass,
  pool: Uint8Array,
  evt: SimpleGameEvent,
  alphabet: Alphabet,
): Uint8Array => {
  const newPool = [...pool];

  if (evt.type === OMGWordsGameEventType.TILE_PLACEMENT_MOVE) {
    const row = evt.row;
    const col = evt.column;
    const dir = evt.direction;
    const playedTiles = evt.playedTiles;

    for (let i = 0; i < playedTiles.length; i++) {
      const ml = playedTiles[i] as MachineLetter;
      if (dir === 0) {
        // HORIZONTAL
        board.removeTile({ row, col: col + i, ml });
      } else {
        // VERTICAL
        board.removeTile({ row: row + i, col, ml });
      }
      // Return tile to pool
      newPool.push(ml);
    }
  }

  return Uint8Array.from(newPool);
};

export const StandaloneEmbed: React.FC<StandaloneEmbedProps> = ({
  gameDocument,
  options = {},
}) => {
  const {
    width = 600,
    height = 700,
    showControls = true,
    showScores = true,
    showMoveList = false,
    theme = "light",
  } = options;

  const [currentEventIndex, setCurrentEventIndex] = useState(0);

  // Debug logging
  // console.log("Embed options:", options);
  // console.log("showControls:", showControls, "showScores:", showScores);
  // console.log("About to render controls section:", showControls, "currentEventIndex:", currentEventIndex, "events length:", gameDocument.events.length);

  const alphabet = useMemo(
    () =>
      alphabetFromName(
        gameDocument.letterDistribution?.toLowerCase() || "english",
      ),
    [gameDocument.letterDistribution],
  );

  const boardLayout =
    gameDocument.boardLayout === "SuperCrosswordGame"
      ? SuperCrosswordGameGridLayout
      : CrosswordGameGridLayout;
  const gridSize = gameDocument.boardLayout === "SuperCrosswordGame" ? 21 : 15;

  // Initialize board and replay events up to current index
  const { boardState, lastPlayedTiles, playerOfTileAt, currentScores } =
    useMemo(() => {
      const board = new BoardClass(boardLayout);
      let pool: Uint8Array;
      if (gameDocument.bag && "tiles" in gameDocument.bag) {
        pool = gameDocument.bag.tiles;
      } else {
        pool = new Uint8Array();
      }
      let lastPlayed: { [key: string]: boolean } = {};
      const playerOfTile: { [key: string]: number } = {};
      const scores = [0, 0];
      const events = gameDocument.events;

      // Replay events up to current index
      for (let i = 0; i < currentEventIndex && i < events.length; i++) {
        const evt = convertToSimpleEvt(events[i]);
        const onturn = evt.playerIndex;

        switch (evt.type) {
          case OMGWordsGameEventType.TILE_PLACEMENT_MOVE:
            [lastPlayed, pool] = placeOnBoard(board, pool, evt, alphabet);
            for (const k in lastPlayed) {
              playerOfTile[k] = onturn;
            }
            // Update score for the player who made the move
            scores[onturn] = events[i].cumulative;
            break;
          case OMGWordsGameEventType.PHONY_TILES_RETURNED:
            // Unplace the move BEFORE this one
            if (i > 0) {
              const toUnplace = convertToSimpleEvt(events[i - 1]);
              pool = unplaceOnBoard(board, pool, toUnplace, alphabet);
              // Clear last played tiles
              lastPlayed = {};
            }
            break;
          case OMGWordsGameEventType.EXCHANGE:
          case OMGWordsGameEventType.PASS:
            // These don't change the score but might update cumulative
            if (events[i].cumulative > 0) {
              scores[onturn] = events[i].cumulative;
            }
            break;
        }
      }

      // console.log("Board state after replay:", {
      //   currentEventIndex,
      //   boardLetters: board.letters.filter(l => l !== 0).length,
      //   boardArray: board.letters,
      //   scores,
      //   lastPlayed,
      //   playerOfTile
      // });

      return {
        boardState: board.letters,
        lastPlayedTiles: lastPlayed,
        playerOfTileAt: playerOfTile,
        currentScores: scores,
      };
    }, [
      currentEventIndex,
      gameDocument.events,
      gameDocument.bag,
      alphabet,
      boardLayout,
    ]);

  const handlePrevEvent = useCallback(() => {
    setCurrentEventIndex((prev) => Math.max(0, prev - 1));
  }, []);

  const handleNextEvent = useCallback(() => {
    setCurrentEventIndex((prev) =>
      Math.min(gameDocument.events.length, prev + 1),
    );
  }, [gameDocument.events.length]);

  const handleFirstEvent = useCallback(() => {
    setCurrentEventIndex(0);
  }, []);

  const handleLastEvent = useCallback(() => {
    setCurrentEventIndex(gameDocument.events.length);
  }, [gameDocument.events.length]);

  // Get player names for display

  const currentEvent =
    currentEventIndex > 0 ? gameDocument.events[currentEventIndex - 1] : null;
  const playerOnTurn = currentEvent ? currentEvent.playerIndex : 0;

  // Format event for display
  const formatEvent = (event: OMGWordsGameEvent) => {
    if (event.type === OMGWordsGameEventType.TILE_PLACEMENT_MOVE) {
      const col = String.fromCharCode(65 + event.column);
      const row = event.row + 1;
      const pos = `${row}${col}`;

      // Convert played tiles to readable format
      const tiles = Array.from(event.playedTiles)
        .map((t: number) => {
          const ml = t as MachineLetter;
          const isBlank = (ml & 0x80) !== 0;
          if (isBlank) {
            const letter = String.fromCharCode(64 + (ml & 0x7f));
            return letter.toLowerCase();
          }
          return alphabet.letters?.[ml] || "?";
        })
        .join("");

      return `${pos} ${tiles} (${event.score} pts)${event.isBingo ? " BINGO!" : ""}`;
    } else if (event.type === OMGWordsGameEventType.EXCHANGE) {
      return `Exchanged ${event.exchanged.length} tiles`;
    } else if (event.type === OMGWordsGameEventType.PASS) {
      return "Passed";
    } else if (event.type === OMGWordsGameEventType.CHALLENGE_BONUS) {
      return `Challenge bonus: +${event.bonus}`;
    }
    return "";
  };

  return (
    <DndProvider backend={TouchBackend}>
      <div
        className={`standalone-embed standalone-embed--${theme}`}
        style={{
          width,
          maxWidth: "100%",
          height,
          display: "flex",
          flexDirection: "column",
          border: "1px solid #e0e0e0",
          borderRadius: "8px",
          overflow: "hidden",
          backgroundColor: theme === "dark" ? "#1a1a1a" : "#fff",
        }}
      >
        {/* Player scores at the top */}
        {showScores && (
          <div
            style={{
              padding: "8px",
              borderBottom: "1px solid #e0e0e0",
              backgroundColor: theme === "dark" ? "#2a2a2a" : "#f5f5f5",
            }}
          >
            <StaticPlayerCards
              p0Score={currentScores[0]}
              p1Score={currentScores[1]}
              playerOnTurn={playerOnTurn}
            />
          </div>
        )}

        {/* Board in the middle */}
        <div className="standalone-embed__board-container">
          {/* Board placeholder - need to implement without store dependency */}
          <div
            style={{
              width: "400px",
              height: "400px",
              background: "#f0f0f0",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            Board requires store context - see comment
          </div>
        </div>

        {/* Controls at the bottom */}
        {showControls && (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              gap: "12px",
              padding: "12px",
              borderTop: "1px solid #e0e0e0",
              backgroundColor: theme === "dark" ? "#2a2a2a" : "#f5f5f5",
            }}
          >
            <button
              onClick={handleFirstEvent}
              disabled={currentEventIndex === 0}
              style={{
                width: "36px",
                height: "36px",
                border: "1px solid #d0d0d0",
                borderRadius: "4px",
                backgroundColor: "#fff",
                cursor: currentEventIndex === 0 ? "not-allowed" : "pointer",
                opacity: currentEventIndex === 0 ? 0.4 : 1,
              }}
              title="Go to beginning"
            >
              ⏮
            </button>
            <button
              onClick={handlePrevEvent}
              disabled={currentEventIndex === 0}
              style={{
                width: "36px",
                height: "36px",
                border: "1px solid #d0d0d0",
                borderRadius: "4px",
                backgroundColor: "#fff",
                cursor: currentEventIndex === 0 ? "not-allowed" : "pointer",
                opacity: currentEventIndex === 0 ? 0.4 : 1,
              }}
              title="Previous move"
            >
              ◀
            </button>
            <div
              style={{
                padding: "0 16px",
                fontSize: "14px",
                fontWeight: 500,
                color: theme === "dark" ? "#aaa" : "#666",
              }}
            >
              Move {currentEventIndex} / {gameDocument.events.length}
            </div>
            <button
              onClick={handleNextEvent}
              disabled={currentEventIndex >= gameDocument.events.length}
              style={{
                width: "36px",
                height: "36px",
                border: "1px solid #d0d0d0",
                borderRadius: "4px",
                backgroundColor: "#fff",
                cursor:
                  currentEventIndex >= gameDocument.events.length
                    ? "not-allowed"
                    : "pointer",
                opacity:
                  currentEventIndex >= gameDocument.events.length ? 0.4 : 1,
              }}
              title="Next move"
            >
              ▶
            </button>
            <button
              onClick={handleLastEvent}
              disabled={currentEventIndex >= gameDocument.events.length}
              style={{
                width: "36px",
                height: "36px",
                border: "1px solid #d0d0d0",
                borderRadius: "4px",
                backgroundColor: "#fff",
                cursor:
                  currentEventIndex >= gameDocument.events.length
                    ? "not-allowed"
                    : "pointer",
                opacity:
                  currentEventIndex >= gameDocument.events.length ? 0.4 : 1,
              }}
              title="Go to end"
            >
              ⏭
            </button>
          </div>
        )}

        {/* Move details (optional) */}
        {showMoveList && currentEvent && (
          <div
            style={{
              padding: "12px",
              borderTop: "1px solid #e0e0e0",
              backgroundColor: theme === "dark" ? "#252525" : "#fafafa",
            }}
          >
            <div
              style={{
                fontWeight: 600,
                marginBottom: "4px",
                color: theme === "dark" ? "#fff" : "#333",
              }}
            >
              {gameDocument.players[currentEvent.playerIndex].nickname}
            </div>
            <div
              style={{
                fontSize: "14px",
                color: theme === "dark" ? "#bbb" : "#666",
              }}
            >
              {formatEvent(currentEvent)}
            </div>
          </div>
        )}
      </div>
    </DndProvider>
  );
};
