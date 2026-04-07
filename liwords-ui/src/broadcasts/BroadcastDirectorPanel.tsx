import React, { useState } from "react";
import {
  Card,
  Button,
  Input,
  Space,
  Tag,
  Divider,
  Switch,
  App,
  Popconfirm,
} from "antd";
import {
  UserAddOutlined,
  UserDeleteOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  addBroadcastAnnotators,
  removeBroadcastAnnotators,
  addBroadcastDirectors,
  removeBroadcastDirectors,
  updateBroadcast,
  triggerPoll,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import type { Broadcast } from "../gen/api/proto/broadcast_service/broadcast_service_pb";
import { flashError } from "../utils/hooks/connect";

type Props = {
  broadcast: Broadcast;
  annotatorUsernames: string[];
  directorUsernames: string[];
};

export const BroadcastDirectorPanel: React.FC<Props> = ({
  broadcast,
  annotatorUsernames,
  directorUsernames,
}) => {
  const { notification } = App.useApp();
  const queryClient = useQueryClient();
  const [newAnnotator, setNewAnnotator] = useState("");
  const [newDirector, setNewDirector] = useState("");

  const invalidate = () =>
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "GetBroadcast" }],
    });

  const addAnnotatorMutation = useMutation(addBroadcastAnnotators, {
    onSuccess: () => { setNewAnnotator(""); invalidate(); },
    onError: (e) => flashError(e),
  });

  const removeAnnotatorMutation = useMutation(removeBroadcastAnnotators, {
    onError: (e) => flashError(e),
    onSuccess: invalidate,
  });

  const addDirectorMutation = useMutation(addBroadcastDirectors, {
    onSuccess: () => { setNewDirector(""); invalidate(); },
    onError: (e) => flashError(e),
  });

  const removeDirectorMutation = useMutation(removeBroadcastDirectors, {
    onError: (e) => flashError(e),
    onSuccess: invalidate,
  });

  const updateMutation = useMutation(updateBroadcast, {
    onSuccess: invalidate,
    onError: (e) => flashError(e),
  });

  const pollMutation = useMutation(triggerPoll, {
    onSuccess: () => notification.success({ message: "Poll triggered" }),
    onError: (e) => flashError(e),
  });

  const toggleActive = (active: boolean) => {
    updateMutation.mutate({
      slug: broadcast.slug,
      name: broadcast.name,
      description: broadcast.description,
      broadcastUrl: broadcast.broadcastUrl,
      broadcastUrlFormat: broadcast.broadcastUrlFormat,
      pollIntervalSeconds: broadcast.pollIntervalSeconds,
      pollStartTime: broadcast.pollStartTime,
      pollEndTime: broadcast.pollEndTime,
      lexicon: broadcast.lexicon,
      boardLayout: broadcast.boardLayout,
      letterDistribution: broadcast.letterDistribution,
      challengeRule: broadcast.challengeRule,
      active,
    });
  };

  return (
    <Card
      title="Director Panel"
      size="small"
      style={{ marginTop: 24 }}
      styles={{ header: { paddingBlock: 10 } }}
      extra={
        <Space>
          <Switch
            checked={broadcast.active}
            onChange={toggleActive}
            checkedChildren="Active"
            unCheckedChildren="Inactive"
            loading={updateMutation.isPending}
          />
          <Button
            icon={<ReloadOutlined />}
            size="small"
            loading={pollMutation.isPending}
            onClick={() => pollMutation.mutate({ slug: broadcast.slug })}
          >
            Force poll
          </Button>
        </Space>
      }
    >
      <Divider orientation="left" plain>Annotators</Divider>
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

      <Divider orientation="left" plain>Directors</Divider>
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
            <Tag
              key={u}
              closable
              onClose={(e) => e.preventDefault()}
            >
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
