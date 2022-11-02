import { MessageType } from '../gen/api/proto/ipc/ipc_pb';
import {
  SeekRequest,
  SeekState,
  SoughtGameProcessEvent,
} from '../gen/api/proto/ipc/omgseeks_pb';
import {
  GameRequest,
  GameRules,
  RatingMode,
} from '../gen/api/proto/ipc/omgwords_pb';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { encodeToSocketFmt } from '../utils/protobuf';
import { BotTypesEnumProperties } from './bots';

export const defaultLetterDistribution = (lexicon: string): string => {
  const lowercasedLexicon = lexicon.toLowerCase();
  if (lowercasedLexicon.startsWith('rd')) {
    return 'german';
  } else if (lowercasedLexicon.startsWith('nsf')) {
    return 'norwegian';
  } else if (lowercasedLexicon.startsWith('fra')) {
    return 'french';
  } else {
    return 'english';
  }
};

export const sendSeek = (
  game: SoughtGame,
  sendSocketMsg: (msg: Uint8Array) => void
): void => {
  const sr = new SeekRequest();
  const gr = new GameRequest();
  const rules = new GameRules();
  rules.boardLayoutName = 'CrosswordGame';
  rules.variantName = game.variant;
  rules.letterDistributionName = defaultLetterDistribution(game.lexicon);

  gr.challengeRule = game.challengeRule;
  gr.lexicon = game.lexicon;
  gr.initialTimeSeconds = game.initialTimeSecs;
  gr.maxOvertimeMinutes = game.maxOvertimeMinutes;
  gr.incrementSeconds = game.incrementSecs;
  gr.rules = rules;
  gr.ratingMode = game.rated ? RatingMode.RATED : RatingMode.CASUAL;
  gr.playerVsBot = game.playerVsBot;
  if (game.playerVsBot) {
    gr.botType = BotTypesEnumProperties[game.botType].botCode(game.lexicon);
  }

  sr.userState = SeekState.READY;
  sr.minimumRatingRange = game.minRatingRange;
  sr.maximumRatingRange = game.maxRatingRange;

  if (!game.receiverIsPermanent) {
    sr.gameRequest = gr;
    console.log('this is a seek request');
  } else {
    // We make it a match request if the receiver is non-empty, or if playerVsBot.
    sr.gameRequest = gr;
    sr.receivingUser = game.receiver;
    sr.tournamentId = game.tournamentID;
    sr.receiverIsPermanent = true;
    console.log('this is a match request');
  }
  console.log('sr: ', sr);
  sendSocketMsg(encodeToSocketFmt(MessageType.SEEK_REQUEST, sr.toBinary()));
};

export const sendAccept = (
  seekID: string,
  sendSocketMsg: (msg: Uint8Array) => void
): void => {
  // Eventually use the ID.
  const sa = new SoughtGameProcessEvent();
  sa.requestId = seekID;
  sendSocketMsg(
    encodeToSocketFmt(MessageType.SOUGHT_GAME_PROCESS_EVENT, sa.toBinary())
  );
};
