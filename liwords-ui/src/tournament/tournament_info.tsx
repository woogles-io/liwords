/* eslint-disable jsx-a11y/anchor-is-valid */
import React from 'react';
import { Card, Divider } from 'antd';
import ReactMarkdown from 'react-markdown';
import { useTournamentStoreContext } from '../store/store';
import { UsernameWithContext } from '../shared/usernameWithContext';
import { CompetitorStatus } from './competitor_status';

type TournamentInfoProps = {
  setSelectedGameTab: (tab: string) => void;
};

export const TournamentInfo = (props: TournamentInfoProps) => {
  const { tournamentContext, competitorContext } = useTournamentStoreContext();

  const directors = tournamentContext.metadata.directors.map((username, i) => (
    <span key={username}>
      {i > 0 && ', '}
      <UsernameWithContext username={username} omitSendMessage />
    </span>
  ));

  return (
    <div className="tournament-info">
      {/* Mobile version of the status widget, hidden by css elsewhere */}
      {competitorContext.isRegistered && <CompetitorStatus />}
      <Card title="Tournament Information" className="tournament">
        <h3 className="tournament-name">{tournamentContext.metadata.name}</h3>
        <h4>Directors: {directors}</h4>
        <ReactMarkdown linkTarget="_blank">
          {tournamentContext.metadata.description}
        </ReactMarkdown>
        <Divider />
        Recent games can now be found in the{' '}
        <a onClick={() => props.setSelectedGameTab('RECENT')}>
          RECENT GAMES
        </a>{' '}
        tab in the center panel.
      </Card>
    </div>
  );
};
