import React, { useState } from "react";
// Also rendered standalone in the game editor, so it can't rely on the director
// panel having pulled these layout classes in.
import "./broadcasts.scss";
import { Button, Dropdown, Input, Popconfirm, Tag, Typography } from "antd";
import { DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  listMyStreamSlots,
  createStreamSlot,
  pointStreamSlot,
  deleteStreamSlot,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import {
  StreamSlotTargetKind,
  type BroadcastSlot,
  type StreamSlot,
} from "../gen/api/proto/broadcast_service/broadcast_service_pb";
import { flashError } from "../utils/hooks/connect";
import { OBSPanel } from "./OBSPanel";
import { OBSHelpButton } from "./OBSHelp";

type Props = {
  /**
   * The broadcast whose page this panel is rendered on, when there is one.
   * Omitted in the game editor, where a slot can still be set to follow your
   * own annotations — streaming must not require creating a broadcast.
   */
  broadcastSlug?: string;
  broadcastName?: string;
  /** That broadcast's own slots — the candidates for "mirror a broadcast slot". */
  slots?: BroadcastSlot[];
  username: string;
};

/**
 * Stream slots are the user's own permanent OBS pointers. Where a broadcast
 * slot names a table inside one event, a stream slot names whatever its owner
 * chooses — a broadcast slot, or their own latest annotated game — and its URL
 * carries only the username and slot name, so it outlives the event.
 *
 * Because the target kind lives on the slot rather than in the URL, a slot can
 * move between those two without the OBS browser source changing: stream club
 * nights off your own annotations, then mirror a broadcast slot when you
 * graduate to a real broadcast.
 */
export const StreamSlotsPanel: React.FC<Props> = ({
  broadcastSlug,
  broadcastName,
  slots = [],
  username,
}) => {
  const queryClient = useQueryClient();
  const [newSlotName, setNewSlotName] = useState("");

  const { data: streamSlotsData } = useQuery(
    listMyStreamSlots,
    {},
    { refetchInterval: 10_000 },
  );

  const invalidate = () => {
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "ListMyStreamSlots" }],
    });
    // This broadcast's slots carry a stream_slot_count that just changed.
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "ListSlots" }],
    });
  };

  const createMutation = useMutation(createStreamSlot, {
    onSuccess: () => {
      setNewSlotName("");
      invalidate();
    },
    onError: (e) => flashError(e),
  });

  const pointMutation = useMutation(pointStreamSlot, {
    onSuccess: invalidate,
    onError: (e) => flashError(e),
  });

  const deleteMutation = useMutation(deleteStreamSlot, {
    onSuccess: invalidate,
    onError: (e) => flashError(e),
  });

  const streamSlots = streamSlotsData?.slots ?? [];

  const doCreate = () => {
    const name = newSlotName.trim().toLowerCase();
    if (!name) return;
    createMutation.mutate({ slotName: name });
  };

  const describeTarget = (ss: StreamSlot) => {
    switch (ss.targetKind) {
      case StreamSlotTargetKind.LATEST_ANNOTATION:
        return <Tag color="blue">your latest annotated game</Tag>;
      case StreamSlotTargetKind.BROADCAST_SLOT: {
        const here = ss.targetBroadcastSlug === broadcastSlug;
        return (
          <Tag color={here ? "purple" : undefined}>
            {here ? broadcastName : ss.targetBroadcastName} /{" "}
            {ss.targetSlotName}
          </Tag>
        );
      }
      default:
        return <Tag>not pointed</Tag>;
    }
  };

  // What the target currently resolves to, so you can tell at a glance whether
  // the OBS source is actually showing anything.
  const describeResolved = (ss: StreamSlot) => {
    if (ss.targetKind === StreamSlotTargetKind.NONE) return null;
    if (ss.targetKind === StreamSlotTargetKind.BROADCAST_SLOT && ss.round === 0)
      return <Tag color="red">target missing</Tag>;
    if (ss.gameUuid)
      return (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {ss.player1Name} vs {ss.player2Name}
          {ss.round > 0 ? ` (R${ss.round}, T${ss.tableNumber})` : ""}
        </Typography.Text>
      );
    return (
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {ss.targetKind === StreamSlotTargetKind.LATEST_ANNOTATION
          ? "nothing annotated yet"
          : `R${ss.round}, T${ss.tableNumber} — no game claimed`}
      </Typography.Text>
    );
  };

  const targetMenuItems = (ss: StreamSlot) => [
    {
      key: "__latest",
      label:
        ss.targetKind === StreamSlotTargetKind.LATEST_ANNOTATION
          ? "My latest annotated game (current)"
          : "My latest annotated game",
      onClick: () =>
        pointMutation.mutate({
          slotName: ss.slotName,
          targetKind: StreamSlotTargetKind.LATEST_ANNOTATION,
        }),
    },
    ...(slots.length > 0 && broadcastSlug
      ? [
          {
            type: "divider" as const,
            key: "d1",
          },
          ...slots.map((s) => ({
            key: `bs:${s.slotName}`,
            label:
              ss.targetKind === StreamSlotTargetKind.BROADCAST_SLOT &&
              ss.targetBroadcastSlug === broadcastSlug &&
              ss.targetSlotName === s.slotName
                ? `${broadcastName}: ${s.slotName} (current)`
                : `${broadcastName}: ${s.slotName}`,
            onClick: () =>
              pointMutation.mutate({
                slotName: ss.slotName,
                targetKind: StreamSlotTargetKind.BROADCAST_SLOT,
                targetBroadcastSlug: broadcastSlug,
                targetSlotName: s.slotName,
              }),
          })),
        ]
      : []),
    ...(ss.targetKind !== StreamSlotTargetKind.NONE
      ? [
          { type: "divider" as const, key: "d2" },
          {
            key: "__clear",
            danger: true,
            label: "Clear",
            onClick: () =>
              pointMutation.mutate({
                slotName: ss.slotName,
                targetKind: StreamSlotTargetKind.NONE,
              }),
          },
        ]
      : []),
  ];

  return (
    <div>
      {/* Lead with the account: these URLs embed the username, so someone
          signed in as their personal account rather than their org's would
          otherwise build a whole OBS scene against the wrong one. */}
      <Typography.Text className="panel-hint">
        You are logged in as <strong>{username}</strong>. A stream slot is a
        permanent OBS URL of your own. Point one at your latest annotated game,
        or at a broadcast slot to follow whatever table that event is featuring.
        Because the URL only contains your username and the slot name, changing
        what you cover is a re-point here instead of an edit in OBS.{" "}
        <OBSHelpButton section={broadcastSlug ? "streaming" : "solo"} />
      </Typography.Text>

      {streamSlots.length === 0 && (
        <Typography.Text className="panel-empty">
          No stream slots yet — add one below.
        </Typography.Text>
      )}

      {streamSlots.map((ss) => (
        <div className="slot-row" key={ss.slotName}>
          <div className="slot-row-main">
            <span className="slot-row-name">{ss.slotName}</span>
            {describeTarget(ss)}
            {describeResolved(ss)}
          </div>
          <div className="slot-row-actions">
            <Dropdown trigger={["click"]} menu={{ items: targetMenuItems(ss) }}>
              <Button size="small" loading={pointMutation.isPending}>
                Point at…
              </Button>
            </Dropdown>
            <OBSPanel
              compact
              username={username}
              streamSlotName={ss.slotName}
              streamSlotFollowsSelf={
                ss.targetKind === StreamSlotTargetKind.LATEST_ANNOTATION
              }
            />
            <Popconfirm
              title={`Delete stream slot "${ss.slotName}"?`}
              description="Any OBS browser source using its URL will stop working."
              onConfirm={() => deleteMutation.mutate({ slotName: ss.slotName })}
            >
              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
                loading={deleteMutation.isPending}
              />
            </Popconfirm>
          </div>
        </div>
      ))}

      {broadcastSlug && slots.length === 0 && streamSlots.length > 0 && (
        <Typography.Text className="panel-empty">
          This broadcast has no slots yet — add one in the Broadcast slots tab
          to be able to mirror it.
        </Typography.Text>
      )}

      {/* Add stream slot form */}
      <div className="add-form">
        <div className="add-form-field">
          <span className="add-form-label">Stream slot name</span>
          <Input
            size="small"
            placeholder="e.g. stream1"
            value={newSlotName}
            onChange={(e) => setNewSlotName(e.target.value)}
            onPressEnter={doCreate}
            style={{ width: 140 }}
          />
        </div>
        <Button
          size="small"
          icon={<PlusOutlined />}
          loading={createMutation.isPending}
          onClick={doCreate}
          disabled={!newSlotName.trim()}
        >
          Add stream slot
        </Button>
        <Typography.Text
          className="add-form-label"
          style={{ paddingBottom: 6 }}
        >
          Lowercase letters, numbers and dashes.
        </Typography.Text>
      </div>
    </div>
  );
};
