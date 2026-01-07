import { create, toBinary } from "@bufbuild/protobuf";
import { MessageType } from "../gen/api/proto/ipc/ipc_pb";
import {
  DeclineSeekRequestSchema,
  SeekRequestSchema,
  SeekState,
  SoughtGameProcessEventSchema,
} from "../gen/api/proto/ipc/omgseeks_pb";
import {
  GameRequestSchema,
  GameRulesSchema,
  RatingMode,
} from "../gen/api/proto/ipc/omgwords_pb";
import { SoughtGame } from "../store/reducers/lobby_reducer";
import { encodeToSocketFmt } from "../utils/protobuf";
import { BotTypesEnumProperties } from "./bots";

export const defaultLetterDistribution = (lexicon: string): string => {
  const lowercasedLexicon = lexicon.toLowerCase();
  if (lowercasedLexicon.startsWith("rd")) {
    return "german";
  } else if (lowercasedLexicon.startsWith("nsf")) {
    return "norwegian";
  } else if (lowercasedLexicon.startsWith("fra")) {
    return "french";
  } else if (lowercasedLexicon.startsWith("disc")) {
    return "catalan";
  } else if (lowercasedLexicon.startsWith("osps")) {
    return "polish";
  } else if (lowercasedLexicon.startsWith("file")) {
    return "spanish";
  } else {
    return "english";
  }
};

export const sendSeek = (
  game: SoughtGame,
  sendSocketMsg: (msg: Uint8Array) => void,
): void => {
  const sr = create(SeekRequestSchema);
  const gr = create(GameRequestSchema);
  const rules = create(GameRulesSchema);
  rules.boardLayoutName = "CrosswordGame";
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
  gr.gameMode = game.gameMode ?? 0;
  if (game.playerVsBot) {
    gr.botType = BotTypesEnumProperties[game.botType].botCode(game.lexicon);
  }

  sr.userState = SeekState.READY;
  sr.minimumRatingRange = game.minRatingRange;
  sr.maximumRatingRange = game.maxRatingRange;
  sr.requireEstablishedRating = game.requireEstablishedRating || false;
  sr.onlyFollowedPlayers = game.onlyFollowedPlayers || false;

  if (!game.receiverIsPermanent) {
    sr.gameRequest = gr;
    console.log("this is a seek request");
  } else {
    // We make it a match request if the receiver is non-empty, or if playerVsBot.
    sr.gameRequest = gr;
    sr.receivingUser = game.receiver;
    sr.tournamentId = game.tournamentID;
    sr.receiverIsPermanent = true;
    console.log("this is a match request");
  }
  console.log("sr: ", sr);
  sendSocketMsg(
    encodeToSocketFmt(
      MessageType.SEEK_REQUEST,
      toBinary(SeekRequestSchema, sr),
    ),
  );
};

export const sendAccept = (
  seekID: string,
  sendSocketMsg: (msg: Uint8Array) => void,
): void => {
  // Eventually use the ID.
  const sa = create(SoughtGameProcessEventSchema, {});
  sa.requestId = seekID;
  sendSocketMsg(
    encodeToSocketFmt(
      MessageType.SOUGHT_GAME_PROCESS_EVENT,
      toBinary(SoughtGameProcessEventSchema, sa),
    ),
  );
};

export const sendDecline = (
  seekID: string,
  sendSocketMsg: (msg: Uint8Array) => void,
): void => {
  const evt = create(DeclineSeekRequestSchema, { requestId: seekID });
  sendSocketMsg(
    encodeToSocketFmt(
      MessageType.DECLINE_SEEK_REQUEST,
      toBinary(DeclineSeekRequestSchema, evt),
    ),
  );
};
