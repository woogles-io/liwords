import React, { useState, useMemo, useEffect } from "react";
import {
  Col,
  Row,
  Card,
  Spin,
  Button,
  Select,
  Space,
  Tag,
  Alert,
  notification,
  Modal,
  Checkbox,
} from "antd";
import { ArrowLeftOutlined, TrophyOutlined } from "@ant-design/icons";
import { useParams, Link } from "react-router";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import moment from "moment";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { TopBar } from "../navigation/topbar";
import { Chat } from "../chat/chat";
import {
  getLeague,
  getAllSeasons,
  getAllDivisionStandings,
  getSeasonRegistrations,
  registerForSeason,
  unregisterFromSeason,
  openRegistration,
} from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { getSelfRoles } from "../gen/api/proto/user_service/user_service-AuthorizationService_connectquery";
import { DivisionStandings } from "./standings";
import { LeagueCorrespondenceGames } from "./league_correspondence_games";
import { DivisionSelector, getDefaultDivisionId } from "./division_selector";
import { useLoginStateStoreContext } from "../store/store";
import { flashError } from "../utils/hooks/connect";
import { UsernameWithContext } from "../shared/usernameWithContext";
import "./leagues.scss";

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
};

// Helper function to format season dates in local time
const formatSeasonDates = (
  startDate: Timestamp | undefined,
  endDate: Timestamp | undefined,
): string => {
  if (!startDate || !endDate) return "";

  // Convert protobuf Timestamp to Date
  const startDateObj = timestampDate(startDate);
  const endDateObj = timestampDate(endDate);

  // Format in user's local timezone (moment uses local timezone by default)
  const start = moment(startDateObj);
  const end = moment(endDateObj);

  // Format: "Jan 15, 2025 3:00 PM - Feb 28, 2025 11:59 PM"
  return `${start.format("MMM D, YYYY h:mm A")} - ${end.format("MMM D, YYYY h:mm A")}`;
};

const formatLocalTime = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return "TBD";

  const date = timestampDate(timestamp);

  // Format in user's local timezone
  const localTime = date.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
    timeZoneName: "long",
  });

  return localTime;
};

export const LeaguePage = (props: Props) => {
  const { slug } = useParams<{ slug: string }>();
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn, userID } = loginState;
  const queryClient = useQueryClient();

  const [selectedSeasonId, setSelectedSeasonId] = useState<string | null>(null);
  const [selectedDivisionId, setSelectedDivisionId] = useState<string>("");
  const [showPlayersModal, setShowPlayersModal] = useState<boolean>(false);
  const [showCommitmentModal, setShowCommitmentModal] =
    useState<boolean>(false);
  const [hasAgreedToCommitment, setHasAgreedToCommitment] =
    useState<boolean>(false);

  // Fetch league data
  const { data: leagueData, isPending: leaguePending } = useQuery(
    getLeague,
    {
      leagueId: slug || "",
    },
    { enabled: !!slug },
  );

  // Fetch all seasons (regardless of status)
  const { data: allSeasonsData, isPending: allSeasonsPending } = useQuery(
    getAllSeasons,
    {
      leagueId: slug || "",
    },
    { enabled: !!slug },
  );

  // Fetch user roles for admin checks
  const { data: selfRoles } = useQuery(getSelfRoles, {}, { enabled: loggedIn });

  // Data processing - get all seasons and find the active one
  const league = leagueData?.league;
  const allSeasons = useMemo(() => {
    const seasons = allSeasonsData?.seasons || [];
    // Sort by season number descending (newest first)
    return [...seasons].sort(
      (a, b) => (b.seasonNumber || 0) - (a.seasonNumber || 0),
    );
  }, [allSeasonsData?.seasons]);

  // Find the current season (status = ACTIVE = 1)
  const currentSeason = useMemo(() => {
    return allSeasons.find((s) => s.status === 1) || null;
  }, [allSeasons]);

  // Determine which season to display standings for
  const displaySeasonId = useMemo(() => {
    if (selectedSeasonId) return selectedSeasonId;
    // Default to current season if available, otherwise first season in list
    return currentSeason?.uuid || allSeasons[0]?.uuid || null;
  }, [selectedSeasonId, currentSeason, allSeasons]);

  // Fetch standings for selected season
  const { data: standingsData, isLoading: standingsLoading } = useQuery(
    getAllDivisionStandings,
    {
      seasonId: displaySeasonId || "",
    },
    { enabled: !!displaySeasonId },
  );

  // Set default selected division when standings data changes
  useEffect(() => {
    if (standingsData?.divisions && standingsData.divisions.length > 0) {
      const defaultDivId = getDefaultDivisionId(
        standingsData.divisions,
        userID,
      );
      setSelectedDivisionId(defaultDivId);
    }
  }, [standingsData, userID]);

  // Fetch registrations for selected season
  const { data: registrationsData } = useQuery(
    getSeasonRegistrations,
    {
      seasonId: displaySeasonId || "",
    },
    { enabled: !!displaySeasonId },
  );

  // Find the season that has REGISTRATION_OPEN status from all seasons
  const registrationOpenSeason = useMemo(() => {
    return allSeasons.find((s) => s.status === 4) || null;
  }, [allSeasons]);

  // Fetch registrations for the registration-open season (to check if user is registered)
  const { data: registrationOpenSeasonRegistrations } = useQuery(
    getSeasonRegistrations,
    {
      seasonId: registrationOpenSeason?.uuid || "",
    },
    { enabled: !!registrationOpenSeason?.uuid },
  );

  // Find the most recent SCHEDULED season (status = 0)
  const scheduledSeason = useMemo(() => {
    const scheduled = allSeasons.filter((s) => s.status === 0);
    return scheduled.length > 0 ? scheduled[scheduled.length - 1] : null;
  }, [allSeasons]);

  // Get the displayed season object
  const displayedSeason = useMemo(() => {
    const season = allSeasons.find((s) => s.uuid === displaySeasonId) || null;
    console.log("Displayed Season:", {
      season,
      hasStartDate: !!season?.startDate,
      hasEndDate: !!season?.endDate,
      startDate: season?.startDate,
      endDate: season?.endDate,
    });
    return season;
  }, [allSeasons, displaySeasonId]);

  // Check if user is registered for the displayed season
  const isUserRegistered = useMemo(() => {
    if (!displayedSeason || !userID || !registrationsData?.registrations) {
      return false;
    }

    // Check if user appears in registrations list
    return registrationsData.registrations.some((reg) => reg.userId === userID);
  }, [displayedSeason, userID, registrationsData]);

  // Get all registrants for the displayed season
  const registrants = useMemo(() => {
    if (!registrationsData?.registrations) return [];

    return registrationsData.registrations.map((reg) => ({
      userId: reg.userId || "",
      username: reg.username || "",
      divisionNumber: reg.divisionNumber || 0,
    }));
  }, [registrationsData]);

  // Check if user can manage leagues (Admin or Manager role)
  const canManageLeagues = useMemo(() => {
    return !!(
      selfRoles?.roles.includes("Admin") || selfRoles?.roles.includes("Manager")
    );
  }, [selfRoles?.roles]);

  // Register/Unregister mutations
  const registerMutation = useMutation(registerForSeason, {
    onSuccess: async () => {
      // Refetch queries with proper Connect Query syntax
      await queryClient.refetchQueries({
        queryKey: ["connect-query", { methodName: "GetSeasonRegistrations" }],
      });
      await queryClient.refetchQueries({
        queryKey: ["connect-query", { methodName: "GetAllSeasons" }],
      });
      notification.success({
        message: "Registration Successful",
        description: "You have been successfully registered for the season!",
      });
    },
    onError: (error) => {
      flashError(error);
    },
  });

  const unregisterMutation = useMutation(unregisterFromSeason, {
    onSuccess: async () => {
      // Refetch queries with proper Connect Query syntax
      await queryClient.refetchQueries({
        queryKey: ["connect-query", { methodName: "GetSeasonRegistrations" }],
      });
      notification.success({
        message: "Unregistration Successful",
        description: "You have been successfully unregistered from the season.",
      });
    },
    onError: (error) => {
      flashError(error);
    },
  });

  // Admin mutation to open registration
  const openRegistrationMutation = useMutation(openRegistration, {
    onSuccess: async (response) => {
      // Refetch queries with proper Connect Query syntax
      await queryClient.refetchQueries({
        queryKey: ["connect-query", { methodName: "GetAllSeasons" }],
      });
      const seasonNumber = response.season?.seasonNumber || "next";
      notification.success({
        message: "Registration Opened",
        description: `Registration has been opened for Season ${seasonNumber}.`,
      });
    },
    onError: (error) => {
      flashError(error);
    },
  });

  // Handler functions
  const handleRegister = () => {
    // Show commitment modal before registering
    setShowCommitmentModal(true);
    setHasAgreedToCommitment(false);
  };

  const handleConfirmRegistration = () => {
    if (!slug || !displayedSeason?.uuid || !hasAgreedToCommitment) return;
    registerMutation.mutate({
      leagueId: slug,
      userId: userID,
      seasonId: displayedSeason.uuid,
    });
    setShowCommitmentModal(false);
    setHasAgreedToCommitment(false);
  };

  const handleUnregister = () => {
    if (!displayedSeason?.uuid) return;
    unregisterMutation.mutate({
      seasonId: displayedSeason.uuid,
      userId: userID,
    });
  };

  const handleOpenRegistration = () => {
    if (!slug || !scheduledSeason) return;
    openRegistrationMutation.mutate({
      leagueId: slug,
      seasonId: scheduledSeason.uuid,
    });
  };

  // Check if registration is open (any season has REGISTRATION_OPEN status)
  const isRegistrationOpen = useMemo(() => {
    return registrationOpenSeason !== null;
  }, [registrationOpenSeason]);

  const isLoading = leaguePending || allSeasonsPending;

  if (isLoading) {
    return (
      <>
        <Row>
          <Col span={24}>
            <TopBar />
          </Col>
        </Row>
        <div className="leagues-container">
          <div className="loading-container">
            <Spin size="large" />
          </div>
        </div>
      </>
    );
  }

  if (!league) {
    return (
      <>
        <Row>
          <Col span={24}>
            <TopBar />
          </Col>
        </Row>
        <div className="leagues-container">
          <Alert
            message="League Not Found"
            description="The league you are looking for does not exist."
            type="error"
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
        <Link to="/leagues" className="back-to-leagues-link">
          <ArrowLeftOutlined style={{ marginRight: 8 }} />
          Back to Leagues
        </Link>

        <Row gutter={16}>
          {/* Left Column - Chat Only */}
          <Col xs={24} lg={6}>
            {/* League Chat */}
            {loggedIn && league && (
              <div className="chat-area">
                <Chat
                  sendChat={props.sendChat}
                  defaultChannel={`chat.league.${league.uuid.replace(/-/g, "")}`}
                  defaultDescription={`League Chat: ${league.name}`}
                  leagueID={league.uuid}
                />
              </div>
            )}
          </Col>

          {/* Center Column - Seasons & Standings */}
          <Col xs={24} lg={12}>
            <Card className="season-navigation-card">
              <div className="season-header">
                <h2>Seasons</h2>
                {/* Admin button to open registration - show when there's a SCHEDULED season */}
                {canManageLeagues &&
                  loggedIn &&
                  !isRegistrationOpen &&
                  scheduledSeason && (
                    <Button
                      type="default"
                      onClick={handleOpenRegistration}
                      loading={openRegistrationMutation.isPending}
                    >
                      Open Registration
                    </Button>
                  )}
              </div>

              {/* Player registration status/buttons for displayed season */}
              {loggedIn && displayedSeason && (
                <div style={{ marginBottom: 16 }}>
                  {isUserRegistered ? (
                    <Space>
                      <Tag color="green">
                        Registered for Season {displayedSeason.seasonNumber}
                      </Tag>
                      {/* Only allow unregister if season is REGISTRATION_OPEN */}
                      {displayedSeason.status === 4 && (
                        <Button
                          onClick={handleUnregister}
                          loading={unregisterMutation.isPending}
                        >
                          Unregister
                        </Button>
                      )}
                    </Space>
                  ) : (
                    <>
                      {/* Only allow registration if season is REGISTRATION_OPEN */}
                      {displayedSeason.status === 4 && (
                        <Button
                          type="primary"
                          onClick={handleRegister}
                          loading={registerMutation.isPending}
                        >
                          Register for Season {displayedSeason.seasonNumber}
                        </Button>
                      )}
                    </>
                  )}
                </div>
              )}

              <div style={{ marginBottom: 16 }}>
                <label style={{ marginRight: 8, fontWeight: 500 }}>
                  Select Season:
                </label>
                <Select
                  value={displaySeasonId || undefined}
                  onChange={setSelectedSeasonId}
                  style={{ width: 350 }}
                  popupMatchSelectWidth={true}
                  options={allSeasons.map((season) => {
                    // Determine status label
                    let statusLabel = "";
                    let statusColor = "";
                    if (season.status === 0) {
                      statusLabel = "Scheduled";
                      statusColor = "";
                    } else if (season.status === 1) {
                      statusLabel = "Active";
                      statusColor = "blue";
                    } else if (season.status === 2) {
                      statusLabel = "Completed";
                      statusColor = "default";
                    } else if (season.status === 3) {
                      statusLabel = "Cancelled";
                      statusColor = "red";
                    } else if (season.status === 4) {
                      statusLabel = "Registration Open";
                      statusColor = "green";
                    }

                    return {
                      value: season.uuid,
                      label: (
                        <span
                          style={{
                            display: "flex",
                            justifyContent: "space-between",
                            alignItems: "center",
                          }}
                        >
                          <span>Season {season.seasonNumber}</span>
                          {statusLabel && (
                            <Tag color={statusColor}>{statusLabel}</Tag>
                          )}
                        </span>
                      ),
                    };
                  })}
                />
                {/* Show dates for selected season */}
                {displayedSeason &&
                  displayedSeason.startDate &&
                  displayedSeason.endDate && (
                    <div
                      className="season-dates-selector"
                      style={{ marginTop: 8, fontSize: "13px", color: "#666" }}
                    >
                      {formatSeasonDates(
                        displayedSeason.startDate,
                        displayedSeason.endDate,
                      )}
                    </div>
                  )}

                {/* Player count display */}
                {displayedSeason && registrants.length > 0 && (
                  <div style={{ marginTop: 8 }}>
                    <span
                      className="clickable-link"
                      onClick={() => setShowPlayersModal(true)}
                    >
                      ({registrants.length}{" "}
                      {registrants.length === 1 ? "player" : "players"}
                      {displayedSeason.status === 4 ||
                      displayedSeason.status === 0
                        ? " registered"
                        : ""}
                      )
                    </span>
                  </div>
                )}
              </div>

              {/* Registration reminder banner - show when viewing a different season while registration is open */}
              {registrationOpenSeason &&
                displaySeasonId !== registrationOpenSeason.uuid &&
                (() => {
                  // Check if user is registered for the registration-open season
                  const isRegisteredForNewSeason =
                    registrationOpenSeasonRegistrations?.registrations.some(
                      (reg) => reg.userId === userID,
                    );

                  return (
                    <Alert
                      message={
                        isRegisteredForNewSeason
                          ? `You're registered for Season ${registrationOpenSeason.seasonNumber}!`
                          : `Registration is now open for Season ${registrationOpenSeason.seasonNumber}!`
                      }
                      description={
                        <div>
                          <p style={{ marginBottom: 8 }}>
                            {isRegisteredForNewSeason
                              ? "Season starts on "
                              : "Registration closes on "}
                            {registrationOpenSeason.startDate &&
                              moment(
                                timestampDate(registrationOpenSeason.startDate),
                              ).format("MMMM D, YYYY [at] h:mm A")}
                          </p>
                          <Button
                            type="primary"
                            size="small"
                            onClick={() =>
                              setSelectedSeasonId(registrationOpenSeason.uuid)
                            }
                          >
                            {isRegisteredForNewSeason
                              ? `View Season ${registrationOpenSeason.seasonNumber}`
                              : `View & Register for Season ${registrationOpenSeason.seasonNumber}`}
                          </Button>
                        </div>
                      }
                      type={isRegisteredForNewSeason ? "success" : "info"}
                      showIcon
                      style={{ marginTop: 16 }}
                    />
                  );
                })()}

              {/* Help banner when viewing registration-open season but not registered */}
              {displayedSeason?.status === 4 &&
                !isUserRegistered &&
                loggedIn && (
                  <Alert
                    message="Registration Instructions"
                    description={
                      <div>
                        To register for this season, please click the 'Register
                        for Season' button above.&nbsp;
                        <strong>Note:</strong> Please only register if you are
                        willing to play all league games without forfeits!
                      </div>
                    }
                    type="info"
                    showIcon
                    style={{ marginTop: 16 }}
                  />
                )}
            </Card>

            {standingsLoading && (
              <div className="loading-container" style={{ marginTop: 16 }}>
                <Spin size="large" />
              </div>
            )}

            {!standingsLoading &&
              standingsData &&
              standingsData.divisions.length === 0 && (
                <Alert
                  message="No divisions yet"
                  description="Divisions will be created when the season starts."
                  type="info"
                  style={{ marginTop: 16 }}
                />
              )}

            {!standingsLoading &&
              standingsData &&
              standingsData.divisions.length > 0 && (
                <div className="standings-container" style={{ marginTop: 16 }}>
                  {/* Division Selector */}
                  <DivisionSelector
                    divisions={standingsData.divisions}
                    selectedDivisionId={selectedDivisionId}
                    onDivisionChange={setSelectedDivisionId}
                    currentUserId={userID}
                  />

                  {/* Show champion banner for completed seasons */}
                  {displayedSeason?.status === 2 &&
                    standingsData.divisions.length > 0 &&
                    standingsData.divisions[0].standings.length > 0 && (
                      <div className="season-champion-banner">
                        <div className="champion-content">
                          <TrophyOutlined className="trophy-icon" />
                          <div className="champion-text">
                            <h3>
                              Season {displayedSeason.seasonNumber} Champion
                            </h3>
                            <p className="champion-name">
                              Congratulations to{" "}
                              {standingsData.divisions[0].standings[0].username}
                              !
                            </p>
                          </div>
                        </div>
                      </div>
                    )}

                  {/* Show only selected division */}
                  {standingsData.divisions
                    .filter((division) => division.uuid === selectedDivisionId)
                    .map((division) => (
                      <DivisionStandings
                        key={division.uuid}
                        division={division}
                        totalDivisions={standingsData.divisions.length}
                        seasonId={displaySeasonId || ""}
                        seasonNumber={displayedSeason?.seasonNumber || 0}
                        currentUserId={userID}
                      />
                    ))}
                  <div className="standings-legend">
                    <p>
                      <strong>Promotion/Relegation Zones:</strong> Green
                      highlighted rows indicate players in the promotion zone
                      (top players in each division that will move up to the
                      next division). Red rows indicate relegation zones (bottom
                      players will move down). The number of players in the
                      promotion and relegation zones is determined by the size
                      of the division.
                    </p>
                    <p>
                      <strong>Note:</strong> If additional divisions are added
                      in future seasons to maintain balance, some players from
                      the bottom division may be relegated to accommodate the
                      new structure. These relegation and promotion zones are
                      not always 100% set in stone, depending on how many
                      players join or leave season to season.
                    </p>
                    <div style={{ marginTop: 12 }}>
                      <span className="legend-item">
                        <span className="promotion-indicator"></span>
                        Promotion Zone
                      </span>
                      <span className="legend-item">
                        <span className="relegation-indicator"></span>
                        Relegation Zone
                      </span>
                    </div>
                  </div>
                </div>
              )}
          </Col>

          {/* Right Column - Settings & Games */}
          <Col xs={24} lg={6}>
            {/* League Info & Settings Card */}
            {league && (
              <Card className="league-info-card" style={{ marginBottom: 16 }}>
                {/* League Name & Description */}
                <h3
                  style={{
                    fontSize: "16px",
                    margin: 0,
                    marginBottom: 6,
                    fontWeight: 600,
                  }}
                >
                  {league.name}
                </h3>
                <p
                  style={{
                    fontSize: "12px",
                    margin: 0,
                    marginBottom: 12,
                    color: "#666",
                    lineHeight: "1.4",
                  }}
                >
                  {league.description}
                </p>

                {/* Current Season Info */}
                {displayedSeason &&
                  displayedSeason.startDate &&
                  displayedSeason.endDate && (
                    <div className="season-info-compact">
                      <strong>
                        {displayedSeason.status === 1
                          ? "Current Season"
                          : `Season ${displayedSeason.seasonNumber}`}
                        :
                      </strong>{" "}
                      {formatSeasonDates(
                        displayedSeason.startDate,
                        displayedSeason.endDate,
                      )}
                    </div>
                  )}

                {/* Settings Table */}
                {league.settings && (
                  <div className="settings-table">
                    <div className="settings-row">
                      <span className="settings-label">Lexicon:</span>
                      <span className="settings-value">
                        {league.settings.lexicon}
                      </span>
                    </div>
                    <div className="settings-row">
                      <span className="settings-label">Challenge:</span>
                      <span className="settings-value">
                        {league.settings.challengeRule === 5
                          ? "Triple"
                          : league.settings.challengeRule === 4
                            ? "10pt"
                            : league.settings.challengeRule === 3
                              ? "5pt"
                              : league.settings.challengeRule === 2
                                ? "Double"
                                : league.settings.challengeRule === 1
                                  ? "Single"
                                  : "Void"}
                      </span>
                    </div>
                    <div className="settings-row">
                      <span className="settings-label">Time Control:</span>
                      <span className="settings-value">
                        {league.settings.timeControl?.incrementSeconds
                          ? `${league.settings.timeControl.incrementSeconds / 3600}h/turn`
                          : "None"}
                      </span>
                    </div>
                    <div className="settings-row">
                      <span className="settings-label">Time Bank:</span>
                      <span className="settings-value">
                        {league.settings.timeControl?.timeBankMinutes
                          ? `${league.settings.timeControl.timeBankMinutes / 60}h`
                          : "None"}
                      </span>
                    </div>
                    <div className="settings-row">
                      <span className="settings-label">Season Length:</span>
                      <span className="settings-value">
                        {league.settings.seasonLengthDays} days
                      </span>
                    </div>
                    <div className="settings-row">
                      <span className="settings-label">Division Size:</span>
                      <span className="settings-value">
                        {league.settings.idealDivisionSize} players
                      </span>
                    </div>
                  </div>
                )}
              </Card>
            )}

            {/* League Games */}
            {loggedIn && slug && (
              <LeagueCorrespondenceGames leagueSlug={slug} />
            )}
          </Col>
        </Row>

        {/* Players List Modal */}
        <Modal
          title={`Season ${displayedSeason?.seasonNumber} Players`}
          open={showPlayersModal}
          onCancel={() => setShowPlayersModal(false)}
          footer={null}
          width={500}
          zIndex={2000}
        >
          <div style={{ maxHeight: "400px", overflowY: "auto" }}>
            <p style={{ marginBottom: 12 }}>
              {registrants.length}{" "}
              {registrants.length === 1 ? "player" : "players"}
              {displayedSeason?.status === 4 || displayedSeason?.status === 0
                ? " registered"
                : ""}
            </p>
            <div
              style={{
                display: "flex",
                flexWrap: "wrap",
                gap: "8px 16px",
              }}
            >
              {registrants.map((registrant) => (
                <UsernameWithContext
                  key={registrant.userId}
                  username={registrant.username}
                  userID={registrant.userId}
                />
              ))}
            </div>
          </div>
        </Modal>

        {/* Registration Commitment Modal */}
        <Modal
          title="League Registration Commitment"
          open={showCommitmentModal}
          onCancel={() => {
            setShowCommitmentModal(false);
            setHasAgreedToCommitment(false);
          }}
          footer={[
            <Button
              key="cancel"
              onClick={() => {
                setShowCommitmentModal(false);
                setHasAgreedToCommitment(false);
              }}
            >
              Cancel
            </Button>,
            <Button
              key="register"
              type="primary"
              disabled={!hasAgreedToCommitment}
              onClick={handleConfirmRegistration}
            >
              Register
            </Button>,
          ]}
          width={600}
          zIndex={2000}
        >
          <div style={{ lineHeight: 1.8 }}>
            <p style={{ marginBottom: 16, fontSize: "15px" }}>
              Welcome to {league?.name}! Before registering for Season{" "}
              {displayedSeason?.seasonNumber}, please read and agree to the
              following commitment:
            </p>

            <div className="registration-commitment-box">
              <p style={{ marginBottom: 12, fontWeight: 500 }}>
                In order to keep the league fun and fair for everyone, I commit
                to:
              </p>
              <ul style={{ marginBottom: 0, paddingLeft: 20 }}>
                <li>
                  Checking the app regularly to avoid forfeiting games on time
                </li>
                <li>Playing fairly without external assistance</li>
                <li>Completing all my games for the season</li>
                <li>Having fun</li>
              </ul>
            </div>

            <p style={{ marginBottom: 8, fontWeight: 500 }}>
              Season {displayedSeason?.seasonNumber} starts at:
            </p>
            <div className="registration-time-box">
              {formatLocalTime(displayedSeason?.startDate)}
            </div>

            <Checkbox
              checked={hasAgreedToCommitment}
              onChange={(e) => setHasAgreedToCommitment(e.target.checked)}
              style={{ fontSize: "15px" }}
            >
              <strong>
                I understand and commit to playing my games promptly and fairly
              </strong>
            </Checkbox>
          </div>
        </Modal>
      </div>
    </>
  );
};
