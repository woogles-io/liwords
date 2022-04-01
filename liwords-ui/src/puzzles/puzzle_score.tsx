import React, { ReactNode, useMemo } from 'react';
import { Button, Card } from 'antd';
import moment from 'moment';
import { StarFilled, StarOutlined } from '@ant-design/icons';
import { PuzzleStatus } from '../gen/api/proto/puzzle_service/puzzle_service_pb';

const MAX_STARS = 3;
const calculateScore = (solved: boolean, attempts: number) => {
  // Score 3 for 1 attempt, 2 for 2 attempts, 1 for all others
  return solved ? Math.max(1, MAX_STARS + 1 - attempts) : 0;
};

type Props = {
  attempts: number;
  dateSolved?: Date;
  loadNewPuzzle: () => void;
  showSolution: () => void;
  solved: number;
};

export const PuzzleScore = React.memo((props: Props) => {
  const { attempts, dateSolved, loadNewPuzzle, showSolution, solved } = props;
  const score = calculateScore(!!dateSolved, attempts);
  const renderScore = useMemo(() => {
    const ret: ReactNode[] = [];
    for (let i = score; i > 0; i--) {
      ret.push(<StarFilled />);
    }
    while (ret.length < MAX_STARS) {
      ret.push(<StarOutlined className="unearned" />);
    }
    return ret;
  }, [score]);

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
  return (
    <Card className="puzzle-score" title="Puzzle Mode">
      <div className="stars">{renderScore}</div>
      {attemptsText}
      {actions}
    </Card>
  );
});
