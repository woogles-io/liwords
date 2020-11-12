import React from 'react';
import { Card, Divider } from 'antd';
import { RecentTourneyGames } from '../tournament/recent_games';
import { RecentGame } from '../tournament/recent_game';

type TournamentInfoProps = {
  tournamentID: string;
  tournamentInfo: TournamentMetadata;
  games: Array<RecentGame>;
};

export type RecentTournamentGames = {
  games: Array<RecentGame>;
};

export type TournamentMetadata = {
  name: string;
  description: string;
  director_username: string;
};

export const TournamentInfo = (props: TournamentInfoProps) => {
  return (
    <div className="announcements">
      <Card title="Tournament Information">
        <h3>{props.tournamentInfo.name}</h3>
        <h4>Executive Director: {props.tournamentInfo.director_username}</h4>
        <p>{props.tournamentInfo.description}</p>
        <Divider />
        <RecentTourneyGames
          games={props.games}
          fetchPrev={() => {}}
          fetchNext={() => {}}
        />
      </Card>
    </div>
  );
};
