import {
  MachineLetter,
  EmptyRackSpaceMachineLetter,
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
