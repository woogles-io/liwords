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
  const directors = tournamentContext.metadata.directors.map((username, i) => (
    <span className="director" key={username}>
      {i > 0 && ', '}
      <UsernameWithContext username={username} omitSendMessage />
    </span>
  ));
  const type = isClubType(metadata.type) ? 'Club' : 'Tournament';
  return (
    <div className="tournament-info">
      {/* Mobile version of the status widget, hidden by css elsewhere */}
      {competitorContext.isRegistered && (
        <CompetitorStatus
          sendReady={() =>
            readyForTournamentGame(
              props.sendSocketMsg,
              tournamentContext.metadata.id,
              competitorContext
            )
          }
        />
      )}
      <Card title={tournamentContext.metadata.name} className="tournament">
        <h4>Directed by: {directors}</h4>
        <h5 className="section-header">{type} Details</h5>
        <ReactMarkdown linkTarget="_blank">
          {tournamentContext.metadata.description}
        </ReactMarkdown>
      </Card>
    </div>
  );
};
