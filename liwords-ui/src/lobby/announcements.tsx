import React, { useEffect, useState } from "react";
import { Card, Tabs, Tooltip } from "antd";
import ReactMarkdown from "react-markdown";
import { Link } from "react-router";
import {
  GlobalOutlined,
  DesktopOutlined,
  TeamOutlined,
  TrophyOutlined,
} from "@ant-design/icons";
import { Announcement } from "../gen/api/proto/config_service/config_service_pb";
import { useClient } from "../utils/hooks/connect";
import { ConfigService } from "../gen/api/proto/config_service/config_service_pb";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import type { TournamentMetadata } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { LeagueService } from "../gen/api/proto/league_service/league_service_pb";
import type { League, Season } from "../gen/api/proto/ipc/league_pb";
import "./upcoming_tournaments.scss";

export type Announcements = {
  announcements: Array<Announcement>;
};

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMs < 0) {
    const absSecs = Math.abs(diffSecs);
    const absMins = Math.abs(diffMins);
    const absHours = Math.abs(diffHours);
    const absDays = Math.abs(diffDays);

    if (absSecs < 60) return "just now";
    if (absMins < 60) return `${absMins} min ago`;
    if (absHours < 24) return `${absHours} hour${absHours > 1 ? "s" : ""} ago`;
    return `${absDays} day${absDays > 1 ? "s" : ""} ago`;
  }

  if (diffSecs < 60) return "starting soon";
  if (diffMins < 60) return `in ${diffMins} min`;
  if (diffHours < 24) return `in ${diffHours} hour${diffHours > 1 ? "s" : ""}`;
  return `in ${diffDays} day${diffDays > 1 ? "s" : ""}`;
};

const formatLocalDate = (date: Date): string => {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(date);
};

const formatLocalTime = (date: Date): string => {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(date);
};

const TournamentCard = ({ tournament }: { tournament: TournamentMetadata }) => {
  const startDate = tournament.scheduledStartTime
    ? new Date(Number(tournament.scheduledStartTime.seconds) * 1000)
    : null;
  const endDate = tournament.scheduledEndTime
    ? new Date(Number(tournament.scheduledEndTime.seconds) * 1000)
    : null;

  const now = new Date();
  const isOngoing = startDate && endDate && now >= startDate && now <= endDate;
  const isUpcoming = startDate && now < startDate;

  return (
    <div className="tournament-card">
      <div className="tournament-header">
        <Link to={tournament.slug} className="tournament-name">
          {tournament.name}
          {tournament.irlMode && (
            <Tooltip title="In Real Life Tournament">
              <GlobalOutlined className="tournament-icon" />
            </Tooltip>
          )}
          {tournament.monitored && (
            <Tooltip title="Monitored Tournament">
              <DesktopOutlined className="tournament-icon" />
            </Tooltip>
          )}
        </Link>
        {tournament.firstDirector && (
          <span className="tournament-director">
            {tournament.firstDirector}
          </span>
        )}
      </div>

      <div className="tournament-details">
        <div className="tournament-meta">
          <span className="tournament-registrants">
            <TeamOutlined className="registrants-icon" />
            {tournament.registrantCount}
          </span>
          {isUpcoming && (
            <Tooltip
              title={`Registrations are ${tournament.registrationOpen ? "open" : "closed"}`}
            >
              <span
                className={`registration-badge ${tournament.registrationOpen ? "open" : "closed"}`}
              >
                {tournament.registrationOpen ? "Open" : "Closed"}
              </span>
            </Tooltip>
          )}
          {isOngoing && <span className="registration-badge live">Live</span>}
        </div>

        <div className="tournament-time">
          {isOngoing && startDate && (
            <>
              <strong>Started:</strong> {formatRelativeTime(startDate)} •{" "}
              {formatLocalTime(startDate)}
            </>
          )}
          {isUpcoming && startDate && (
            <>
              <strong>Starts:</strong> {formatRelativeTime(startDate)} •{" "}
              {formatLocalTime(startDate)}
            </>
          )}
          {!isOngoing && !isUpcoming && endDate && (
            <>
              <strong>Ended:</strong> {formatRelativeTime(endDate)} •{" "}
              {formatLocalTime(endDate)}
            </>
          )}
        </div>
      </div>
    </div>
  );
};

interface LeagueWithSeason {
  league: League;
  currentSeason?: Season;
  champion?: string;
  previousSeasonChampion?: {
    seasonNumber: number;
    username: string;
  };
}

const LeagueStatusCard = ({ leagueData }: { leagueData: LeagueWithSeason }) => {
  const { league, currentSeason, champion, previousSeasonChampion } = leagueData;

  const getStatusMessage = () => {
    if (!currentSeason) return null;

    // Check for registration open
    if (currentSeason.status === 4) {
      // SEASON_REGISTRATION_OPEN
      const startDate = currentSeason.startDate
        ? new Date(Number(currentSeason.startDate.seconds) * 1000)
        : null;
      return {
        type: "registration" as const,
        message: `Season ${currentSeason.seasonNumber} registration is open`,
        date: startDate,
        dateLabel: "Starts",
      };
    }

    // Check for active season
    if (currentSeason.status === 1) {
      // SEASON_ACTIVE
      const endDate = currentSeason.endDate
        ? new Date(Number(currentSeason.endDate.seconds) * 1000)
        : null;
      return {
        type: "live" as const,
        message: `Season ${currentSeason.seasonNumber} is live!`,
        date: endDate,
        dateLabel: "Ends",
      };
    }

    // Check for completed season with champion
    if (currentSeason.status === 2 && champion) {
      // SEASON_COMPLETED
      return {
        type: "completed" as const,
        message: `Season ${currentSeason.seasonNumber} has ended!`,
        champion,
      };
    }

    return null;
  };

  const status = getStatusMessage();
  if (!status) return null;

  return (
    <div className="league-status-card">
      <Link to={`/league/${league.slug}`} className="league-name">
        <TrophyOutlined className="league-icon" />
        {league.name}
      </Link>
      <div className="league-status-details">
        <span className={`status-badge ${status.type}`}>
          {status.type === "live" && "Live"}
          {status.type === "registration" && "Registration Open"}
          {status.type === "completed" && "Completed"}
        </span>
        <span className="status-message">{status.message}</span>
        {status.type !== "completed" && status.date && (
          <span className="status-date">
            <strong>{status.dateLabel}:</strong> {formatLocalDate(status.date)}
          </span>
        )}
        {status.type === "completed" && status.champion && (
          <span className="status-champion">
            Champion: <strong>{status.champion}</strong>
          </span>
        )}
        {previousSeasonChampion && status.type === "live" && (
          <span className="status-previous-champion">
            Season {previousSeasonChampion.seasonNumber} champion:{" "}
            <strong>{previousSeasonChampion.username}</strong>
          </span>
        )}
      </div>
    </div>
  );
};

const TournamentsAndLeaguesContent = () => {
  const [upcomingTournaments, setUpcomingTournaments] = useState<
    Array<TournamentMetadata>
  >([]);
  const [pastTournaments, setPastTournaments] = useState<
    Array<TournamentMetadata>
  >([]);
  const [leagueStatuses, setLeagueStatuses] = useState<LeagueWithSeason[]>([]);
  const tournamentClient = useClient(TournamentService);
  const leagueClient = useClient(LeagueService);

  useEffect(() => {
    (async () => {
      try {
        const resp = await tournamentClient.getRecentAndUpcomingTournaments({});
        const now = new Date();
        const filtered = resp.tournaments.filter((t: TournamentMetadata) => {
          if (!t.scheduledEndTime) return true;
          const endDate = new Date(Number(t.scheduledEndTime.seconds) * 1000);
          return endDate > now;
        });
        setUpcomingTournaments(filtered);
      } catch (error) {
        console.error("Error fetching upcoming tournaments:", error);
      }
    })();
  }, [tournamentClient]);

  useEffect(() => {
    (async () => {
      try {
        const resp = await tournamentClient.getPastTournaments({ limit: 50 });
        setPastTournaments(resp.tournaments);
      } catch (error) {
        console.error("Error fetching past tournaments:", error);
      }
    })();
  }, [tournamentClient]);

  useEffect(() => {
    (async () => {
      try {
        const leaguesResp = await leagueClient.getAllLeagues({
          activeOnly: true,
        });
        const statuses: LeagueWithSeason[] = [];

        for (const league of leaguesResp.leagues) {
          try {
            // Get 2 most recent seasons - includes champion info from server
            const recentSeasonsResp = await leagueClient.getRecentSeasons({
              leagueId: league.slug,
              limit: 2,
            });
            const recentSeasons = recentSeasonsResp.seasons;
            if (recentSeasons.length === 0) continue;

            // Most recent season is first (ordered by season_number DESC)
            const currentSeason = recentSeasons[0];

            // Get champion if current season is completed
            // The server includes champion in divisions[0].standings[0] for completed seasons
            let champion: string | undefined;
            if (currentSeason && currentSeason.status === 2) {
              // SEASON_COMPLETED
              const div1 = currentSeason.divisions.find(
                (d) => d.divisionNumber === 1,
              );
              if (div1 && div1.standings.length > 0) {
                const championStanding = div1.standings.find(
                  (s) => s.result === 4,
                );
                if (championStanding) {
                  champion = championStanding.username;
                }
              }
            }

            // Get previous season champion if current season is active and within 7 days of start
            let previousSeasonChampion:
              | { seasonNumber: number; username: string }
              | undefined;
            if (
              currentSeason &&
              currentSeason.status === 1 &&
              recentSeasons.length > 1
            ) {
              // SEASON_ACTIVE
              const startDate = currentSeason.startDate
                ? new Date(Number(currentSeason.startDate.seconds) * 1000)
                : null;
              const now = new Date();
              const sevenDaysMs = 7 * 24 * 60 * 60 * 1000;

              if (
                startDate &&
                now.getTime() - startDate.getTime() < sevenDaysMs
              ) {
                // Get the previous season (second in list)
                const prevSeason = recentSeasons[1];
                if (
                  prevSeason &&
                  prevSeason.seasonNumber === currentSeason.seasonNumber - 1
                ) {
                  // Server includes champion in divisions for completed seasons
                  const div1 = prevSeason.divisions.find(
                    (d) => d.divisionNumber === 1,
                  );
                  if (div1 && div1.standings.length > 0) {
                    const championStanding = div1.standings.find(
                      (s) => s.result === 4,
                    );
                    if (championStanding) {
                      previousSeasonChampion = {
                        seasonNumber: prevSeason.seasonNumber,
                        username: championStanding.username,
                      };
                    }
                  }
                }
              }
            }

            statuses.push({
              league,
              currentSeason,
              champion,
              previousSeasonChampion,
            });
          } catch (error) {
            console.error(
              `Error fetching season for league ${league.name}:`,
              error,
            );
          }
        }

        setLeagueStatuses(statuses);
      } catch (error) {
        console.error("Error fetching leagues:", error);
      }
    })();
  }, [leagueClient]);

  const activeLeagues = leagueStatuses.filter((ls) => {
    if (!ls.currentSeason) return false;
    // Show if active, registration open, or recently completed with champion
    if (ls.currentSeason.status === 4) return true; // Registration open
    if (ls.currentSeason.status === 1) return true; // Active
    if (ls.currentSeason.status === 2 && ls.champion) return true; // Completed with champion
    return false;
  });

  return (
    <div className="tournaments-leagues-content">
      {upcomingTournaments.length > 0 && (
        <>
          <h4>Upcoming Tournaments</h4>
          <div className="tournaments-list">
            {upcomingTournaments.map((t) => (
              <TournamentCard key={t.id} tournament={t} />
            ))}
          </div>
        </>
      )}

      {activeLeagues.length > 0 && (
        <>
          <h4>Leagues</h4>
          <div className="leagues-list">
            {activeLeagues.map((ls) => (
              <LeagueStatusCard key={ls.league.uuid} leagueData={ls} />
            ))}
          </div>
        </>
      )}

      {upcomingTournaments.length === 0 &&
        pastTournaments.length === 0 &&
        activeLeagues.length === 0 && (
          <div className="empty-state">No tournaments or leagues</div>
        )}

      {pastTournaments.length > 0 && (
        <>
          <h4>Past Tournaments</h4>
          <div className="tournaments-list">
            {pastTournaments.map((t) => (
              <TournamentCard key={t.id} tournament={t} />
            ))}
          </div>
        </>
      )}
    </div>
  );
};

const AnnouncementsContent = () => {
  const [announcements, setAnnouncements] = useState<Array<Announcement>>([]);
  const configClient = useClient(ConfigService);

  useEffect(() => {
    (async () => {
      const resp = await configClient.getAnnouncements({});
      setAnnouncements(resp.announcements);
    })();
  }, [configClient]);

  const renderAnnouncements = announcements.map((a, idx) => (
    <a href={a.link} target="_blank" rel="noopener noreferrer" key={idx}>
      <li>
        <h4>{a.title}</h4>
        <div>
          <ReactMarkdown
            components={{
              img: ({ src }) => <img src={src} style={{ maxWidth: 300 }} />,
            }}
          >
            {a.body}
          </ReactMarkdown>
        </div>
      </li>
    </a>
  ));

  return <ul className="announcements-list">{renderAnnouncements}</ul>;
};

export const AnnouncementsWidget = () => {
  const items = [
    {
      key: "announcements",
      label: "Announcements",
      children: <AnnouncementsContent />,
    },
    {
      key: "tournaments-leagues",
      label: "Events",
      children: <TournamentsAndLeaguesContent />,
    },
  ];

  return (
    <Card className="announcements-card">
      <Tabs defaultActiveKey="announcements" items={items} centered />
    </Card>
  );
};
