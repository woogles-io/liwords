import React, { useState, useMemo } from "react";
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
  App,
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
  getSeasonRegistrations,
  getAllDivisionStandings,
  getAllSeasons,
  movePlayerToDivision,
  updateSeasonDates,
  unregisterFromSeason,
} from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { flashError } from "../utils/hooks/connect";
import { useLoginStateStoreContext } from "../store/store";
import { SeasonStatus } from "../gen/api/proto/ipc/league_pb";
import { ChallengeRule } from "../gen/api/proto/ipc/omgwords_pb";
import {
  dayjsToProtobufTimestampIgnoringNanos,
  dayjs,
} from "../utils/datetime";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { AllLexica } from "../shared/lexica";
import "./leagues.scss";

const { TextArea } = Input;
const { Option } = Select;

export const LeagueAdmin = () => {
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn } = loginState;
  const { notification } = App.useApp();

  const [leagueForm] = Form.useForm();
  const [seasonForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [movePlayerForm] = Form.useForm();
  const [editSeasonDatesForm] = Form.useForm();
  const [createdLeagueSlug, setCreatedLeagueSlug] = useState<string>("");
  const [selectedLeagueId, setSelectedLeagueId] = useState<string>("");
  const [selectedMoveLeagueId, setSelectedMoveLeagueId] = useState<string>("");
  const [selectedEditDatesLeagueId, setSelectedEditDatesLeagueId] =
    useState<string>("");
  const [selectedEditDatesSeasonId, setSelectedEditDatesSeasonId] =
    useState<string>("");
  const [selectedPlayerId, setSelectedPlayerId] = useState<string>("");
  const [selectedPlayerDivisionId, setSelectedPlayerDivisionId] =
    useState<string>("");

  // Fetch all leagues for the edit dropdown
  const { data: leaguesData } = useQuery(getAllLeagues, {
    activeOnly: false,
  });

  // Fetch all seasons for the selected league (for move player)
  const { data: allSeasonsData } = useQuery(
    getAllSeasons,
    { leagueId: selectedMoveLeagueId },
    { enabled: !!selectedMoveLeagueId },
  );

  // Fetch all seasons for the edit dates league
  const { data: editDatesSeasonsData, refetch: refetchEditDatesSeasons } =
    useQuery(
      getAllSeasons,
      { leagueId: selectedEditDatesLeagueId },
      { enabled: !!selectedEditDatesLeagueId },
    );

  // Find the latest season (try SCHEDULED first, then fall back to highest season number)
  const latestSeason = useMemo(() => {
    const seasons = allSeasonsData?.seasons || [];
    if (seasons.length === 0) return null;

    // Sort by season number descending (newest first)
    const sortedSeasons = [...seasons].sort(
      (a, b) => (b.seasonNumber || 0) - (a.seasonNumber || 0),
    );

    // Try to find SCHEDULED season first
    const scheduledSeason = sortedSeasons.find(
      (s) => s.status === SeasonStatus.SEASON_SCHEDULED,
    );
    if (scheduledSeason) return scheduledSeason;

    // Otherwise return the latest season by number
    return sortedSeasons[0];
  }, [allSeasonsData?.seasons]);

  // Fetch registrations for the latest season
  const { data: registrationsData, refetch: refetchRegistrations } = useQuery(
    getSeasonRegistrations,
    { seasonId: latestSeason?.uuid || "" },
    { enabled: !!latestSeason?.uuid },
  );

  // Fetch divisions for the latest season
  const { data: divisionsData, refetch: refetchDivisions } = useQuery(
    getAllDivisionStandings,
    { seasonId: latestSeason?.uuid || "" },
    { enabled: !!latestSeason?.uuid },
  );

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

  const movePlayerMutation = useMutation(movePlayerToDivision, {
    onSuccess: (response) => {
      notification.success({
        message: "Player Moved",
        description: response.message || "Player moved successfully!",
      });
      // Refetch the data to show updated divisions
      refetchRegistrations();
      refetchDivisions();
      // Reset selection
      setSelectedPlayerId("");
      setSelectedPlayerDivisionId("");
      movePlayerForm.resetFields();
    },
    onError: (error) => {
      flashError(error);
    },
  });

  const unregisterUserMutation = useMutation(unregisterFromSeason, {
    onSuccess: () => {
      notification.success({
        message: "User Removed",
        description: "User has been removed from the season successfully!",
      });
      // Refetch the data to show updated divisions
      refetchRegistrations();
      refetchDivisions();
      // Reset selection
      setSelectedPlayerId("");
      setSelectedPlayerDivisionId("");
      movePlayerForm.resetFields();
    },
    onError: (error) => {
      flashError(error);
    },
  });

  const updateSeasonDatesMutation = useMutation(updateSeasonDates, {
    onSuccess: () => {
      notification.success({
        message: "Season Dates Updated",
        description: "Season dates have been updated successfully!",
      });
      refetchEditDatesSeasons();
      editSeasonDatesForm.resetFields(["startDate", "endDate"]);
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
    } catch {
      // Errors are already handled by individual mutations
    }
  };

  const handlePlayerSelect = (playerId: string) => {
    setSelectedPlayerId(playerId);
    // Find the player's current division
    const registration = registrationsData?.registrations?.find(
      (r) => r.userId === playerId,
    );
    if (registration?.divisionId) {
      setSelectedPlayerDivisionId(registration.divisionId);
    }
  };

  const handleMovePlayer = (values: { toDivisionId: string }) => {
    if (!selectedPlayerId || !selectedPlayerDivisionId) {
      notification.error({
        message: "Selection Error",
        description: "Please select a player first",
      });
      return;
    }

    if (!latestSeason?.uuid) {
      notification.error({
        message: "Season Error",
        description: "No season found for this league",
      });
      return;
    }

    // Check if user selected "remove from league" option
    if (values.toDivisionId === "REMOVE_USER") {
      handleRemoveUser();
      return;
    }

    movePlayerMutation.mutate({
      userId: selectedPlayerId,
      seasonId: latestSeason.uuid,
      fromDivisionId: selectedPlayerDivisionId,
      toDivisionId: values.toDivisionId,
    });
  };

  const handleRemoveUser = () => {
    if (!selectedPlayerId) {
      notification.error({
        message: "Selection Error",
        description: "Please select a player first",
      });
      return;
    }

    if (!latestSeason?.uuid) {
      notification.error({
        message: "Season Error",
        description: "No season found for this league",
      });
      return;
    }

    unregisterUserMutation.mutate({
      seasonId: latestSeason.uuid,
      userId: selectedPlayerId,
    });
  };

  const handleUpdateSeasonDates = (vals: Store) => {
    if (!selectedEditDatesSeasonId) {
      notification.error({
        message: "Selection Error",
        description: "Please select a season first",
      });
      return;
    }

    const startTimestamp = vals.startDate
      ? dayjsToProtobufTimestampIgnoringNanos(vals.startDate)
      : undefined;
    const endTimestamp = vals.endDate
      ? dayjsToProtobufTimestampIgnoringNanos(vals.endDate)
      : undefined;

    updateSeasonDatesMutation.mutate({
      seasonId: selectedEditDatesSeasonId,
      startDate: startTimestamp,
      endDate: endTimestamp,
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
                  <Form.Item
                    label="Lexicon"
                    name="lexicon"
                    rules={[
                      { required: true, message: "Please select a lexicon" },
                    ]}
                  >
                    <Select placeholder="Select a lexicon">
                      {Object.entries(AllLexica).map(([code, lexicon]) => (
                        <Option key={code} value={code}>
                          {lexicon.shortDescription}
                        </Option>
                      ))}
                    </Select>
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
                help="Target size for divisions (actual sizes will range from 13-20 players). Promotion/relegation is automatically calculated as ceil((size+1)/5)."
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
                      <Select placeholder="Select a lexicon">
                        {Object.entries(AllLexica).map(([code, lexicon]) => (
                          <Option key={code} value={code}>
                            {lexicon.shortDescription}
                          </Option>
                        ))}
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

          {/* Edit Season Dates */}
          <Card title="Edit Season Dates">
            <Alert
              message="Edit Season Dates"
              description="Update the start and end dates of a season. Only works when season is SCHEDULED or REGISTRATION_OPEN."
              type="info"
              style={{ marginBottom: 16 }}
            />

            <Form.Item label="Select League">
              <Select
                placeholder="Choose a league"
                onChange={(value) => {
                  setSelectedEditDatesLeagueId(value);
                  setSelectedEditDatesSeasonId("");
                  editSeasonDatesForm.resetFields();
                }}
                value={selectedEditDatesLeagueId || undefined}
              >
                {leaguesData?.leagues?.map((league) => (
                  <Option key={league.uuid} value={league.uuid}>
                    {league.name} ({league.slug})
                  </Option>
                ))}
              </Select>
            </Form.Item>

            {selectedEditDatesLeagueId &&
              editDatesSeasonsData?.seasons &&
              editDatesSeasonsData.seasons.length > 0 && (
                <>
                  <Form.Item label="Select Season">
                    <Select
                      placeholder="Choose a season"
                      onChange={(value) => {
                        setSelectedEditDatesSeasonId(value);
                        // Pre-populate the form with current dates
                        const season = editDatesSeasonsData.seasons?.find(
                          (s) => s.uuid === value,
                        );
                        if (season) {
                          editSeasonDatesForm.setFieldsValue({
                            startDate: season.startDate
                              ? dayjs(timestampDate(season.startDate))
                              : undefined,
                            endDate: season.endDate
                              ? dayjs(timestampDate(season.endDate))
                              : undefined,
                          });
                        }
                      }}
                      value={selectedEditDatesSeasonId || undefined}
                    >
                      {editDatesSeasonsData.seasons.map((season) => (
                        <Option key={season.uuid} value={season.uuid}>
                          Season {season.seasonNumber} (
                          {SeasonStatus[season.status]})
                        </Option>
                      ))}
                    </Select>
                  </Form.Item>

                  {selectedEditDatesSeasonId && (
                    <Form
                      form={editSeasonDatesForm}
                      layout="vertical"
                      onFinish={handleUpdateSeasonDates}
                    >
                      <Row gutter={16}>
                        <Col span={12}>
                          <Form.Item
                            label="Start Date"
                            name="startDate"
                            rules={[
                              {
                                required: true,
                                message: "Please select start date",
                              },
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
                              {
                                required: true,
                                message: "Please select end date",
                              },
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

                      <Form.Item>
                        <Button
                          type="primary"
                          htmlType="submit"
                          loading={updateSeasonDatesMutation.isPending}
                        >
                          Update Season Dates
                        </Button>
                      </Form.Item>
                    </Form>
                  )}
                </>
              )}
          </Card>

          {/* Move Player Between Divisions */}
          <Card title="Move Player Between Divisions">
            <Alert
              message="Move Player"
              description="Move a player from one division to another, or remove them from the league entirely. Only works when season is SCHEDULED."
              type="info"
              style={{ marginBottom: 16 }}
            />

            <Form.Item label="Select League">
              <Select
                placeholder="Choose a league"
                onChange={(value) => {
                  setSelectedMoveLeagueId(value);
                  setSelectedPlayerId("");
                  setSelectedPlayerDivisionId("");
                  movePlayerForm.resetFields();
                }}
                value={selectedMoveLeagueId || undefined}
              >
                {leaguesData?.leagues?.map((league) => (
                  <Option key={league.uuid} value={league.uuid}>
                    {league.name} ({league.slug})
                  </Option>
                ))}
              </Select>
            </Form.Item>

            {selectedMoveLeagueId && latestSeason && (
              <>
                <Alert
                  message={`Latest Season: ${latestSeason.seasonNumber} (Status: ${SeasonStatus[latestSeason.status]})`}
                  type="info"
                  style={{ marginBottom: 16 }}
                />

                {latestSeason.status !== SeasonStatus.SEASON_SCHEDULED && (
                  <Alert
                    message="Season must be SCHEDULED to move players"
                    type="warning"
                    style={{ marginBottom: 16 }}
                  />
                )}

                <Form
                  form={movePlayerForm}
                  layout="vertical"
                  onFinish={handleMovePlayer}
                >
                  <Form.Item label="Select Player" required>
                    <Select
                      placeholder="Choose a player"
                      onChange={handlePlayerSelect}
                      value={selectedPlayerId || undefined}
                      showSearch
                      optionFilterProp="children"
                      disabled={
                        latestSeason.status !== SeasonStatus.SEASON_SCHEDULED
                      }
                    >
                      {registrationsData?.registrations?.map((reg) => {
                        const divisionInfo = divisionsData?.divisions?.find(
                          (d) => d.uuid === reg.divisionId,
                        );
                        const divisionLabel = divisionInfo
                          ? `Division ${divisionInfo.divisionNumber}`
                          : "No Division";
                        return (
                          <Option key={reg.userId} value={reg.userId}>
                            {reg.username} ({divisionLabel})
                          </Option>
                        );
                      })}
                    </Select>
                  </Form.Item>

                  <Form.Item
                    label="Target Division"
                    name="toDivisionId"
                    rules={[
                      {
                        required: true,
                        message: "Please select target division or remove user",
                      },
                    ]}
                  >
                    <Select
                      placeholder="Choose target division or remove user"
                      disabled={
                        !selectedPlayerId ||
                        latestSeason.status !== SeasonStatus.SEASON_SCHEDULED
                      }
                    >
                      <Option
                        key="REMOVE_USER"
                        value="REMOVE_USER"
                        style={{ color: "red", fontWeight: "bold" }}
                      >
                        ⚠️ Remove user from league
                      </Option>
                      {divisionsData?.divisions
                        ?.filter((d) => d.uuid !== selectedPlayerDivisionId)
                        .map((division) => (
                          <Option key={division.uuid} value={division.uuid}>
                            Division {division.divisionNumber} (
                            {division.standings?.length || 0} players)
                          </Option>
                        ))}
                    </Select>
                  </Form.Item>

                  <Form.Item>
                    <Button
                      type="primary"
                      htmlType="submit"
                      loading={
                        movePlayerMutation.isPending ||
                        unregisterUserMutation.isPending
                      }
                      disabled={
                        !selectedPlayerId ||
                        latestSeason.status !== SeasonStatus.SEASON_SCHEDULED
                      }
                    >
                      Move Player / Remove User
                    </Button>
                  </Form.Item>
                </Form>
              </>
            )}
          </Card>
        </Space>
      </div>
    </>
  );
};
