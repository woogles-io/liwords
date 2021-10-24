import {
  GameRequest,
  GameRules,
  MessageType,
  RatingMode,
  SeekRequest,
  SeekState,
  SoughtGameProcessEvent,
} from '../gen/api/proto/realtime/realtime_pb';
import { ChallengeRuleMap } from '../gen/macondo/api/proto/macondo/macondo_pb';
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
  rules.setBoardLayoutName('CrosswordGame');
  rules.setVariantName(game.variant);
  rules.setLetterDistributionName(defaultLetterDistribution(game.lexicon));

  gr.setChallengeRule(
    game.challengeRule as ChallengeRuleMap[keyof ChallengeRuleMap]
  );
  gr.setLexicon(game.lexicon);
  gr.setInitialTimeSeconds(game.initialTimeSecs);
  gr.setMaxOvertimeMinutes(game.maxOvertimeMinutes);
  gr.setIncrementSeconds(game.incrementSecs);
  gr.setRules(rules);
  gr.setRatingMode(game.rated ? RatingMode.RATED : RatingMode.CASUAL);
  gr.setPlayerVsBot(game.playerVsBot);
  if (game.playerVsBot) {
    gr.setBotType(BotTypesEnumProperties[game.botType].botCode(game.lexicon));
  }

  sr.setUserState(SeekState.READY);

  if (!game.receiverIsPermanent) {
    sr.setGameRequest(gr);
    console.log('this is a seek request');
  } else {
    // We make it a match request if the receiver is non-empty, or if playerVsBot.
    sr.setGameRequest(gr);
    sr.setReceivingUser(game.receiver);
    sr.setTournamentId(game.tournamentID);
    sr.setReceiverIsPermanent(true);
    console.log('this is a match request');
  }
  console.log('sr: ', sr);
  sendSocketMsg(
    encodeToSocketFmt(MessageType.SEEK_REQUEST, sr.serializeBinary())
  );
};

export const sendAccept = (
  seekID: string,
  sendSocketMsg: (msg: Uint8Array) => void
): void => {
  // Eventually use the ID.
  const sa = new SoughtGameProcessEvent();
  sa.setRequestId(seekID);
  sendSocketMsg(
    encodeToSocketFmt(
      MessageType.SOUGHT_GAME_PROCESS_EVENT,
      sa.serializeBinary()
    )
  );
};
