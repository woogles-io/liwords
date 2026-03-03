import React, { useCallback, useState } from "react";
import { Modal, Tag, Typography } from "antd";
import { QuestionCircleOutlined } from "@ant-design/icons";
import { fromJsonString } from "@bufbuild/protobuf";
import { useQuery } from "@connectrpc/connect-query";
import { getAnalysisResult } from "../gen/api/proto/analysis_service/analysis_service-AnalysisService_connectquery";
import {
  GameAnalysisResult,
  GameAnalysisResultSchema,
  GamePhase,
  MistakeSize,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { RedoOutlined } from "@ant-design/icons";

const { Text, Paragraph } = Typography;

const phaseLabel = (phase: GamePhase): string => {
  switch (phase) {
    case GamePhase.PHASE_EARLY_MID:
      return "Early/Mid";
    case GamePhase.PHASE_EARLY_PREENDGAME:
    case GamePhase.PHASE_PREENDGAME:
      return "Pre-EG";
    case GamePhase.PHASE_ENDGAME:
      return "Endgame";
    default:
      return "";
  }
};

const mistakeColor = (size: MistakeSize): string => {
  switch (size) {
    case MistakeSize.NO_MISTAKE:
      return "green";
    case MistakeSize.SMALL:
      return "blue";
    case MistakeSize.MEDIUM:
      return "orange";
    case MistakeSize.LARGE:
      return "red";
    default:
      return "default";
  }
};

const mistakeLabel = (size: MistakeSize): string => {
  switch (size) {
    case MistakeSize.NO_MISTAKE:
      return "Optimal";
    case MistakeSize.SMALL:
      return "Small";
    case MistakeSize.MEDIUM:
      return "Medium";
    case MistakeSize.LARGE:
      return "Large";
    default:
      return "?";
  }
};

const MistakeIndexHelp: React.FC = () => (
  <>
    <Paragraph>
      The <strong>Mistake Index</strong> measures the total error in a game.
      Each suboptimal move earns penalty points, which are summed across all
      turns. <strong>Lower is better.</strong> A perfect game scores 0.
    </Paragraph>

    <Paragraph>
      <strong>Penalty per mistake:</strong>
    </Paragraph>
    <ul>
      <li>
        <Tag color="blue">Small</Tag> = 0.2 pts
      </li>
      <li>
        <Tag color="orange">Medium</Tag> = 0.5 pts
      </li>
      <li>
        <Tag color="red">Large</Tag> = 1.0 pts
      </li>
    </ul>

    <Paragraph>
      <strong>Mistake thresholds — early/mid game &amp; pre-endgame</strong>{" "}
      (based on win% loss from simulation):
    </Paragraph>
    <ul>
      <li>
        <Tag color="blue">Small</Tag> ≤ 3% win probability loss
      </li>
      <li>
        <Tag color="orange">Medium</Tag> 4–7% win probability loss
      </li>
      <li>
        <Tag color="red">Large</Tag> &gt; 7% win probability loss
      </li>
    </ul>

    <Paragraph>
      <strong>
        Mistake thresholds — endgame &amp; pre-endgame spread tiebreak
      </strong>{" "}
      (based on spread loss in points):
    </Paragraph>
    <ul>
      <li>
        <Tag color="blue">Small</Tag> 1–7 points
      </li>
      <li>
        <Tag color="orange">Medium</Tag> 8–15 points
      </li>
      <li>
        <Tag color="red">Large</Tag> 16+ points
      </li>
      <li>
        <Tag color="red">Large</Tag> Blown endgame (win turned into loss/tie),
        regardless of spread
      </li>
    </ul>

    <Paragraph>
      <strong>Estimated ELO mapping:</strong> (thanks to Joey Krafchick for his
      methodology —{" "}
      <a
        href="https://nbaniac.com/odds-overview"
        target="_blank"
        rel="noreferrer"
      >
        nbaniac.com/odds-overview
      </a>
      )
    </Paragraph>
    <table className="ca-elo-table">
      <thead>
        <tr>
          <th>Mistake Index</th>
          <th>Est. ELO</th>
        </tr>
      </thead>
      <tbody>
        {[
          ["0.0", "2300"],
          ["0.2", "2250"],
          ["0.5", "2200"],
          ["0.8", "2150"],
          ["1.2", "2100"],
          ["1.5", "2050"],
          ["1.7", "2000"],
          ["1.9", "1950"],
          ["2.3", "1900"],
          ["2.6", "1850"],
          ["2.9", "1800"],
          ["3.3", "1750"],
          ["3.8", "1700"],
          ["4.2", "1650"],
          ["4.2+", "< 1650"],
        ].map(([mi, elo]) => (
          <tr key={mi}>
            <td>{mi}</td>
            <td>{elo}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </>
);

type ComputerAnalysisProps = {
  gameID: string;
  currentTurn: number; // number of completed turns (examinableGameContext.turns.length)
  onBack: () => void;
};

export const ComputerAnalysis: React.FC<ComputerAnalysisProps> = ({
  gameID,
  currentTurn,
  onBack,
}) => {
  const [helpOpen, setHelpOpen] = useState(false);

  const { data: result, isLoading } = useQuery(
    getAnalysisResult,
    { gameId: gameID },
    {
      select: (resp) => {
        if (!resp.found || !resp.resultProto.length) return null;
        const jsonStr = new TextDecoder().decode(resp.resultProto);
        return fromJsonString(GameAnalysisResultSchema, jsonStr);
      },
    },
  );

  const turnsForCurrentPosition = result?.turns.filter(
    (t) => t.turnNumber === currentTurn + 1,
  );

  const renderTurnRow = useCallback((turn: GameAnalysisResult["turns"][0]) => {
    const notes: string[] = [];
    if (turn.isPhony) notes.push("Phony");
    if (turn.phonyChallenged) notes.push("Challenged");
    if (turn.missedChallenge) notes.push("Missed chal.");
    if (turn.missedBingo) notes.push("Missed bingo");
    if (turn.blownEndgame) notes.push("Blown EG");

    return (
      <div
        key={`${turn.turnNumber}-${turn.playerIndex}`}
        className="ca-turn-row"
      >
        <div className="ca-turn-header">
          <span className="ca-player-name">{turn.playerName}</span>
          <span className="ca-sep">·</span>
          <span className="ca-phase">{phaseLabel(turn.phase)}</span>
          <Tag color={mistakeColor(turn.mistakeSize)} className="ca-tag">
            {mistakeLabel(turn.mistakeSize)}
          </Tag>
        </div>
        <div className="ca-move-row">
          <span className="ca-label">Played</span>
          <Text code>{turn.playedMove}</Text>
          <Text type="secondary" className="ca-score">
            ({turn.playedScore})
          </Text>
        </div>
        {!turn.wasOptimal && (
          <div className="ca-move-row ca-best-move">
            <span className="ca-label">Best</span>
            <Text code>{turn.optimalMove}</Text>
            <Text type="secondary" className="ca-score">
              ({turn.optimalScore})
            </Text>
            <Text type="danger" className="ca-score">
              {turn.phase === GamePhase.PHASE_ENDGAME
                ? `−${turn.spreadLoss}pts`
                : `−${(turn.winProbLoss * 100).toFixed(1)}%`}
            </Text>
          </div>
        )}
        {notes.length > 0 && <div className="ca-notes">{notes.join(", ")}</div>}
      </div>
    );
  }, []);

  const toolbar = (
    <div className="ca-toolbar">
      <strong className="ca-title">BestBot Analysis</strong>
      <button className="ca-back-btn" onClick={onBack}>
        ← Static Analyzer
      </button>
    </div>
  );

  const playerSummaryBar = result ? (
    <div className="ca-summary-bar">
      {result.playerSummaries.map((ps) => (
        <div key={ps.playerName} className="ca-summary-player">
          <span className="ca-summary-name">{ps.playerName}</span>
          <span className="ca-summary-mi">
            MI <strong>{ps.mistakeIndex.toFixed(1)}</strong>
          </span>
          {ps.estimatedElo > 0 && (
            <span className="ca-summary-elo">
              ~{Math.round(ps.estimatedElo)} ELO
            </span>
          )}
        </div>
      ))}
      <button
        className="ca-help-btn"
        onClick={() => setHelpOpen(true)}
        title="About Mistake Index"
        aria-label="About Mistake Index"
      >
        <QuestionCircleOutlined />
      </button>
    </div>
  ) : null;

  return (
    <>
      <Modal
        title="About Mistake Index"
        open={helpOpen}
        onCancel={() => setHelpOpen(false)}
        footer={null}
        width={480}
      >
        <MistakeIndexHelp />
      </Modal>

      {isLoading ? (
        <div className="computer-analysis">
          {toolbar}
          <div className="ca-loading">
            <RedoOutlined spin /> Loading analysis…
          </div>
        </div>
      ) : !result ? (
        <div className="computer-analysis">
          {toolbar}
          <div className="ca-empty">Analysis data unavailable.</div>
        </div>
      ) : (
        <div className="computer-analysis">
          {toolbar}
          <div className="ca-turns">
            {turnsForCurrentPosition && turnsForCurrentPosition.length > 0 ? (
              turnsForCurrentPosition.map(renderTurnRow)
            ) : (
              <div className="ca-empty-turn">
                Navigate to a turn to see analysis.
              </div>
            )}
          </div>
          {playerSummaryBar}
        </div>
      )}
    </>
  );
};
