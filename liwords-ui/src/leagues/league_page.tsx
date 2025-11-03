import React, { useState, useMemo } from "react";
import { Col, Row, Card, Spin, Button, Tabs, Space, Tag, Alert } from "antd";
import { useParams } from "react-router";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { TopBar } from "../navigation/topbar";
import {
  getLeague,
  getCurrentSeason,
  getPastSeasons,
  getAllDivisionStandings,
  registerForSeason,
  unregisterFromSeason,
} from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { DivisionStandings } from "./standings";
import { useLoginStateStoreContext } from "../store/store";
import { flashError } from "../utils/hooks/connect";
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

  // Fetch current season
  const { data: currentSeasonData, isLoading: currentSeasonLoading } = useQuery(
    getCurrentSeason,
    {
      leagueId: slug || "",
    },
    { enabled: !!slug },
  );

  // Fetch past seasons
  const { data: pastSeasonsData, isLoading: pastSeasonsLoading } = useQuery(
    getPastSeasons,
    {
      leagueId: slug || "",
    },
    { enabled: !!slug },
  );

  // Determine which season to display standings for
  const displaySeasonId = useMemo(() => {
    if (selectedSeasonId) return selectedSeasonId;
    return currentSeasonData?.season?.uuid || null;
  }, [selectedSeasonId, currentSeasonData]);

  // Fetch standings for selected season
  const { data: standingsData, isLoading: standingsLoading } = useQuery(
    getAllDivisionStandings,
    {
      seasonId: displaySeasonId || "",
    },
    { enabled: !!displaySeasonId },
  );

  // Data processing - compute before using in queries
  const league = leagueData?.league;
  const currentSeason = currentSeasonData?.season;
  const pastSeasons = useMemo(
    () => pastSeasonsData?.seasons || [],
    [pastSeasonsData?.seasons],
  );

  const allSeasons = useMemo(() => {
    const seasons = [];
    if (currentSeason) {
      seasons.push({ ...currentSeason, isCurrent: true });
    }
    seasons.push(...pastSeasons.map((s) => ({ ...s, isCurrent: false })));
    return seasons;
  }, [currentSeason, pastSeasons]);

  // Find the season that has REGISTRATION_OPEN status from all seasons
  const registrationOpenSeason = useMemo(() => {
    return allSeasons.find((s) => s.status === 4) || null;
  }, [allSeasons]);

  // Fetch standings for registration-open season to check registration status
  const { data: registrationSeasonData } = useQuery(
    getAllDivisionStandings,
    {
      seasonId: registrationOpenSeason?.uuid || "",
    },
    { enabled: !!registrationOpenSeason?.uuid },
  );

  // Check if user is registered for the registration-open season
  const isUserRegistered = useMemo(() => {
    if (
      !registrationOpenSeason ||
      !userID ||
      !registrationSeasonData?.divisions
    )
      return false;

    // Check if user appears in any division's players list
    return registrationSeasonData.divisions.some((division) =>
      division.players?.some((player) => player.userId === userID),
    );
  }, [registrationOpenSeason, userID, registrationSeasonData]);

  // Register/Unregister mutations
  const registerMutation = useMutation(registerForSeason, {
    onSuccess: () => {
      // Invalidate queries to refetch fresh data
      queryClient.invalidateQueries();
      alert("Successfully registered for the season!");
    },
    onError: (error) => {
      flashError(error);
    },
  });

  const unregisterMutation = useMutation(unregisterFromSeason, {
    onSuccess: () => {
      // Invalidate queries to refetch fresh data
      queryClient.invalidateQueries();
      alert("Successfully unregistered from the season!");
    },
    onError: (error) => {
      flashError(error);
    },
  });

  // Handler functions
  const handleRegister = () => {
    if (!slug || !registrationOpenSeason?.uuid) return;
    registerMutation.mutate({
      leagueId: slug,
      userId: userID,
      seasonId: registrationOpenSeason.uuid,
    });
  };

  const handleUnregister = () => {
    if (!registrationOpenSeason?.uuid) return;
    unregisterMutation.mutate({
      seasonId: registrationOpenSeason.uuid,
      userId: userID,
    });
  };

  // Check if registration is open (any season has REGISTRATION_OPEN status)
  const isRegistrationOpen = useMemo(() => {
    return registrationOpenSeason !== null;
  }, [registrationOpenSeason]);

  const isLoading = leagueLoading || currentSeasonLoading || pastSeasonsLoading;

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

        {league.settings && (
          <Card className="league-settings-card">
            <Row gutter={16}>
              <Col span={6}>
                <div className="setting-item">
                  <strong>Lexicon</strong>
                  <div>{league.settings.lexicon}</div>
                </div>
              </Col>
              <Col span={6}>
                <div className="setting-item">
                  <strong>Season Length</strong>
                  <div>{league.settings.seasonLengthDays} days</div>
                </div>
              </Col>
              <Col span={8}>
                <div className="setting-item">
                  <strong>Ideal Division Size</strong>
                  <div>{league.settings.idealDivisionSize} players</div>
                </div>
              </Col>
            </Row>
          </Card>
        )}

        <Card className="season-navigation-card">
          <div className="season-header">
            <h2>Seasons</h2>
            {isRegistrationOpen && loggedIn && (
              <Space>
                {isUserRegistered ? (
                  <>
                    <Tag color="green">
                      Registered for Season{" "}
                      {registrationOpenSeason?.seasonNumber}
                    </Tag>
                    <Button
                      onClick={handleUnregister}
                      loading={unregisterMutation.isPending}
                    >
                      Unregister
                    </Button>
                  </>
                ) : (
                  <Button
                    type="primary"
                    onClick={handleRegister}
                    loading={registerMutation.isPending}
                  >
                    Register for Season {registrationOpenSeason?.seasonNumber}
                  </Button>
                )}
              </Space>
            )}
          </div>

          <Tabs
            activeKey={displaySeasonId || undefined}
            onChange={setSelectedSeasonId}
            items={allSeasons.map((season) => ({
              key: season.uuid,
              label: (
                <span>
                  Season {season.seasonNumber}
                  {season.isCurrent && <Tag color="blue">Current</Tag>}
                </span>
              ),
            }))}
          />
        </Card>

        {standingsLoading && (
          <div className="loading-container">
            <Spin size="large" />
          </div>
        )}

        {!standingsLoading && standingsData && (
          <div className="standings-container">
            <h3>
              Season{" "}
              {allSeasons.find((s) => s.uuid === displaySeasonId)?.seasonNumber}{" "}
              Standings
            </h3>
            {standingsData.divisions.length === 0 && (
              <Alert
                message="No divisions yet"
                description="Divisions will be created when the season starts."
                type="info"
              />
            )}
            {standingsData.divisions.map((division) => (
              <DivisionStandings key={division.uuid} division={division} />
            ))}
          </div>
        )}
      </div>
    </>
  );
};
