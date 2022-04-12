import React, { ReactNode, useMemo } from 'react';
import { Button, Card } from 'antd';
import { ChallengeRule, PlayerMetadata } from '../gameroom/game_info';
import { UsernameWithContext } from '../shared/usernameWithContext';
import moment from 'moment';
import { timeCtrlToDisplayName, timeToString } from '../store/constants';
import { PuzzleStatus } from '../gen/api/proto/puzzle_service/puzzle_service_pb';
import { StarFilled, StarOutlined } from '@ant-design/icons';

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
  attempts: number;
  dateSolved?: Date;
  loadNewPuzzle: () => void;
  showSolution: () => void;
};

export const renderStars = (stars: number) => {
  const ret: ReactNode[] = [];
  for (let i = stars; i > 0; i--) {
    ret.push(<StarFilled />);
  }
  while (ret.length < MAX_STARS) {
    ret.push(<StarOutlined className="unearned" />);
  }
  return <div className="stars">{ret}</div>;
};

const MAX_STARS = 3;
export const calculatePuzzleScore = (solved: boolean, attempts: number) => {
  // Score 3 for 1 attempt, 2 for 2 attempts, 1 for all others
  return solved ? Math.max(1, MAX_STARS + 1 - attempts) : 0;
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
    attempts,
    dateSolved,
    loadNewPuzzle,
    showSolution,
  } = props;

  // TODO: should be determined on the back end and not hardcoded
  const puzzleType = 'Equity puzzle';
  const score = calculatePuzzleScore(!!dateSolved, attempts);

  const attemptsText = useMemo(() => {
    if (solved === PuzzleStatus.CORRECT) {
      const solveDate = moment(dateSolved).format('MMMM D, YYYY');
      return (
        <p className="attempts-made">{`Solved in ${attempts} ${
          attempts === 1 ? 'attempt' : 'attempts'
        } on ${solveDate}`}</p>
      );
    }
    switch (solved) {
      case PuzzleStatus.CORRECT:
        const solveDate = moment(dateSolved).format('MMMM D, YYYY');
        return (
          <p className="attempts-made">{`Solved in ${attempts} ${
            attempts === 1 ? 'attempt' : 'attempts'
          } on ${solveDate}`}</p>
        );
      case PuzzleStatus.UNANSWERED:
        return (
          <p className="attempts-made">
            {`You have made ${attempts} ${
              attempts === 1 ? 'attempt' : 'attempts'
            }`}
          </p>
        );
      case PuzzleStatus.INCORRECT:
      default:
        return (
          <p className="attempts-made">{`You gave up after ${attempts} ${
            attempts === 1 ? 'attempt' : 'attempts'
          }`}</p>
        );
    }
  }, [attempts, dateSolved, solved]);

  const actions = useMemo(() => {
    if (!solved || solved === PuzzleStatus.UNANSWERED) {
      return (
        <div className="actions">
          <Button type="default" onClick={loadNewPuzzle}>
            Skip
          </Button>
          <Button type="default" onClick={showSolution}>
            Give up
          </Button>
        </div>
      );
    }
    return (
      <div className="actions">
        <Button type="primary" onClick={loadNewPuzzle}>
          Next puzzle
        </Button>
      </div>
    );
  }, [loadNewPuzzle, showSolution, solved]);

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

  if (solved === PuzzleStatus.CORRECT) {
    const solveDate = moment(dateSolved).format('MMMM D, YYYY');
    return (
      <p className="attempts-made">{`Solved in ${attempts} ${
        attempts === 1 ? 'attempt' : 'attempts'
      } on ${solveDate}`}</p>
    );
  }

  if (solved === PuzzleStatus.UNANSWERED) {
    return (
      <Card className="puzzle-info" title={`Puzzle Mode`} extra={puzzleType}>
        <p className="game-settings">{`${
          variantName || 'classic'
        } • ${lexicon}`}</p>
        <p className="instructions">
          There is a star play in this position that is significantly better
          than the second-best play. What would HastyBot play?
        </p>
        <p className="progress">{attemptsText}</p>
        {actions}
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
    <Card className="puzzle-info" title={`Puzzle Mode`} extra={puzzleType}>
      {solved !== PuzzleStatus.UNANSWERED && renderStars(score)}
      <p>{playerInfo}</p>
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
      <p className="progress">{attemptsText}</p>
      {actions}
    </Card>
  );
});
