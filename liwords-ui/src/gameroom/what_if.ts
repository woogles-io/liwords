import { create } from "@bufbuild/protobuf";
import { Alphabet, machineWordToRunes } from "../constants/alphabets";
import {
  GameEvent,
  GameEventSchema,
  GameEvent_Direction,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { Board } from "../utils/cwgame/board";
import {
  Direction,
  EphemeralTile,
  MachineLetter,
} from "../utils/cwgame/common";
import {
  calculateTemporaryScore,
  contiguousTilesFromTileSet,
} from "../utils/cwgame/scoring";

// The player to move, as a made-up move needs them: their index, score and
// rack; and the other player's score and rack, and the tiles in the bag.
export type WhatIfSide = {
  playerIndex: number;
  score: number;
  rack: Array<MachineLetter>;
  otherScore: number;
  otherRack: Array<MachineLetter>;
  bag: number;
};

// The points a rack's tiles are worth, a blank nothing.
const rackPoints = (rack: Array<MachineLetter>, alphabet: Alphabet) =>
  rack.reduce(
    (sum, t) => sum + (t & 0x80 ? 0 : (alphabet.letters[t]?.score ?? 0)),
    0,
  );

// The events of playing the tiles placed, scored by the page alone (no word
// checked): the play, and, when it empties the rack with the bag empty, twice
// the other rack's points for going out. Null when the tiles placed make no
// play.
export const whatIfPlay = (
  placed: Set<EphemeralTile>,
  board: Board,
  alphabet: Alphabet,
  side: WhatIfSide,
): Array<GameEvent> | null => {
  const run = placed.size ? contiguousTilesFromTileSet(placed, board) : null;
  const score = calculateTemporaryScore(placed, board, alphabet);
  if (!run || score === undefined) return null;
  const [tiles, direction] = run;
  const rackRunes = machineWordToRunes(side.rack, alphabet);
  const cumulative = side.score + score;
  const play = create(GameEventSchema, {
    type: GameEvent_Type.TILE_PLACEMENT_MOVE,
    playerIndex: side.playerIndex,
    rack: rackRunes,
    row: tiles[0].row,
    column: tiles[0].col,
    direction:
      direction === Direction.Vertical
        ? GameEvent_Direction.VERTICAL
        : GameEvent_Direction.HORIZONTAL,
    playedTiles: machineWordToRunes(
      tiles.map((t) => (t.fresh ? t.letter : 0)),
      alphabet,
      true,
      true,
    ),
    score,
    cumulative,
    isBingo: placed.size === 7,
  });
  const out = side.bag === 0 && placed.size === side.rack.length;
  if (!out) return [play];
  const endRackPoints = 2 * rackPoints(side.otherRack, alphabet);
  return [
    play,
    create(GameEventSchema, {
      type: GameEvent_Type.END_RACK_PTS,
      playerIndex: side.playerIndex,
      rack: machineWordToRunes(side.otherRack, alphabet),
      endRackPoints,
      cumulative: cumulative + endRackPoints,
    }),
  ];
};

// The event of passing, the score as it was.
export const whatIfPass = (
  alphabet: Alphabet,
  side: WhatIfSide,
): Array<GameEvent> => [
  create(GameEventSchema, {
    type: GameEvent_Type.PASS,
    playerIndex: side.playerIndex,
    rack: machineWordToRunes(side.rack, alphabet),
    cumulative: side.score,
  }),
];
