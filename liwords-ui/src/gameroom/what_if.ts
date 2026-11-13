import { create } from "@bufbuild/protobuf";
import { Alphabet, machineWordToRunes } from "../constants/alphabets";
import {
  GameEvent,
  GameEventSchema,
  GameEvent_Direction,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { isSpanish } from "../store/constants";
import { GameState } from "../store/reducers/game_reducer";
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
// rack; the other player's score and rack; the tiles in the bag, and the
// tiles unseen (both racks among them).
export type WhatIfSide = {
  playerIndex: number;
  score: number;
  rack: Array<MachineLetter>;
  otherScore: number;
  otherRack: Array<MachineLetter>;
  bag: number;
  unseen: Array<MachineLetter>;
};

// The most tiles a rack holds.
export const RACK_SIZE = 7;

// Whether the mover can hold the tiles placed: each from the rack they are
// known to hold, or, for the part of the rack not known (fewer tiles known
// than a full rack, the bag allowing), from the tiles in the bag. A tile
// placed twice that the game has once is not theirs to place.
const holdsPlaced = (placed: Set<EphemeralTile>, side: WhatIfSide) => {
  const rack = [...side.rack];
  const bagTiles = [...side.unseen];
  for (const t of [...side.rack, ...side.otherRack]) {
    const at = bagTiles.indexOf(t);
    if (at >= 0) bagTiles.splice(at, 1);
  }
  let unknown = Math.min(RACK_SIZE - side.rack.length, side.bag);
  for (const { letter } of placed) {
    const t = (letter & 0x80 ? 0 : letter) as MachineLetter;
    const at = rack.indexOf(t);
    if (at >= 0) {
      rack.splice(at, 1);
      continue;
    }
    const inBag = bagTiles.indexOf(t);
    if (unknown <= 0 || inBag < 0) return false;
    bagTiles.splice(inBag, 1);
    unknown -= 1;
  }
  return true;
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
// play, or are not the mover's to place.
export const whatIfPlay = (
  placed: Set<EphemeralTile>,
  board: Board,
  alphabet: Alphabet,
  side: WhatIfSide,
): Array<GameEvent> | null => {
  const run = placed.size ? contiguousTilesFromTileSet(placed, board) : null;
  const score = calculateTemporaryScore(placed, board, alphabet);
  if (!run || score === undefined || !holdsPlaced(placed, side)) return null;
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

// The fewest tiles the bag must hold for an exchange.
export const EXCHANGE_MIN_BAG = 7;
// Spanish's: a tile in the bag is enough.
export const EXCHANGE_MIN_BAG_SPANISH = 1;

export const exchangeMinBag = (lexicon: string) =>
  isSpanish(lexicon) ? EXCHANGE_MIN_BAG_SPANISH : EXCHANGE_MIN_BAG;

// The event of exchanging the tiles put back, the score as it was. Null
// unless the tiles are some of the mover's own and the bag holds enough for
// an exchange.
export const whatIfExchange = (
  back: Array<MachineLetter>,
  alphabet: Alphabet,
  side: WhatIfSide,
  minBag: number,
): Array<GameEvent> | null => {
  if (!back.length || side.bag < minBag) return null;
  const left = [...side.rack];
  for (const t of back) {
    const at = left.indexOf(t & 0x80 ? 0 : t);
    if (at < 0) return null;
    left.splice(at, 1);
  }
  return [
    create(GameEventSchema, {
      type: GameEvent_Type.EXCHANGE,
      playerIndex: side.playerIndex,
      rack: machineWordToRunes(side.rack, alphabet),
      exchanged: machineWordToRunes(
        back.map((t) => (t & 0x80 ? 0 : t) as MachineLetter),
        alphabet,
      ),
      cumulative: side.score,
    }),
  ];
};

// The tiles the mover draws after a move that takes count tiles off the
// rack, as asked: exactly that many, the bag allowing, each one still unseen
// once the mover's own rack (the tiles moved among them) is set aside. Null
// when the tiles asked are not that.
export const whatIfDraw = (
  asked: Array<MachineLetter>,
  count: number,
  rack: Array<MachineLetter>,
  unseen: Array<MachineLetter>,
  bag: number,
): Array<MachineLetter> | null => {
  if (asked.length !== Math.min(count, bag)) return null;
  const pool = [...unseen];
  for (const t of rack) {
    const at = pool.indexOf(t);
    if (at >= 0) pool.splice(at, 1);
  }
  for (const t of asked) {
    const at = pool.indexOf(t);
    if (at < 0) return null;
    pool.splice(at, 1);
  }
  return asked;
};

// The player to move in a position, as a made-up move needs them, and the
// tiles unseen there (the racks among them); null with other than two
// players.
export const whatIfSideOf = (
  gs: GameState,
): { side: WhatIfSide; unseen: Array<MachineLetter> } | null => {
  if (gs.players.length !== 2) return null;
  const me = gs.players[gs.onturn];
  const other = gs.players[1 - gs.onturn];
  const unseen: Array<MachineLetter> = [];
  for (const [t, n] of Object.entries(gs.pool)) {
    for (let i = 0; i < n; ++i) unseen.push(Number(t) as MachineLetter);
  }
  return {
    side: {
      playerIndex: gs.onturn,
      score: me.score,
      rack: me.currentRack,
      otherScore: other.score,
      otherRack: other.currentRack,
      bag: Math.max(
        0,
        unseen.length - me.currentRack.length - other.currentRack.length,
      ),
      unseen,
    },
    unseen,
  };
};

// The tiles a play the wasm writes places, as tiles placed on the board: a
// wasm blank (negative) as a designated blank.
export const placedOfPlay = (play: {
  down: boolean;
  lane: number;
  idx: number;
  word: Array<number>;
}) => {
  const placed = new Set<EphemeralTile>();
  play.word.forEach((t, i) => {
    if (t === 0) return;
    placed.add({
      row: play.down ? play.idx + i : play.lane,
      col: play.down ? play.lane : play.idx + i,
      letter: (t < 0 ? -t | 0x80 : t) as MachineLetter,
    });
  });
  return placed;
};
