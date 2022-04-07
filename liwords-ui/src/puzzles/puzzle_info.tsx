import React, { useMemo } from 'react';
import { Card } from 'antd';
import { ChallengeRule, PlayerMetadata } from '../gameroom/game_info';
import { UsernameWithContext } from '../shared/usernameWithContext';
import moment from 'moment';
import { timeCtrlToDisplayName, timeToString } from '../store/constants';
import { PuzzleStatus } from '../gen/api/proto/puzzle_service/puzzle_service_pb';

export const challengeMap = {
  FIVE_POINT: '5 point',
  TEN_POINT: '10 point',
  SINGLE: 'Single',
  DOUBLE: 'Double',
  TRIPLE: 'Triple',
  VOID: 'Void',
};

type Props = {
  solved: number;
  gameDate?: Date;
  gameUrl?: string;
  lexicon: string;
  variantName: string;
  player1: Partial<PlayerMetadata> | undefined;
  player2: Partial<PlayerMetadata> | undefined;
  ratingMode?: string;
  challengeRule: ChallengeRule | undefined;
  initial_time_seconds?: number;
  increment_seconds?: number;
  max_overtime_minutes?: number;
};

export const PuzzleInfo = React.memo((props: Props) => {
  const {
    solved,
    gameDate,
    gameUrl,
    challengeRule,
    ratingMode,
    initial_time_seconds,
    increment_seconds,
    max_overtime_minutes,
    lexicon,
    variantName,
    player1,
    player2,
  } = props;
  // TODO: probably should be determined on the back end and not hardcoded
  const puzzleType = 'Equity puzzle';

  const formattedGameDate = useMemo(() => {
    return moment(gameDate).format('MMMM D, YYYY');
  }, [gameDate]);

  const gameLink = useMemo(() => {
    if (gameUrl) {
      return (
        <span>
          {' • '}
          <a href={gameUrl} target="_blank" rel="noopener noreferrer">
            View Game
          </a>
        </span>
      );
    }
  }, [gameUrl]);
  if (solved === PuzzleStatus.UNANSWERED) {
    return (
      <Card
        className="puzzle-info"
        title={puzzleType}
        extra={`${variantName || 'classic'} • ${lexicon}`}
      >
        <p>
          There is a star play in this position that is significantly better
          than the second-best play. What would HastyBot play?
        </p>
      </Card>
    );
  }
  const ratedDisplay = ratingMode === 'RATED' ? 'Rated' : 'Unrated';
  const challengeDisplay = challengeRule ? challengeMap[challengeRule] : '';

  const player1NameDisplay = player1?.nickname ? (
    <UsernameWithContext username={player1?.nickname || ''} />
  ) : (
    <span>unknown player</span>
  );
  const player2NameDisplay = player2?.nickname ? (
    <UsernameWithContext username={player2?.nickname || ''} />
  ) : (
    <span>unknown player</span>
  );
  const playerInfo = (
    <p className="player-title">
      Game played by {player1NameDisplay} vs {player2NameDisplay}
    </p>
  );

  return (
    <Card className="puzzle-info" title={playerInfo}>
      <p>
        {formattedGameDate}
        {gameLink}
      </p>
      <p className="game-settings">{`${
        timeCtrlToDisplayName(
          initial_time_seconds || 0,
          increment_seconds || 0,
          max_overtime_minutes || 0
        )[0]
      } ${timeToString(
        initial_time_seconds || 0,
        increment_seconds || 0,
        max_overtime_minutes || 0
      )} • ${variantName || 'classic'} • ${lexicon}`}</p>
      <p>
        {challengeDisplay}
        {challengeDisplay && ratedDisplay ? ' • ' : ''}
        {ratedDisplay}
      </p>
    </Card>
  );
});
