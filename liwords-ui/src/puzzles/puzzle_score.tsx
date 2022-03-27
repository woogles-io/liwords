import React, { ReactNode, useMemo } from 'react';
import { Button, Card } from 'antd';
import moment from 'moment';
import { StarFilled, StarOutlined } from '@ant-design/icons';

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
};

export const PuzzleScore = React.memo((props: Props) => {
  const { attempts, dateSolved, loadNewPuzzle, showSolution } = props;
  const score = calculateScore(!!dateSolved, attempts);
  const renderScore = useMemo(() => {
    let ret: ReactNode[] = [];
    for (let i = score; i > 0; i--) {
      ret.push(<StarFilled />);
    }
    while (ret.length < MAX_STARS) {
      ret.push(<StarOutlined className="unearned" />);
    }
    return ret;
  }, [score]);

  const attemptsText = useMemo(() => {
    if (dateSolved) {
      const solveDate = moment(dateSolved).format('MMMM D, YYYY');
      return (
        <p className="attempts-made">{`Solved in ${attempts} ${
          attempts === 1 ? 'attempt' : 'attempts'
        } on ${solveDate}`}</p>
      );
    }
    return (
      <p className="attempts-made">
        {`You have made ${attempts} ${
          attempts === 1 ? '1 attempt' : 'attempts'
        }`}
      </p>
    );
  }, [attempts, dateSolved]);

  const actions = useMemo(() => {
    if (dateSolved) {
      return (
        <div className="actions">
          <Button type="primary" onClick={loadNewPuzzle}>
            Next puzzle
          </Button>
        </div>
      );
    }
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
  }, [dateSolved]);
  return (
    <Card className="puzzle-score" title="Puzzle Mode">
      <div className="stars">{renderScore}</div>
      {attemptsText}
      {actions}
    </Card>
  );
});
