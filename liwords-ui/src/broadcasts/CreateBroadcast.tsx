import React from "react";
import {
  Form,
  Input,
  InputNumber,
  Button,
  Card,
  Space,
  Select,
  App,
} from "antd";
import { DatePicker } from "antd";
import { useNavigate } from "react-router";
import { useMutation } from "@connectrpc/connect-query";
import { TopBar } from "../navigation/topbar";
import { useLoginStateStoreContext } from "../store/store";
import { LexiconFormItem } from "../shared/lexicon_display";
import { ChallengeRulesFormItem } from "../lobby/challenge_rules_form_item";
import { createBroadcast } from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import { dayjsToProtobufTimestampIgnoringNanos } from "../utils/datetime";
import { defaultLetterDistribution } from "../lobby/sought_game_interactions";
import { flashError } from "../utils/hooks/connect";

const { TextArea } = Input;

export const CreateBroadcast: React.FC = () => {
  const navigate = useNavigate();
  const { loginState } = useLoginStateStoreContext();
  const { notification } = App.useApp();
  const [form] = Form.useForm();

  const createMutation = useMutation(createBroadcast, {
    onSuccess: (resp) => {
      notification.success({ message: "Broadcast created" });
      navigate(`/broadcasts/${resp.slug}`);
    },
    onError: (e) => flashError(e),
  });

  const isAdmin = loginState.perms.includes("adm");
  if (!loginState.loggedIn || !isAdmin) {
    return (
      <div>
        <TopBar />
        <p style={{ padding: 24 }}>Not authorized.</p>
      </div>
    );
  }

  const onFinish = (vals: Record<string, unknown>) => {
    const lexicon = (vals.lexicon as string) ?? "CSW24";
    createMutation.mutate({
      slug: vals.slug as string,
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
      boardLayout: "CrosswordGame",
      letterDistribution: defaultLetterDistribution(lexicon),
      challengeRule: (vals.challengerule as number) ?? 0,
    });
  };

  return (
    <div>
      <TopBar />
      <div style={{ maxWidth: 640, margin: "32px auto", padding: "0 16px" }}>
        <Card title="Create Broadcast">
          <Form
            form={form}
            layout="vertical"
            onFinish={onFinish}
            initialValues={{
              broadcastUrlFormat: "tsh_newt_json",
              pollIntervalSeconds: 300,
              lexicon: "CSW24",
              challengerule: 0,
            }}
          >
            <Form.Item
              label="Slug"
              name="slug"
              rules={[{ required: true }]}
              extra="URL-friendly ID, e.g. thailand-2026"
            >
              <Input />
            </Form.Item>
            <Form.Item label="Name" name="name" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item label="Description" name="description">
              <TextArea rows={2} />
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
            <Form.Item
              label="Poll end time"
              name="pollEndTime"
              rules={[
                {
                  validator: async (_, value) => {
                    const start = form.getFieldValue("pollStartTime");
                    if (start && value && !value.isAfter(start)) {
                      throw new Error("End time must be after start time");
                    }
                  },
                },
              ]}
            >
              <DatePicker showTime />
            </Form.Item>
            <LexiconFormItem />
            <ChallengeRulesFormItem disabled={false} />
            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={createMutation.isPending}
                >
                  Create
                </Button>
                <Button onClick={() => navigate(-1)}>Cancel</Button>
              </Space>
            </Form.Item>
          </Form>
        </Card>
      </div>
    </div>
  );
};
