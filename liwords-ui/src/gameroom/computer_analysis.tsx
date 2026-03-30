import React, { useCallback, useState } from "react";
import { Button, Modal, Tag, Typography } from "antd";
import { QuestionCircleOutlined } from "@ant-design/icons";
import { fromJsonString } from "@bufbuild/protobuf";
import { useQuery } from "@connectrpc/connect-query";
import { getAnalysisResult } from "../gen/api/proto/analysis_service/analysis_service-AnalysisService_connectquery";
import {
  EndgameVariation,
  GameAnalysisResult,
  GameAnalysisResultSchema,
  GamePhase,
  MistakeSize,
  PEGOutcomeType,
  PEGPlayInfo,
  SimmedPlayInfo,
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

// --- Enriched data sub-components (v2+ only) ---

const pegOutcomeLabel = (outcome: PEGOutcomeType): string => {
  switch (outcome) {
    case PEGOutcomeType.PEG_OUTCOME_WIN:
      return "Win";
    case PEGOutcomeType.PEG_OUTCOME_DRAW:
      return "Draw";
    case PEGOutcomeType.PEG_OUTCOME_LOSS:
      return "Loss";
    default:
      return "?";
  }
};

const pegOutcomeColor = (outcome: PEGOutcomeType): string => {
  switch (outcome) {
    case PEGOutcomeType.PEG_OUTCOME_WIN:
      return "green";
    case PEGOutcomeType.PEG_OUTCOME_DRAW:
      return "#b8860b";
    case PEGOutcomeType.PEG_OUTCOME_LOSS:
      return "red";
    default:
      return "inherit";
  }
};

const SimPlaysTable: React.FC<{
  plays: SimmedPlayInfo[];
  iterations: number;
}> = ({ plays, iterations }) => (
  <div className="ca-enriched-section ca-sim-table">
    <div className="ca-enriched-header">
      Simulation{iterations > 0 ? ` (${iterations.toLocaleString()} iters)` : ""}
    </div>
    <table>
      <thead>
        <tr>
          <th>Move</th>
          <th>Score</th>
          <th>Leave</th>
          <th>Win%</th>
          <th>Equity</th>
        </tr>
      </thead>
      <tbody>
        {plays.map((p, i) => (
          <tr key={i} className={p.isPlayedMove ? "ca-sim-played" : ""}>
            <td>
              <Text code>{p.moveDescription}</Text>
            </td>
            <td>{p.score}</td>
            <td>
              <Text code>{p.leave}</Text>
            </td>
            <td>{(p.winProb * 100).toFixed(1)}%</td>
            <td>
              {p.equity >= 0 ? "+" : ""}
              {p.equity.toFixed(1)}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const PEGPlaysTable: React.FC<{ plays: PEGPlayInfo[] }> = ({ plays }) => {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);
  return (
    <div className="ca-enriched-section ca-peg-table">
      <div className="ca-enriched-header">Pre-Endgame</div>
      <table>
        <thead>
          <tr>
            <th>Move</th>
            <th>Score</th>
            <th>Win%</th>
            {plays.some((p) => p.hasSpread) && <th>Spread</th>}
          </tr>
        </thead>
        <tbody>
          {plays.map((p, i) => (
            <React.Fragment key={i}>
              <tr
                className={p.isPlayedMove ? "ca-sim-played" : ""}
                onClick={() =>
                  setExpandedIndex(expandedIndex === i ? null : i)
                }
                style={{ cursor: "pointer" }}
              >
                <td>
                  <Text code>{p.moveDescription}</Text>
                </td>
                <td>{p.score}</td>
                <td>{(p.winProb * 100).toFixed(1)}%</td>
                {plays.some((pl) => pl.hasSpread) && (
                  <td>
                    {p.hasSpread
                      ? `${p.avgSpread >= 0 ? "+" : ""}${p.avgSpread.toFixed(1)}`
                      : "—"}
                  </td>
                )}
              </tr>
              {expandedIndex === i && p.outcomes.length > 0 && (
                <tr>
                  <td colSpan={4} className="ca-peg-outcomes">
                    {p.outcomes.map((o, j) => (
                      <span
                        key={j}
                        style={{ color: pegOutcomeColor(o.outcome) }}
                        className="ca-peg-outcome-item"
                      >
                        <Text code>{o.tiles}</Text>→{pegOutcomeLabel(o.outcome)}
                        ({o.count})
                      </span>
                    ))}
                  </td>
                </tr>
              )}
            </React.Fragment>
          ))}
        </tbody>
      </table>
    </div>
  );
};

const EndgameSequence: React.FC<{
  principalVariation?: EndgameVariation;
  otherVariations: EndgameVariation[];
}> = ({ principalVariation, otherVariations }) => {
  const [showOthers, setShowOthers] = useState(false);
  if (!principalVariation) return null;
  const spreadSign = principalVariation.finalSpread >= 0 ? "+" : "";
  return (
    <div className="ca-enriched-section ca-endgame-seq">
      <div className="ca-enriched-header">
        Endgame (best spread: {spreadSign}
        {principalVariation.finalSpread})
      </div>
      <ol className="ca-endgame-moves">
        {principalVariation.moves.map((m) => (
          <li key={m.moveNumber}>
            <Text code>{m.moveDescription}</Text>{" "}
            <span className="ca-endgame-score">({m.score})</span>
          </li>
        ))}
      </ol>
      {otherVariations.length > 0 && (
        <div className="ca-endgame-others">
          <button
            className="ca-endgame-toggle"
            onClick={() => setShowOthers(!showOthers)}
          >
            {showOthers ? "▾" : "▸"} Alternative lines ({otherVariations.length})
          </button>
          {showOthers &&
            otherVariations.map((v, i) => {
              const vSign = v.finalSpread >= 0 ? "+" : "";
              return (
                <div key={i} className="ca-endgame-alt">
                  <span className="ca-endgame-alt-spread">
                    Spread: {vSign}
                    {v.finalSpread}
                  </span>
                  <ol className="ca-endgame-moves">
                    {v.moves.map((m) => (
                      <li key={m.moveNumber}>
                        <Text code>{m.moveDescription}</Text>{" "}
                        <span className="ca-endgame-score">({m.score})</span>
                      </li>
                    ))}
                  </ol>
                </div>
              );
            })}
        </div>
      )}
    </div>
  );
};

type ComputerAnalysisProps = {
  gameID: string;
  currentTurn: number; // number of completed turns (examinableGameContext.turns.length)
  onBack: () => void;
  isLegacy?: boolean;
  onRequestReanalysis?: () => void;
};

export const ComputerAnalysis: React.FC<ComputerAnalysisProps> = ({
  gameID,
  currentTurn,
  onBack,
  isLegacy,
  onRequestReanalysis,
}) => {
  const [helpOpen, setHelpOpen] = useState(false);

  const { data: result, isLoading } = useQuery(
    getAnalysisResult,
    { gameId: gameID },
    {
      select: (resp) => {
        if (!resp.found) return null;
        // Prefer typed result (v2+), fall back to deprecated bytes
        if (resp.result && resp.result.turns.length > 0) return resp.result;
        if (resp.resultProto.length) {
          const jsonStr = new TextDecoder().decode(resp.resultProto);
          return fromJsonString(GameAnalysisResultSchema, jsonStr);
        }
        return null;
      },
    },
  );

  const turnsForCurrentPosition = result?.turns.filter(
    (t) => t.turnNumber === currentTurn + 1,
  );

  const isV2 = (result?.analysisVersion ?? 0) >= 2;

  const renderTurnRow = useCallback(
    (turn: GameAnalysisResult["turns"][0]) => {
      const notes: string[] = [];
      if (turn.isPhony) notes.push("Phony");
      if (turn.phonyChallenged) notes.push("Challenged");
      if (turn.missedChallenge) notes.push("Missed chal.");
      if (turn.missedBingo) notes.push("Missed bingo");
      if (turn.blownEndgame) notes.push("Blown EG");
      if (turn.knownOppRack) notes.push(`Opp. rack: ${turn.knownOppRack}`);

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
          {notes.length > 0 && (
            <div className="ca-notes">{notes.join(", ")}</div>
          )}
          {isV2 && turn.topSimPlays.length > 0 && (
            <SimPlaysTable
              plays={turn.topSimPlays}
              iterations={turn.simIterations}
            />
          )}
          {isV2 && turn.topPegPlays.length > 0 && (
            <PEGPlaysTable plays={turn.topPegPlays} />
          )}
          {isV2 && turn.principalVariation && (
            <EndgameSequence
              principalVariation={turn.principalVariation}
              otherVariations={turn.otherVariations}
            />
          )}
        </div>
      );
    },
    [isV2],
  );

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
          {isLegacy && (
            <div className="ca-legacy-banner">
              <span>Analyzed with older engine</span>
              <Button size="small" type="link" onClick={onRequestReanalysis}>
                Request Updated Analysis
              </Button>
            </div>
          )}
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
