import React from 'react';
import { Card, Row, Tooltip } from 'antd';
import { initialTimeLabel } from '../store/constants';

// At some point we should get this from the pb but then we have to use
// twirp for this and we really shouldn't need to. Wait on it probably.
// See game_service.proto
export type GameMetadata = {
  players: Array<PlayerMetadata>;
  lexicon: string;
  variant: string;
  time_control_name: string;
  initial_time_seconds: number;
  tournament_name: string;
  max_overtime_minutes: number;
  increment_seconds: number;
  challenge_rule:
    | 'FIVE_POINT'
    | 'TEN_POINT'
    | 'SINGLE'
    | 'DOUBLE'
    | 'TRIPLE'
    | 'VOID';
  rating_mode: string;
  game_end_reason: string; // ?
  created_at?: string;
  winner?: number;
  scores?: Array<number>;
  game_id?: string;
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
};

export const GameInfo = (props: Props) => {
  let variant;
  if (props.meta.variant === 'classic') {
    variant = 'Classic';
  }

  const rated = props.meta.rating_mode === 'RATED' ? 'Rated' : 'Unrated';
  const challenge = {
    FIVE_POINT: '5 point',
    TEN_POINT: '10 point',
    SINGLE: 'Single',
    DOUBLE: 'Double',
    TRIPLE: 'Triple',
    VOID: 'Void',
  }[props.meta.challenge_rule];

  return (
    <Card className="game-info">
      <Row className="variant">
        {`${initialTimeLabel(props.meta.initial_time_seconds)} ${
          props.meta.increment_seconds || 0
        }`}{' '}
        •
        <Tooltip title="The maximum amount of overtime, in minutes">
          <span>
            &nbsp;{`OT: ${props.meta.max_overtime_minutes || 0}`}&nbsp;
          </span>
        </Tooltip>
        • {variant} • {props.meta.lexicon}
      </Row>
      <Row>
        {challenge} challenge • {rated}
      </Row>
    </Card>
  );
};
