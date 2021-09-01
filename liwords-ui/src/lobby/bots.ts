import { BotRequest } from '../gen/macondo/api/proto/macondo/macondo_pb';

export enum BotTypesEnum {
  MASTER,
  EXPERT,
  INTERMEDIATE,
  EASY,
  BEGINNER,
}

export const BotTypesEnumProperties = {
  [BotTypesEnum.MASTER]: {
    userVisible: 'Master',
    botName: 'HastyBot',
    description: 'HastyBot knows all the wordz, some more text here',
    botCode: (lexicon: string) => BotRequest.BotCode.HASTY_BOT,
  },
  [BotTypesEnum.EXPERT]: {
    userVisible: 'Expert',
    botName: 'FOOBot',
    description: 'Etc',
    botCode: (lexicon: string) => BotRequest.BotCode.LEVEL4_PROBABILISTIC,
  },
  [BotTypesEnum.INTERMEDIATE]: {
    userVisible: 'Intermediate',
    botName: 'FOOBot',
    description: 'Etc',
    botCode: (lexicon: string) =>
      lexicon.startsWith('CSW') || lexicon.startsWith('NWL')
        ? BotRequest.BotCode.LEVEL4_CEL_BOT
        : BotRequest.BotCode.LEVEL3_PROBABILISTIC,
  },
  [BotTypesEnum.EASY]: {
    userVisible: 'Easy',
    botName: 'FOOBot',
    description: 'Etc',
    botCode: (lexicon: string) =>
      lexicon.startsWith('CSW') || lexicon.startsWith('NWL')
        ? BotRequest.BotCode.LEVEL2_CEL_BOT
        : BotRequest.BotCode.LEVEL2_PROBABILISTIC,
  },
  [BotTypesEnum.BEGINNER]: {
    userVisible: 'Beginner',
    botName: 'FOOBot',
    description: 'Etc',
    botCode: (lexicon: string) =>
      lexicon.startsWith('CSW') || lexicon.startsWith('NWL')
        ? BotRequest.BotCode.LEVEL1_CEL_BOT
        : BotRequest.BotCode.LEVEL1_PROBABILISTIC,
  },
};
