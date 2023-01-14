import { EphemeralTile, Direction, Blank, EmptySpace, isBlank } from './common';
import { contiguousTilesFromTileSet } from './scoring';
import { Board } from './board';
import {
  GameEvent,
  GameEvent_Direction,
} from '../../gen/api/proto/macondo/macondo_pb';
import {
  ClientGameplayEvent,
  ClientGameplayEvent_EventType,
  PlayerInfo,
} from '../../gen/api/proto/ipc/omgwords_pb';
import { indexToPlayerOrder, PlayerOrder } from '../../store/constants';

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

  const evt = new ClientGameplayEvent({
    positionCoords: wordPos,
    tiles: wordStr,
    type: ClientGameplayEvent_EventType.TILE_PLACEMENT,
    gameId: gameID,
  });

  return evt;
};

export const exchangeMoveEvent = (rack: string, gameID: string) => {
  const evt = new ClientGameplayEvent({
    tiles: rack,
    type: ClientGameplayEvent_EventType.EXCHANGE,
    gameId: gameID,
  });

  return evt;
};

export const passMoveEvent = (gameID: string) => {
  const evt = new ClientGameplayEvent({
    type: ClientGameplayEvent_EventType.PASS,
    gameId: gameID,
  });
  return evt;
};

export const resignMoveEvent = (gameID: string) => {
  const evt = new ClientGameplayEvent({
    type: ClientGameplayEvent_EventType.RESIGN,
    gameId: gameID,
  });
  return evt;
};

export const challengeMoveEvent = (gameID: string) => {
  const evt = new ClientGameplayEvent({
    type: ClientGameplayEvent_EventType.CHALLENGE_PLAY,
    gameId: gameID,
  });
  return evt;
};

export const tilePlacementEventDisplay = (evt: GameEvent, board: Board) => {
  // modify a tile placement move for display purposes.
  const row = evt.row;
  const col = evt.column;
  const ri = evt.direction === GameEvent_Direction.HORIZONTAL ? 0 : 1;
  const ci = 1 - ri;

  let m = '';
  let openParen = false;
  for (
    let i = 0, r = row, c = col;
    i < evt.playedTiles.length;
    i += 1, r += ri, c += ci
  ) {
    const t = evt.playedTiles[i];
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

// nicknameFromEvt gets the nickname of the user who performed an
// event.
export const nicknameFromEvt = (
  evt: GameEvent,
  players: Array<PlayerInfo>
): string => {
  return players[evt.playerIndex]?.nickname;
};

export const playerOrderFromEvt = (
  evt: GameEvent,
  nickToPlayerOrder: { [nick: string]: PlayerOrder }
): PlayerOrder => {
  return indexToPlayerOrder(evt.playerIndex);
};

export const computeLeave = (tilesPlayed: string, rack: string) => {
  // tilesPlayed is either from evt.getPlayedTiles(), which is like "TRUNCa.E",
  // or from evt.getExchanged(), which is like "AE?".
  // rack is a pre-sorted rack; spaces will be returned where gaps should be.

  const leave: Array<string | null> = Array.from(rack);
  for (const letter of tilesPlayed) {
    if (letter !== ThroughTileMarker) {
      const t = isBlank(letter) ? Blank : letter;
      const usedTileIndex = leave.lastIndexOf(t);
      if (usedTileIndex >= 0) {
        // make it a non-string to disqualify multiple matches in this loop.
        leave[usedTileIndex] = null;
      }
    }
  }
  for (let i = 0; i < leave.length; ++i) {
    if (leave[i] === null) {
      // this is intentionally done in a separate pass.
      leave[i] = EmptySpace;
    }
  }
  return leave.join('');
};
