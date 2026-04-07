import React, { useEffect } from "react";
import {
  Form,
  Input,
  InputNumber,
  Button,
  Card,
  Space,
  Select,
  App,
  Spin,
} from "antd";
import { DatePicker } from "antd";
import { useNavigate, useParams } from "react-router";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { TopBar } from "../navigation/topbar";
import { useLoginStateStoreContext } from "../store/store";
import { LexiconFormItem } from "../shared/lexicon_display";
import { ChallengeRulesFormItem } from "../lobby/challenge_rules_form_item";
import {
  getBroadcast,
  updateBroadcast,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import {
  dayjsToProtobufTimestampIgnoringNanos,
  protobufTimestampToDayjsIgnoringNanos,
} from "../utils/datetime";
import { defaultLetterDistribution } from "../lobby/sought_game_interactions";
import { flashError } from "../utils/hooks/connect";

const { TextArea } = Input;

export const EditBroadcast: React.FC = () => {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const { loginState } = useLoginStateStoreContext();
  const { notification } = App.useApp();
  const [form] = Form.useForm();

  const { data, isLoading } = useQuery(
    getBroadcast,
    { slug: slug ?? "", division: "" },
    { enabled: !!slug },
  );

  const broadcast = data?.broadcast;

  const canEdit =
    loginState.loggedIn &&
    (loginState.perms.includes("adm") ||
      broadcast?.creatorUsername === loginState.username);

  useEffect(() => {
    if (!broadcast) return;
    form.setFieldsValue({
      name: broadcast.name,
      description: broadcast.description,
      broadcastUrl: broadcast.broadcastUrl,
      broadcastUrlFormat: broadcast.broadcastUrlFormat || "tsh_newt_json",
      pollIntervalSeconds: broadcast.pollIntervalSeconds,
      pollStartTime: broadcast.pollStartTime
        ? protobufTimestampToDayjsIgnoringNanos(broadcast.pollStartTime)
        : null,
      pollEndTime: broadcast.pollEndTime
        ? protobufTimestampToDayjsIgnoringNanos(broadcast.pollEndTime)
        : null,
      lexicon: broadcast.lexicon,
      challengerule: broadcast.challengeRule,
    });
  }, [broadcast, form]);

  const updateMutation = useMutation(updateBroadcast, {
    onSuccess: () => {
      notification.success({ message: "Broadcast updated" });
      navigate(`/broadcasts/${slug}`);
    },
    onError: (e) => flashError(e),
  });

  if (isLoading) {
    return (
      <div>
        <TopBar />
        <div style={{ textAlign: "center", marginTop: 80 }}>
          <Spin size="large" />
        </div>
      </div>
    );
  }

  if (!broadcast || !canEdit) {
    return (
      <div>
        <TopBar />
        <p style={{ padding: 24 }}>Not authorized.</p>
      </div>
    );
  }

  const onFinish = (vals: Record<string, unknown>) => {
    const lexicon = (vals.lexicon as string) ?? broadcast.lexicon;
    updateMutation.mutate({
      slug: slug ?? "",
      name: vals.name as string,
      description: (vals.description as string) ?? "",
      broadcastUrl: vals.broadcastUrl as string,
      broadcastUrlFormat:
        (vals.broadcastUrlFormat as string) ?? "tsh_newt_json",
      pollIntervalSeconds: (vals.pollIntervalSeconds as number) ?? 300,
      pollStartTime: vals.pollStartTime
        ? dayjsToProtobufTimestampIgnoringNanos(
            vals.pollStartTime as Parameters<
              typeof dayjsToProtobufTimestampIgnoringNanos
            >[0],
          )
        : undefined,
      pollEndTime: vals.pollEndTime
        ? dayjsToProtobufTimestampIgnoringNanos(
            vals.pollEndTime as Parameters<
              typeof dayjsToProtobufTimestampIgnoringNanos
            >[0],
          )
        : undefined,
      lexicon,
      boardLayout: broadcast.boardLayout || "CrosswordGame",
      letterDistribution: defaultLetterDistribution(lexicon),
      challengeRule: (vals.challengerule as number) ?? 0,
      active: broadcast.active,
    });
  };

  return (
    <div>
      <TopBar />
      <div style={{ maxWidth: 640, margin: "32px auto", padding: "0 16px" }}>
        <Card title={`Edit Broadcast: ${broadcast.name}`}>
          <Form form={form} layout="vertical" onFinish={onFinish}>
            <Form.Item label="Slug">
              <Input value={slug} disabled />
            </Form.Item>
            <Form.Item label="Name" name="name" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item label="Description" name="description">
              <TextArea rows={3} />
            </Form.Item>
            <Form.Item
              label="Feed URL"
              name="broadcastUrl"
              rules={[{ required: true }]}
              extra="URL of the TSH tourney.js file"
            >
              <Input />
            </Form.Item>
            <Form.Item label="Feed format" name="broadcastUrlFormat">
              <Select>
                <Select.Option value="tsh_newt_json">
                  tsh_newt_json
                </Select.Option>
              </Select>
            </Form.Item>
            <Form.Item
              label="Poll interval (seconds)"
              name="pollIntervalSeconds"
            >
              <InputNumber min={30} max={3600} />
            </Form.Item>
            <Form.Item label="Poll start time" name="pollStartTime">
              <DatePicker showTime />
            </Form.Item>
            <Form.Item label="Poll end time" name="pollEndTime">
              <DatePicker showTime />
            </Form.Item>
            <LexiconFormItem />
            <ChallengeRulesFormItem disabled={false} />
            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={updateMutation.isPending}
                >
                  Save
                </Button>
                <Button onClick={() => navigate(`/broadcasts/${slug}`)}>
                  Cancel
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Card>
      </div>
    </div>
  );
};
