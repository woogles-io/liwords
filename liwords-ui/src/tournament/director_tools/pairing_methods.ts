// Different pairing methods should show different options to the director.

import {
  PairingMethod,
  RoundControl,
} from "../../gen/api/proto/ipc/tournament_pb";

// Whether the method pairs without looking at results, so its rounds are
// paired in advance. Mirrors pair.IsStandingsIndependent on the server.
export const isPrePaired = (m: PairingMethod): boolean =>
  m === PairingMethod.ROUND_ROBIN ||
  m === PairingMethod.TEAM_ROUND_ROBIN ||
  m === PairingMethod.INTERLEAVED_ROUND_ROBIN ||
  m === PairingMethod.SNAKED_ROUND_ROBIN ||
  m === PairingMethod.ASSIGNED_TEAM_ROUND_ROBIN ||
  m === PairingMethod.RANDOM ||
  m === PairingMethod.INITIAL_FONTES ||
  m === PairingMethod.MANUAL;

export type RoundSetting = {
  beginRound: number;
  endRound: number;
  setting: RoundControl;
};

export const settingsEqual = (s1: RoundControl, s2: RoundControl): boolean => {
  return (
    s1.pairingMethod === s2.pairingMethod &&
    s1.gamesPerRound === s2.gamesPerRound &&
    s1.factor === s2.factor &&
    s1.maxRepeats === s2.maxRepeats &&
    s1.allowOverMaxRepeats === s2.allowOverMaxRepeats &&
    s1.repeatRelativeWeight === s2.repeatRelativeWeight &&
    s1.winDifferenceRelativeWeight === s2.winDifferenceRelativeWeight &&
    s1.resetRound === s2.resetRound &&
    s1.pairingLag === s2.pairingLag &&
    s1.playWithinTeam === s2.playWithinTeam //&&
    //    s1.controlLossActivationRound === s2.controlLossActivationRound &&
    //    s1.controlLossSims === s2.controlLossSims &&
    //    s1.controlLossThreshold === s2.controlLossThreshold &&
    //    s1.divisionSims === s2.divisionSims &&
    //    s1.gibsonSpreads === s2.gibsonSpreads &&
    // s1.hopefulnessThresholds === s2.hopefulnessThresholds &&
    // s1.placePrizes === s2.placePrizes
  );
};

// [fieldType, fieldName, displayName, help, minValue?]. minValue is the
// smallest value the director may enter (and the display default when the
// stored value is unset); it defaults to 0 when omitted.
export type PairingMethodField = [
  string,
  keyof RoundControl,
  string,
  string,
  number?,
];

export const fieldsForMethod = (
  m: PairingMethod,
): Array<PairingMethodField> => {
  const fields = new Array<PairingMethodField>();
  switch (m) {
    case PairingMethod.RANDOM:
    case PairingMethod.ROUND_ROBIN:
    case PairingMethod.KING_OF_THE_HILL:
    case PairingMethod.MANUAL:
    case PairingMethod.INITIAL_FONTES:
      return [];
    // @ts-expect-error fallthrough is purposeful:
    case PairingMethod.FACTOR:
      fields.push([
        "number",
        "factor",
        "Factor",
        "Your selected factor (use 2 for 1v3, 2v4 for example).",
      ]);
    case PairingMethod.SWISS:
      fields.push(
        [
          "number",
          "maxRepeats",
          "Max Pairings Between \nAny Two Players",
          'Use "1" for no repeats, "2" for 1 max repeat, and so on. ' +
            "The pairing system will try to meet your requirement, but it is not guaranteed.",
        ],
        [
          "number",
          "repeatRelativeWeight",
          "Repeat Relative Weight",
          "The larger this number relative to other weights, the less likely a repeat will be. Choose a whole number between 1 and 100.",
        ],
        [
          "number",
          "winDifferenceRelativeWeight",
          "Win Difference Relative Weight",
          "The smaller this number relative to other weights, the more mismatched your pairings will be, in terms of win difference. Choose a whole number between 1 and 100.",
        ],
      );
      break;

    case PairingMethod.TEAM_ROUND_ROBIN:
    case PairingMethod.ASSIGNED_TEAM_ROUND_ROBIN:
      fields.push([
        "number",
        "gamesPerRound",
        "Games per Round",
        "The number of games per round. For example, set this to two if you wish each team member to play the other team member twice.",
      ]);
      break;

    case PairingMethod.INTERLEAVED_ROUND_ROBIN:
    case PairingMethod.SNAKED_ROUND_ROBIN:
      fields.push([
        "boolean",
        "playWithinTeam",
        "Play Within Team",
        "This pairing system divides players into two 'teams'. If this option is selected, " +
          "players will play a round robin within their own team. If false, players will play " +
          "against players from the other team. When both teams have an odd number of players, " +
          "the two players who would otherwise sit out a round play each other instead, so each " +
          "player plays one game per round robin against the other team.",
      ]);
      break;

    case PairingMethod.AUSTRALIAN_DRAW:
      fields.push([
        "number",
        "resetRound",
        "Reset Round",
        "Reset round N: opponents from rounds before N may be re-paired; round N " +
          "onward avoids repeats. N=1 avoids all repeats. The first-round draw only " +
          "ever happens in round 1 -- a later reset does NOT re-trigger it. " +
          "Example: N=9 for rounds 9-16 lets round 1-8 opponents recur.",
        1,
      ]);
      break;

    case PairingMethod.WIN_GROUPS:
      fields.push([
        "number",
        "resetRound",
        "Reset Round",
        "Reset round N: opponents from rounds before N may be re-paired; round N " +
          "onward avoids repeats. N=1 avoids all repeats. " +
          "Example: N=9 for rounds 9-16 lets round 1-8 opponents recur.",
        1,
      ]);
      break;

    case PairingMethod.PAIRING_METHOD_COP:
      // COP fields are handled by a custom component, not this field system
      return [];
  }

  return fields;
};
