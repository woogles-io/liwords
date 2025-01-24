import { BotRequest_BotCode } from "../gen/api/vendor/macondo/macondo_pb";

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

// isEnglish (but not CEL, that is handled separately)
const isEnglish = (lexicon: string) =>
  lexicon.startsWith("CSW") || lexicon.startsWith("NWL");

export const BotTypesEnumProperties = {
  [BotTypesEnum.GRANDMASTER]: {
    userVisible: "Grandmaster",
    botName: "BestBot",
    shortDescription: "470 point average",
    description: (lexicon: string) =>
      "Our DeepBlueDog, BestBot is the best word-game AI out there",
    botCode: (lexicon: string) => BotRequest_BotCode.SIMMING_BOT,
    image: bestbot,
    voidonly: false,
  },
  [BotTypesEnum.MASTER]: {
    userVisible: "Master",
    botName: "HastyBot",
    shortDescription: "460 point average",
    description: (lexicon: string) =>
      "Knows all the words; always makes the best play before simulation",
    botCode: (lexicon: string) => BotRequest_BotCode.HASTY_BOT,
    image: hastybot,
    voidonly: false,
  },
  [BotTypesEnum.EXPERT]: {
    userVisible: "Expert",
    botName: "STEEBot",
    shortDescription: "410 point average",
    description: (lexicon: string) =>
      "Knows all the words, but will make some strategic mistakes",
    botCode: (lexicon: string) => BotRequest_BotCode.LEVEL4_PROBABILISTIC,
    image: steebot,
    voidonly: false,
  },
  [BotTypesEnum.INTERMEDIATE]: {
    userVisible: "Intermediate",
    botName: "BetterBot",
    shortDescription: "370 point average",
    description: (lexicon: string) =>
      isEnglish(lexicon)
        ? "Perfect play while still only using common words"
        : "BetterBot. A bit better than BasicBot.",
    botCode: (lexicon: string) =>
      isEnglish(lexicon)
        ? BotRequest_BotCode.LEVEL4_CEL_BOT
        : BotRequest_BotCode.LEVEL3_PROBABILISTIC,
    image: betterbot,
    voidonly: true,
  },
  [BotTypesEnum.EASY]: {
    userVisible: "Basic",
    botName: "BasicBot",
    shortDescription: "330 point average",
    description: (lexicon: string) =>
      isEnglish(lexicon)
        ? "Higher scoring, but still emphasizes common English words only"
        : "Beating BeginnerBot? Basicbot is your next frenemy, scoring more.",
    botCode: (lexicon: string) =>
      isEnglish(lexicon)
        ? BotRequest_BotCode.LEVEL2_CEL_BOT
        : BotRequest_BotCode.LEVEL2_PROBABILISTIC,
    image: basicbot,
    voidonly: true,
  },
  [BotTypesEnum.BEGINNER]: {
    userVisible: "Beginner",
    botName: "BeginnerBot",
    shortDescription: "240 point average",
    description: (lexicon: string) =>
      isEnglish(lexicon)
        ? "New to OMGWords? Low scoring plays and common words only"
        : "New to OMGWords? BeginnerBot sticks to lower-scoring plays.",
    botCode: (lexicon: string) =>
      isEnglish(lexicon)
        ? BotRequest_BotCode.LEVEL1_CEL_BOT
        : BotRequest_BotCode.LEVEL1_PROBABILISTIC,
    image: beginnerbot,
    voidonly: true,
  },
};
