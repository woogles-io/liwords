import React, { useState } from "react";
import ReactMarkdown from "react-markdown";
import {
  Spin,
  Button,
  Tag,
  Table,
  Typography,
  Space,
  App,
  Select,
  Popover,
} from "antd";
import type { TableColumnsType } from "antd";
import { LinkOutlined, PlayCircleOutlined } from "@ant-design/icons";
import { useParams, Link, useNavigate } from "react-router";
import {
  useQuery,
  useMutation,
  useTransport,
  callUnaryMethod,
} from "@connectrpc/connect-query";
import {
  getBroadcast,
  getBroadcastGames,
  claimGame,
  listSlots,
  assignSlot,
  getSlotCurrentGame,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import type {
  BroadcastRoundGame,
  BroadcastSlot,
} from "../gen/api/proto/broadcast_service/broadcast_service_pb";
import { TopBar } from "../navigation/topbar";
import { useLoginStateStoreContext } from "../store/store";
import { BroadcastDirectorPanel } from "./BroadcastDirectorPanel";
import { BroadcastAnnotatorPanel } from "./BroadcastAnnotatorPanel";
import { flashError } from "../utils/hooks/connect";
import { useQueryClient } from "@tanstack/react-query";

const { Title, Text } = Typography;

export const BroadcastRoom: React.FC = () => {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const { loginState } = useLoginStateStoreContext();
  const { message, modal } = App.useApp();
  const queryClient = useQueryClient();
  const transport = useTransport();
  const [selectedRound, setSelectedRound] = useState<number>(0);
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

  const { data: slotsData } = useQuery(
    listSlots,
    { slug: slug ?? "" },
    { enabled: !!slug, refetchInterval: 10_000 },
  );

  const claimMutation = useMutation(claimGame, {
    onSuccess: (resp) => {
      if (resp.gameId) {
        navigate(`/editor/${resp.gameId}`);
      }
    },
    onError: (e) => message.error(`Could not claim game: ${e.message}`),
  });

  const assignSlotMutation = useMutation(assignSlot, {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["connect-query", { methodName: "ListSlots" }],
      });
    },
    onError: (e) => flashError(e),
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

  const slots: BroadcastSlot[] = slotsData?.slots ?? [];

  // Assign a slot, checking for an in-progress game at the slot's current target first.
  const doAssign = async (
    slot: BroadcastSlot,
    newDivision: string,
    newRound: number,
    newTableNumber: number,
  ) => {
    const mutate = () =>
      assignSlotMutation.mutate({
        slug: slug ?? "",
        slotName: slot.slotName,
        division: newDivision,
        round: newRound,
        tableNumber: newTableNumber,
      });

    try {
      const current = await callUnaryMethod(transport, getSlotCurrentGame, {
        slug: slug ?? "",
        slotName: slot.slotName,
      });

      if (current.gameUuid && !current.annotationDone) {
        const currentDesc =
          `round ${current.round}, table ${current.tableNumber}` +
          (current.division ? ` (div ${current.division})` : "");
        modal.confirm({
          title: `Reassign slot "${slot.slotName}"?`,
          content: (
            <span>
              This slot is currently assigned to the{" "}
              <strong>
                {current.player1Name} vs {current.player2Name}
              </strong>{" "}
              game ({currentDesc}). That game might be currently streaming live.
              Are you sure you want to reassign it?
            </span>
          ),
          okText: "Yes, reassign",
          okButtonProps: { danger: true },
          cancelText: "Cancel",
          onOk: mutate,
        });
        return;
      }
    } catch {
      // Not a director or slot not found — fall through and let the server reject if needed.
    }

    mutate();
  };

  // Build a map from tableNumber → slots that currently point to (activeDivision, activeRound, tableNumber).
  const slotsByTable = new Map<number, BroadcastSlot[]>();
  for (const slot of slots) {
    if (slot.round !== activeRound) continue;
    if (slot.division !== "" && slot.division !== activeDivision) continue;
    const existing = slotsByTable.get(slot.tableNumber) ?? [];
    existing.push(slot);
    slotsByTable.set(slot.tableNumber, existing);
  }

  const showSlotColumn = slots.length > 0;

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
    ...(showSlotColumn
      ? [
          {
            title: "Slot",
            key: "slot",
            width: 130,
            render: (_: unknown, row: BroadcastRoundGame) => {
              const rowSlots = slotsByTable.get(row.tableNumber) ?? [];
              if (!isDirector && !isAdmin) {
                return rowSlots.length > 0 ? (
                  <Space size={4} wrap>
                    {rowSlots.map((s) => (
                      <Tag key={s.slotName} color="purple">
                        {s.slotName}
                      </Tag>
                    ))}
                  </Space>
                ) : null;
              }
              // Director view: show current tags + popover to move other slots here.
              const assignedSlotNames = new Set(
                rowSlots.map((s) => s.slotName),
              );
              const movableSlots = slots.filter(
                (s) => !assignedSlotNames.has(s.slotName),
              );
              return (
                <Space size={4} wrap>
                  {rowSlots.map((s) => (
                    <Tag key={s.slotName} color="purple">
                      {s.slotName}
                    </Tag>
                  ))}
                  {movableSlots.length > 0 && (
                    <Popover
                      trigger="click"
                      content={
                        <Space direction="vertical" size={4}>
                          <Text type="secondary" style={{ fontSize: 12 }}>
                            Move slot here:
                          </Text>
                          {movableSlots.map((s) => (
                            <Button
                              key={s.slotName}
                              size="small"
                              onClick={() =>
                                doAssign(
                                  s,
                                  activeDivision,
                                  activeRound,
                                  row.tableNumber,
                                )
                              }
                            >
                              {s.slotName}
                            </Button>
                          ))}
                        </Space>
                      }
                    >
                      <Button size="small" type="dashed">
                        + Assign
                      </Button>
                    </Popover>
                  )}
                </Space>
              );
            },
          } as TableColumnsType<BroadcastRoundGame>[number],
        ]
      : []),
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
              {!row.scoresFinalized && (
                <Tag color="blue" icon={<PlayCircleOutlined />}>
                  LIVE
                </Tag>
              )}
              <Link to={`/anno/${row.gameUuid}`}>
                <LinkOutlined /> {row.scoresFinalized ? "Review" : "Watch"}
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
            divisions={broadcastData.divisions ?? []}
            activeDivision={activeDivision}
            activeRound={activeRound}
          />
        )}
      </div>
    </div>
  );
};
