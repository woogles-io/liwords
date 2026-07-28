import React, { useState } from "react";
import {
  Button,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Tag,
  Typography,
} from "antd";
import { DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  createSlot,
  deleteSlot,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import type { BroadcastSlot } from "../gen/api/proto/broadcast_service/broadcast_service_pb";
import { flashError } from "../utils/hooks/connect";
import { OBSPanel } from "./OBSPanel";
import { OBSHelpButton } from "./OBSHelp";

type Props = {
  slug: string;
  slots: BroadcastSlot[];
  divisions: string[];
  activeDivision: string;
  activeRound: number;
};

/**
 * The broadcast's own slots: named pointers at a (division, round, table) that
 * the event's directors move as rounds progress. Reassignment happens in the
 * pairings table above, so this section is only create, delete and grab-the-URL.
 */
export const BroadcastSlotsSection: React.FC<Props> = ({
  slug,
  slots,
  divisions,
  activeDivision,
  activeRound,
}) => {
  const queryClient = useQueryClient();
  const [showAddForm, setShowAddForm] = useState(false);
  const [newSlotName, setNewSlotName] = useState("");
  const [newSlotDivision, setNewSlotDivision] = useState(activeDivision);
  const [newSlotRound, setNewSlotRound] = useState(activeRound);
  const [newSlotTable, setNewSlotTable] = useState(1);

  const invalidateSlots = () =>
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "ListSlots" }],
    });

  const createSlotMutation = useMutation(createSlot, {
    onSuccess: () => {
      setNewSlotName("");
      setShowAddForm(false);
      invalidateSlots();
    },
    onError: (e) => flashError(e),
  });

  const deleteSlotMutation = useMutation(deleteSlot, {
    onSuccess: invalidateSlots,
    onError: (e) => flashError(e),
  });

  // Next free table for the current div/round, as a convenience default.
  const nextTableForCurrentTarget = () => {
    const matching = slots
      .filter((s) => s.division === newSlotDivision && s.round === newSlotRound)
      .map((s) => s.tableNumber);
    return matching.length > 0 ? Math.max(...matching) + 1 : 1;
  };

  const doCreateSlot = () => {
    if (!newSlotName.trim()) return;
    createSlotMutation.mutate({
      slug,
      slotName: newSlotName.trim(),
      division: newSlotDivision,
      round: newSlotRound,
      tableNumber: newSlotTable,
    });
  };

  const deleteWarning = (slot: BroadcastSlot) =>
    slot.streamSlotCount > 0
      ? `${slot.streamSlotCount} stream slot${
          slot.streamSlotCount === 1 ? "" : "s"
        } currently mirror${
          slot.streamSlotCount === 1 ? "s" : ""
        } this — they will go blank.`
      : undefined;

  // Once there is at least one slot, adding is rare — keep the four-field form
  // out of the way until it's asked for.
  const addFormOpen = showAddForm || slots.length === 0;

  return (
    <div>
      <Typography.Text className="panel-hint">
        Slots point at a division, round and table. Reassign them from the games
        table above when a new round starts.{" "}
        <OBSHelpButton section="director" />
      </Typography.Text>

      {slots.length === 0 && (
        <Typography.Text className="panel-empty">
          No slots yet — add one below.
        </Typography.Text>
      )}

      {slots.map((slot) => (
        <div className="slot-row" key={slot.slotName}>
          <div className="slot-row-main">
            <span className="slot-row-name">{slot.slotName}</span>
            <span className="slot-row-detail">
              {slot.division ? `Div ${slot.division} · ` : ""}R{slot.round} · T
              {slot.tableNumber}
            </span>
            {slot.streamSlotCount > 0 && (
              <Tag color="purple">streamed by {slot.streamSlotCount}</Tag>
            )}
          </div>
          <div className="slot-row-actions">
            <OBSPanel compact broadcastSlug={slug} slotName={slot.slotName} />
            <Popconfirm
              title={`Delete slot "${slot.slotName}"?`}
              description={deleteWarning(slot)}
              onConfirm={() =>
                deleteSlotMutation.mutate({ slug, slotName: slot.slotName })
              }
            >
              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
                loading={deleteSlotMutation.isPending}
              />
            </Popconfirm>
          </div>
        </div>
      ))}

      {!addFormOpen && (
        <div className="add-form">
          <Button
            size="small"
            icon={<PlusOutlined />}
            onClick={() => {
              setNewSlotTable(nextTableForCurrentTarget());
              setShowAddForm(true);
            }}
          >
            Add slot
          </Button>
        </div>
      )}

      {addFormOpen && (
        <div className="add-form">
          <div className="add-form-field">
            <span className="add-form-label">Slot name</span>
            <Input
              size="small"
              placeholder="e.g. featured"
              value={newSlotName}
              onChange={(e) => setNewSlotName(e.target.value)}
              style={{ width: 130 }}
            />
          </div>
          <div className="add-form-field">
            <span className="add-form-label">Division</span>
            {divisions.length > 1 ? (
              <Select
                size="small"
                value={newSlotDivision}
                onChange={setNewSlotDivision}
                options={divisions.map((d) => ({
                  value: d,
                  label: `Div ${d}`,
                }))}
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
          <div className="add-form-field">
            <span className="add-form-label">Round</span>
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
          <div className="add-form-field">
            <span className="add-form-label">Table</span>
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
            type="primary"
            icon={<PlusOutlined />}
            loading={createSlotMutation.isPending}
            onClick={doCreateSlot}
            disabled={!newSlotName.trim() || !newSlotDivision}
          >
            Add slot
          </Button>
          {slots.length > 0 && (
            <Button size="small" onClick={() => setShowAddForm(false)}>
              Cancel
            </Button>
          )}
        </div>
      )}
    </div>
  );
};
