import { useEffect, useState } from "react";
import { Card, Tooltip } from "antd";
import { Link } from "react-router";
import {
  BookOutlined,
  GlobalOutlined,
  DesktopOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { useClient } from "../utils/hooks/connect";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import type { TournamentMetadata } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { MatchLexiconDisplay } from "../shared/lexicon_display";
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

  // Distinct lexica configured across the tournament's divisions. Empty for
  // IRL-mode or not-yet-configured tournaments, in which case we show nothing.
  const lexica = Array.from(
    new Set(
      (tournament.divisions ?? [])
        .map((d) => d.gameRequest?.lexicon)
        .filter((l): l is string => !!l),
    ),
  );

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
          {lexica.length > 0 && (
            <span className="tournament-lexicon">
              <BookOutlined className="lexicon-icon" />
              {lexica.map((lex, i) => (
                <span key={lex}>
                  {i > 0 ? ", " : ""}
                  <MatchLexiconDisplay lexiconCode={lex} />
                </span>
              ))}
            </span>
          )}
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

  // upcomingTournaments holds tournaments that have not yet ended (or have no
  // end time). Split them: not-yet-started ones stay "Upcoming", while
  // already-started ones become "Live". Tournaments with a start time in the
  // past but no end time are treated as live (ongoing indefinitely), and any
  // not-ended tournament without a start time is also shown as live so it is
  // not silently dropped.
  const now = new Date();
  const notStartedTournaments = upcomingTournaments.filter((t) => {
    if (!t.scheduledStartTime) return false;
    const start = new Date(Number(t.scheduledStartTime.seconds) * 1000);
    return now < start;
  });
  // Source list is ordered by start time ascending; reversing yields
  // descending (latest start first) for the live section.
  const liveTournaments = upcomingTournaments
    .filter((t) => {
      if (!t.scheduledStartTime) return true;
      const start = new Date(Number(t.scheduledStartTime.seconds) * 1000);
      return now >= start;
    })
    .slice()
    .reverse();

  return (
    <Card className="upcoming-tournaments-widget" title="Tournaments">
      <div className="tournaments-container">
        {notStartedTournaments.length > 0 && (
          <>
            <h4>Upcoming Tournaments</h4>
            <div className="tournaments-list">
              {notStartedTournaments.map((t) => (
                <TournamentCard key={t.id} tournament={t} />
              ))}
            </div>
          </>
        )}

        {liveTournaments.length > 0 && (
          <>
            <h4>Live Tournaments</h4>
            <div className="tournaments-list">
              {liveTournaments.map((t) => (
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
