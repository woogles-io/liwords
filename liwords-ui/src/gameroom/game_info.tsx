import React from 'react';
import { Card } from 'antd';
import { timeCtrlToDisplayName, timeToString } from '../store/constants';
import { VariantIcon } from '../shared/variant_icons';

// At some point we should get this from the pb but then we have to use
// twirp for this and we really shouldn't need to. Wait on it probably.
// See game_service.proto
export type GameMetadata = {
  players: Array<PlayerMetadata>;
  time_control_name: string;
  tournament_id: string;
  game_end_reason: string;
  created_at?: string;
  winner?: number;
  scores?: Array<number>;
  game_id?: string;
  game_request: GameRequest;
};

type GameRules = {
  board_layout_name: string;
  letter_distribution_name: string;
  variant_name: string;
};

export type ChallengeRule =
  | 'FIVE_POINT'
  | 'TEN_POINT'
  | 'SINGLE'
  | 'DOUBLE'
  | 'TRIPLE'
  | 'VOID';

export type GameRequest = {
  lexicon: string;
  rules: GameRules;
  initial_time_seconds: number;
  increment_seconds: number;
  challenge_rule: ChallengeRule;
  rating_mode: string;
  max_overtime_minutes: number;
  original_request_id: string;
};

export const defaultGameInfo: GameMetadata = {
  players: new Array<PlayerMetadata>(),
  game_request: {
    lexicon: '',
    rules: {
      variant_name: '',
      board_layout_name: 'CrosswordGame',
      letter_distribution_name: 'english',
    },
    initial_time_seconds: 0,
    increment_seconds: 0,
    challenge_rule: 'VOID' as ChallengeRule,
    rating_mode: 'RATED',
    max_overtime_minutes: 0,
    original_request_id: '',
  },
  tournament_id: '',
  game_end_reason: 'NONE',
  time_control_name: '',
};

export type SingleGameStreakInfo = {
  game_id: string;
  winner: number;
};

export type StreakInfoResponse = {
  streak: Array<SingleGameStreakInfo>;
  players: Array<string>;
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
      vcode={props.meta.game_request.rules.variant_name || 'classic'}
      withName
    />
  );
  const rated =
    props.meta.game_request.rating_mode === 'RATED' ? 'Rated' : 'Unrated';
  const challenge = {
    FIVE_POINT: '5 point',
    TEN_POINT: '10 point',
    SINGLE: 'Single',
    DOUBLE: 'Double',
    TRIPLE: 'Triple',
    VOID: 'Void',
  }[props.meta.game_request.challenge_rule];

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
              props.meta.game_request.initial_time_seconds,
              props.meta.game_request.increment_seconds,
              props.meta.game_request.max_overtime_minutes
            )[0]
          } ${timeToString(
            props.meta.game_request.initial_time_seconds,
            props.meta.game_request.increment_seconds,
            props.meta.game_request.max_overtime_minutes
          )}`}{' '}
          • {variant} • {props.meta.game_request.lexicon}
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
