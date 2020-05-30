import { EphemeralTile, Direction } from './common';
import { ClientGameplayEvent } from '../../gen/api/proto/game_service_pb';
import { contiguousTilesFromTileSet } from './scoring';
import { Board } from './game';

const ThroughTileMarker = '.';
// convert a set of ephemeral tiles to a protobuf game event.
export const tilesetToMoveEvent = (tiles: Set<EphemeralTile>, board: Board) => {
  const ret = contiguousTilesFromTileSet(tiles, board);
  if (ret === null) {
    // the play is not rules-valid. Deal with it in the caller.
    return null;
  }

  const [wordTiles, wordDir] = ret;
  let wordStr = '';
  let wordPos = '';
  wordTiles.forEach((t) => {
    wordStr += t.fresh ? t.letter : ThroughTileMarker;
  });

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

  return evt;
};
