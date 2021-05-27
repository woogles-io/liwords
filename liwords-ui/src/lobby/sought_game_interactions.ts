import {
  GameRequest,
  GameRules,
  MatchRequest,
  MessageType,
  RatingMode,
  SeekRequest,
  SoughtGameProcessEvent,
} from '../gen/api/proto/realtime/realtime_pb';
import { ChallengeRuleMap } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { encodeToSocketFmt } from '../utils/protobuf';

export const sendSeek = (
  game: SoughtGame,
  sendSocketMsg: (msg: Uint8Array) => void
): void => {
  const sr = new SeekRequest();
  const mr = new MatchRequest();
  const gr = new GameRequest();
  const rules = new GameRules();
  rules.setBoardLayoutName('CrosswordGame');
  rules.setVariantName(game.variant);
  rules.setLetterDistributionName('english');

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

  if (game.receiver.getDisplayName() === '' && game.playerVsBot === false) {
    sr.setGameRequest(gr);

    sendSocketMsg(
      encodeToSocketFmt(MessageType.SEEK_REQUEST, sr.serializeBinary())
    );
  } else {
    // We make it a match request if the receiver is non-empty, or if playerVsBot.
    mr.setGameRequest(gr);
    mr.setReceivingUser(game.receiver);
    mr.setTournamentId(game.tournamentID);
    sendSocketMsg(
      encodeToSocketFmt(MessageType.MATCH_REQUEST, mr.serializeBinary())
    );
  }
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
