import { useEffect, useState } from "react";
import { Card, Tooltip } from "antd";
import { Link } from "react-router";
import {
  GlobalOutlined,
  DesktopOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { useClient } from "../utils/hooks/connect";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import type { TournamentMetadata } from "../gen/api/proto/tournament_service/tournament_service_pb";
import "./upcoming_tournaments.scss";

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMs < 0) {
    // Past time
    const absSecs = Math.abs(diffSecs);
    const absMins = Math.abs(diffMins);
    const absHours = Math.abs(diffHours);
    const absDays = Math.abs(diffDays);

    if (absSecs < 60) return "just now";
    if (absMins < 60) return `${absMins} min ago`;
    if (absHours < 24) return `${absHours} hour${absHours > 1 ? "s" : ""} ago`;
    return `${absDays} day${absDays > 1 ? "s" : ""} ago`;
  }

  // Future time
  if (diffSecs < 60) return "starting soon";
  if (diffMins < 60) return `in ${diffMins} min`;
  if (diffHours < 24) return `in ${diffHours} hour${diffHours > 1 ? "s" : ""}`;
  return `in ${diffDays} day${diffDays > 1 ? "s" : ""}`;
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

export const UpcomingTournamentsWidget = () => {
  const [upcomingTournaments, setUpcomingTournaments] = useState<
    Array<TournamentMetadata>
  >([]);
  const [pastTournaments, setPastTournaments] = useState<
    Array<TournamentMetadata>
  >([]);
  const tournamentClient = useClient(TournamentService);

  useEffect(() => {
    (async () => {
      try {
        const resp = await tournamentClient.getRecentAndUpcomingTournaments({});
        // Filter out tournaments that have already ended
        const now = new Date();
        const filtered = resp.tournaments.filter((t: TournamentMetadata) => {
          if (!t.scheduledEndTime) return true;
          const endDate = new Date(Number(t.scheduledEndTime.seconds) * 1000);
          return endDate > now; // Only show if end time is in the future
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

  return (
    <Card className="upcoming-tournaments-widget" title="Tournaments">
      <div className="tournaments-container">
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

        {upcomingTournaments.length === 0 && pastTournaments.length === 0 && (
          <div className="empty-state">No tournaments</div>
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
    </Card>
  );
};
