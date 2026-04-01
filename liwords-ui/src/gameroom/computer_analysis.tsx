import React, { useCallback, useMemo, useState } from "react";
import { Button, Modal, Table, Tag, Typography } from "antd";
import { QuestionCircleOutlined } from "@ant-design/icons";
import { fromJsonString } from "@bufbuild/protobuf";
import { useQuery } from "@connectrpc/connect-query";
import { getAnalysisResult } from "../gen/api/proto/analysis_service/analysis_service-AnalysisService_connectquery";
import {
  EndgameMove,
  EndgameVariation,
  GameAnalysisResult,
  GameAnalysisResultSchema,
  GamePhase,
  MistakeSize,
  PEGOutcomeType,
  PEGPlayInfo,
  PlyStats,
  SimmedPlayInfo,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import {
  Alphabet,
  machineLetterToRune,
  runesToMachineWord,
} from "../constants/alphabets";
import { MachineLetter, MachineWord } from "../utils/cwgame/common";
import {
  AnalyzerMove,
  liwordsLetterToWolgesLetter,
  usePlaceMoveCallback,
} from "./analyzer";
import { useExaminableGameContextStoreContext } from "../store/store";
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
      <div>~{elo} Elo rating</div>
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
        label={{
          value: "Mistake Score",
          position: "insideBottom",
          offset: -12,
          fontSize: 11,
        }}
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
      turns. <strong>Lower is better.</strong> A perfect game scores 0. Only
      mistakes of at least 0.25% win probability are counted — smaller
      deviations are considered to be within the noise.
    </Paragraph>

    <Table
      size="small"
      pagination={false}
      dataSource={[
        {
          key: "small",
          size: <Tag color="blue">Small</Tag>,
          penalty: "0.2",
          winPct: "0.25%–3%",
          blowout: "≤ 15 pts",
          endgame: "1–7 pts",
        },
        {
          key: "medium",
          size: <Tag color="orange">Medium</Tag>,
          penalty: "0.5",
          winPct: "> 3%–7%",
          blowout: "16–30 pts",
          endgame: "8–15 pts",
        },
        {
          key: "large",
          size: <Tag color="red">Large</Tag>,
          penalty: "1.0",
          winPct: "> 7%",
          blowout: "> 30 pts",
          endgame: "16+ pts, or blown EG",
        },
      ]}
      columns={[
        { title: "Size", dataIndex: "size", key: "size" },
        { title: "Penalty", dataIndex: "penalty", key: "penalty" },
        {
          title: (
            <>
              Early/Mid
              <br />
              win%
            </>
          ),
          dataIndex: "winPct",
          key: "winPct",
        },
        {
          title: (
            <>
              Early/Mid
              <br />
              spread†
            </>
          ),
          dataIndex: "blowout",
          key: "blowout",
        },
        {
          title: (
            <>
              Endgame /<br />
              Pre-EG spread
            </>
          ),
          dataIndex: "endgame",
          key: "endgame",
        },
      ]}
    />
    <Paragraph style={{ marginTop: 8, marginBottom: 0, fontSize: 11 }}>
      † Spread thresholds apply in early/mid game only when the position is
      already decided (win% &lt; 0.5% or &gt; 99.5%).
    </Paragraph>

    <Paragraph style={{ marginTop: 12 }}>
      Mistake Score is based on BestBot's analysis. BestBot is the strongest
      Scrabble AI we know of, but it is not infallible — there will be moves you
      disagree with, and occasionally it will be wrong. Over the long run we
      believe this metric is meaningful and that BestBot plays at a very high
      level, but please don't treat every flagged mistake as gospel. Use it as a
      guide, not a verdict.
    </Paragraph>

    <Paragraph>
      <strong>Estimated Elo rating:</strong> (thanks to Joey Krafchick —{" "}
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

// applyMoveToBoard returns a new board with the given tiles placed on it.
// Through-tiles (ml === 0) are skipped — the board already has a tile there.
const applyMoveToBoard = (
  boardLetters: MachineLetter[],
  dim: number,
  row: number,
  col: number,
  vertical: boolean,
  liwordsTiles: MachineWord,
): MachineLetter[] => {
  const newBoard = [...boardLetters];
  let r = row;
  let c = col;
  for (const ml of liwordsTiles) {
    if (ml !== 0) {
      newBoard[r * dim + c] = ml;
    }
    if (vertical) ++r;
    else ++c;
  }
  return newBoard;
};

type BoardContext = {
  dim: number;
  boardLetters: MachineLetter[];
  alphabet: Alphabet;
};

// parseMoveDescription parses a moveDescription string like "7A QUITc." into
// an AnalyzerMove that can be placed on the board and displayed with
// through-tile parenthesization (e.g. "QUI(T)c").
const parseMoveDescription = (
  desc: string,
  leave: string,
  score: number,
  equity: number,
  boardCtx: BoardContext,
): AnalyzerMove => {
  const { dim, boardLetters, alphabet } = boardCtx;
  const trimmed = desc.trim();

  // Exchange / Pass
  if (trimmed.startsWith("(")) {
    const leaveNum = runesToMachineWord(leave, alphabet);
    const isPass = /pass/i.test(trimmed);
    const exchMatch = trimmed.match(/^\(exch\s+(.+)\)$/i);
    const tilesStr = exchMatch ? exchMatch[1] : "";
    const liwordsTiles: MachineWord = tilesStr
      ? runesToMachineWord(tilesStr, alphabet)
      : [];
    const wolgesTiles = liwordsTiles.map(liwordsLetterToWolgesLetter);
    return {
      jsonKey: trimmed,
      displayMove: isPass ? "Pass" : `Exch. ${tilesStr}`,
      coordinates: "",
      leave: leaveNum,
      leaveWithGaps: leaveNum,
      score,
      equity,
      vertical: false,
      col: 0,
      row: 0,
      tiles: wolgesTiles,
      isExchange: true,
    };
  }

  const spaceIdx = trimmed.indexOf(" ");
  const coordStr = spaceIdx >= 0 ? trimmed.slice(0, spaceIdx) : trimmed;
  const tilesStr = spaceIdx >= 0 ? trimmed.slice(spaceIdx + 1) : "";

  let vertical = false;
  let row = 0;
  let col = 0;

  if (coordStr.length > 0) {
    if (coordStr[0] >= "0" && coordStr[0] <= "9") {
      // Horizontal: row-first e.g. "7A"
      const m = coordStr.match(/^(\d+)([A-Za-z]+)/);
      if (m) {
        row = parseInt(m[1], 10) - 1;
        col = m[2].toUpperCase().charCodeAt(0) - 65;
        vertical = false;
      }
    } else {
      // Vertical: col-first e.g. "A7"
      const m = coordStr.match(/^([A-Za-z]+)(\d+)/);
      if (m) {
        col = m[1].toUpperCase().charCodeAt(0) - 65;
        row = parseInt(m[2], 10) - 1;
        vertical = true;
      }
    }
  }

  const liwordsTiles: MachineWord = tilesStr
    ? runesToMachineWord(tilesStr, alphabet)
    : [];
  const leaveNum = runesToMachineWord(leave, alphabet);

  let displayMove = "";
  let r = row;
  let c = col;
  let inParen = false;
  const wolgesTiles: MachineLetter[] = [];

  for (const t of liwordsTiles) {
    if (t === 0) {
      // Through-tile: read the board letter and wrap in parentheses
      if (!inParen) {
        displayMove += "(";
        inParen = true;
      }
      displayMove += machineLetterToRune(
        boardLetters[r * dim + c],
        alphabet,
        false,
        true,
      );
      wolgesTiles.push(0);
    } else {
      if (inParen) {
        displayMove += ")";
        inParen = false;
      }
      displayMove += machineLetterToRune(t, alphabet, false, true);
      wolgesTiles.push(liwordsLetterToWolgesLetter(t));
    }
    if (vertical) ++r;
    else ++c;
  }
  if (inParen) displayMove += ")";

  return {
    jsonKey: trimmed,
    displayMove,
    coordinates: coordStr,
    leave: leaveNum,
    leaveWithGaps: leaveNum,
    score,
    equity,
    vertical,
    col,
    row,
    tiles: wolgesTiles,
    isExchange: false,
  };
};

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
      return "#4caf50";
    case PEGOutcomeType.PEG_OUTCOME_DRAW:
      return "#b8860b";
    case PEGOutcomeType.PEG_OUTCOME_LOSS:
      return "red";
    default:
      return "inherit";
  }
};

const PlyDetailsRow: React.FC<{
  plyStats: PlyStats[];
  colSpan: number;
  iterations: number;
}> = ({ plyStats, colSpan, iterations }) => {
  // Skip plies where the simulation didn't run (mean and stdev both 0)
  const activePlies = plyStats.filter(
    (ps) => ps.scoreMean !== 0 || ps.scoreStdev !== 0,
  );
  return (
    <tr>
      <td colSpan={colSpan} className="ca-ply-details">
        <div className="ca-ply-iters">
          {iterations.toLocaleString()} iterations
        </div>
        <table className="ca-ply-table">
          <tbody>
            {activePlies.map((ps) => (
              <tr key={ps.ply}>
                <td>
                  Ply {ps.ply} ({ps.ply % 2 === 1 ? "Opp" : "You"})
                </td>
                <td>
                  Avg {ps.scoreMean.toFixed(1)}, σ {ps.scoreStdev.toFixed(1)}
                </td>
                <td>Bingo {(ps.bingoPct * 100).toFixed(1)}%</td>
              </tr>
            ))}
          </tbody>
        </table>
      </td>
    </tr>
  );
};

const SimPlaysTable: React.FC<{
  plays: SimmedPlayInfo[];
  iterations: number;
  boardCtx: BoardContext;
  onClickPlay: (move: AnalyzerMove) => void;
}> = ({ plays, iterations, boardCtx, onClickPlay }) => {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);

  // Sort client-side: win% desc, then equity desc as tiebreaker.
  // This guarantees correct order regardless of server-side edge cases
  // (e.g. ignored/pruned plays appended out of order by extractTopSimPlays).
  const WIN_PCT_EPSILON = 1e-4;
  const sortedPlays = useMemo(
    () =>
      [...plays].sort((a, b) => {
        const wDiff = b.winProb - a.winProb;
        if (Math.abs(wDiff) > WIN_PCT_EPSILON) return wDiff > 0 ? 1 : -1;
        return b.equity - a.equity;
      }),
    [plays],
  );

  const parsedPlays = useMemo(
    () =>
      sortedPlays.map((p) =>
        parseMoveDescription(
          p.moveDescription,
          p.leave,
          p.score,
          p.equity,
          boardCtx,
        ),
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [sortedPlays, boardCtx.dim, boardCtx.boardLetters, boardCtx.alphabet],
  );

  return (
    <div className="ca-enriched-section ca-sim-table">
      <div className="ca-enriched-header">
        Simulation
        {iterations > 0 ? ` (${iterations.toLocaleString()} iters)` : ""}
      </div>
      <table className="ca-clickable-table">
        <thead>
          <tr>
            <th>Coord</th>
            <th>Move</th>
            <th>Score</th>
            <th>Leave</th>
            <th>Win%</th>
            <th>Equity</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {sortedPlays.map((p, i) => {
            const parsed = parsedPlays[i];
            const expanded = expandedIndex === i;
            const hasPlies = p.plyStats.length > 0;
            return (
              <React.Fragment key={i}>
                <tr
                  className={p.isPlayedMove ? "ca-sim-played" : ""}
                  onClick={() => onClickPlay(parsed)}
                >
                  <td className="ca-coord">{parsed.coordinates}</td>
                  <td className="ca-move-word">{parsed.displayMove}</td>
                  <td className="ca-bold">{p.score}</td>
                  <td>{p.leave}</td>
                  <td>{(p.winProb * 100).toFixed(1)}%</td>
                  <td>
                    {p.equity >= 0 ? "+" : ""}
                    {p.equity.toFixed(1)}
                  </td>
                  <td>
                    {hasPlies && (
                      <button
                        className="ca-expand-btn"
                        onClick={(e) => {
                          e.stopPropagation();
                          setExpandedIndex(expanded ? null : i);
                        }}
                        title={
                          expanded ? "Hide ply details" : "Show ply details"
                        }
                      >
                        {expanded ? "▾" : "▸"}
                      </button>
                    )}
                  </td>
                </tr>
                {expanded && hasPlies && (
                  <PlyDetailsRow
                    plyStats={p.plyStats}
                    colSpan={7}
                    iterations={p.iterations}
                  />
                )}
              </React.Fragment>
            );
          })}
        </tbody>
      </table>
      <div className="ca-sim-note">
        ≥5-ply simulation; stops early when the best play is identified with 99%
        confidence.
      </div>
    </div>
  );
};

const PEGPlaysTable: React.FC<{
  plays: PEGPlayInfo[];
  boardCtx: BoardContext;
  onClickPlay: (move: AnalyzerMove) => void;
}> = ({ plays, boardCtx, onClickPlay }) => {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);

  const parsedPlays = useMemo(
    () =>
      plays.map((p) =>
        parseMoveDescription(p.moveDescription, p.leave, p.score, 0, boardCtx),
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [plays, boardCtx.dim, boardCtx.boardLetters, boardCtx.alphabet],
  );

  const hasSpread = plays.some((p) => p.hasSpread);
  const colSpan = hasSpread ? 4 : 3;

  return (
    <div className="ca-enriched-section ca-peg-table">
      <div className="ca-enriched-header">Pre-Endgame</div>
      <table className="ca-clickable-table">
        <thead>
          <tr>
            <th>Coord</th>
            <th>Move</th>
            <th>Score</th>
            <th>Win%</th>
            {hasSpread && <th>Spread</th>}
            <th></th>
          </tr>
        </thead>
        <tbody>
          {plays.map((p, i) => {
            const parsed = parsedPlays[i];
            const expanded = expandedIndex === i;
            return (
              <React.Fragment key={i}>
                <tr
                  className={p.isPlayedMove ? "ca-sim-played" : ""}
                  onClick={() => onClickPlay(parsed)}
                >
                  <td className="ca-coord">{parsed.coordinates}</td>
                  <td className="ca-move-word">{parsed.displayMove}</td>
                  <td className="ca-bold">{p.score}</td>
                  <td>{(p.winProb * 100).toFixed(1)}%</td>
                  {hasSpread && (
                    <td>
                      {p.hasSpread
                        ? `${p.avgSpread >= 0 ? "+" : ""}${p.avgSpread.toFixed(1)}`
                        : "—"}
                    </td>
                  )}
                  <td>
                    {p.outcomes.length > 0 && (
                      <button
                        className="ca-expand-btn"
                        onClick={(e) => {
                          e.stopPropagation();
                          setExpandedIndex(expanded ? null : i);
                        }}
                        title={expanded ? "Hide outcomes" : "Show outcomes"}
                      >
                        {expanded ? "▾" : "▸"}
                      </button>
                    )}
                  </td>
                </tr>
                {expanded && p.outcomes.length > 0 && (
                  <tr>
                    <td colSpan={colSpan + 2} className="ca-peg-outcomes">
                      {[...p.outcomes]
                        .sort((a, b) => a.tiles.localeCompare(b.tiles))
                        .map((o, j) => (
                          <span
                            key={j}
                            style={{ color: pegOutcomeColor(o.outcome) }}
                            className="ca-peg-outcome-item"
                          >
                            <Text code>{o.tiles}</Text>→{" "}
                            {pegOutcomeLabel(o.outcome)}({o.count})
                          </span>
                        ))}
                    </td>
                  </tr>
                )}
              </React.Fragment>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

// parseEndgameVariation parses each move in sequence, threading a virtual
// board so that through-tiles on later moves resolve correctly.
const parseEndgameVariation = (
  moves: EndgameMove[],
  boardCtx: BoardContext,
): AnalyzerMove[] => {
  let currentBoard = boardCtx.boardLetters;
  const results: AnalyzerMove[] = [];
  for (const m of moves) {
    const ctx = { ...boardCtx, boardLetters: currentBoard };
    const parsed = parseMoveDescription(m.moveDescription, "", m.score, 0, ctx);
    results.push(parsed);
    if (!parsed.isExchange) {
      // Extract tile string and get liwords tiles to update the virtual board
      const trimmed = m.moveDescription.trim();
      const spaceIdx = trimmed.indexOf(" ");
      const tilesStr = spaceIdx >= 0 ? trimmed.slice(spaceIdx + 1) : "";
      const liwordsTiles = tilesStr
        ? runesToMachineWord(tilesStr, boardCtx.alphabet)
        : [];
      currentBoard = applyMoveToBoard(
        currentBoard,
        boardCtx.dim,
        parsed.row,
        parsed.col,
        parsed.vertical,
        liwordsTiles,
      );
    }
  }
  return results;
};

const EndgameVariationMoves: React.FC<{
  moves: EndgameMove[];
  boardCtx: BoardContext;
  onClickPlay: (move: AnalyzerMove) => void;
}> = ({ moves, boardCtx, onClickPlay }) => {
  const parsedMoves = useMemo(
    () => parseEndgameVariation(moves, boardCtx),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [moves, boardCtx.dim, boardCtx.boardLetters, boardCtx.alphabet],
  );
  return (
    <ol className="ca-endgame-moves">
      {moves.map((m, i) => {
        const parsed = parsedMoves[i];
        return (
          <li
            key={m.moveNumber}
            className="ca-endgame-move-clickable"
            onClick={() => onClickPlay(parsed)}
          >
            <Text code>
              {parsed
                ? parsed.coordinates
                  ? `${parsed.coordinates} ${parsed.displayMove}`
                  : parsed.displayMove
                : m.moveDescription}
            </Text>{" "}
            <span className="ca-endgame-score">({m.score})</span>
          </li>
        );
      })}
    </ol>
  );
};

const EndgameSequence: React.FC<{
  principalVariation?: EndgameVariation;
  otherVariations: EndgameVariation[];
  boardCtx: BoardContext;
  onClickPlay: (move: AnalyzerMove) => void;
}> = ({ principalVariation, otherVariations, boardCtx, onClickPlay }) => {
  const [showOthers, setShowOthers] = useState(false);
  if (!principalVariation) return null;

  // Filter out duplicate: macondo includes the PV as the first element of
  // otherVariations, so skip any variation that matches the PV exactly.
  const dedupedVariations = otherVariations.filter(
    (v) =>
      v.finalSpread !== principalVariation.finalSpread ||
      v.moves[0]?.moveDescription !==
        principalVariation.moves[0]?.moveDescription,
  );

  const spreadSign = principalVariation.finalSpread >= 0 ? "+" : "";
  return (
    <div className="ca-enriched-section ca-endgame-seq">
      <div className="ca-enriched-header">
        Endgame (best spread: {spreadSign}
        {principalVariation.finalSpread})
      </div>
      <EndgameVariationMoves
        moves={principalVariation.moves}
        boardCtx={boardCtx}
        onClickPlay={onClickPlay}
      />
      {dedupedVariations.length > 0 && (
        <div className="ca-endgame-others">
          <button
            className="ca-endgame-toggle"
            onClick={() => setShowOthers(!showOthers)}
          >
            {showOthers ? "▾" : "▸"} Alternative lines (
            {dedupedVariations.length})
          </button>
          {showOthers &&
            dedupedVariations.map((v, i) => {
              const vSign = v.finalSpread >= 0 ? "+" : "";
              return (
                <div key={i} className="ca-endgame-alt">
                  <span className="ca-endgame-alt-spread">
                    Spread: {vSign}
                    {v.finalSpread}
                  </span>
                  <EndgameVariationMoves
                    moves={v.moves}
                    boardCtx={boardCtx}
                    onClickPlay={onClickPlay}
                  />
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
  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const placeMove = usePlaceMoveCallback();

  const boardCtx: BoardContext = useMemo(
    () => ({
      dim: examinableGameContext.board.dim,
      boardLetters: examinableGameContext.board.letters,
      alphabet: examinableGameContext.alphabet,
    }),
    [
      examinableGameContext.board.dim,
      examinableGameContext.board.letters,
      examinableGameContext.alphabet,
    ],
  );

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
    (turn: GameAnalysisResult["turns"][0], ctx: BoardContext) => {
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
          {(() => {
            // Find leaves from sim plays so rack updates correctly on click
            const playedSimPlay = turn.topSimPlays.find((p) => p.isPlayedMove);
            const optimalSimPlay = turn.wasOptimal
              ? playedSimPlay
              : turn.topSimPlays.find(
                  (p) => p.moveDescription.trim() === turn.optimalMove.trim(),
                );
            const parsedPlayed = parseMoveDescription(
              turn.playedMove,
              playedSimPlay?.leave ?? "",
              turn.playedScore,
              0,
              ctx,
            );
            const parsedOptimal =
              !turn.wasOptimal && turn.optimalMove
                ? parseMoveDescription(
                    turn.optimalMove,
                    optimalSimPlay?.leave ?? "",
                    turn.optimalScore,
                    0,
                    ctx,
                  )
                : null;
            return (
              <>
                <div
                  className="ca-move-row ca-move-clickable"
                  onClick={() => placeMove(parsedPlayed)}
                >
                  <span className="ca-label">Played</span>
                  <Text code>
                    {parsedPlayed.coordinates
                      ? `${parsedPlayed.coordinates} ${parsedPlayed.displayMove}`
                      : parsedPlayed.displayMove}
                  </Text>
                  <Text type="secondary" className="ca-score">
                    ({turn.playedScore})
                  </Text>
                </div>
                {parsedOptimal && (
                  <div
                    className="ca-move-row ca-best-move ca-move-clickable"
                    onClick={() => placeMove(parsedOptimal)}
                  >
                    <span className="ca-label">Best</span>
                    <Text code>
                      {parsedOptimal.coordinates
                        ? `${parsedOptimal.coordinates} ${parsedOptimal.displayMove}`
                        : parsedOptimal.displayMove}
                    </Text>
                    <Text type="secondary" className="ca-score">
                      ({turn.optimalScore})
                    </Text>
                    <Text type="danger" className="ca-score">
                      {turn.phase === GamePhase.PHASE_ENDGAME ||
                      turn.winProbLoss === 0
                        ? `−${turn.spreadLoss}pts`
                        : `−${(turn.winProbLoss * 100).toFixed(1)}%`}
                    </Text>
                  </div>
                )}
              </>
            );
          })()}
          {notes.length > 0 && (
            <div className="ca-notes">{notes.join(", ")}</div>
          )}
          {isV2 && turn.topSimPlays.length > 0 && (
            <SimPlaysTable
              plays={turn.topSimPlays}
              iterations={turn.simIterations}
              boardCtx={ctx}
              onClickPlay={placeMove}
            />
          )}
          {isV2 && turn.topPegPlays.length > 0 && (
            <PEGPlaysTable
              plays={turn.topPegPlays}
              boardCtx={ctx}
              onClickPlay={placeMove}
            />
          )}
          {isV2 && turn.principalVariation && (
            <EndgameSequence
              principalVariation={turn.principalVariation}
              otherVariations={turn.otherVariations}
              boardCtx={ctx}
              onClickPlay={placeMove}
            />
          )}
        </div>
      );
    },
    [isV2, placeMove],
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
              ~{Math.round(ps.estimatedElo)} Elo rating
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
              turnsForCurrentPosition.map((t) => renderTurnRow(t, boardCtx))
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
