/* eslint-disable jsx-a11y/anchor-is-valid */
import React from 'react';
import { Card } from 'antd';
import ReactMarkdown from 'react-markdown';
import { useTournamentStoreContext } from '../store/store';
import { UsernameWithContext } from '../shared/usernameWithContext';
import { CompetitorStatus } from './competitor_status';
import { readyForTournamentGame } from '../store/reducers/tournament_reducer';
import { isClubType } from '../store/constants';

type TournamentInfoProps = {
  setSelectedGameTab: (tab: string) => void;
  sendSocketMsg: (msg: Uint8Array) => void;
};

export const TournamentInfo = (props: TournamentInfoProps) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { competitorState: competitorContext, metadata } = tournamentContext;
  const directors = tournamentContext.directors.map((username, i) => (
    <span className="director" key={username}>
      {i > 0 && ', '}
      <UsernameWithContext username={username} omitSendMessage />
    </span>
  ));
  const type = isClubType(metadata.getType()) ? 'Club' : 'Tournament';
  const title = (
    <span style={{ color: tournamentContext.metadata.getColor() }}>
      {tournamentContext.metadata.getName()}
    </span>
  );
  return (
    <div className="tournament-info">
      {/* Mobile version of the status widget, hidden by css elsewhere */}
      {competitorContext.isRegistered && (
        <CompetitorStatus
          sendReady={() =>
            readyForTournamentGame(
              props.sendSocketMsg,
              tournamentContext.metadata.getId(),
              competitorContext
            )
          }
        />
      )}
      <Card title={title} className="tournament">
        {tournamentContext.metadata.getLogo() && (
          <img
            src={tournamentContext.metadata.getLogo()}
            alt={tournamentContext.metadata.getName()}
            style={{
              width: 150,
              textAlign: 'center',
              margin: '0 auto 18px',
              display: 'block',
            }}
          />
        )}
        <h4>Directed by: {directors}</h4>
        <h5 className="section-header">{type} Details</h5>
        <ReactMarkdown linkTarget="_blank">
          {tournamentContext.metadata.getDescription()}
        </ReactMarkdown>
        {tournamentContext.metadata.getDisclaimer() && (
          <>
            <h5 className="section-header">{type} Notice</h5>
            <ReactMarkdown linkTarget="_blank">
              {tournamentContext.metadata.getDisclaimer()}
            </ReactMarkdown>
          </>
        )}
      </Card>
    </div>
  );
};
