import { create } from '@bufbuild/protobuf';
import {
  GameInfoResponseSchema,
  GameRequestSchema,
  GameRulesSchema,
  PlayerInfoSchema,
} from '../gen/api/proto/ipc/omgwords_pb';
import { RatingMode } from '../gen/api/proto/ipc/omgwords_pb';
import {
  GameDocument,
  GameInfoResponse,
} from '../gen/api/proto/ipc/omgwords_pb';

export const syntheticGameInfo = (doc: GameDocument): GameInfoResponse => {
  return create(GameInfoResponseSchema, {
    players: doc.players.map((p) =>
      create(PlayerInfoSchema, {
        userId: p.userId,
        nickname: p.nickname,
        fullName: p.realName,
      })
    ),
    timeControlName: 'Annotated',
    tournamentId: '', // maybe can populate from a description later
    gameEndReason: doc.endReason,
    scores: doc.currentScores,
    winner: doc.winner,
    createdAt: doc.createdAt,
    gameId: doc.uid,
    // no last update
    type: doc.type,
    gameRequest: create(GameRequestSchema, {
      lexicon: doc.lexicon,
      rules: create(GameRulesSchema, {
        boardLayoutName: doc.boardLayout,
        letterDistributionName: doc.letterDistribution,
        variantName: doc.variant,
      }),
      incrementSeconds: doc.timers?.incrementSeconds,
      challengeRule: doc.challengeRule.valueOf(),
      ratingMode: RatingMode.CASUAL,
      requestId: 'none',
      maxOvertimeMinutes: doc.timers?.maxOvertime,
      originalRequestId: 'none',
    }),
  });
};
