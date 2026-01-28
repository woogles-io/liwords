import React, { ReactNode, useMemo } from "react";
import { Button, Card } from "antd";
import { UsernameWithContext } from "../shared/usernameWithContext";
import moment from "moment";
import { timeCtrlToDisplayName } from "../store/constants";
import { PuzzleStatus } from "../gen/api/proto/puzzle_service/puzzle_service_pb";
import { StarFilled, StarOutlined } from "@ant-design/icons";
import { Hints } from "./hints";
import { PuzzleShareButton } from "./puzzle_share";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";
import { challengeRuleNames } from "../constants/challenge_rules";
import { PlayerInfo } from "../gen/api/proto/ipc/omgwords_pb";

type Props = {
  solved: number;
  gameDate?: Date;
  gameUrl?: string;
  lexicon: string;
  variantName: string;
  player1: Partial<PlayerInfo> | undefined;
  player2: Partial<PlayerInfo> | undefined;
  puzzleID?: string;
  ratingMode?: string;
  challengeRule: ChallengeRule | undefined;
  initialTimeSeconds?: number;
  incrementSeconds?: number;
  maxOvertimeMinutes?: number;
  attempts: number;
  userRating?: number;
  puzzleRating?: number;
  dateSolved?: Date;
  loadNewPuzzle: () => void;
  showSolution: () => void;
};

export const renderStars = (stars: number, useEmoji = false) => {
  const ret: ReactNode[] = [];
  let eRet = "";
  for (let i = stars; i > 0; i--) {
    if (useEmoji) {
      eRet += "⭐";
    } else ret.push(<StarFilled key={`star-${i}`} />);
  }
  if (useEmoji) {
    return eRet;
  }
  while (ret.length < MAX_STARS) {
    ret.push(
      <StarOutlined className="unearned" key={`unearned-star-${ret.length}`} />,
    );
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
    initialTimeSeconds,
    incrementSeconds,
    maxOvertimeMinutes,
    lexicon,
    variantName,
    player1,
    player2,
    puzzleID,
    attempts,
    dateSolved,
    loadNewPuzzle,
    showSolution,
  } = props;

  // TODO: should be determined on the back end and not hardcoded
  const puzzleType = "Equity puzzle";
  const score = calculatePuzzleScore(!!dateSolved, attempts);
  const attemptsText = useMemo(() => {
    if (solved === PuzzleStatus.CORRECT) {
      const solveDate = moment(dateSolved).format("MMMM D, YYYY");
      return (
        <p className="attempts-made">{`Solved in ${attempts} ${
          attempts === 1 ? "attempt" : "attempts"
        } on ${solveDate}`}</p>
      );
    }
    switch (solved) {
      case PuzzleStatus.CORRECT:
        const solveDate = moment(dateSolved).format("MMMM D, YYYY");
        return (
          <p className="attempts-made">{`Solved in ${attempts} ${
            attempts === 1 ? "attempt" : "attempts"
          } on ${solveDate}`}</p>
        );
      case PuzzleStatus.UNANSWERED:
        return (
          <p className="attempts-made">
            {`You have made ${attempts} ${
              attempts === 1 ? "attempt" : "attempts"
            }`}
          </p>
        );
      case PuzzleStatus.INCORRECT:
      default:
        return (
          <p className="attempts-made">{`You gave up after ${attempts} ${
            attempts === 1 ? "attempt" : "attempts"
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
        <PuzzleShareButton
          puzzleID={puzzleID}
          attempts={attempts}
          solved={solved}
        />
      </div>
    );
  }, [loadNewPuzzle, showSolution, solved, puzzleID, attempts]);

  const formattedGameDate = useMemo(() => {
    return moment(gameDate).format("MMMM D, YYYY");
  }, [gameDate]);

  const gameLink = useMemo(() => {
    if (gameUrl) {
      return (
        <span>
          {" • "}
          <a href={gameUrl} target="_blank" rel="noopener noreferrer">
            View Game
          </a>
        </span>
      );
    }
  }, [gameUrl]);

  const challengeDisplay = challengeRule
    ? challengeRuleNames[challengeRule]
    : "";

  const player1NameDisplay = player1?.nickname ? (
    <UsernameWithContext username={player1?.nickname || ""} />
  ) : (
    <span>unknown player</span>
  );
  const player2NameDisplay = player2?.nickname ? (
    <UsernameWithContext username={player2?.nickname || ""} />
  ) : (
    <span>unknown player</span>
  );
  const gamePlayersInfo = (
    <span className="player-title">
      Game played by {player1NameDisplay} vs {player2NameDisplay}
    </span>
  );
  const stillSolving = solved === PuzzleStatus.UNANSWERED;
  let settingsDisplay;
  if (!initialTimeSeconds && !incrementSeconds && !maxOvertimeMinutes) {
    settingsDisplay = (
      <p className="game-settings">{`${
        variantName || "classic"
      } • ${lexicon}`}</p>
    );
  } else {
    settingsDisplay = (
      <p className="game-settings">{`${
        timeCtrlToDisplayName(
          initialTimeSeconds || 0,
          incrementSeconds || 0,
          maxOvertimeMinutes || 0,
        )[0]
      } • ${variantName || "classic"} • ${lexicon}`}</p>
    );
  }

  return (
    <Card className="puzzle-info" title={`Puzzle Mode`} extra={puzzleType}>
      <div className="puzzle-details">
        {!stillSolving && (
          <>
            {renderStars(score)}
            <p>{gamePlayersInfo}</p>
            <p>
              {formattedGameDate}
              {gameLink}
            </p>
          </>
        )}
        {settingsDisplay}
        <div>
          {challengeDisplay}
          {challengeDisplay && ratingMode ? " • " : ""}
          {ratingMode}
        </div>
        <p className="instructions">
          There is a star play in this position that is significantly better
          than the second-best play. What would HastyBot play?
        </p>

        <div className="progress">{attemptsText}</div>
        {stillSolving && (
          <Hints
            puzzleID={props.puzzleID}
            solved={solved}
            attempts={attempts}
          />
        )}
        {actions}
      </div>
    </Card>
  );
});
