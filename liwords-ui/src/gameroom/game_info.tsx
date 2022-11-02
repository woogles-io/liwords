import React from 'react';
import { Card } from 'antd';
import { timeCtrlToDisplayName, timeToString } from '../store/constants';
import { VariantIcon } from '../shared/variant_icons';
import { MatchLexiconDisplay } from '../shared/lexicon_display';
import { GameEndReason, GameRequest } from '../gen/api/proto/ipc/omgwords_pb';
import {
  BotRequest_BotCode,
  ChallengeRule,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import { RatingMode } from '../gen/api/proto/ipc/omgwords_pb';
import { GameRules } from '../gen/api/proto/ipc/omgwords_pb';
import { BotTypesEnum } from '../lobby/bots';
import { BotRequest } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { challengeRuleNames } from '../constants/challenge_rules';

// At some point we should get this from the pb but then we have to use
// twirp for this and we really shouldn't need to. Wait on it probably.
// See game_service.proto
export type GameMetadata = {
  players: Array<PlayerMetadata>;
  time_control_name: string;
  tournament_id: string;
  game_end_reason: GameEndReason;
  created_at?: string;
  winner?: number;
  scores?: Array<number>;
  game_id?: string;
  game_request: GameRequest;
};

export const defaultGameInfo: GameMetadata = {
  players: new Array<PlayerMetadata>(),
  game_request: new GameRequest({
    lexicon: '',
    rules: new GameRules({
      variantName: '',
      boardLayoutName: 'CrosswordGame',
      letterDistributionName: 'english',
    }),
    initialTimeSeconds: 0,
    incrementSeconds: 0,
    challengeRule: ChallengeRule.VOID,
    ratingMode: RatingMode.RATED,
    maxOvertimeMinutes: 0,
    originalRequestId: '',
    playerVsBot: false,
    botType: BotRequest_BotCode.HASTY_BOT,
  }),
  tournament_id: '',
  game_end_reason: GameEndReason.NONE,
  time_control_name: '',
};

export type PlayersStreakInfo = {
  nickname: string;
  uuid: number;
};

export type SingleGameStreakInfo = {
  game_id: string;
  winner: number;
};

export type StreakInfoResponse = {
  streak: Array<SingleGameStreakInfo>;
  playersInfo: Array<PlayersStreakInfo>;
};

export type DefineWordsResponse = {
  results: {
    [key: string]: { v: boolean; d: string };
  };
};

export type RecentGamesResponse = {
  game_info: Array<GameMetadata>;
};

export type PlayerMetadata = {
  user_id: string;
  nickname: string;
  full_name: string;
  country_code: string;
  rating: string;
  title: string;
  avatar_url: string;
  is_bot: boolean;
  first: boolean;
};

export type GCGResponse = {
  gcg: string;
};

type Props = {
  meta: GameMetadata;
  tournamentName: string;
  colorOverride?: string;
  logoUrl?: string;
};

export const GameInfo = React.memo((props: Props) => {
  const variant = (
    <VariantIcon
      vcode={props.meta.game_request.rules?.variantName || 'classic'}
      withName
    />
  );
  const rated =
    props.meta.game_request.ratingMode === RatingMode.RATED
      ? 'Rated'
      : 'Unrated';

  const challenge = challengeRuleNames[props.meta.game_request.challengeRule];

  const card = (
    <Card className="game-info">
      <div className="metadata">
        {props.meta.tournament_id && (
          <p
            className="tournament-name"
            style={{ color: props.colorOverride || 'ignore' }}
          >
            {props.tournamentName}
          </p>
        )}
        <p className="variant">
          {`${
            timeCtrlToDisplayName(
              props.meta.game_request.initialTimeSeconds,
              props.meta.game_request.incrementSeconds,
              props.meta.game_request.maxOvertimeMinutes
            )[0]
          } ${timeToString(
            props.meta.game_request.initialTimeSeconds,
            props.meta.game_request.incrementSeconds,
            props.meta.game_request.maxOvertimeMinutes
          )}`}{' '}
          • {variant} •{' '}
          <MatchLexiconDisplay lexiconCode={props.meta.game_request.lexicon} />
        </p>
        <p>
          {challenge} challenge • {rated}
        </p>
      </div>
      {props.logoUrl && (
        <div className="logo-container">
          <img
            className="club-logo"
            src={props.logoUrl}
            alt={props.tournamentName}
          />
        </div>
      )}
    </Card>
  );
  return card;
});
