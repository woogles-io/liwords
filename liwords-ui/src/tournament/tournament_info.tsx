import React, { ReactNode, useMemo } from "react";
import { Card, Tooltip, Button } from "antd";
import ReactMarkdown from "react-markdown";
import { useTournamentStoreContext } from "../store/store";
import { useLoginStateStoreContext } from "../store/store";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { CompetitorStatus } from "./competitor_status";
import { readyForTournamentGame } from "../tournament/ready";
import { isClubType } from "../store/constants";
import { GlobalOutlined, CameraOutlined } from "@ant-design/icons";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { useSearchParams } from "react-router";

type TournamentInfoProps = {
  setSelectedGameTab: (tab: string) => void;
  sendSocketMsg: (msg: Uint8Array) => void;
};

function LinkRenderer(props: { href?: string; children?: ReactNode }) {
  return (
    <a href={props.href} target="_blank" rel="noreferrer">
      {props.children}
    </a>
  );
}

export const TournamentInfo = (props: TournamentInfoProps) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { metadata } = tournamentContext;
  const competitorState = useTournamentCompetitorState();
  const [searchParams, setSearchParams] = useSearchParams();

  const handleOpenMonitoring = () => {
    const newParams = new URLSearchParams(searchParams);
    newParams.set("monitoring", "true");
    setSearchParams(newParams);
  };

  // HACK: Parse :readonly suffix and sort directors (full first, then read-only)
  // TODO: Replace with proper permissions field when backend schema is updated
  const directors = useMemo(() => {
    const parsedDirectors = tournamentContext.directors.map((username) => {
      const isReadOnly = username.endsWith(":readonly");
      const displayName = isReadOnly ? username.slice(0, -9) : username;
      return {
        displayName,
        isReadOnly,
        originalName: username,
      };
    });

    // Sort: full directors first, then read-only directors
    parsedDirectors.sort((a, b) => {
      if (a.isReadOnly === b.isReadOnly) return 0;
      return a.isReadOnly ? 1 : -1;
    });

    return parsedDirectors.map((director, i) => (
      <span
        className={director.isReadOnly ? "director readonly" : "director"}
        key={director.originalName}
      >
        {i > 0 && ", "}
        <UsernameWithContext username={director.displayName} omitSendMessage />
      </span>
    ));
  }, [tournamentContext.directors]);
  const type = isClubType(metadata.type) ? "Club" : "Tournament";
  const title = (
    <span style={{ color: tournamentContext.metadata.color }}>
      {tournamentContext.metadata.name}
    </span>
  );
  return (
    <div className="tournament-info">
      {/* Mobile version of the status widget, hidden by css elsewhere */}
      {competitorState.isRegistered && (
        <CompetitorStatus
          sendReady={() =>
            readyForTournamentGame(
              props.sendSocketMsg,
              tournamentContext.metadata.id,
              competitorState,
            )
          }
        />
      )}
      <Card
        title={title}
        className="tournament"
        extra={
          tournamentContext.metadata.irlMode ? (
            <Tooltip title="In Real Life Mode">
              <GlobalOutlined style={{ color: "#955f9a" }} />
            </Tooltip>
          ) : null
        }
      >
        {tournamentContext.metadata.logo && (
          <img
            src={tournamentContext.metadata.logo}
            alt={tournamentContext.metadata.name}
            style={{
              width: 150,
              textAlign: "center",
              margin: "0 auto 18px",
              display: "block",
            }}
          />
        )}
        <h4>Directed by: {directors}</h4>
        {tournamentContext.metadata.monitored && loginState.loggedIn && (
          <>
            <h5 className="section-header">Monitoring Required</h5>
            <p>
              This tournament requires all participants to share their camera
              and screen during games for invigilation purposes.
            </p>
            <Button
              type="primary"
              icon={<CameraOutlined />}
              onClick={handleOpenMonitoring}
              block
              style={{ marginTop: "12px" }}
            >
              Set up Monitoring
            </Button>
          </>
        )}
        <h5 className="section-header">{type} Details</h5>
        <ReactMarkdown components={{ a: LinkRenderer }}>
          {tournamentContext.metadata.description}
        </ReactMarkdown>
        {tournamentContext.metadata.disclaimer && (
          <>
            <h5 className="section-header">{type} Notice</h5>
            <ReactMarkdown components={{ a: LinkRenderer }}>
              {tournamentContext.metadata.disclaimer}
            </ReactMarkdown>
          </>
        )}
      </Card>
    </div>
  );
};
