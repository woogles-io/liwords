import { Division, SinglePairing } from "../store/reducers/tournament_reducer";
import {
  pairingBasisRound,
  roundsCompleteThrough,
} from "../store/selectors/round_selectors";

// The tables of a round played over the board, numbered the usual way: table
// 1 holds the game of the best-placed player, table 2 the game of the
// best-placed player not already seated, and so on.
//
// Places come from the standings the round was paired from, or by seed before
// any results, never from the round's own results: those come in while the
// round is played, and a table must keep its number while players sit at it.
// For the same reason it is the same for everyone looking, whoever they are.
// A round paired ahead of those standings (a round robin's later rounds, say)
// has no numbers until they are final, rather than numbers that would move.
//
// A bye, a forfeit, or a round not played, which pairs a player with no one
// else, takes no table.
export const tableNumbers = (
  division: Division,
  round: number,
  pairings: Array<SinglePairing>,
): Map<SinglePairing, number> | undefined => {
  const basis = pairingBasisRound(division, round);
  if (!roundsCompleteThrough(division, basis)) {
    return undefined;
  }
  const places = new Map<string, number>();
  if (basis >= 0) {
    division.standingsMap[basis]?.standings.forEach((s, idx) =>
      places.set(s.playerId, idx),
    );
  }
  // Anyone those standings leave out places after everyone in them, by seed.
  const place = (playerID: string) =>
    places.get(playerID) ??
    places.size + (division.playerIndexMap[playerID] ?? 0);
  const tables = new Map<SinglePairing, number>();
  pairings
    .filter((p) => p.players[0].id !== p.players[1].id)
    .map((pairing) => ({
      pairing,
      best: Math.min(
        place(pairing.players[0].id),
        place(pairing.players[1].id),
      ),
    }))
    .sort((a, b) => a.best - b.best)
    .forEach(({ pairing }, idx) => tables.set(pairing, idx + 1));
  return tables;
};
