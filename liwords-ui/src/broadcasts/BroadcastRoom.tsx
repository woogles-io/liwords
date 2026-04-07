import React, { useState } from "react";
import ReactMarkdown from "react-markdown";
import { Spin, Button, Tag, Table, Typography, Space, App, Select } from "antd";
import type { TableColumnsType } from "antd";
import {
  LinkOutlined,
  PlayCircleOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import { useParams, Link, useNavigate } from "react-router";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import {
  getBroadcast,
  getBroadcastGames,
  claimGame,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import type { BroadcastRoundGame } from "../gen/api/proto/broadcast_service/broadcast_service_pb";
import { TopBar } from "../navigation/topbar";
import { useLoginStateStoreContext } from "../store/store";
import { BroadcastDirectorPanel } from "./BroadcastDirectorPanel";
import { BroadcastAnnotatorPanel } from "./BroadcastAnnotatorPanel";
import { flashError } from "../utils/hooks/connect";

const { Title, Text } = Typography;

export const BroadcastRoom: React.FC = () => {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const { loginState } = useLoginStateStoreContext();
  const { message } = App.useApp();
  const [selectedRound, setSelectedRound] = useState<number>(0);
  // Empty string means "let the server pick the first division". Gets set only
  // when the user explicitly changes divisions via the dropdown.
  const [selectedDivision, setSelectedDivision] = useState<string>("");

  const {
    data: broadcastData,
    isLoading: broadcastLoading,
    error: broadcastError,
  } = useQuery(
    getBroadcast,
    { slug: slug ?? "", division: selectedDivision },
    { enabled: !!slug },
  );

  // The effective division for display: use what the server resolved (from response),
  // falling back to what the user selected.
  const activeDivision =
    broadcastData?.divisions?.[0] && !selectedDivision
      ? broadcastData.divisions[0]
      : selectedDivision;

  const activeRound =
    selectedRound ||
    broadcastData?.broadcast?.currentRound ||
    broadcastData?.totalRounds ||
    1;

  const { data: gamesData, isLoading: gamesLoading } = useQuery(
    getBroadcastGames,
    { slug: slug ?? "", round: activeRound, division: activeDivision },
    { enabled: !!slug && activeRound > 0, refetchInterval: 30_000 },
  );

  const claimMutation = useMutation(claimGame, {
    onSuccess: (resp) => {
      if (resp.gameId) {
        navigate(`/editor/${resp.gameId}`);
      }
    },
    onError: (e) => message.error(`Could not claim game: ${e.message}`),
  });

  if (broadcastLoading) {
    return (
      <div className="broadcast-room">
        <TopBar />
        <div style={{ textAlign: "center", marginTop: 80 }}>
          <Spin size="large" />
        </div>
      </div>
    );
  }

  if (broadcastError || !broadcastData?.broadcast) {
    return (
      <div className="broadcast-room">
        <TopBar />
        <div style={{ textAlign: "center", marginTop: 80 }}>
          <Text type="danger">Broadcast not found.</Text>
        </div>
      </div>
    );
  }

  const broadcast = broadcastData.broadcast;
  const totalRounds = broadcastData.totalRounds || 1;
  const isAnnotator = broadcastData.annotatorUsernames.includes(
    loginState.username,
  );
  const isDirector = broadcastData.directorUsernames.includes(
    loginState.username,
  );
  const isAdmin = loginState.perms.includes("adm");

  const roundOptions = Array.from({ length: totalRounds }, (_, i) => ({
    value: i + 1,
    label: `Round ${i + 1}`,
  }));

  const columns: TableColumnsType<BroadcastRoundGame> = [
    {
      title: "Table",
      dataIndex: "tableNumber",
      key: "tableNumber",
      width: 70,
      render: (n: number) => <Text strong>#{n}</Text>,
    },
    {
      title: "Players",
      key: "players",
      render: (_, row) => (
        <span>
          {row.player1Name} <Text type="secondary">vs</Text> {row.player2Name}
        </span>
      ),
    },
    {
      title: "Score",
      key: "score",
      width: 120,
      render: (_, row) => {
        if (row.player1Score === 0 && row.player2Score === 0) {
          return <Text type="secondary">—</Text>;
        }
        return (
          <span>
            {row.player1Score} – {row.player2Score}
            {row.scoresFinalized && (
              <Tag color="green" style={{ marginLeft: 6 }}>
                Final
              </Tag>
            )}
          </span>
        );
      },
    },
    {
      title: "Annotation",
      key: "annotation",
      width: 200,
      render: (_, row) => {
        if (row.gameUuid) {
          return (
            <Space>
              <Tag color="blue" icon={<PlayCircleOutlined />}>
                LIVE
              </Tag>
              <Link to={`/anno/${row.gameUuid}`}>
                <LinkOutlined /> Watch
              </Link>
            </Space>
          );
        }
        if (isAnnotator || isDirector || isAdmin) {
          return (
            <Button
              size="small"
              type="primary"
              loading={
                claimMutation.isPending &&
                claimMutation.variables?.tableNumber === row.tableNumber
              }
              onClick={() =>
                claimMutation.mutate({
                  slug: slug ?? "",
                  round: activeRound,
                  tableNumber: row.tableNumber,
                  division: activeDivision,
                })
              }
            >
              Claim &amp; Annotate
            </Button>
          );
        }
        return <Text type="secondary">Not annotated</Text>;
      },
    },
  ];

  return (
    <div className="broadcast-room">
      <TopBar />
      <div style={{ maxWidth: 900, margin: "0 auto", padding: "24px 16px" }}>
        <Space direction="vertical" size="small" style={{ width: "100%" }}>
          <Space align="center">
            <Title level={2} style={{ marginBottom: 0 }}>
              {broadcast.name}
            </Title>
            {!broadcast.active && <Tag color="default">Archived</Tag>}
            {(isAdmin || broadcast.creatorUsername === loginState.username) && (
              <Button
                size="small"
                onClick={() => navigate(`/broadcasts/${slug}/edit`)}
              >
                Edit
              </Button>
            )}
          </Space>
          {broadcast.description && (
            <div className="broadcast-description">
              <ReactMarkdown>{broadcast.description}</ReactMarkdown>
            </div>
          )}
        </Space>

        <Space style={{ marginTop: 16 }} wrap>
          {(broadcastData.divisions?.length ?? 0) >= 1 && (
            <>
              <Text strong>Division:</Text>
              <Select
                value={activeDivision}
                onChange={(val) => {
                  setSelectedDivision(val);
                  setSelectedRound(0);
                }}
                options={broadcastData.divisions.map((d) => ({
                  value: d,
                  label: `Division ${d}`,
                }))}
                style={{ minWidth: 140 }}
              />
            </>
          )}
          <Text strong>Round:</Text>
          <Select
            value={activeRound}
            onChange={(val) => setSelectedRound(val)}
            options={roundOptions}
            style={{ minWidth: 130 }}
          />
        </Space>

        <Table<BroadcastRoundGame>
          style={{ marginTop: 16 }}
          rowKey={(r) => `${r.round}-${r.tableNumber}`}
          dataSource={gamesData?.games ?? []}
          columns={columns}
          loading={gamesLoading}
          pagination={false}
          size="small"
        />

        {(isAnnotator || isDirector || isAdmin) && (
          <BroadcastAnnotatorPanel slug={slug ?? ""} />
        )}

        {(isDirector || isAdmin) && (
          <BroadcastDirectorPanel
            broadcast={broadcast}
            annotatorUsernames={broadcastData.annotatorUsernames}
            directorUsernames={broadcastData.directorUsernames}
          />
        )}
      </div>
    </div>
  );
};
