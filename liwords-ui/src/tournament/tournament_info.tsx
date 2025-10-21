import React, { ReactNode } from "react";
import { Card, Tooltip } from "antd";
import ReactMarkdown from "react-markdown";
import { useTournamentStoreContext } from "../store/store";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { CompetitorStatus } from "./competitor_status";
import { readyForTournamentGame } from "../tournament/ready";
import { isClubType } from "../store/constants";
import { TeamOutlined, CameraOutlined } from "@ant-design/icons";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { Link } from "react-router";

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
  const { metadata } = tournamentContext;
  const competitorState = useTournamentCompetitorState();
  const directors = tournamentContext.directors.map((username, i) => (
    <span className="director" key={username}>
      {i > 0 && ", "}
      <UsernameWithContext username={username} omitSendMessage />
    </span>
  ));
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
              <TeamOutlined style={{ color: "#955f9a" }} />
              <TeamOutlined style={{ color: "#955f9a" }} />
              <TeamOutlined style={{ color: "#955f9a" }} />
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
        {tournamentContext.metadata.monitored && (
          <div
            style={{
              marginBottom: "16px",
              padding: "8px",
              backgroundColor: "rgba(0, 0, 0, 0.04)",
              borderRadius: "4px",
              border: "1px solid rgba(0, 0, 0, 0.1)",
            }}
            className="monitoring-link-container"
          >
            <Link
              to={`${tournamentContext.metadata.slug}/monitoring`}
              style={{ display: "flex", alignItems: "center", gap: "8px" }}
            >
              <CameraOutlined style={{ fontSize: "18px" }} />
              <strong>Set up monitoring</strong>
            </Link>
          </div>
        )}
        <h4>Directed by: {directors}</h4>
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
