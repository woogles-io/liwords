import { AnalyzerMove } from "./analyzer";

// lineEndsGame reports whether an endgame line, played out from the turn it
// was solved for, actually finishes the game.
//
// The solver can return a line that stops early while its spread still counts
// the moves after it (a transposition-table hit hands back a value without the
// moves behind it). Summing the moves of such a line gives the wrong total, so
// the UI says when one is cut short instead of presenting it as complete.
//
// moverTiles and oppTiles are how many tiles each side holds when the line
// starts; with the bag empty, every tile off the board is on one of the racks.
// A line ends the game when a side plays out its last tile, or on two passes
// in a row (the solver plays endgames with that rule).
export const lineEndsGame = (
  moves: ReadonlyArray<AnalyzerMove>,
  moverTiles: number,
  oppTiles: number,
): boolean => {
  const onRack = [moverTiles, oppTiles];
  let consecutivePasses = 0;
  for (let i = 0; i < moves.length; i++) {
    const m = moves[i];
    // Through-squares are 0 in a placement's tiles; an exchange or pass lays
    // nothing.
    const laid = m.isExchange ? 0 : m.tiles.filter((t) => t !== 0).length;
    if (laid === 0) {
      consecutivePasses++;
      if (consecutivePasses >= 2) return true;
      continue;
    }
    consecutivePasses = 0;
    const side = i % 2;
    onRack[side] -= laid;
    if (onRack[side] <= 0) return true;
  }
  return false;
};
