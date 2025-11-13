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
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { TopBar } from "../navigation/topbar";
import {
  createLeague,
  bootstrapSeason,
  getAllLeagues,
  updateLeagueSettings,
  updateLeagueMetadata,
} from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { flashError } from "../utils/hooks/connect";
import { useLoginStateStoreContext } from "../store/store";
import { SeasonStatus } from "../gen/api/proto/ipc/league_pb";
import { ChallengeRule } from "../gen/api/proto/ipc/omgwords_pb";
import { dayjsToProtobufTimestampIgnoringNanos } from "../utils/datetime";
import "./leagues.scss";

const { TextArea } = Input;
const { Option } = Select;

export const LeagueAdmin = () => {
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn } = loginState;

  const [leagueForm] = Form.useForm();
  const [seasonForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [createdLeagueSlug, setCreatedLeagueSlug] = useState<string>("");
  const [selectedLeagueId, setSelectedLeagueId] = useState<string>("");

  // Fetch all leagues for the edit dropdown
  const { data: leaguesData } = useQuery(getAllLeagues, {
    activeOnly: false,
  });

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

  const updateLeagueSettingsMutation = useMutation(updateLeagueSettings, {
    onSuccess: (response) => {
      notification.success({
        message: "League Settings Updated",
        description: `League settings updated successfully!`,
      });
    },
    onError: (error) => {
      flashError(error);
    },
  });

  const updateLeagueMetadataMutation = useMutation(updateLeagueMetadata, {
    onSuccess: (response) => {
      notification.success({
        message: "League Metadata Updated",
        description: `League "${response.league?.name}" updated successfully!`,
      });
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
    challengeRule: number;
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
        challengeRule: values.challengeRule,
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

  const handleLeagueSelect = (leagueId: string) => {
    setSelectedLeagueId(leagueId);
    const selectedLeague = leaguesData?.leagues?.find(
      (l) => l.uuid === leagueId,
    );
    if (selectedLeague && selectedLeague.settings) {
      editForm.setFieldsValue({
        name: selectedLeague.name,
        description: selectedLeague.description,
        seasonLengthDays: selectedLeague.settings.seasonLengthDays,
        incrementSeconds:
          selectedLeague.settings.timeControl?.incrementSeconds || 0,
        timeBankMinutes:
          selectedLeague.settings.timeControl?.timeBankMinutes || 0,
        lexicon: selectedLeague.settings.lexicon,
        variant: selectedLeague.settings.variant,
        idealDivisionSize: selectedLeague.settings.idealDivisionSize,
        challengeRule: selectedLeague.settings.challengeRule,
      });
    }
  };

  const handleUpdateLeague = async (values: {
    name: string;
    description: string;
    seasonLengthDays: number;
    incrementSeconds: number;
    timeBankMinutes: number;
    lexicon: string;
    variant: string;
    idealDivisionSize: number;
    challengeRule: number;
  }) => {
    // Update both metadata and settings
    try {
      await updateLeagueMetadataMutation.mutateAsync({
        leagueId: selectedLeagueId,
        name: values.name,
        description: values.description,
      });

      await updateLeagueSettingsMutation.mutateAsync({
        leagueId: selectedLeagueId,
        settings: {
          seasonLengthDays: values.seasonLengthDays,
          timeControl: {
            incrementSeconds: values.incrementSeconds,
            timeBankMinutes: values.timeBankMinutes,
          },
          lexicon: values.lexicon,
          variant: values.variant,
          idealDivisionSize: values.idealDivisionSize,
          challengeRule: values.challengeRule,
        },
      });

      notification.success({
        message: "League Updated",
        description: `League "${values.name}" updated successfully!`,
      });
    } catch (error) {
      // Errors are already handled by individual mutations
    }
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
                challengeRule: ChallengeRule.ChallengeRule_DOUBLE,
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

              <Form.Item
                label="Challenge Rule"
                name="challengeRule"
                rules={[
                  { required: true, message: "Please select challenge rule" },
                ]}
              >
                <Select>
                  <Option value={ChallengeRule.ChallengeRule_VOID}>Void</Option>
                  <Option value={ChallengeRule.ChallengeRule_SINGLE}>
                    Single
                  </Option>
                  <Option value={ChallengeRule.ChallengeRule_DOUBLE}>
                    Double
                  </Option>
                  <Option value={ChallengeRule.ChallengeRule_FIVE_POINT}>
                    Five Point
                  </Option>
                  <Option value={ChallengeRule.ChallengeRule_TEN_POINT}>
                    Ten Point
                  </Option>
                  <Option value={ChallengeRule.ChallengeRule_TRIPLE}>
                    Triple
                  </Option>
                </Select>
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

          {/* Edit Existing League */}
          <Card title="Edit Existing League">
            <Alert
              message="Edit League"
              description="Select a league to edit its name, description, and settings."
              type="info"
              style={{ marginBottom: 16 }}
            />

            <Form.Item label="Select League">
              <Select
                placeholder="Choose a league to edit"
                onChange={handleLeagueSelect}
                value={selectedLeagueId || undefined}
              >
                {leaguesData?.leagues?.map((league) => (
                  <Option key={league.uuid} value={league.uuid}>
                    {league.name} ({league.slug})
                    {league.isActive ? "" : " - INACTIVE"}
                  </Option>
                ))}
              </Select>
            </Form.Item>

            {selectedLeagueId && (
              <Form
                form={editForm}
                layout="vertical"
                onFinish={handleUpdateLeague}
              >
                <Form.Item
                  label="League Name"
                  name="name"
                  rules={[
                    { required: true, message: "Please enter a league name" },
                  ]}
                >
                  <Input placeholder="League name" />
                </Form.Item>

                <Form.Item label="Description" name="description">
                  <TextArea rows={3} placeholder="Description" />
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
                        {
                          required: true,
                          message: "Please enter time per turn",
                        },
                      ]}
                    >
                      <InputNumber min={1} style={{ width: "100%" }} />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item
                      label="Time Bank (minutes)"
                      name="timeBankMinutes"
                      rules={[
                        {
                          required: true,
                          message: "Please enter time bank minutes",
                        },
                      ]}
                    >
                      <InputNumber min={0} style={{ width: "100%" }} />
                    </Form.Item>
                  </Col>
                </Row>

                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item
                      label="Lexicon"
                      name="lexicon"
                      rules={[
                        { required: true, message: "Please select lexicon" },
                      ]}
                    >
                      <Select>
                        <Option value="NWL23">NWL23</Option>
                        <Option value="CSW21">CSW21</Option>
                        <Option value="ECWL">ECWL</Option>
                        <Option value="FRA20">FRA20</Option>
                      </Select>
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item
                      label="Variant"
                      name="variant"
                      rules={[
                        { required: true, message: "Please select variant" },
                      ]}
                    >
                      <Select>
                        <Option value="classic">Classic</Option>
                        <Option value="wordsmog">Wordsmog</Option>
                      </Select>
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
                >
                  <InputNumber min={2} max={50} style={{ width: "100%" }} />
                </Form.Item>

                <Form.Item
                  label="Challenge Rule"
                  name="challengeRule"
                  rules={[
                    { required: true, message: "Please select challenge rule" },
                  ]}
                >
                  <Select>
                    <Option value={ChallengeRule.ChallengeRule_VOID}>
                      Void
                    </Option>
                    <Option value={ChallengeRule.ChallengeRule_SINGLE}>
                      Single
                    </Option>
                    <Option value={ChallengeRule.ChallengeRule_DOUBLE}>
                      Double
                    </Option>
                    <Option value={ChallengeRule.ChallengeRule_FIVE_POINT}>
                      Five Point
                    </Option>
                    <Option value={ChallengeRule.ChallengeRule_TEN_POINT}>
                      Ten Point
                    </Option>
                    <Option value={ChallengeRule.ChallengeRule_TRIPLE}>
                      Triple
                    </Option>
                  </Select>
                </Form.Item>

                <Form.Item>
                  <Button type="primary" htmlType="submit">
                    Update League Settings
                  </Button>
                </Form.Item>
              </Form>
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
