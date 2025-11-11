import React, { useState } from "react";
import {
  Col,
  Row,
  Card,
  Form,
  Input,
  InputNumber,
  Button,
  DatePicker,
  Select,
  Space,
  Alert,
  notification,
} from "antd";
import { Store } from "rc-field-form/lib/interface";
import { useMutation } from "@connectrpc/connect-query";
import { TopBar } from "../navigation/topbar";
import {
  createLeague,
  bootstrapSeason,
} from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { flashError } from "../utils/hooks/connect";
import { useLoginStateStoreContext } from "../store/store";
import { SeasonStatus } from "../gen/api/proto/ipc/league_pb";
import { dayjsToProtobufTimestampIgnoringNanos } from "../utils/datetime";
import "./leagues.scss";

const { TextArea } = Input;
const { Option } = Select;

export const LeagueAdmin = () => {
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn } = loginState;

  const [leagueForm] = Form.useForm();
  const [seasonForm] = Form.useForm();
  const [createdLeagueSlug, setCreatedLeagueSlug] = useState<string>("");

  const createLeagueMutation = useMutation(createLeague, {
    onSuccess: (response) => {
      if (response.league) {
        setCreatedLeagueSlug(response.league.slug);
        leagueForm.resetFields();
        notification.success({
          message: "League Created",
          description: `League created successfully! Slug: ${response.league.slug}`,
        });
      }
    },
    onError: (error) => {
      flashError(error);
    },
  });

  const bootstrapSeasonMutation = useMutation(bootstrapSeason, {
    onSuccess: () => {
      seasonForm.resetFields();
      notification.success({
        message: "Season Bootstrapped",
        description: "Season has been bootstrapped successfully!",
      });
    },
    onError: (error) => {
      flashError(error);
    },
  });

  const handleCreateLeague = (values: {
    name: string;
    description: string;
    slug: string;
    seasonLengthDays: number;
    incrementSeconds: number;
    timeBankMinutes: number;
    lexicon: string;
    variant: string;
    idealDivisionSize: number;
  }) => {
    createLeagueMutation.mutate({
      name: values.name,
      description: values.description,
      slug: values.slug,
      settings: {
        seasonLengthDays: values.seasonLengthDays,
        timeControl: {
          incrementSeconds: values.incrementSeconds,
          timeBankMinutes: values.timeBankMinutes,
        },
        lexicon: values.lexicon,
        variant: values.variant,
        idealDivisionSize: values.idealDivisionSize,
        challengeRule: 0, // ChallengeRule.DOUBLE
      },
    });
  };

  const handleBootstrapSeason = (vals: Store) => {
    const startTimestamp = vals.startDate
      ? dayjsToProtobufTimestampIgnoringNanos(vals.startDate)
      : undefined;
    const endTimestamp = vals.endDate
      ? dayjsToProtobufTimestampIgnoringNanos(vals.endDate)
      : undefined;

    bootstrapSeasonMutation.mutate({
      leagueId: vals.leagueId,
      startDate: startTimestamp,
      endDate: endTimestamp,
      status: vals.status,
    });
  };

  if (!loggedIn) {
    return (
      <>
        <Row>
          <Col span={24}>
            <TopBar />
          </Col>
        </Row>
        <div className="leagues-container">
          <Alert
            message="Authentication Required"
            description="You must be logged in to access the admin page."
            type="warning"
          />
        </div>
      </>
    );
  }

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <div className="leagues-container">
        <h1>League Administration</h1>
        <p>
          Create new leagues and bootstrap their first seasons. This page
          requires <code>can_manage_leagues</code> permission.
        </p>

        <Space direction="vertical" size="large" style={{ width: "100%" }}>
          {/* Create League Form */}
          <Card title="Create New League">
            <Form
              form={leagueForm}
              layout="vertical"
              onFinish={handleCreateLeague}
              initialValues={{
                seasonLengthDays: 14,
                incrementSeconds: 21600, // 6 hours
                timeBankMinutes: 4320, // 72 hours
                lexicon: "NWL23",
                variant: "classic",
                idealDivisionSize: 15,
              }}
            >
              <Form.Item
                label="League Name"
                name="name"
                rules={[
                  { required: true, message: "Please enter a league name" },
                ]}
              >
                <Input placeholder="e.g., WoogLeague" />
              </Form.Item>

              <Form.Item label="Description" name="description">
                <TextArea
                  rows={3}
                  placeholder="A competitive correspondence-based league..."
                />
              </Form.Item>

              <Form.Item
                label="Slug"
                name="slug"
                rules={[{ required: true, message: "Please enter a slug" }]}
              >
                <Input placeholder="e.g., woogleague" />
              </Form.Item>

              <Form.Item
                label="Season Length (days)"
                name="seasonLengthDays"
                rules={[
                  { required: true, message: "Please enter season length" },
                ]}
              >
                <InputNumber min={1} max={90} style={{ width: "100%" }} />
              </Form.Item>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    label="Time per turn (seconds)"
                    name="incrementSeconds"
                    rules={[
                      { required: true, message: "Please enter time per turn" },
                    ]}
                  >
                    <InputNumber min={1} style={{ width: "100%" }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    label="Time bank (minutes)"
                    name="timeBankMinutes"
                    rules={[
                      { required: true, message: "Please enter time bank" },
                    ]}
                  >
                    <InputNumber min={1} style={{ width: "100%" }} />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item label="Lexicon" name="lexicon">
                    <Input placeholder="e.g., NWL23" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label="Variant" name="variant">
                    <Input placeholder="e.g., classic" />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item
                label="Ideal Division Size"
                name="idealDivisionSize"
                rules={[
                  {
                    required: true,
                    message: "Please enter ideal division size",
                  },
                ]}
                help="Target size for divisions (actual sizes will range from 13-20 players). Promotion/relegation is automatically calculated as ceil(size/6)."
              >
                <InputNumber min={10} max={20} style={{ width: "100%" }} />
              </Form.Item>

              <Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={createLeagueMutation.isPending}
                >
                  Create League
                </Button>
              </Form.Item>
            </Form>

            {createdLeagueSlug && (
              <Alert
                message="League Created!"
                description={
                  <>
                    Successfully created league. View it at{" "}
                    <a href={`/leagues/${createdLeagueSlug}`}>
                      /leagues/{createdLeagueSlug}
                    </a>
                  </>
                }
                type="success"
                closable
              />
            )}
          </Card>

          {/* Bootstrap Season Form */}
          <Card title="Bootstrap Initial Season">
            <Alert
              message="Bootstrap Season"
              description="This creates the first season for a league. Can only be used when the league has zero seasons."
              type="info"
              style={{ marginBottom: 16 }}
            />
            <Form
              form={seasonForm}
              layout="vertical"
              onFinish={handleBootstrapSeason}
            >
              <Form.Item
                label="League ID or Slug"
                name="leagueId"
                rules={[
                  {
                    required: true,
                    message: "Please enter league ID or slug",
                  },
                ]}
              >
                <Input placeholder="e.g., woogleague or UUID" />
              </Form.Item>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    label="Start Date"
                    name="startDate"
                    rules={[
                      { required: true, message: "Please select start date" },
                    ]}
                  >
                    <DatePicker
                      showTime
                      style={{ width: "100%" }}
                      format="YYYY-MM-DD HH:mm:ss"
                    />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    label="End Date"
                    name="endDate"
                    rules={[
                      { required: true, message: "Please select end date" },
                    ]}
                  >
                    <DatePicker
                      showTime
                      style={{ width: "100%" }}
                      format="YYYY-MM-DD HH:mm:ss"
                    />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item
                label="Status"
                name="status"
                rules={[{ required: true, message: "Please select status" }]}
                initialValue={SeasonStatus.SEASON_SCHEDULED}
              >
                <Select placeholder="Select season status">
                  <Option value={SeasonStatus.SEASON_SCHEDULED}>
                    Scheduled
                  </Option>
                  <Option value={SeasonStatus.SEASON_REGISTRATION_OPEN}>
                    Registration Open
                  </Option>
                  <Option value={SeasonStatus.SEASON_ACTIVE}>Active</Option>
                </Select>
              </Form.Item>

              <Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={bootstrapSeasonMutation.isPending}
                >
                  Bootstrap Season
                </Button>
              </Form.Item>
            </Form>
          </Card>
        </Space>
      </div>
    </>
  );
};
