import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  Alert,
  Button,
  Card,
  DatePicker,
  Divider,
  Form,
  Input,
  InputNumber,
  message,
  Radio,
  Select,
  Steps,
  Switch,
  Typography,
} from "antd";
import type { RadioChangeEvent } from "antd";
import { useNavigate } from "react-router";
import {
  PlusOutlined,
  CopyOutlined,
  TrophyOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import {
  TType,
  TournamentService,
} from "../gen/api/proto/tournament_service/tournament_service_pb";
import type { TournamentMetadata } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { flashError, useClient } from "../utils/hooks/connect";
import { TopBar } from "../navigation/topbar";
import { useLoginStateStoreContext } from "../store/store";
import {
  SettingsForm,
  DisplayedGameSetting,
} from "./director_tools/game_settings_form";
import { Modal } from "../utils/focus_modal";
import { GameRequest } from "../gen/api/proto/ipc/omgwords_pb";
import {
  TournamentGameResult,
  RoundControl,
} from "../gen/api/proto/ipc/tournament_pb";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import {
  doesCurrentUserUse24HourTime,
  protobufTimestampToDayjsIgnoringNanos,
} from "../utils/datetime";
import type { Dayjs } from "dayjs";
import "./tournament_wizard.scss";

const { Title, Paragraph, Text } = Typography;

type DivisionConfig = {
  name: string;
  numRounds: number;
  gameRequest?: GameRequest;
  roundControls?: RoundControl[];
};

type WizardData = {
  // Step 0: Tournament type
  tournamentMode: "online" | "irl";
  monitored: boolean;
  // Step 1: Basic info
  name: string;
  slug: string;
  description: string;
  scheduledStartTime: Dayjs | null;
  scheduledEndTime: Dayjs | null;
  // Step 2: Divisions & Settings
  divisions: DivisionConfig[];
  gameRequest: GameRequest | undefined;
  // Copy source
  copyFromSlug?: string;
  gameSettingsVaryPerDivision?: boolean;
};

const STEPS = [
  { title: "Tournament Type" },
  { title: "Basic Info" },
  { title: "Divisions & Settings" },
  { title: "Review & Create" },
];

export const TournamentWizard = () => {
  const [currentStep, setCurrentStep] = useState(0);
  const [settingsModalVisible, setSettingsModalVisible] = useState(false);
  const [creating, setCreating] = useState(false);
  const [myTournaments, setMyTournaments] = useState<TournamentMetadata[]>([]);
  const [showCopyPicker, setShowCopyPicker] = useState(false);
  const [loadingMyTournaments, setLoadingMyTournaments] = useState(false);
  const [slugManuallyEdited, setSlugManuallyEdited] = useState(false);

  const { loginState } = useLoginStateStoreContext();
  const navigate = useNavigate();
  const tournamentClient = useClient(TournamentService);
  const timeFormat = doesCurrentUserUse24HourTime() ? "HH:mm" : "hh:mm A";

  const canCreate =
    loginState.perms.includes("toc") || loginState.perms.includes("adm");

  const [wizardData, setWizardData] = useState<WizardData>({
    tournamentMode: "online",
    monitored: false,
    name: "",
    slug: "",
    description: "",
    scheduledStartTime: null,
    scheduledEndTime: null,
    divisions: [{ name: "Division 1", numRounds: 10 }],
    gameRequest: undefined,
  });

  const updateField = useCallback(
    <K extends keyof WizardData>(field: K, value: WizardData[K]) => {
      setWizardData((prev) => ({ ...prev, [field]: value }));
    },
    [],
  );

  const slugify = (name: string) =>
    name
      .toLowerCase()
      .replace(/[^a-z0-9-\s]/g, "")
      .trim()
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-");

  const updateName = useCallback(
    (name: string) => {
      setWizardData((prev) => ({
        ...prev,
        name,
        slug: slugManuallyEdited ? prev.slug : slugify(name),
      }));
    },
    [slugManuallyEdited],
  );

  const updateSlug = useCallback((raw: string) => {
    const sanitized = raw
      .toLowerCase()
      .replace(/[^a-z0-9-]/g, "-")
      .replace(/-+/g, "-");
    setSlugManuallyEdited(true);
    setWizardData((prev) => ({ ...prev, slug: sanitized }));
  }, []);

  const loadMyTournaments = useCallback(async () => {
    if (!canCreate) return;
    setLoadingMyTournaments(true);
    try {
      // Try the dedicated GetMyTournaments endpoint first
      try {
        const resp = await tournamentClient.getMyTournaments({});
        setMyTournaments(resp.tournaments);
        setLoadingMyTournaments(false);
        return;
      } catch {
        // Fall back to filtering past tournaments
      }
      const resp = await tournamentClient.getPastTournaments({ limit: 50 });
      const myTourneys = resp.tournaments.filter(
        (t) => t.firstDirector === loginState.username,
      );
      setMyTournaments(myTourneys);
    } catch {
      // Silently fail - copy feature is optional
    }
    setLoadingMyTournaments(false);
  }, [canCreate, tournamentClient, loginState.username]);

  useEffect(() => {
    loadMyTournaments();
  }, [loadMyTournaments]);

  const copyFromTournament = useCallback(
    async (slug: string) => {
      try {
        const resp = await tournamentClient.getTournamentMetadata({ slug });
        const metadata = resp.metadata;
        if (!metadata) {
          message.error({ content: "Tournament not found", duration: 3 });
          return;
        }

        // Build divisions from the metadata division summaries
        const divisions: DivisionConfig[] = metadata.divisions.map((d) => ({
          name: d.name,
          numRounds: d.roundControls.length || 1,
          gameRequest: d.gameRequest,
          roundControls:
            d.roundControls.length > 0 ? d.roundControls : undefined,
        }));

        // Determine shared game request: if all divisions have the same settings
        // (or there's only one division), use it as the global game request.
        // Otherwise leave it undefined and store per-division.
        let sharedGameRequest: GameRequest | undefined;
        let gameSettingsVary = false;
        if (divisions.length > 0 && divisions[0].gameRequest) {
          const firstJSON = JSON.stringify(divisions[0].gameRequest);
          const allSame = divisions.every(
            (d) => JSON.stringify(d.gameRequest) === firstJSON,
          );
          if (allSame) {
            sharedGameRequest = divisions[0].gameRequest;
            // Clear per-division game request since global covers it
            divisions.forEach((d) => {
              d.gameRequest = undefined;
            });
          } else {
            gameSettingsVary = true;
          }
        }

        setWizardData((prev) => ({
          ...prev,
          tournamentMode: metadata.irlMode ? "irl" : "online",
          monitored: metadata.monitored,
          name: metadata.name + " (Copy)",
          description: metadata.description,
          divisions: divisions.length > 0 ? divisions : prev.divisions,
          gameRequest: sharedGameRequest,
          gameSettingsVaryPerDivision: gameSettingsVary,
          copyFromSlug: slug,
        }));

        setShowCopyPicker(false);
        message.info({
          content: gameSettingsVary
            ? "Tournament copied! Note: game settings vary per division — configure them after creation in Director Tools."
            : "Tournament settings copied! Review and modify as needed.",
          duration: 5,
        });
      } catch (e) {
        flashError(e);
      }
    },
    [tournamentClient],
  );

  const handleCreate = useCallback(async () => {
    if (!wizardData.name.trim()) {
      message.error({ content: "Tournament name is required", duration: 3 });
      return;
    }

    setCreating(true);
    try {
      const userSlug = wizardData.slug.trim();
      const generatedSlug = userSlug
        ? `/tournament/${userSlug}`
        : `/tournament/${wizardData.name
            .trim()
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, "-")
            .replace(/^-|-$/g, "")}-${Math.random().toString(36).slice(2, 6)}`;

      const resp = await tournamentClient.newTournament({
        name: wizardData.name.trim(),
        description: wizardData.description,
        slug: generatedSlug,
        type: TType.STANDARD,
        directorUsernames: [loginState.username],
        scheduledStartTime: wizardData.scheduledStartTime
          ? timestampFromMs(wizardData.scheduledStartTime.unix() * 1000)
          : undefined,
        scheduledEndTime: wizardData.scheduledEndTime
          ? timestampFromMs(wizardData.scheduledEndTime.unix() * 1000)
          : undefined,
      });

      const tournamentId = resp.id;
      const slug = resp.slug;

      // Add divisions
      for (const div of wizardData.divisions) {
        try {
          await tournamentClient.addDivision({
            id: tournamentId,
            division: div.name,
          });
        } catch (e) {
          console.error("Error adding division:", e);
        }
      }

      // Set metadata for IRL mode and monitoring
      if (wizardData.tournamentMode === "irl" || wizardData.monitored) {
        try {
          await tournamentClient.setTournamentMetadata({
            metadata: {
              id: tournamentId,
              name: wizardData.name.trim(),
              irlMode: wizardData.tournamentMode === "irl",
              monitored: wizardData.monitored,
            },
            setOnlySpecified: true,
          });
        } catch (e) {
          console.error("Error setting metadata:", e);
        }
      }

      // Set division controls and round controls for each division
      for (const div of wizardData.divisions) {
        const effectiveGameRequest = div.gameRequest ?? wizardData.gameRequest;
        if (effectiveGameRequest || wizardData.tournamentMode === "irl") {
          try {
            await tournamentClient.setDivisionControls({
              id: tournamentId,
              division: div.name,
              gameRequest: effectiveGameRequest,
              suspendedResult: TournamentGameResult.FORFEIT_LOSS,
              suspendedSpread: -50,
              autoStart: false,
            });
          } catch (e) {
            console.error("Error setting division controls:", e);
          }
        }
        if (div.roundControls && div.roundControls.length > 0) {
          try {
            await tournamentClient.setRoundControls({
              id: tournamentId,
              division: div.name,
              roundControls: div.roundControls,
            });
          } catch (e) {
            console.error("Error setting round controls:", e);
          }
        }
      }

      message.success({
        content: "Tournament created successfully!",
        duration: 3,
      });

      // Navigate to the newly created tournament
      navigate(`/${slug}`);
    } catch (e) {
      flashError(e);
    }
    setCreating(false);
  }, [wizardData, tournamentClient, loginState.username, navigate]);

  const canProceed = useMemo(() => {
    switch (currentStep) {
      case 0:
        return true;
      case 1: {
        if (wizardData.name.trim().length === 0) return false;
        const { scheduledStartTime, scheduledEndTime } = wizardData;
        if (
          scheduledStartTime &&
          scheduledEndTime &&
          !scheduledEndTime.isAfter(scheduledStartTime)
        )
          return false;
        return true;
      }
      case 2:
        return (
          wizardData.divisions.length > 0 &&
          wizardData.divisions.every((d) => d.name.trim().length > 0)
        );
      case 3:
        return true;
      default:
        return false;
    }
  }, [currentStep, wizardData]);

  if (!loginState.loggedIn) {
    return (
      <>
        <TopBar />
        <div className="tournament-wizard">
          <Alert
            type="warning"
            message="Please log in to create a tournament."
            showIcon
          />
        </div>
      </>
    );
  }

  if (!canCreate) {
    return (
      <>
        <TopBar />
        <div className="tournament-wizard">
          <Alert
            type="info"
            message="Tournament Creation"
            description="You need the Tournament Creator role to create tournaments. Please contact a site administrator to request this permission."
            showIcon
          />
        </div>
      </>
    );
  }

  return (
    <>
      <TopBar />
      <div className="tournament-wizard">
        <div className="wizard-header">
          <Title level={3}>Create a Tournament</Title>
          <Button
            icon={<CopyOutlined />}
            onClick={() => setShowCopyPicker(true)}
            disabled={myTournaments.length === 0}
          >
            Copy from existing
          </Button>
        </div>

        <Steps
          current={currentStep}
          items={STEPS}
          className="wizard-steps"
          size="small"
        />

        <div className="wizard-content">
          {currentStep === 0 && (
            <StepTournamentType
              wizardData={wizardData}
              updateField={updateField}
            />
          )}
          {currentStep === 1 && (
            <StepBasicInfo
              wizardData={wizardData}
              updateField={updateField}
              updateName={updateName}
              updateSlug={updateSlug}
              timeFormat={timeFormat}
            />
          )}
          {currentStep === 2 && (
            <StepDivisions
              wizardData={wizardData}
              updateField={updateField}
              settingsModalVisible={settingsModalVisible}
              setSettingsModalVisible={setSettingsModalVisible}
              onSetGameRequest={(gr) => {
                setWizardData((prev) => ({
                  ...prev,
                  gameRequest: gr,
                  gameSettingsVaryPerDivision: false,
                  divisions: prev.divisions.map((d) => ({
                    ...d,
                    gameRequest: undefined,
                  })),
                }));
              }}
            />
          )}
          {currentStep === 3 && <StepReview wizardData={wizardData} />}
        </div>

        <div className="wizard-nav">
          <Button
            disabled={currentStep === 0}
            onClick={() => setCurrentStep((s) => s - 1)}
          >
            Back
          </Button>
          <div className="wizard-nav-right">
            {currentStep < STEPS.length - 1 && (
              <Button
                type="primary"
                disabled={!canProceed}
                onClick={() => setCurrentStep((s) => s + 1)}
              >
                Next
              </Button>
            )}
            {currentStep === STEPS.length - 1 && (
              <Button
                type="primary"
                loading={creating}
                onClick={handleCreate}
                icon={<TrophyOutlined />}
              >
                Create Tournament
              </Button>
            )}
          </div>
        </div>

        <Modal
          title="Copy from existing tournament"
          open={showCopyPicker}
          onCancel={() => setShowCopyPicker(false)}
          footer={null}
        >
          <Paragraph>
            Select a tournament you&apos;ve directed to copy its settings. This
            will copy the tournament type, description, and game settings (not
            players or pairings).
          </Paragraph>
          {loadingMyTournaments ? (
            <Text type="secondary">Loading your tournaments...</Text>
          ) : myTournaments.length === 0 ? (
            <Text type="secondary">
              No past tournaments found where you are the director.
            </Text>
          ) : (
            <div className="copy-tournament-list">
              {myTournaments.map((t) => (
                <Card
                  key={t.id}
                  size="small"
                  className="copy-tournament-card"
                  hoverable
                  onClick={() => copyFromTournament(t.slug)}
                >
                  <div className="copy-tournament-name">{t.name}</div>
                  <div className="copy-tournament-meta">
                    {t.irlMode ? "IRL" : "Online"}
                    {t.monitored ? " | Monitored" : ""}
                    {t.scheduledStartTime &&
                      ` | ${new Date(Number(t.scheduledStartTime.seconds) * 1000).toLocaleDateString()}`}
                  </div>
                </Card>
              ))}
            </div>
          )}
        </Modal>
      </div>
    </>
  );
};

// Step 0: Tournament Type
const StepTournamentType = ({
  wizardData,
  updateField,
}: {
  wizardData: WizardData;
  updateField: <K extends keyof WizardData>(
    field: K,
    value: WizardData[K],
  ) => void;
}) => {
  const handleModeChange = (e: RadioChangeEvent) => {
    const newMode = e.target.value as WizardData["tournamentMode"];
    updateField("tournamentMode", newMode);
    if (newMode === "irl") {
      updateField("monitored", false);
    }
  };

  return (
    <div className="step-content">
      <Title level={4}>What kind of tournament are you running?</Title>

      <Radio.Group
        value={wizardData.tournamentMode}
        onChange={handleModeChange}
        className="tournament-type-radio"
      >
        <Radio.Button value="online" className="type-option">
          <div className="type-option-content">
            <strong>Online Tournament</strong>
            <p>
              Players compete on the platform. Games are played digitally
              (online, on Woogles.io) with automatic scoring and result
              tracking.
            </p>
          </div>
        </Radio.Button>
        <Radio.Button value="irl" className="type-option">
          <div className="type-option-content">
            <strong>In Real Life (IRL) Tournament</strong>
            <p>
              For over-the-board play with physical boards and tiles. The
              platform handles pairings, standings, and score entry, but the
              games are played OFFLINE. Player usernames do not need to be
              registered on the site. Once IRL mode is enabled, it cannot be
              turned off.
            </p>
          </div>
        </Radio.Button>
      </Radio.Group>

      <Divider />

      <Title level={4}>Monitoring / Invigilation</Title>
      <div className="monitoring-option">
        <Switch
          checked={wizardData.monitored}
          onChange={(checked) => updateField("monitored", checked)}
          disabled={wizardData.tournamentMode === "irl"}
        />
        <div className="monitoring-description">
          <Text strong>Enable monitoring</Text>
          <Paragraph type="secondary">
            Requires participants to share camera and screenshot streams for
            tournament oversight. Recommended for competitive online tournaments
            where fair play verification is important. Uses vdo.ninja for stream
            management.
          </Paragraph>
          {wizardData.tournamentMode === "irl" && (
            <Paragraph type="warning">
              Monitoring is not available for IRL tournaments.
            </Paragraph>
          )}
        </div>
      </div>
    </div>
  );
};

// Step 1: Basic Info
const StepBasicInfo = ({
  wizardData,
  updateField,
  updateName,
  updateSlug,
  timeFormat,
}: {
  wizardData: WizardData;
  updateField: <K extends keyof WizardData>(
    field: K,
    value: WizardData[K],
  ) => void;
  updateName: (name: string) => void;
  updateSlug: (slug: string) => void;
  timeFormat: string;
}) => {
  const endTimeError =
    wizardData.scheduledStartTime &&
    wizardData.scheduledEndTime &&
    !wizardData.scheduledEndTime.isAfter(wizardData.scheduledStartTime)
      ? "End time must be after start time."
      : undefined;

  return (
    <div className="step-content">
      <Title level={4}>Tournament Details</Title>
      <Form layout="vertical">
        <Form.Item
          label="Tournament Name"
          required
          help="The name that will appear in tournament listings."
        >
          <Input
            value={wizardData.name}
            onChange={(e) => updateName(e.target.value)}
            placeholder="e.g., Spring Scrabble Championship 2026"
            maxLength={100}
          />
        </Form.Item>

        <Form.Item
          label="URL Slug (optional)"
          help='A custom URL for your tournament (e.g., "spring-championship-2026"). If left blank, one will be generated from the tournament name.'
        >
          <Input
            value={wizardData.slug}
            onChange={(e) => updateSlug(e.target.value)}
            placeholder="spring-championship-2026"
            addonBefore="/tournament/"
          />
        </Form.Item>

        <Form.Item
          label="Scheduled Start Time"
          help="Use your local time zone. The tournament will still only start when the director does so manually."
        >
          <DatePicker
            showTime={{ format: timeFormat }}
            format={`YYYY-MM-DD ${timeFormat}`}
            value={wizardData.scheduledStartTime}
            onChange={(val) => updateField("scheduledStartTime", val)}
            showNow={false}
            style={{ width: "100%" }}
          />
        </Form.Item>

        <Form.Item
          label="Scheduled End Time"
          validateStatus={endTimeError ? "error" : undefined}
          help={endTimeError}
        >
          <DatePicker
            showTime={{ format: timeFormat }}
            format={`YYYY-MM-DD ${timeFormat}`}
            value={wizardData.scheduledEndTime}
            onChange={(val) => updateField("scheduledEndTime", val)}
            showNow={false}
            style={{ width: "100%" }}
          />
        </Form.Item>

        <Form.Item
          label="Description (Markdown)"
          help="Describe your tournament. Supports Markdown formatting. Include details like entry fees, prizes, rules, schedule, etc."
        >
          <Input.TextArea
            rows={8}
            value={wizardData.description}
            onChange={(e) => updateField("description", e.target.value)}
            placeholder={`## Welcome to the tournament!\n\n**Date:** ...\n**Entry Fee:** ...\n**Prizes:** ...\n\n### Rules\n- ...`}
          />
        </Form.Item>
      </Form>
    </div>
  );
};

// Step 2: Divisions & Game Settings
const StepDivisions = ({
  wizardData,
  updateField,
  settingsModalVisible,
  setSettingsModalVisible,
  onSetGameRequest,
}: {
  wizardData: WizardData;
  updateField: <K extends keyof WizardData>(
    field: K,
    value: WizardData[K],
  ) => void;
  settingsModalVisible: boolean;
  setSettingsModalVisible: (v: boolean) => void;
  onSetGameRequest: (gr: GameRequest) => void;
}) => {
  const addDivision = () => {
    const newDivisions = [
      ...wizardData.divisions,
      {
        name: `Division ${wizardData.divisions.length + 1}`,
        numRounds: 10,
      },
    ];
    updateField("divisions", newDivisions);
  };

  const removeDivision = (idx: number) => {
    if (wizardData.divisions.length <= 1) return;
    const newDivisions = wizardData.divisions.filter((_, i) => i !== idx);
    updateField("divisions", newDivisions);
  };

  const updateDivision = (
    idx: number,
    field: keyof DivisionConfig,
    value: string | number,
  ) => {
    const newDivisions = [...wizardData.divisions];
    newDivisions[idx] = { ...newDivisions[idx], [field]: value };
    updateField("divisions", newDivisions);
  };

  return (
    <div className="step-content">
      <Title level={4}>Divisions</Title>
      <Paragraph type="secondary">
        Set up divisions for your tournament. Most tournaments have a single
        division, but you can add more to separate players by skill level or
        other criteria.
      </Paragraph>

      {wizardData.divisions.map((div, idx) => (
        <Card key={idx} size="small" className="division-card">
          <div className="division-row">
            <Form.Item label="Division Name" style={{ flex: 1 }}>
              <Input
                value={div.name}
                onChange={(e) => updateDivision(idx, "name", e.target.value)}
              />
            </Form.Item>
            {wizardData.divisions.length > 1 && (
              <Button
                danger
                icon={<DeleteOutlined />}
                onClick={() => removeDivision(idx)}
                style={{ alignSelf: "flex-start" }}
              />
            )}
          </div>
        </Card>
      ))}

      <Button
        icon={<PlusOutlined />}
        onClick={addDivision}
        style={{ marginTop: 8 }}
      >
        Add Division
      </Button>

      {wizardData.tournamentMode !== "irl" && (
        <>
          <Divider />

          <Title level={4}>Game Settings</Title>
          <Paragraph type="secondary">
            Configure the default game settings for all divisions. These include
            lexicon, time controls, challenge rules, and whether games are
            rated. You can customize per-division settings later from the
            director tools.
          </Paragraph>

          {wizardData.gameSettingsVaryPerDivision ? (
            <Alert
              type="info"
              message="Game settings vary per division — they have been copied and will be applied per division. You can update them after creation in Director Tools."
              style={{ marginBottom: 12 }}
              showIcon
            />
          ) : (
            DisplayedGameSetting(wizardData.gameRequest)
          )}

          {!wizardData.gameSettingsVaryPerDivision && (
            <Button onClick={() => setSettingsModalVisible(true)}>
              {wizardData.gameRequest
                ? "Change Game Settings"
                : "Configure Game Settings"}
            </Button>
          )}

          <Modal
            title="Game Settings"
            open={settingsModalVisible}
            onCancel={() => setSettingsModalVisible(false)}
            className="seek-modal"
            okButtonProps={{ style: { display: "none" } }}
            destroyOnClose
          >
            <SettingsForm
              setGameRequest={(gr) => {
                onSetGameRequest(gr);
                setSettingsModalVisible(false);
              }}
              gameRequest={wizardData.gameRequest}
            />
          </Modal>
        </>
      )}
    </div>
  );
};

// Step 3: Review
const StepReview = ({ wizardData }: { wizardData: WizardData }) => (
  <div className="step-content">
    <Title level={4}>Review Your Tournament</Title>
    <Paragraph>
      Please review your tournament settings before creating it. You can go back
      to any step to make changes.
    </Paragraph>

    <Card size="small" title="Tournament Type" className="review-card">
      <p>
        <strong>Mode:</strong>{" "}
        {wizardData.tournamentMode === "irl" ? "In Real Life (IRL)" : "Online"}
      </p>
      <p>
        <strong>Monitoring:</strong>{" "}
        {wizardData.monitored ? "Enabled" : "Disabled"}
      </p>
    </Card>

    <Card size="small" title="Basic Info" className="review-card">
      <p>
        <strong>Name:</strong> {wizardData.name || "(not set)"}
      </p>
      {wizardData.slug ? (
        <p>
          <strong>URL:</strong> /tournament/{wizardData.slug}
        </p>
      ) : (
        <p>
          <strong>URL:</strong> auto-generated from tournament name
        </p>
      )}
      {wizardData.scheduledStartTime && (
        <p>
          <strong>Start:</strong>{" "}
          {wizardData.scheduledStartTime.format("YYYY-MM-DD HH:mm")}
        </p>
      )}
      {wizardData.scheduledEndTime && (
        <p>
          <strong>End:</strong>{" "}
          {wizardData.scheduledEndTime.format("YYYY-MM-DD HH:mm")}
        </p>
      )}
      {wizardData.description && (
        <p>
          <strong>Description:</strong> {wizardData.description.slice(0, 100)}
          {wizardData.description.length > 100 ? "..." : ""}
        </p>
      )}
    </Card>

    <Card size="small" title="Divisions" className="review-card">
      {wizardData.divisions.map((div, idx) => (
        <p key={idx}>
          <strong>{div.name}</strong>
          {div.roundControls && div.roundControls.length > 0 && (
            <span style={{ fontWeight: "normal", marginLeft: 8, opacity: 0.7 }}>
              ({div.roundControls.length} round
              {div.roundControls.length !== 1 ? "s" : ""})
            </span>
          )}
        </p>
      ))}
    </Card>

    {wizardData.tournamentMode !== "irl" && (
      <Card size="small" title="Game Settings" className="review-card">
        {wizardData.gameSettingsVaryPerDivision
          ? "Varies per division (copied from source tournament)"
          : DisplayedGameSetting(wizardData.gameRequest)}
      </Card>
    )}

    <Alert
      type="info"
      message="After creation, you can further configure round controls, pairing methods, add players, and manage other settings from the Director Tools in your tournament page."
      showIcon
      style={{ marginTop: 16 }}
    />
  </div>
);
