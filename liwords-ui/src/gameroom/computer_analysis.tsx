import React, { useCallback, useState } from "react";
import { Button, Modal, Table, Tag, Typography } from "antd";
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
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

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

// ELO data points matching macondo's estimateELO function
const ELO_DATA_POINTS = [
  { mi: 0.0, elo: 2300 },
  { mi: 0.2, elo: 2250 },
  { mi: 0.5, elo: 2200 },
  { mi: 0.8, elo: 2150 },
  { mi: 1.2, elo: 2100 },
  { mi: 1.5, elo: 2050 },
  { mi: 1.7, elo: 2000 },
  { mi: 1.9, elo: 1950 },
  { mi: 2.3, elo: 1900 },
  { mi: 2.6, elo: 1850 },
  { mi: 2.9, elo: 1800 },
  { mi: 3.3, elo: 1750 },
  { mi: 3.8, elo: 1700 },
  { mi: 4.2, elo: 1650 },
];

function estimateELO(mi: number): number {
  if (mi <= 0) return 2300;
  if (mi > 4.2) return 1650 - 125 * (mi - 4.2);
  for (let i = 1; i < ELO_DATA_POINTS.length; i++) {
    if (mi <= ELO_DATA_POINTS[i].mi) {
      const lower = ELO_DATA_POINTS[i - 1];
      const upper = ELO_DATA_POINTS[i];
      const t = (mi - lower.mi) / (upper.mi - lower.mi);
      return lower.elo + t * (upper.elo - lower.elo);
    }
  }
  return 1650;
}

// Generate graph data points at fine resolution (Mistake Score 0 to 10)
const ELO_GRAPH_DATA = Array.from({ length: 141 }, (_, i) => {
  const mi = (i / 140) * 10;
  return { mi: parseFloat(mi.toFixed(2)), elo: Math.round(estimateELO(mi)) };
});

type ELOTooltipProps = {
  active?: boolean;
  payload?: { value: number; payload: { mi: number } }[];
};

const ELOTooltip: React.FC<ELOTooltipProps> = ({ active, payload }) => {
  if (!active || !payload?.length) return null;
  const dark = localStorage.getItem("darkMode") === "true";
  const { mi } = payload[0].payload;
  const elo = payload[0].value;
  return (
    <div
      style={{
        background: dark ? "#3a3a3a" : "#ffffff",
        border: `1px solid ${dark ? "#515151" : "#bebebe"}`,
        color: dark ? "#ffffff" : "#414141",
        borderRadius: 6,
        padding: "4px 8px",
        fontSize: 11,
      }}
    >
      <div style={{ color: dark ? "#cccccc" : "#999999" }}>
        Score {mi.toFixed(2)}
      </div>
      <div>~{elo} ELO</div>
    </div>
  );
};

const MIELOGraph: React.FC = () => (
  <ResponsiveContainer width="100%" height={240}>
    <LineChart
      data={ELO_GRAPH_DATA}
      margin={{ top: 4, right: 12, bottom: 20, left: 8 }}
    >
      <CartesianGrid strokeDasharray="3 3" />
      <XAxis
        dataKey="mi"
        type="number"
        domain={[0, 10]}
        ticks={[0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10]}
        label={{ value: "Mistake Score", position: "insideBottom", offset: -12, fontSize: 11 }}
        tick={{ fontSize: 10 }}
      />
      <YAxis
        domain={[800, 2400]}
        ticks={[800, 1000, 1200, 1400, 1600, 1800, 2000, 2200, 2400]}
        tick={{ fontSize: 10 }}
        width={38}
      />
      <Tooltip content={<ELOTooltip />} wrapperStyle={{ outline: "none" }} />
      <Line
        type="monotone"
        dataKey="elo"
        stroke="#1677ff"
        dot={false}
        strokeWidth={2}
      />
    </LineChart>
  </ResponsiveContainer>
);

const MistakeIndexHelp: React.FC = () => (
  <>
    <Paragraph>
      The <strong>Mistake Score</strong> measures the total error in a game.
      Each suboptimal move earns penalty points, which are summed across all
      turns. <strong>Lower is better.</strong> A perfect game scores 0.
    </Paragraph>

    <Table
      size="small"
      pagination={false}
      dataSource={[
        { key: "small",  size: <Tag color="blue">Small</Tag>,   penalty: "0.2", winPct: "≤ 3%",      blowout: "≤ 15 pts",  endgame: "1–7 pts" },
        { key: "medium", size: <Tag color="orange">Medium</Tag>, penalty: "0.5", winPct: "> 3%–7%",   blowout: "16–30 pts", endgame: "8–15 pts" },
        { key: "large",  size: <Tag color="red">Large</Tag>,    penalty: "1.0", winPct: "> 7%",       blowout: "> 30 pts",  endgame: "16+ pts, or blown EG" },
      ]}
      columns={[
        { title: "Size",                          dataIndex: "size",    key: "size" },
        { title: "Penalty",                       dataIndex: "penalty", key: "penalty" },
        { title: <>Early/Mid<br />win%</>,         dataIndex: "winPct",  key: "winPct" },
        { title: <>Early/Mid<br />spread†</>,      dataIndex: "blowout", key: "blowout" },
        { title: <>Endgame /<br />Pre-EG spread</>, dataIndex: "endgame", key: "endgame" },
      ]}
    />
    <Paragraph style={{ marginTop: 8, marginBottom: 0, fontSize: 11 }}>
      † Spread thresholds apply in early/mid game only when the position is
      already decided (win% &lt; 0.5% or &gt; 99.5%).
    </Paragraph>

    <Paragraph>
      <strong>Estimated ELO:</strong> (thanks to Joey Krafchick —{" "}
      <a
        href="https://nbaniac.com/odds-overview"
        target="_blank"
        rel="noreferrer"
      >
        nbaniac.com/odds-overview
      </a>
      ) Hover for values.
    </Paragraph>
    <div className="ca-elo-graph">
      <MIELOGraph />
    </div>
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
    <div className="ca-sim-note">
      5-ply simulation; stops early when the best play is identified with 99%
      confidence.
    </div>
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
                {turn.phase === GamePhase.PHASE_ENDGAME || turn.winProbLoss === 0
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
            Mistakes <strong>{ps.mistakeIndex.toFixed(1)}</strong>
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
        title="About Mistake Score"
        aria-label="About Mistake Score"
      >
        <QuestionCircleOutlined />
      </button>
    </div>
  ) : null;

  return (
    <>
      <Modal
        title="About Mistake Score"
        open={helpOpen}
        onCancel={() => setHelpOpen(false)}
        footer={null}
        width={680}
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
