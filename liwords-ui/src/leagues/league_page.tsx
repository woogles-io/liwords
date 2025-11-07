import React, { useState, useMemo } from "react";
import { Col, Row, Card, Spin, Button, Select, Space, Tag, Alert } from "antd";
import { useParams } from "react-router";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { TopBar } from "../navigation/topbar";
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
import { useLoginStateStoreContext } from "../store/store";
import { flashError } from "../utils/hooks/connect";
import { UsernameWithContext } from "../shared/usernameWithContext";
import "./leagues.scss";

export const LeaguePage = () => {
  const { slug } = useParams<{ slug: string }>();
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn, userID } = loginState;
  const queryClient = useQueryClient();

  const [selectedSeasonId, setSelectedSeasonId] = useState<string | null>(null);

  // Fetch league data
  const { data: leagueData, isLoading: leagueLoading } = useQuery(
    getLeague,
    {
      leagueId: slug || "",
    },
    { enabled: !!slug },
  );

  // Fetch all seasons (regardless of status)
  const { data: allSeasonsData, isLoading: allSeasonsLoading } = useQuery(
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

  // Find the most recent SCHEDULED season (status = 0)
  const scheduledSeason = useMemo(() => {
    const scheduled = allSeasons.filter((s) => s.status === 0);
    return scheduled.length > 0 ? scheduled[scheduled.length - 1] : null;
  }, [allSeasons]);

  // Get the displayed season object
  const displayedSeason = useMemo(() => {
    return allSeasons.find((s) => s.uuid === displaySeasonId) || null;
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
      alert("Successfully registered for the season!");
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
      alert("Successfully unregistered from the season!");
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
      alert(`Successfully opened registration for Season ${seasonNumber}!`);
    },
    onError: (error) => {
      flashError(error);
    },
  });

  // Handler functions
  const handleRegister = () => {
    if (!slug || !displayedSeason?.uuid) return;
    registerMutation.mutate({
      leagueId: slug,
      userId: userID,
      seasonId: displayedSeason.uuid,
    });
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

  const isLoading = leagueLoading || allSeasonsLoading;

  if (isLoading) {
    return (
      <>
        <Row>
          <Col span={24}>
            <TopBar />
          </Col>
        </Row>
        <div className="loading-container">
          <Spin size="large" />
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
        <div className="league-header">
          <h1>{league.name}</h1>
          <p>{league.description}</p>
        </div>

        <Row gutter={16}>
          {/* Left Column - League Info & Registrants */}
          <Col xs={24} lg={6}>
            {league.settings && (
              <Card
                className="league-settings-card"
                style={{ marginBottom: 16 }}
              >
                <h3>League Settings</h3>
                <div className="setting-item">
                  <strong>Lexicon</strong>
                  <div>{league.settings.lexicon}</div>
                </div>
                <div className="setting-item" style={{ marginTop: 12 }}>
                  <strong>Season Length</strong>
                  <div>{league.settings.seasonLengthDays} days</div>
                </div>
                <div className="setting-item" style={{ marginTop: 12 }}>
                  <strong>Ideal Division Size</strong>
                  <div>{league.settings.idealDivisionSize} players</div>
                </div>
              </Card>
            )}

            {/* Show registrants for seasons without divisions (SCHEDULED, REGISTRATION_OPEN) */}
            {!standingsLoading &&
              displayedSeason &&
              registrants.length > 0 &&
              standingsData?.divisions.length === 0 && (
                <Card className="registrants-card">
                  <h3>Registrants</h3>
                  <p style={{ marginBottom: 12 }}>
                    {registrants.length}{" "}
                    {registrants.length === 1 ? "player" : "players"} registered
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
                </Card>
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
              </div>
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
                  <div className="standings-legend">
                    <p>
                      <strong>Promotion/Relegation Zones:</strong> Green
                      highlighted rows indicate players in the promotion zone
                      (top{" "}
                      {Math.ceil(
                        (standingsData.divisions[0]?.standings.length || 0) / 6,
                      )}{" "}
                      players per division will move up). Red highlighted rows
                      indicate relegation zones (bottom players will move down).
                    </p>
                    <p>
                      <strong>Note:</strong> If additional divisions are added
                      in future seasons to maintain balance, some players from
                      the bottom division may be relegated to accommodate the
                      new structure.
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
                  {standingsData.divisions.map((division) => (
                    <DivisionStandings
                      key={division.uuid}
                      division={division}
                      totalDivisions={standingsData.divisions.length}
                      seasonId={displaySeasonId || ""}
                      currentUserId={userID}
                    />
                  ))}
                </div>
              )}
          </Col>

          {/* Right Column - Future Stats */}
          <Col xs={24} lg={6}>
            {/* Reserved for future stats */}
          </Col>
        </Row>
      </div>
    </>
  );
};
