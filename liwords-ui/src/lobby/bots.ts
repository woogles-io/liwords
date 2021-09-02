import { BotRequest } from '../gen/macondo/api/proto/macondo/macondo_pb';

export enum BotTypesEnum {
  MASTER,
  EXPERT,
  INTERMEDIATE,
  EASY,
  BEGINNER,
}

// isEnglish (but not CEL, that is handled separately)
const isEnglish = (lexicon: string) =>
  lexicon.startsWith('CSW') || lexicon.startsWith('NWL');

export const BotTypesEnumProperties = {
  [BotTypesEnum.MASTER]: {
    userVisible: 'Master',
    botName: 'HastyBot',
    shortDescription: '460 point average',
    description: (lexicon: string) =>
      'The Bot, the myth, the legend. HastyBot always finds the best play.',
    botCode: (lexicon: string) => BotRequest.BotCode.HASTY_BOT,
  },
  [BotTypesEnum.EXPERT]: {
    userVisible: 'Expert',
    botName: 'STEEBot',
    shortDescription: '410 point average',
    description: (lexicon: string) =>
      isEnglish(lexicon)
        ? 'Ready for the weird words? Not quite an expert, STEEBot knows all the words but will make some mistakes.'
        : 'Not quite an expert, STEEBot knows all the words but will make mistakes.',
    botCode: (lexicon: string) => BotRequest.BotCode.LEVEL4_PROBABILISTIC,
  },
  [BotTypesEnum.INTERMEDIATE]: {
    userVisible: 'Intermediate',
    botName: 'BetterBot',
    shortDescription: '370 point average',
    description: (lexicon: string) =>
      isEnglish(lexicon)
        ? 'BetterBot is the best bot for common-words only, with perfect play compared to its lower-rated counterparts.'
        : 'BetterBot. A bit better than BasicBot.',
    botCode: (lexicon: string) =>
      isEnglish(lexicon)
        ? BotRequest.BotCode.LEVEL4_CEL_BOT
        : BotRequest.BotCode.LEVEL3_PROBABILISTIC,
  },
  [BotTypesEnum.EASY]: {
    userVisible: 'Basic',
    botName: 'BasicBot',
    shortDescription: '330 point average',
    description: (lexicon: string) =>
      isEnglish(lexicon)
        ? 'Beating Beginnerbot? Basicbot is your next frenemy, scoring more, but still emphasizing common English words.'
        : 'Beating Beginnerbot? Basicbot is your next frenemy, scoring more.',
    botCode: (lexicon: string) =>
      isEnglish(lexicon)
        ? BotRequest.BotCode.LEVEL2_CEL_BOT
        : BotRequest.BotCode.LEVEL2_PROBABILISTIC,
  },
  [BotTypesEnum.BEGINNER]: {
    userVisible: 'Beginner',
    botName: 'BeginnerBot',
    shortDescription: '240 point average',
    description: (lexicon: string) =>
      isEnglish(lexicon)
        ? 'New to OMGWords? Beginnerbot sticks to lower-scoring plays and common words'
        : 'New to OMGWords? Beginnerbot sticks to lower-scoring plays',
    botCode: (lexicon: string) =>
      isEnglish(lexicon)
        ? BotRequest.BotCode.LEVEL1_CEL_BOT
        : BotRequest.BotCode.LEVEL1_PROBABILISTIC,
  },
};
