import React, { useState } from "react";
import "./broadcasts.scss";
import { Card, Button, Grid, Select, Space, Tabs, App } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  addBroadcastAnnotators,
  removeBroadcastAnnotators,
  addBroadcastDirectors,
  removeBroadcastDirectors,
  triggerPoll,
  listSlots,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import type { Broadcast } from "../gen/api/proto/broadcast_service/broadcast_service_pb";
import { flashError } from "../utils/hooks/connect";
import { StreamSlotsPanel } from "./StreamSlotsPanel";
import { BroadcastSlotsSection } from "./BroadcastSlotsSection";
import { UserListSection } from "./UserListSection";
import { useLoginStateStoreContext } from "../store/store";

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
  const { loginState } = useLoginStateStoreContext();
  const screens = Grid.useBreakpoint();
  const [activeTab, setActiveTab] = useState("slots");

  const invalidateBroadcast = () =>
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "GetBroadcast" }],
    });

  const { data: slotsData } = useQuery(
    listSlots,
    { slug: broadcast.slug },
    { refetchInterval: 10_000 },
  );
  const slots = slotsData?.slots ?? [];

  const addAnnotatorMutation = useMutation(addBroadcastAnnotators, {
    onSuccess: invalidateBroadcast,
    onError: (e) => flashError(e),
  });

  const removeAnnotatorMutation = useMutation(removeBroadcastAnnotators, {
    onSuccess: invalidateBroadcast,
    onError: (e) => flashError(e),
  });

  const addDirectorMutation = useMutation(addBroadcastDirectors, {
    onSuccess: invalidateBroadcast,
    onError: (e) => flashError(e),
  });

  const removeDirectorMutation = useMutation(removeBroadcastDirectors, {
    onSuccess: invalidateBroadcast,
    onError: (e) => flashError(e),
  });

  const pollMutation = useMutation(triggerPoll, {
    onSuccess: () => notification.success({ message: "Poll triggered" }),
    onError: (e) => flashError(e),
  });

  const peopleCount = annotatorUsernames.length + directorUsernames.length;

  const tabItems = [
    {
      key: "slots",
      label: `Broadcast slots (${slots.length})`,
      children: (
        <BroadcastSlotsSection
          slug={broadcast.slug}
          slots={slots}
          divisions={divisions}
          activeDivision={activeDivision}
          activeRound={activeRound}
        />
      ),
    },
    // Stream slots belong to the viewer, not the broadcast, so this tab is only
    // meaningful for a signed-in user.
    ...(loginState.loggedIn
      ? [
          {
            key: "stream",
            label: "My stream",
            children: (
              <StreamSlotsPanel
                broadcastSlug={broadcast.slug}
                broadcastName={broadcast.name}
                slots={slots}
                username={loginState.username}
              />
            ),
          },
        ]
      : []),
    {
      key: "people",
      label: `People (${peopleCount})`,
      children: (
        <div className="panel-people">
          <UserListSection
            title="Annotators"
            hint="Annotators can claim tables and enter moves for this broadcast."
            usernames={annotatorUsernames}
            adding={addAnnotatorMutation.isPending}
            emptyText="No annotators yet."
            onAdd={(username) =>
              addAnnotatorMutation.mutate({
                slug: broadcast.slug,
                usernames: [username],
              })
            }
            onRemove={(username) =>
              removeAnnotatorMutation.mutate({
                slug: broadcast.slug,
                usernames: [username],
              })
            }
          />
          <UserListSection
            title="Directors"
            hint="Directors manage slots, annotators and this panel."
            usernames={directorUsernames}
            adding={addDirectorMutation.isPending}
            emptyText="No directors yet."
            confirmRemove={(u) => `Remove ${u} as director?`}
            onAdd={(username) =>
              addDirectorMutation.mutate({
                slug: broadcast.slug,
                usernames: [username],
              })
            }
            onRemove={(username) =>
              removeDirectorMutation.mutate({
                slug: broadcast.slug,
                usernames: [username],
              })
            }
          />
        </div>
      ),
    },
  ];

  return (
    <Card
      title="Director Panel"
      size="small"
      className="director-panel"
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
      {/* The global tabs mixin hides antd's overflow arrows, so tabs that don't
          fit would be unreachable. Below sm, swap the bar for a select — the
          same fallback BroadcastRoom uses for its own tabs. */}
      {screens.sm ? (
        <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} />
      ) : (
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <Select
            value={activeTab}
            onChange={setActiveTab}
            style={{ width: "100%", marginTop: 12 }}
            options={tabItems.map((t) => ({ value: t.key, label: t.label }))}
          />
          {tabItems.find((t) => t.key === activeTab)?.children}
        </Space>
      )}
    </Card>
  );
};
