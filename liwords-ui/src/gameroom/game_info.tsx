import React from 'react';
import { Card, Row } from 'antd';
import { GameInfoResponse } from '../gen/api/proto/game_service/game_service_pb';
// At some point we should get this from the pb but then we have to use
// twirp for this and we really shouldn't need to. Wait on it probably.
// See game_service.proto
export type GameMetadata = {
  players: Array<PlayerInfo>;
  lexicon: string;
  variant: string;
  time_control: string;
  tournament_name: string;
  challenge_rule: 'FIVE_POINT' | 'TEN_POINT' | 'SINGLE' | 'DOUBLE' | 'VOID';
  rating_mode: number;
  done: boolean;
};

type PlayerInfo = {
  user_id: string;
  nickname: string;
  full_name: string;
  country_code: string;
  rating: string;
  title: string;
};

type Props = {
  meta: GameMetadata;
};

export const GameInfo = (props: Props) => {
  let variant;
  let rated;
  if (props.meta.variant === 'classic') {
    variant = 'Classic';
  }

  if (props.meta.rating_mode === 0) {
    rated = 'Rated';
  } else {
    rated = 'Unrated';
  }
  // Default to void, see macondo.proto; void is zero or default value of enum.
  let challenge = 'Void';

  if (props.meta.challenge_rule) {
    challenge = {
      FIVE_POINT: '5 point',
      TEN_POINT: '10 point',
      SINGLE: 'Single',
      DOUBLE: 'Double',
      VOID: 'Void',
    }[props.meta.challenge_rule];
  }

  return (
    <Card>
      <Row>
        {props.meta.time_control} - {variant} - {props.meta.lexicon}
      </Row>
      <Row>
        {challenge} challenge - {rated}
      </Row>
    </Card>
  );
};
