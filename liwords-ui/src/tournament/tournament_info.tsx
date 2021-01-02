import React from 'react';
import { Card } from 'antd';
import ReactMarkdown from 'react-markdown';
import { useTournamentStoreContext } from '../store/store';
import { UsernameWithContext } from '../shared/usernameWithContext';

type TournamentInfoProps = {};

export const TournamentInfo = (props: TournamentInfoProps) => {
  const { tournamentContext } = useTournamentStoreContext();

  const directors = tournamentContext.metadata.directors.map((username, i) => (
    <span key={username}>
      {i > 0 && ', '}
      <UsernameWithContext username={username} omitSendMessage />
    </span>
  ));

  return (
    <div className="tournament-info">
      <Card title="Tournament Information">
        <h3 className="tournament-name">{tournamentContext.metadata.name}</h3>
        <h4>Directors: {directors}</h4>
        <ReactMarkdown linkTarget="_blank">
          {tournamentContext.metadata.description}
        </ReactMarkdown>
      </Card>
    </div>
  );
};
