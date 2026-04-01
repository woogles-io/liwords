import { BotRequest_BotCode } from "../gen/api/proto/vendored/macondo/macondo_pb";

import bestbot from "../assets/bots/best.png";
import hastybot from "../assets/bots/macondog.png";
import steebot from "../assets/bots/stee.png";
import betterbot from "../assets/bots/better.png";
import basicbot from "../assets/bots/basic.png";
import beginnerbot from "../assets/bots/beginner.png";

export enum BotTypesEnum {
  MASTER,
  EXPERT,
  INTERMEDIATE,
  EASY,
  BEGINNER,
  GRANDMASTER,
}

const hasCommonWordSublexicon = (lexicon: string) =>
  lexicon.startsWith("CSW") ||
  lexicon.startsWith("NWL") ||
  lexicon.startsWith("RD");

export const BotTypesEnumProperties = {
  [BotTypesEnum.GRANDMASTER]: {
    userVisible: "BestBot",
    botName: "BestBot",
    shortDescription: "470 point average",
    description: (lexicon: string) =>
      "Our DeepBlueDog, BestBot is the best word-game AI out there",
    botCode: (lexicon: string) => BotRequest_BotCode.SIMMING_BOT,
    image: bestbot,
    voidonly: false,
  },
  [BotTypesEnum.MASTER]: {
    userVisible: "HastyBot",
    botName: "HastyBot",
    shortDescription: "460 point average",
    description: (lexicon: string) =>
      "Knows all the words; always makes the best play before simulation",
    botCode: (lexicon: string) => BotRequest_BotCode.HASTY_BOT,
    image: hastybot,
    voidonly: false,
  },
  [BotTypesEnum.EXPERT]: {
    userVisible: "STEEBot",
    botName: "STEEBot",
    shortDescription: "410 point average",
    description: (lexicon: string) =>
      "Knows all the words, but will make some strategic mistakes",
    botCode: (lexicon: string) => BotRequest_BotCode.LEVEL4_PROBABILISTIC,
    image: steebot,
    voidonly: false,
  },
  [BotTypesEnum.INTERMEDIATE]: {
    userVisible: "BetterBot",
    botName: "BetterBot",
    shortDescription: "370 point average",
    description: (lexicon: string) =>
      hasCommonWordSublexicon(lexicon)
        ? "Perfect play while still only using common words"
        : "BetterBot. A bit better than BasicBot",
    botCode: (lexicon: string) =>
      hasCommonWordSublexicon(lexicon)
        ? BotRequest_BotCode.LEVEL4_COMMON_WORD_BOT
        : BotRequest_BotCode.LEVEL3_PROBABILISTIC,
    image: betterbot,
    voidonly: false,
  },
  [BotTypesEnum.EASY]: {
    userVisible: "BasicBot",
    botName: "BasicBot",
    shortDescription: "330 point average",
    description: (lexicon: string) =>
      hasCommonWordSublexicon(lexicon)
        ? lexicon.startsWith("RD")
          ? "Higher scoring, but still emphasizes common German words only"
          : "Higher scoring, but still emphasizes common English words only"
        : "Beating BeginnerBot? Basicbot is your next frenemy, scoring more",
    botCode: (lexicon: string) =>
      hasCommonWordSublexicon(lexicon)
        ? BotRequest_BotCode.LEVEL2_COMMON_WORD_BOT
        : BotRequest_BotCode.LEVEL2_PROBABILISTIC,
    image: basicbot,
    voidonly: false,
  },
  [BotTypesEnum.BEGINNER]: {
    userVisible: "BeginnerBot",
    botName: "BeginnerBot",
    shortDescription: "240 point average",
    description: (lexicon: string) =>
      hasCommonWordSublexicon(lexicon)
        ? "New to OMGWords? Low scoring plays and common words only"
        : "New to OMGWords? BeginnerBot sticks to lower-scoring plays",
    botCode: (lexicon: string) =>
      hasCommonWordSublexicon(lexicon)
        ? BotRequest_BotCode.LEVEL1_COMMON_WORD_BOT
        : BotRequest_BotCode.LEVEL1_PROBABILISTIC,
    image: beginnerbot,
    voidonly: true,
  },
};
