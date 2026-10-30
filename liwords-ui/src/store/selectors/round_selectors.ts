// Derived state for lagged pairings. A round's pairing lag is 0 (paired from
// results through the previous round) or 1 (paired from results through two
// rounds back, so it can open while the previous round is still being
// played). Nothing here is sent by the server: it is all derived from the
// current round, the round controls and the pairing outcomes. With every lag
// at 0, openRounds is [currentRound] and playerActiveRound is currentRound.
//
// Rounds are 0-based here, as in the rest of the store.
//
// This module only imports types from the reducer, so that the reducer can
// use it without an import cycle.

import { TournamentGameResult } from "../../gen/api/proto/ipc/tournament_pb";
import type { Division, SinglePairing } from "../reducers/tournament_reducer";

export const pairingLag = (division: Division, round: number): number =>
  division.roundControls?.[round]?.pairingLag ?? 0;

export const isLaggedRound = (division: Division, round: number): boolean =>
  pairingLag(division, round) > 0;

export const hasLaggedRounds = (division: Division): boolean =>
  division.roundControls?.some((rc) => rc.pairingLag > 0) ?? false;

// The last round whose results feed the given round's pairings. A negative
// basis means the round is paired from the initial standings (by seed).
// lag defaults to the round's current pairing lag.
export const pairingBasisRound = (
  division: Division,
  round: number,
  lag: number = pairingLag(division, round),
): number => round - 1 - lag;

export const pairingHasResult = (pairing: SinglePairing | undefined): boolean =>
  !!pairing?.players &&
  pairing.outcomes[0] !== TournamentGameResult.NO_RESULT &&
  pairing.outcomes[1] !== TournamentGameResult.NO_RESULT;

// A round is complete when every player in the division has a pairing with a
// result, which is the same rule the server uses.
export const isRoundComplete = (division: Division, round: number): boolean => {
  const roundPairings = division.pairings?.[round]?.roundPairings;
  if (!roundPairings) {
    return false;
  }
  return division.players.every((_, idx) =>
    pairingHasResult(roundPairings[idx]),
  );
};

// True when rounds 0..round are all complete. A negative round is trivially
// complete.
export const roundsCompleteThrough = (
  division: Division,
  round: number,
): boolean => {
  for (let r = 0; r <= round; r++) {
    if (!isRoundComplete(division, r)) {
      return false;
    }
  }
  return true;
};

// The rounds that are open for play: currentRound, plus currentRound - 1
// when currentRound is lagged and the round before it is still in progress.
// Empty before the tournament starts.
export const openRounds = (division: Division): Array<number> => {
  const { currentRound } = division;
  if (currentRound < 0) {
    return [];
  }
  if (
    currentRound > 0 &&
    isLaggedRound(division, currentRound) &&
    !isRoundComplete(division, currentRound - 1)
  ) {
    return [currentRound - 1, currentRound];
  }
  return [currentRound];
};

export const isOpenRound = (division: Division, round: number): boolean =>
  openRounds(division).includes(round);

// Whether a round that just opened should call the player to it, with a
// sound: not if they are in the middle of a game, or if they still owe a
// game in the round before it, as happens when a lagged round opens before
// the round ahead of it is finished.
export const isCalledToRound = (
  division: Division,
  round: number,
  fullPlayerID: string,
  inGame: boolean,
): boolean =>
  !inGame &&
  playerActiveRound({ ...division, currentRound: round }, fullPlayerID) ===
    round;

// The round a player is playing now: the earliest open round where their
// pairing has no result yet, or else currentRound. A player can only ready up
// for a round once their previous round's game has a result, so this is the
// only round they can act on.
export const playerActiveRound = (
  division: Division,
  fullPlayerID: string,
): number => {
  const playerIdx = division.playerIndexMap[fullPlayerID];
  if (playerIdx !== undefined) {
    for (const round of openRounds(division)) {
      const pairing = division.pairings?.[round]?.roundPairings[playerIdx];
      if (pairing?.players && !pairingHasResult(pairing)) {
        return round;
      }
    }
  }
  return division.currentRound;
};
