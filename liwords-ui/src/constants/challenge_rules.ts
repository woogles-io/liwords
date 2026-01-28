import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";

export const challengeRuleNames = {
  [ChallengeRule.FIVE_POINT]: "5 point",
  [ChallengeRule.TEN_POINT]: "10 point",
  [ChallengeRule.SINGLE]: "Single",
  [ChallengeRule.DOUBLE]: "Double",
  [ChallengeRule.TRIPLE]: "Triple",
  [ChallengeRule.VOID]: "Void",
};

export const challengeRuleNamesShort = {
  [ChallengeRule.FIVE_POINT]: "+5",
  [ChallengeRule.TEN_POINT]: "+10",
  [ChallengeRule.SINGLE]: "x1",
  [ChallengeRule.DOUBLE]: "x2",
  [ChallengeRule.TRIPLE]: "x3",
  [ChallengeRule.VOID]: "Void",
};
