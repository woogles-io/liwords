import React, {
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
} from 'react';
import { GameEvent } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { useMountedState } from '../utils/mounted';
import {
  AnswerResponse,
  PuzzleRequest,
  PuzzleStatus,
} from '../gen/api/proto/puzzle_service/puzzle_service_pb';
import { LiwordsAPIError, postProto } from '../api/api';
import { Button, message } from 'antd';
import { singularCount } from '../utils/plural';
import {
  generateEmptyLearnLayout,
  LearnContext,
  LearnSpaceType,
} from '../learn/learn_overlay';
import {
  BulbFilled,
  EyeInvisibleOutlined,
  EyeOutlined,
} from '@ant-design/icons';

type Props = {
  puzzleID?: string;
  solved: number;
  attempts: number;
};

type HintInfo = {
  key: string;
  message: ReactNode;
  revealed: boolean;
};

type AvailableHints = {
  score?: HintInfo;
  tiles?: HintInfo;
  position?: HintInfo;
};

const RATED_ATTEMPTS = 2;

const readableLane = (row: number, col: number, direction: 0 | 1) => {
  if (direction === 0) {
    return 'row ' + (row + 1).toString();
  } else {
    return 'column ' + String.fromCodePoint(col + 65);
  }
};

export const Hints = (props: Props) => {
  const { puzzleID, solved, attempts } = props;
  const { useState } = useMountedState();
  const [hints, setHints] = useState<AvailableHints>({});
  const [solution, setSolution] = useState<GameEvent | undefined>(undefined);
  const { gridDim, setLearnLayout } = useContext(LearnContext);
  const [boardHighlightingOn, setBoardHighlightingOn] = useState(false);

  const fetchAnswer = useCallback(async () => {
    if (!puzzleID) {
      return;
    }
    const req = new PuzzleRequest();
    req.setPuzzleId(puzzleID);
    try {
      const resp = await postProto(
        AnswerResponse,
        'puzzle_service.PuzzleService',
        'GetPuzzleAnswer',
        req
      );
      console.log('got resp', resp.toObject());
      // Only CorrectAnswer is filled in properly.
      setSolution(resp.getCorrectAnswer());
    } catch (err) {
      // There will be an error if this endpoint is called before the user
      // has submitted a guess.
      message.error({
        content: (err as LiwordsAPIError).message,
        duration: 5,
      });
    }
  }, [puzzleID]);

  const earnedHints = useMemo(() => {
    const usedHints = Object.values(hints).filter((h) => h.revealed).length;
    return Math.min(3 - usedHints, attempts - RATED_ATTEMPTS + 1 - usedHints);
  }, [hints, attempts]);

  useEffect(() => {
    if (!solution && earnedHints > 0) {
      fetchAnswer();
    }
  }, [earnedHints, fetchAnswer, solution]);

  const renderHint = useCallback(
    (hint: HintInfo) => {
      if (!hint.revealed) {
        return null;
      }
      return (
        <div className="puzzle-hint" key={hint.key}>
          <BulbFilled />
          {hint.message}
          {hint.key === 'position-hint' && (
            <div
              onClick={() => {
                console.log('um');
                setBoardHighlightingOn((x) => !x);
              }}
            >
              {boardHighlightingOn ? <EyeInvisibleOutlined /> : <EyeOutlined />}
            </div>
          )}
        </div>
      );
    },
    [boardHighlightingOn, setBoardHighlightingOn]
  );

  useEffect(() => {
    if (solution) {
      // Add score hint
      const scoreHint: HintInfo = {
        key: 'score-hint',
        message: (
          <>
            The score for the play is{' '}
            <span className="tentative-score">{solution.getScore()}</span>
          </>
        ),
        revealed: false,
      };
      // Add tiles hint
      const tilesUsed = Array.from(solution.getPlayedTiles()).filter(
        (x) => x !== '.'
      ).length;
      const isBingo = solution.getIsBingo();
      const tilesMessage = isBingo ? (
        <>The play is a bingo! Use all your tiles.</>
      ) : (
        <>
          The play uses {singularCount(tilesUsed, 'tile', 'tiles')} from the
          rack.
        </>
      );
      const tilesHint: HintInfo = {
        key: 'tile-hint',
        message: tilesMessage,
        revealed: false,
      };

      const positionHint: HintInfo = {
        key: 'position-hint',
        message: (
          <>
            The play should be placed on{' '}
            {readableLane(
              solution.getRow(),
              solution.getColumn(),
              solution.getDirection()
            )}
            .
          </>
        ),
        revealed: false,
      };

      setHints({
        score: scoreHint,
        tiles: tilesHint,
        position: positionHint,
      });
    }
  }, [solution]);

  useEffect(() => {
    if (boardHighlightingOn && solution) {
      const layout = generateEmptyLearnLayout(gridDim, LearnSpaceType.Faded);
      if (solution.getDirection() === 0) {
        layout[solution.getRow()] = new Array(gridDim).fill(
          LearnSpaceType.Highlighted
        );
      } else {
        for (let i = 0; i < gridDim; i++) {
          layout[i][solution.getColumn()] = LearnSpaceType.Highlighted;
        }
      }
      setLearnLayout(layout);
    } else {
      setLearnLayout(generateEmptyLearnLayout(gridDim, LearnSpaceType.Normal));
    }
  }, [boardHighlightingOn, solution, gridDim, setLearnLayout]);

  useEffect(() => {
    if (
      solved == PuzzleStatus.UNANSWERED &&
      solution &&
      hints?.position?.revealed
    ) {
      setBoardHighlightingOn(true);
    }
  }, [hints, solution, solved]);

  useEffect(() => {
    if (solved !== PuzzleStatus.UNANSWERED) {
      setLearnLayout(generateEmptyLearnLayout(gridDim, LearnSpaceType.Normal));
    }
  }, [solved, setLearnLayout, gridDim]);

  const revealHint = useCallback(
    (hintToReveal: 'score' | 'tiles' | 'position') => {
      setHints((x) => ({
        ...x,
        [hintToReveal]: {
          ...x[hintToReveal],
          revealed: true,
        },
      }));
    },
    []
  );

  const actions = useMemo(() => {
    if (attempts >= RATED_ATTEMPTS) {
      return (
        <div className="hint-actions">
          {!hints?.score?.revealed && (
            <Button
              type="primary"
              onClick={() => {
                revealHint('score');
              }}
              disabled={!earnedHints}
            >
              Tell me the play's score.
            </Button>
          )}
          {!hints?.tiles?.revealed && (
            <Button
              type="primary"
              onClick={() => {
                revealHint('tiles');
              }}
              disabled={!earnedHints}
            >
              How many tiles should I play?
            </Button>
          )}
          {!hints?.position?.revealed && (
            <Button
              type="primary"
              onClick={() => {
                revealHint('position');
              }}
              disabled={!earnedHints}
            >
              Where does it go?
            </Button>
          )}
        </div>
      );
    }
    return (
      <div className="hint-actions">
        <p>
          Hints are available after{' '}
          {singularCount(RATED_ATTEMPTS, 'attempt', 'attempts')}.
        </p>
      </div>
    );
  }, [attempts, earnedHints, hints, revealHint]);

  const hintsRemaining = useMemo(() => {
    return Object.values(hints).filter((h) => !h.revealed).length;
  }, [hints]);

  if (solved !== PuzzleStatus.UNANSWERED) {
    return null;
  }
  return (
    <div className="puzzle-hints">
      {earnedHints > 0 && hintsRemaining > 0 && (
        <p className="hint-prompt">
          Having trouble? You've earned{' '}
          {singularCount(earnedHints, 'hint', 'hints')}.
        </p>
      )}
      <div className="displayed-hints">
        {Object.values(hints).map(renderHint)}
      </div>
      {actions}
    </div>
  );
};
