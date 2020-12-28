import React, { useCallback, useEffect } from 'react';
import { Card, Divider } from 'antd';
import axios from 'axios';
import ReactMarkdown from 'react-markdown';
import { RecentTourneyGames } from './recent_games';
import { pageSize, RecentGame } from './recent_game';
import { toAPIUrl } from '../api/api';
import {
  useLobbyStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { ActionType } from '../actions/actions';
import { UsernameWithContext } from '../shared/usernameWithContext';

type TournamentInfoProps = {};

export type RecentTournamentGames = {
  games: Array<RecentGame>;
};

export const TournamentInfo = (props: TournamentInfoProps) => {
  const { lobbyContext, dispatchLobbyContext } = useLobbyStoreContext();
  const { tournamentContext } = useTournamentStoreContext();
  const tournamentID = tournamentContext.metadata.id;

  useEffect(() => {
    if (!tournamentID) {
      return;
    }
    axios
      .post<RecentTournamentGames>(
        toAPIUrl('tournament_service.TournamentService', 'RecentGames'),
        {
          id: tournamentID,
          num_games: pageSize,
          offset: lobbyContext.gamesOffset,
        }
      )
      .then((resp) => {
        dispatchLobbyContext({
          actionType: ActionType.AddTourneyGames,
          payload: resp.data.games,
        });
      });
  }, [tournamentID, dispatchLobbyContext, lobbyContext.gamesOffset]);

  const fetchPrev = useCallback(() => {
    dispatchLobbyContext({
      actionType: ActionType.SetTourneyGamesOffset,
      payload: Math.max(
        lobbyContext.gamesOffset - lobbyContext.gamesPageSize,
        0
      ),
    });
  }, [
    dispatchLobbyContext,
    lobbyContext.gamesOffset,
    lobbyContext.gamesPageSize,
  ]);
  const fetchNext = useCallback(() => {
    dispatchLobbyContext({
      actionType: ActionType.SetTourneyGamesOffset,
      payload: lobbyContext.gamesOffset + lobbyContext.gamesPageSize,
    });
  }, [
    dispatchLobbyContext,
    lobbyContext.gamesOffset,
    lobbyContext.gamesPageSize,
  ]);

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
        <Divider />
        <h3 className="recent-header">Recent Games</h3>
        <RecentTourneyGames
          games={lobbyContext.tourneyGames}
          fetchPrev={lobbyContext.gamesOffset > 0 ? fetchPrev : undefined}
          fetchNext={
            lobbyContext.tourneyGames.length < pageSize ? undefined : fetchNext
          }
        />
      </Card>
    </div>
  );
};
