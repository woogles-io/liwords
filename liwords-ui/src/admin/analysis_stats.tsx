import React, { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import {
  Button,
  Card,
  Col,
  Modal,
  Row,
  Statistic,
  Table,
  Tag,
  Typography,
} from "antd";
import { create, fromJsonString } from "@bufbuild/protobuf";
import { useClient, flashError } from "../utils/hooks/connect";
import {
  AnalysisAdminService,
  AnalyzedGameSummary,
  GetAdminStatsRequestSchema,
  ListAnalyzedGamesRequestSchema,
  RequeueAnalysisRequestSchema,
} from "../gen/api/proto/analysis_service/analysis_service_pb";
import {
  AnalysisService,
  GetAnalysisResultRequestSchema,
} from "../gen/api/proto/analysis_service/analysis_service_pb";
import {
  GameAnalysisResult,
  GameAnalysisResultSchema,
  GamePhase,
  MistakeSize,
} from "../gen/api/proto/vendored/macondo/macondo_pb";

const { Text } = Typography;

const PAGE_SIZE = 50;

const phaseLabel = (phase: GamePhase): string => {
  switch (phase) {
    case GamePhase.PHASE_EARLY_MID:
      return "Early/Mid";
    case GamePhase.PHASE_EARLY_PREENDGAME:
      return "Pre-endgame";
    case GamePhase.PHASE_PREENDGAME:
      return "Preendgame";
    case GamePhase.PHASE_ENDGAME:
      return "Endgame";
    default:
      return "Unknown";
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

type GameDetailModalProps = {
  gameId: string | null;
  onClose: () => void;
};

const GameDetailModal = ({ gameId, onClose }: GameDetailModalProps) => {
  const [result, setResult] = useState<GameAnalysisResult | null>(null);
  const [loading, setLoading] = useState(false);
  const analysisClient = useClient(AnalysisService);

  useEffect(() => {
    if (!gameId) {
      setResult(null);
      return;
    }
    setLoading(true);
    analysisClient
      .getAnalysisResult(create(GetAnalysisResultRequestSchema, { gameId }))
      .then((resp) => {
        if (resp.found && resp.resultProto.length > 0) {
          const jsonStr = new TextDecoder().decode(resp.resultProto);
          const parsed = fromJsonString(GameAnalysisResultSchema, jsonStr);
          setResult(parsed);
        } else {
          setResult(null);
        }
      })
      .catch(flashError)
      .finally(() => setLoading(false));
  }, [gameId, analysisClient]);

  const turnColumns = [
    {
      title: "#",
      dataIndex: "turnNumber",
      key: "turnNumber",
      width: 45,
    },
    {
      title: "Player",
      dataIndex: "playerName",
      key: "playerName",
      width: 100,
    },
    {
      title: "Rack",
      dataIndex: "rack",
      key: "rack",
      width: 80,
      render: (rack: string) => (
        <Text code style={{ fontSize: 12 }}>
          {rack}
        </Text>
      ),
    },
    {
      title: "Phase",
      dataIndex: "phase",
      key: "phase",
      width: 90,
      render: (phase: GamePhase) => (
        <Text style={{ fontSize: 12 }}>{phaseLabel(phase)}</Text>
      ),
    },
    {
      title: "Played",
      key: "played",
      width: 140,
      render: (_: unknown, row: GameAnalysisResult["turns"][0]) => (
        <span>
          <Text code style={{ fontSize: 12 }}>
            {row.playedMove}
          </Text>{" "}
          <Text type="secondary" style={{ fontSize: 11 }}>
            ({row.playedScore})
          </Text>
        </span>
      ),
    },
    {
      title: "Best",
      key: "optimal",
      width: 140,
      render: (_: unknown, row: GameAnalysisResult["turns"][0]) =>
        row.wasOptimal ? (
          <Text type="secondary" style={{ fontSize: 12 }}>
            —
          </Text>
        ) : (
          <span>
            <Text code style={{ fontSize: 12 }}>
              {row.optimalMove}
            </Text>{" "}
            <Text type="secondary" style={{ fontSize: 11 }}>
              ({row.optimalScore})
            </Text>
          </span>
        ),
    },
    {
      title: "Loss",
      key: "loss",
      width: 80,
      render: (_: unknown, row: GameAnalysisResult["turns"][0]) => {
        if (row.wasOptimal) return <Text type="secondary">—</Text>;
        if (row.phase === GamePhase.PHASE_ENDGAME) {
          return (
            <Text type="danger" style={{ fontSize: 12 }}>
              {row.spreadLoss > 0 ? `-${row.spreadLoss}` : row.spreadLoss} pts
            </Text>
          );
        }
        const pct = (row.winProbLoss * 100).toFixed(1);
        return (
          <Text type="danger" style={{ fontSize: 12 }}>
            -{pct}%
          </Text>
        );
      },
    },
    {
      title: "Quality",
      key: "quality",
      width: 80,
      render: (_: unknown, row: GameAnalysisResult["turns"][0]) => (
        <Tag color={mistakeColor(row.mistakeSize)} style={{ fontSize: 11 }}>
          {mistakeLabel(row.mistakeSize)}
        </Tag>
      ),
    },
    {
      title: "Notes",
      key: "notes",
      width: 120,
      render: (_: unknown, row: GameAnalysisResult["turns"][0]) => {
        const flags: string[] = [];
        if (row.isPhony) flags.push("Phony");
        if (row.phonyChallenged) flags.push("Challenged");
        if (row.missedChallenge) flags.push("Missed challenge");
        if (row.missedBingo) flags.push("Missed bingo");
        if (row.blownEndgame) flags.push("Blown endgame");
        return flags.length > 0 ? (
          <span style={{ fontSize: 11 }}>{flags.join(", ")}</span>
        ) : null;
      },
    },
  ];

  const summaryColumns = [
    { title: "Player", dataIndex: "playerName", key: "playerName", width: 120 },
    { title: "Turns", dataIndex: "turnsPlayed", key: "turnsPlayed", width: 60 },
    {
      title: "Optimal",
      dataIndex: "optimalMoves",
      key: "optimalMoves",
      width: 70,
    },
    {
      title: "Avg Win% Loss",
      key: "avgWinLoss",
      width: 110,
      render: (_: unknown, row: GameAnalysisResult["playerSummaries"][0]) =>
        `${(row.avgWinProbLoss * 100).toFixed(2)}%`,
    },
    {
      title: "Mistakes (S/M/L)",
      key: "mistakes",
      width: 130,
      render: (_: unknown, row: GameAnalysisResult["playerSummaries"][0]) =>
        `${row.smallMistakes}/${row.mediumMistakes}/${row.largeMistakes}`,
    },
    {
      title: "Est. Elo",
      key: "elo",
      dataIndex: "estimatedElo",
      width: 80,
      render: (elo: number) => Math.round(elo),
    },
    {
      title: "Missed Bingos",
      key: "missedBingos",
      width: 110,
      render: (_: unknown, row: GameAnalysisResult["playerSummaries"][0]) =>
        `${row.missedBingos}/${row.availableBingos}`,
    },
  ];

  return (
    <Modal
      title={
        <span>
          Analysis:{" "}
          <Link to={`/game/${gameId}`} target="_blank">
            {gameId}
          </Link>
        </span>
      }
      open={!!gameId}
      onCancel={onClose}
      footer={null}
      width={1000}
    >
      {loading ? (
        <div style={{ textAlign: "center", padding: 40 }}>Loading…</div>
      ) : !result ? (
        <div>No analysis data found.</div>
      ) : (
        <>
          <Typography.Title level={5} style={{ marginTop: 0 }}>
            Player Summaries
          </Typography.Title>
          <Table
            dataSource={result.playerSummaries}
            columns={summaryColumns}
            rowKey="playerName"
            pagination={false}
            size="small"
            style={{ marginBottom: 24 }}
          />
          <Typography.Title level={5}>Turn-by-Turn</Typography.Title>
          <Table
            dataSource={result.turns}
            columns={turnColumns}
            rowKey={(row) => `${row.turnNumber}-${row.playerIndex}`}
            pagination={{ pageSize: 30, showSizeChanger: false }}
            size="small"
            scroll={{ x: "max-content" }}
          />
        </>
      )}
    </Modal>
  );
};

export const AnalysisStats = () => {
  const [totalCompleted, setTotalCompleted] = useState(0);
  const [pendingCount, setPendingCount] = useState(0);
  const [processingCount, setProcessingCount] = useState(0);
  const [leaderboard, setLeaderboard] = useState<
    { username: string; analysisCount: number }[]
  >([]);
  const [contributors, setContributors] = useState<
    { username: string; analysisCount: number }[]
  >([]);
  const [games, setGames] = useState<AnalyzedGameSummary[]>([]);
  const [totalGames, setTotalGames] = useState(0);
  const [page, setPage] = useState(0);
  const [statsLoading, setStatsLoading] = useState(false);
  const [gamesLoading, setGamesLoading] = useState(false);
  const [selectedGameId, setSelectedGameId] = useState<string | null>(null);

  const adminClient = useClient(AnalysisAdminService);

  const fetchStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const resp = await adminClient.getAdminStats(
        create(GetAdminStatsRequestSchema, {}),
      );
      setTotalCompleted(resp.totalCompleted);
      setPendingCount(resp.pendingCount);
      setProcessingCount(resp.processingCount);
      setLeaderboard(
        resp.leaderboard.map((e) => ({
          username: e.username,
          analysisCount: e.analysisCount,
        })),
      );
      setContributors(
        resp.contributors.map((e) => ({
          username: e.username,
          analysisCount: e.analysisCount,
        })),
      );
    } catch (e) {
      flashError(e);
    } finally {
      setStatsLoading(false);
    }
  }, [adminClient]);

  const fetchGames = useCallback(
    async (p: number) => {
      setGamesLoading(true);
      try {
        const resp = await adminClient.listAnalyzedGames(
          create(ListAnalyzedGamesRequestSchema, {
            page: p,
            pageSize: PAGE_SIZE,
          }),
        );
        setGames(resp.games);
        setTotalGames(resp.total);
      } catch (e) {
        flashError(e);
      } finally {
        setGamesLoading(false);
      }
    },
    [adminClient],
  );

  const handleRequeue = useCallback(
    async (gameId: string) => {
      try {
        const resp = await adminClient.requeueAnalysis(
          create(RequeueAnalysisRequestSchema, { gameId }),
        );
        Modal.success({
          title: "Re-queued",
          content: `Job ${resp.jobId} reset to pending (queue position: ${resp.queuePosition}).`,
        });
        fetchGames(page);
      } catch (e) {
        flashError(e);
      }
    },
    [adminClient, fetchGames, page],
  );

  useEffect(() => {
    fetchStats();
    fetchGames(0);
  }, [fetchStats, fetchGames]);

  const leaderboardColumns = [
    {
      title: "Rank",
      key: "rank",
      width: 60,
      render: (_: unknown, __: unknown, index: number) => index + 1,
    },
    {
      title: "Username",
      dataIndex: "username",
      key: "username",
      render: (username: string) => (
        <Link to={`/profile/${encodeURIComponent(username)}`} target="_blank">
          {username}
        </Link>
      ),
    },
    {
      title: "Analyses Requested",
      dataIndex: "analysisCount",
      key: "analysisCount",
      width: 160,
    },
  ];

  const gamesColumns = [
    {
      title: "Game ID",
      dataIndex: "gameId",
      key: "gameId",
      render: (gameId: string) => (
        <a onClick={() => setSelectedGameId(gameId)}>{gameId}</a>
      ),
    },
    {
      title: "Type",
      dataIndex: "requestType",
      key: "requestType",
      width: 130,
      render: (t: string) => (
        <Tag color={t === "user_requested" ? "blue" : "default"}>
          {t === "user_requested" ? "User requested" : "Automatic"}
        </Tag>
      ),
    },
    {
      title: "Requested by",
      dataIndex: "requestedByUsername",
      key: "requestedByUsername",
      width: 140,
      render: (username: string) =>
        username ? (
          <Link to={`/profile/${encodeURIComponent(username)}`} target="_blank">
            {username}
          </Link>
        ) : (
          <Text type="secondary">—</Text>
        ),
    },
    {
      title: "Queued at",
      dataIndex: "createdAtMs",
      key: "createdAtMs",
      width: 160,
      render: (ms: bigint) =>
        ms
          ? new Date(Number(ms)).toLocaleString("en-US", {
              year: "numeric",
              month: "short",
              day: "numeric",
              hour: "2-digit",
              minute: "2-digit",
            })
          : "—",
    },
    {
      title: "Claimed at",
      dataIndex: "claimedAtMs",
      key: "claimedAtMs",
      width: 160,
      render: (ms: bigint) =>
        ms
          ? new Date(Number(ms)).toLocaleString("en-US", {
              year: "numeric",
              month: "short",
              day: "numeric",
              hour: "2-digit",
              minute: "2-digit",
            })
          : "—",
    },
    {
      title: "Completed at",
      dataIndex: "completedAtMs",
      key: "completedAtMs",
      width: 160,
      render: (ms: bigint) =>
        ms
          ? new Date(Number(ms)).toLocaleString("en-US", {
              year: "numeric",
              month: "short",
              day: "numeric",
              hour: "2-digit",
              minute: "2-digit",
            })
          : "—",
    },
    {
      title: "Action",
      key: "action",
      width: 110,
      render: (_: unknown, row: AnalyzedGameSummary) => (
        <Button size="small" onClick={() => handleRequeue(row.gameId)}>
          Re-analyze
        </Button>
      ),
    },
  ];

  return (
    <Card title="Analysis Dashboard" style={{ margin: "20px" }}>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="Total Analyzed Games"
              value={totalCompleted}
              loading={statsLoading}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Pending in Queue"
              value={pendingCount}
              loading={statsLoading}
              valueStyle={pendingCount > 0 ? { color: "#faad14" } : undefined}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Currently Processing"
              value={processingCount}
              loading={statsLoading}
              valueStyle={
                processingCount > 0 ? { color: "#1677ff" } : undefined
              }
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Total in Queue"
              value={pendingCount + processingCount}
              loading={statsLoading}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={24} style={{ marginBottom: 24 }}>
        <Col span={12}>
          <Card title="Top Analysis Requesters" size="small">
            <Table
              dataSource={leaderboard}
              columns={leaderboardColumns}
              rowKey="username"
              loading={statsLoading}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>
        <Col span={12}>
          <Card title="Top Contributors" size="small">
            <Table
              dataSource={contributors}
              columns={[
                {
                  title: "Rank",
                  key: "rank",
                  width: 60,
                  render: (_: unknown, __: unknown, index: number) => index + 1,
                },
                {
                  title: "Username",
                  dataIndex: "username",
                  key: "username",
                  render: (username: string) => (
                    <Link
                      to={`/profile/${encodeURIComponent(username)}`}
                      target="_blank"
                    >
                      {username}
                    </Link>
                  ),
                },
                {
                  title: "Analyses Run",
                  dataIndex: "analysisCount",
                  key: "analysisCount",
                  width: 130,
                },
              ]}
              rowKey="username"
              loading={statsLoading}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>
      </Row>

      <Row>
        <Col span={24}>
          <Card title="Analyzed Games" size="small">
            <Table
              dataSource={games}
              columns={gamesColumns}
              rowKey="jobId"
              loading={gamesLoading}
              pagination={{
                pageSize: PAGE_SIZE,
                total: totalGames,
                current: page + 1,
                showSizeChanger: false,
                onChange: (p) => {
                  setPage(p - 1);
                  fetchGames(p - 1);
                },
              }}
              size="small"
              scroll={{ x: "max-content" }}
            />
          </Card>
        </Col>
      </Row>

      <GameDetailModal
        gameId={selectedGameId}
        onClose={() => setSelectedGameId(null)}
      />
    </Card>
  );
};
