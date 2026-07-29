import {
  MachineLetter,
  EmptyBoardSpaceMachineLetter,
  EmptyRackSpaceMachineLetter,
  EphemeralTile,
} from "../utils/cwgame/common";
import { PlayerInfo } from "../gen/api/proto/ipc/omgwords_pb";
import { Client } from "@connectrpc/connect";
import { GameMetadataService } from "../gen/api/proto/game_service/game_service_pb";
import { flashError } from "../utils/hooks/connect";

// Shuffle the letters in a rack, preserving gaps.
export function shuffleLetters(a: Array<MachineLetter>): Array<MachineLetter> {
  const alistWithGaps = [...a];
  const alist = alistWithGaps.filter((x) => x !== EmptyRackSpaceMachineLetter);
  const n = alist.length;

  let somethingChanged = false;
  for (let i = n - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    if (alist[i] !== alist[j]) {
      somethingChanged = true;
      const tmp = alist[i];
      alist[i] = alist[j];
      alist[j] = tmp;
    }
  }

  if (!somethingChanged) {
    // Let's change something if possible.
    const j = Math.floor(Math.random() * n);
    const x = [];
    for (let i = 0; i < n; ++i) {
      if (alist[i] !== alist[j]) {
        x.push(i);
      }
    }

    if (x.length > 0) {
      const i = x[Math.floor(Math.random() * x.length)];
      const tmp = alist[i];
      alist[i] = alist[j];
      alist[j] = tmp;
    }
  }

  // Preserve the gaps.
  let r = 0;
  return alistWithGaps.map((x) =>
    x === EmptyRackSpaceMachineLetter ? x : alist[r++],
  );
}

// Export the game as a GCG file.
export async function gcgExport(
  gameID: string,
  playerMeta: Array<PlayerInfo>,
  gameMetadataClient: Client<typeof GameMetadataService>,
) {
  try {
    const resp = await gameMetadataClient.getGCG({ gameId: gameID });
    const url = window.URL.createObjectURL(new Blob([resp.gcg]));
    const link = document.createElement("a");
    link.href = url;
    let downloadFilename = `${gameID}.gcg`;
    // TODO: allow more characters as applicable
    // Note: does not actively prevent saving .dotfiles or nul.something
    if (playerMeta.every((x) => /^[-0-9A-Za-z_.]+$/.test(x.nickname))) {
      const byStarts: Array<Array<string>> = [[], []];
      for (const x of playerMeta) {
        byStarts[+!!x.first].push(x.nickname);
      }
      downloadFilename = `${[...byStarts[1], ...byStarts[0]].join(
        "-",
      )}-${gameID}.gcg`;
    }
    link.setAttribute("download", downloadFilename);
    document.body.appendChild(link);
    link.onclick = () => {
      link.remove();
      setTimeout(() => {
        window.URL.revokeObjectURL(url);
      }, 1000);
    };
    link.click();
  } catch (e) {
    flashError(e);
  }
}

// Create a backup key for the rack and letters.
export function backupKey(
  letters: Array<MachineLetter>,
  rack: Array<MachineLetter>,
) {
  return JSON.stringify({ letters, rack });
}

export type FullResetInputs = {
  // Board letters as of the last time the board was synced, undefined on the
  // first load.
  lastLetters: Array<MachineLetter> | undefined;
  letters: Array<MachineLetter>;
  dim: number;
  currentRack: Array<MachineLetter>;
  displayedRack: Array<MachineLetter>;
  placedTiles: Set<EphemeralTile>;
  arrowProperties: { row: number; col: number; show: boolean };
  isMyTurn: boolean;
  isExamining: boolean;
  puzzleMode: boolean | undefined;
  boardEditingMode: boolean | undefined;
  // Whether the game gained or lost events since the last sync.
  numEventsChanged: boolean;
  // Whether we have committed a move that has not been synced yet.
  justCommitted: boolean;
};

// Decide whether the tentatively placed tiles and the displayed rack must be
// resynced from the game state.
export function needsFullReset(inputs: FullResetInputs): boolean {
  const {
    lastLetters,
    letters,
    dim,
    currentRack,
    displayedRack,
    placedTiles,
    arrowProperties,
    isMyTurn,
    isExamining,
    puzzleMode,
    boardEditingMode,
    numEventsChanged,
    justCommitted,
  } = inputs;
  if (lastLetters === undefined) {
    // First load.
    return true;
  }
  if (puzzleMode) {
    // XXX: Without this, when exiting from examining an earlier turn on
    // puzzle mode, it does not reset the rack to the latest rack. Why?
    return true;
  }
  if (boardEditingMode) {
    // See comment above. I don't know why it doesn't show the latest rack
    // when we edit more than once.
    return true;
  }
  if (
    JSON.stringify(
      [
        ...[...placedTiles].map((ephemeralTile) => {
          const ml = ephemeralTile.letter;
          return (ml & 0x80) !== 0 ? 0 : ml;
        }),
        ...displayedRack.filter((ml) => ml !== EmptyRackSpaceMachineLetter),
      ].sort(),
    ) !== JSON.stringify([...currentRack].sort())
  ) {
    // First load after receiving rack.
    // Or other cases where the tiles don't match up.
    return true;
  }
  if (isExamining) {
    // Prevent stuck tiles.
    return true;
  }
  if (numEventsChanged && justCommitted) {
    // We have committed a move and the game has moved on since. Whatever is
    // still placed is a leftover of that move, even when the board looks
    // untouched, so recall it.
    return true;
  }
  if (!isMyTurn) {
    // Opponent's turn usually means we have just made a move. (Assumption:
    // there are only two players.) But a GameHistoryRefresher can also
    // arrive mid-turn without any move being made (e.g. the opponent's
    // clock was given more time), in which case the board is unchanged and
    // tentatively placed tiles should be left alone.
    const lettersChanged =
      lastLetters.length !== letters.length ||
      lastLetters.some((ml, idx) => ml !== letters[idx]);
    return lettersChanged || numEventsChanged;
  }
  // Opponent just did something. Check if it affects any premove.
  // TODO: revisit when supporting non-square boards.
  const letterAt = (row: number, col: number, of: Array<MachineLetter>) =>
    row < 0 || row >= dim || col < 0 || col >= dim ? null : of[row * dim + col];
  const letterChanged = (row: number, col: number) =>
    letterAt(row, col, lastLetters) !== letterAt(row, col, letters);
  const hookChanged = (
    row: number,
    col: number,
    drow: number,
    dcol: number,
  ) => {
    while (true) {
      row += drow;
      col += dcol;
      if (letterChanged(row, col)) return true;
      const letter = letterAt(row, col, letters);
      if (letter === null || letter === EmptyBoardSpaceMachineLetter) {
        return false;
      }
    }
  };
  const placedTileAffected = (row: number, col: number) =>
    letterChanged(row, col) ||
    hookChanged(row, col, -1, 0) ||
    hookChanged(row, col, +1, 0) ||
    hookChanged(row, col, 0, -1) ||
    hookChanged(row, col, 0, +1);
  // If no tiles have been placed, but placement arrow is shown,
  // reset based on if that position is affected.
  // This avoids having the placement arrow behind a tile.
  return (
    placedTiles.size === 0 && arrowProperties.show
      ? [arrowProperties as { row: number; col: number }]
      : Array.from(placedTiles)
  ).some(({ row, col }) => placedTileAffected(row, col));
}
