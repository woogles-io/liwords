import { EphemeralTile, Direction, Blank } from './common';
import { contiguousTilesFromTileSet } from './scoring';
import { Board } from './board';
import { GameEvent } from '../../gen/macondo/api/proto/macondo/macondo_pb';
import { ClientGameplayEvent } from '../../gen/api/proto/ipc/omgwords_pb';

export const ThroughTileMarker = '.';
// convert a set of ephemeral tiles to a protobuf game event.
export const tilesetToMoveEvent = (
  tiles: Set<EphemeralTile>,
  board: Board,
  gameID: string
) => {
  const ret = contiguousTilesFromTileSet(tiles, board);
  if (ret === null) {
    // the play is not rules-valid. Deal with it in the caller.
    return null;
  }

  const [wordTiles, wordDir] = ret;
  let wordStr = '';
  let wordPos = '';
  let undesignatedBlank = false;
  wordTiles.forEach((t) => {
    wordStr += t.fresh ? t.letter : ThroughTileMarker;
    if (t.letter === Blank) {
      undesignatedBlank = true;
    }
  });
  if (undesignatedBlank) {
    // Play has an undesignated blank. Not valid.
    console.log('Undesignated blank');
    return null;
  }
  const row = String(wordTiles[0].row + 1);
  const col = String.fromCharCode(wordTiles[0].col + 'A'.charCodeAt(0));

  if (wordDir === Direction.Horizontal) {
    wordPos = row + col;
  } else {
    wordPos = col + row;
  }

  const evt = new ClientGameplayEvent();
  evt.setPositionCoords(wordPos);
  evt.setTiles(wordStr);
  evt.setType(ClientGameplayEvent.EventType.TILE_PLACEMENT);
  evt.setGameId(gameID);
  return evt;
};

export const exchangeMoveEvent = (rack: string, gameID: string) => {
  const evt = new ClientGameplayEvent();
  evt.setTiles(rack);
  evt.setType(ClientGameplayEvent.EventType.EXCHANGE);
  evt.setGameId(gameID);

  return evt;
};

export const passMoveEvent = (gameID: string) => {
  const evt = new ClientGameplayEvent();
  evt.setType(ClientGameplayEvent.EventType.PASS);
  evt.setGameId(gameID);

  return evt;
};

export const resignMoveEvent = (gameID: string) => {
  const evt = new ClientGameplayEvent();
  evt.setType(ClientGameplayEvent.EventType.RESIGN);
  evt.setGameId(gameID);

  return evt;
};

export const challengeMoveEvent = (gameID: string) => {
  const evt = new ClientGameplayEvent();
  evt.setType(ClientGameplayEvent.EventType.CHALLENGE_PLAY);
  evt.setGameId(gameID);

  return evt;
};

export const tilePlacementEventDisplay = (evt: GameEvent, board: Board) => {
  // modify a tile placement move for display purposes.
  const row = evt.getRow();
  const col = evt.getColumn();
  const ri = evt.getDirection() === GameEvent.Direction.HORIZONTAL ? 0 : 1;
  const ci = 1 - ri;

  let m = '';
  let openParen = false;
  for (
    let i = 0, r = row, c = col;
    i < evt.getPlayedTiles().length;
    i += 1, r += ri, c += ci
  ) {
    const t = evt.getPlayedTiles()[i];
    if (t === ThroughTileMarker) {
      if (!openParen) {
        m += '(';
        openParen = true;
      }
      m += board.letterAt(r, c);
    } else {
      if (openParen) {
        m += ')';
        openParen = false;
      }
      m += t;
    }
  }
  if (openParen) {
    m += ')';
  }
  return m;
};

export const computeLeave = (tilesPlayed: string, rack: string) => {
  const tileDict: { [k: string]: number } = {};
  for (let i = 0; i < rack.length; i++) {
    const tile = rack[i];
    if (!tileDict[tile]) {
      tileDict[tile] = 1;
    } else {
      tileDict[tile] += 1;
    }
  }
  const leave = [];
  for (let i = 0; i < tilesPlayed.length; i++) {
    let tile = tilesPlayed[i];
    if (tile.toLowerCase() === tile) {
      tile = Blank;
    }
    tileDict[tile] -= 1;
  }
  for (const tile in tileDict) {
    const n = tileDict[tile];
    for (let i = 0; i < n; i++) {
      leave.push(tile);
    }
  }
  leave.sort();
  return leave.join('');
};
