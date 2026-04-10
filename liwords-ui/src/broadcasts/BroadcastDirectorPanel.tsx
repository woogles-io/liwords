import React, { useState } from "react";
import {
  Card,
  Button,
  Input,
  InputNumber,
  Space,
  Tag,
  Divider,
  App,
  Popconfirm,
  Typography,
  Select,
} from "antd";
import {
  UserAddOutlined,
  ReloadOutlined,
  DeleteOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  addBroadcastAnnotators,
  removeBroadcastAnnotators,
  addBroadcastDirectors,
  removeBroadcastDirectors,
  triggerPoll,
  listSlots,
  createSlot,
  deleteSlot,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import type { Broadcast } from "../gen/api/proto/broadcast_service/broadcast_service_pb";
import { flashError } from "../utils/hooks/connect";
import { OBSPanel } from "./OBSPanel";

type Props = {
  broadcast: Broadcast;
  annotatorUsernames: string[];
  directorUsernames: string[];
  divisions?: string[];
  activeDivision?: string;
  activeRound?: number;
};

export const BroadcastDirectorPanel: React.FC<Props> = ({
  broadcast,
  annotatorUsernames,
  directorUsernames,
  divisions = [],
  activeDivision = "",
  activeRound = 1,
}) => {
  const { notification } = App.useApp();
  const queryClient = useQueryClient();
  const [newAnnotator, setNewAnnotator] = useState("");
  const [newDirector, setNewDirector] = useState("");
  const [newSlotName, setNewSlotName] = useState("");
  const [newSlotDivision, setNewSlotDivision] = useState(activeDivision);
  const [newSlotRound, setNewSlotRound] = useState(activeRound);
  const [newSlotTable, setNewSlotTable] = useState(1);

  const invalidateBroadcast = () =>
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "GetBroadcast" }],
    });

  const invalidateSlots = () =>
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "ListSlots" }],
    });

  // ----- Annotator / director mutations -----

  const addAnnotatorMutation = useMutation(addBroadcastAnnotators, {
    onSuccess: () => {
      setNewAnnotator("");
      invalidateBroadcast();
    },
    onError: (e) => flashError(e),
  });

  const removeAnnotatorMutation = useMutation(removeBroadcastAnnotators, {
    onError: (e) => flashError(e),
    onSuccess: invalidateBroadcast,
  });

  const addDirectorMutation = useMutation(addBroadcastDirectors, {
    onSuccess: () => {
      setNewDirector("");
      invalidateBroadcast();
    },
    onError: (e) => flashError(e),
  });

  const removeDirectorMutation = useMutation(removeBroadcastDirectors, {
    onError: (e) => flashError(e),
    onSuccess: invalidateBroadcast,
  });

  const pollMutation = useMutation(triggerPoll, {
    onSuccess: () => notification.success({ message: "Poll triggered" }),
    onError: (e) => flashError(e),
  });

  // ----- Slot mutations -----

  const { data: slotsData } = useQuery(listSlots, { slug: broadcast.slug });

  const createSlotMutation = useMutation(createSlot, {
    onSuccess: () => {
      setNewSlotName("");
      invalidateSlots();
    },
    onError: (e) => flashError(e),
  });

  const deleteSlotMutation = useMutation(deleteSlot, {
    onSuccess: invalidateSlots,
    onError: (e) => flashError(e),
  });

  const doCreateSlot = () => {
    if (!newSlotName.trim()) return;
    createSlotMutation.mutate({
      slug: broadcast.slug,
      slotName: newSlotName.trim(),
      division: newSlotDivision,
      round: newSlotRound,
      tableNumber: newSlotTable,
    });
  };

  const slots = slotsData?.slots ?? [];

  // Compute next free table for the current div/round as a convenience default.
  const nextTableForCurrentTarget = () => {
    const matching = slots
      .filter(
        (s) => s.division === newSlotDivision && s.round === newSlotRound,
      )
      .map((s) => s.tableNumber);
    return matching.length > 0 ? Math.max(...matching) + 1 : 1;
  };

  return (
    <Card
      title="Director Panel"
      size="small"
      style={{ marginTop: 24 }}
      styles={{ header: { paddingBlock: 10 } }}
      extra={
        <Button
          icon={<ReloadOutlined />}
          size="small"
          loading={pollMutation.isPending}
          onClick={() => pollMutation.mutate({ slug: broadcast.slug })}
        >
          Force poll
        </Button>
      }
    >
      {/* OBS Slots */}
      <Divider orientation="left" plain style={{ marginTop: 24 }}>
        OBS Slots
      </Divider>
      <Typography.Text
        type="secondary"
        style={{ fontSize: 12, display: "block", marginBottom: 8 }}
      >
        Slots point at a (division, round, table). Reassign them from the games
        table above when a new round starts.
      </Typography.Text>
      <Space direction="vertical" style={{ width: "100%" }}>
        {slots.length === 0 && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            No slots yet — add one below.
          </Typography.Text>
        )}
        {slots.map((slot) => {
          const target = slot.division
            ? `Div ${slot.division}, R${slot.round}, T${slot.tableNumber}`
            : `R${slot.round}, T${slot.tableNumber}`;
          return (
            <Card
              key={slot.slotName}
              size="small"
              style={{ background: "rgba(255,255,255,0.04)" }}
            >
              <Space style={{ width: "100%", justifyContent: "space-between" }}>
                <Space>
                  <strong>{slot.slotName}</strong>
                  <Tag>{target}</Tag>
                </Space>
                <Space size="small">
                  <OBSPanel
                    compact
                    broadcastSlug={broadcast.slug}
                    slotName={slot.slotName}
                  />
                  <Popconfirm
                    title={`Delete slot "${slot.slotName}"?`}
                    onConfirm={() =>
                      deleteSlotMutation.mutate({
                        slug: broadcast.slug,
                        slotName: slot.slotName,
                      })
                    }
                  >
                    <Button
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      loading={deleteSlotMutation.isPending}
                    />
                  </Popconfirm>
                </Space>
              </Space>
            </Card>
          );
        })}

        {/* Add slot form */}
        <Space wrap size="small" align="end">
          <div>
            <Typography.Text style={{ fontSize: 11, display: "block", marginBottom: 2 }}>Slot name</Typography.Text>
            <Input
              size="small"
              placeholder="e.g. main"
              value={newSlotName}
              onChange={(e) => setNewSlotName(e.target.value)}
              style={{ width: 120 }}
            />
          </div>
          <div>
            <Typography.Text style={{ fontSize: 11, display: "block", marginBottom: 2 }}>Division</Typography.Text>
            {divisions.length > 1 ? (
              <Select
                size="small"
                value={newSlotDivision}
                onChange={setNewSlotDivision}
                options={divisions.map((d) => ({ value: d, label: `Div ${d}` }))}
                style={{ minWidth: 90 }}
              />
            ) : (
              <Input
                size="small"
                placeholder="e.g. A"
                value={newSlotDivision}
                onChange={(e) => setNewSlotDivision(e.target.value)}
                style={{ width: 90 }}
              />
            )}
          </div>
          <div>
            <Typography.Text style={{ fontSize: 11, display: "block", marginBottom: 2 }}>Round</Typography.Text>
            <InputNumber
              size="small"
              value={newSlotRound}
              min={1}
              onChange={(v) => {
                setNewSlotRound(v ?? 1);
                setNewSlotTable(nextTableForCurrentTarget());
              }}
              style={{ width: 80 }}
            />
          </div>
          <div>
            <Typography.Text style={{ fontSize: 11, display: "block", marginBottom: 2 }}>Table</Typography.Text>
            <InputNumber
              size="small"
              value={newSlotTable}
              min={1}
              onChange={(v) => setNewSlotTable(v ?? 1)}
              style={{ width: 80 }}
            />
          </div>
          <Button
            size="small"
            icon={<PlusOutlined />}
            loading={createSlotMutation.isPending}
            onClick={doCreateSlot}
            disabled={!newSlotName.trim() || !newSlotDivision}
          >
            Add slot
          </Button>
        </Space>
      </Space>

      {/* Annotators */}
      <Divider orientation="left" plain style={{ marginTop: 24 }}>
        Annotators
      </Divider>
      <Space wrap style={{ marginBottom: 8 }}>
        {annotatorUsernames.map((u) => (
          <Tag
            key={u}
            closable
            onClose={() =>
              removeAnnotatorMutation.mutate({
                slug: broadcast.slug,
                usernames: [u],
              })
            }
          >
            {u}
          </Tag>
        ))}
      </Space>
      <Space.Compact style={{ width: "100%" }}>
        <Input
          placeholder="Username"
          value={newAnnotator}
          onChange={(e) => setNewAnnotator(e.target.value)}
          onPressEnter={() => {
            if (newAnnotator.trim()) {
              addAnnotatorMutation.mutate({
                slug: broadcast.slug,
                usernames: [newAnnotator.trim()],
              });
            }
          }}
        />
        <Button
          icon={<UserAddOutlined />}
          loading={addAnnotatorMutation.isPending}
          onClick={() => {
            if (newAnnotator.trim()) {
              addAnnotatorMutation.mutate({
                slug: broadcast.slug,
                usernames: [newAnnotator.trim()],
              });
            }
          }}
        >
          Add
        </Button>
      </Space.Compact>

      {/* Directors */}
      <Divider orientation="left" plain style={{ marginTop: 24 }}>
        Directors
      </Divider>
      <Space wrap style={{ marginBottom: 8 }}>
        {directorUsernames.map((u) => (
          <Popconfirm
            key={u}
            title={`Remove ${u} as director?`}
            onConfirm={() =>
              removeDirectorMutation.mutate({
                slug: broadcast.slug,
                usernames: [u],
              })
            }
          >
            <Tag key={u} closable onClose={(e) => e.preventDefault()}>
              {u}
            </Tag>
          </Popconfirm>
        ))}
      </Space>
      <Space.Compact style={{ width: "100%" }}>
        <Input
          placeholder="Username"
          value={newDirector}
          onChange={(e) => setNewDirector(e.target.value)}
          onPressEnter={() => {
            if (newDirector.trim()) {
              addDirectorMutation.mutate({
                slug: broadcast.slug,
                usernames: [newDirector.trim()],
              });
            }
          }}
        />
        <Button
          icon={<UserAddOutlined />}
          loading={addDirectorMutation.isPending}
          onClick={() => {
            if (newDirector.trim()) {
              addDirectorMutation.mutate({
                slug: broadcast.slug,
                usernames: [newDirector.trim()],
              });
            }
          }}
        >
          Add
        </Button>
      </Space.Compact>
    </Card>
  );
};
