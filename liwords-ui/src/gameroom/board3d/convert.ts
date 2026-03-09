import { GameState } from "../../store/reducers/game_reducer";
import { Alphabet } from "../../constants/alphabets";
import { GameDocument } from "../../gen/api/proto/ipc/omgwords_pb";
import {
  GameEvent_Type,
  GameEvent_Direction,
} from "../../gen/api/proto/vendored/macondo/macondo_pb";
import { MachineLetter } from "../../utils/cwgame/common";
import { Board3DData } from "./types";

function mlToDisplay(ml: MachineLetter, alphabet: Alphabet): string {
  const isBlank = (ml & 0x80) !== 0;
  const idx = isBlank ? ml & 0x7f : ml;
  if (idx === 0) return "?"; // blank tile display
  const letter = alphabet.letters[idx];
  if (!letter) return "";
  const display = letter.tileDisplay ?? letter.rune;
  return isBlank ? display.toLowerCase() : display;
}

export function convertGameStateTo3DData(
  gameContext: GameState,
  alphabet: Alphabet,
  gameDocument?: GameDocument,
  userID?: string,
  tileColor = "orange",
  boardColor = "jade",
): Board3DData {
  const board = gameContext.board;
  const dim = board.dim;

  // Build 2D boardArray
  const boardArray: string[][] = [];
  for (let row = 0; row < dim; row++) {
    const rowArr: string[] = [];
    for (let col = 0; col < dim; col++) {
      const ml = board.letters[row * dim + col];
      if (ml === 0) {
        rowArr.push("");
      } else {
        rowArr.push(mlToDisplay(ml, alphabet));
      }
    }
    boardArray.push(rowArr);
  }

  // Rack: if userID is provided, show that user's rack (for active play).
  // Otherwise, show the on-turn player's rack (for examining/analyzing).
  let currentRack: number[];
  if (userID) {
    const userPlayer = gameContext.players.find((p) => p.userID === userID);
    currentRack = userPlayer?.currentRack ?? [];
  } else {
    const onturn = gameContext.onturn;
    currentRack = gameContext.players[onturn]?.currentRack ?? [];
  }
  const rack: string[] = currentRack
    .filter((ml) => ml !== 0x80)
    .map((ml) => mlToDisplay(ml, alphabet));

  // Remaining tiles: pool minus current rack
  const poolCopy: Record<number, number> = { ...gameContext.pool };
  for (const ml of currentRack) {
    if (ml === 0x80) continue;
    const idx = ml & 0x7f;
    poolCopy[idx] = (poolCopy[idx] ?? 0) - 1;
  }
  const remainingTiles: Record<string, number> = {};
  for (const [mlStr, count] of Object.entries(poolCopy)) {
    if (count <= 0) continue;
    const ml = parseInt(mlStr, 10);
    const letter = alphabet.letters[ml];
    if (!letter) continue;
    const display = letter.tileDisplay ?? letter.rune;
    remainingTiles[display] = (remainingTiles[display] ?? 0) + count;
  }

  // Alphabet scores
  const alphabetScores: Record<string, number> = {};
  for (const letter of alphabet.letters) {
    const display = letter.tileDisplay ?? letter.rune;
    if (display) {
      alphabetScores[display.toUpperCase()] = letter.score;
    }
  }

  // Player names: prefer nickToPlayerOrder (always populated from history/document),
  // fall back to gameDocument.players, then generic defaults.
  let player0Name = "Player 1";
  let player1Name = "Player 2";
  const nickToOrder = gameContext.nickToPlayerOrder ?? {};
  for (const [nick, order] of Object.entries(nickToOrder)) {
    if (order === "p0") player0Name = nick;
    else if (order === "p1") player1Name = nick;
  }
  if (player0Name === "Player 1" && (gameDocument?.players?.length ?? 0) >= 2) {
    player0Name = gameDocument!.players[0].nickname || player0Name;
  }
  if (player1Name === "Player 2" && (gameDocument?.players?.length ?? 0) >= 2) {
    player1Name = gameDocument!.players[1].nickname || player1Name;
  }

  const player0Score = gameContext.players[0]?.score ?? 0;
  const player1Score = gameContext.players[1]?.score ?? 0;
  const playerOnTurn = gameContext.onturn;

  // Last play description
  let lastPlay = "";
  const turns = gameContext.turns;
  if (turns.length > 0) {
    const last = turns[turns.length - 1];
    if (last.type === GameEvent_Type.TILE_PLACEMENT_MOVE) {
      const colLetter = String.fromCharCode(65 + last.column);
      const rowNum = last.row + 1;
      // Standard notation: horizontal = "8A" (row first), vertical = "A8" (col first)
      const coord =
        last.direction === GameEvent_Direction.HORIZONTAL
          ? `${rowNum}${colLetter}`
          : `${colLetter}${rowNum}`;
      lastPlay = `${coord}: ${last.playedTiles} (${last.score})`;
    } else if (last.type === GameEvent_Type.EXCHANGE) {
      lastPlay = `Exchange ${last.exchanged?.length ?? 0} tiles`;
    } else if (last.type === GameEvent_Type.PASS) {
      lastPlay = "Pass";
    }
  }

  return {
    boardArray,
    gridLayout: board.gridLayout,
    boardDimension: dim,
    rack,
    remainingTiles,
    alphabetScores,
    player0Name,
    player1Name,
    player0Score,
    player1Score,
    playerOnTurn,
    lastPlay,
    tileColor,
    boardColor,
  };
}
