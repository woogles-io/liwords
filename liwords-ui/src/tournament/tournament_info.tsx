import React, { useCallback, useEffect } from 'react';
import { Card, Divider } from 'antd';
import axios from 'axios';
import { useParams } from 'react-router-dom';
import { RecentTourneyGames } from './recent_games';
import { pageSize, RecentGame } from './recent_game';
import { toAPIUrl } from '../api/api';
import { useLobbyStoreContext } from '../store/store';
import { ActionType } from '../actions/actions';

type TournamentInfoProps = {
  tournamentID: string;
  tournamentInfo: TournamentMetadata;
};

export type RecentTournamentGames = {
  games: Array<RecentGame>;
};

export type TournamentMetadata = {
  name: string;
  description: string;
  directors: Array<string>;
};

export const TournamentInfo = (props: TournamentInfoProps) => {
  const { lobbyContext, dispatchLobbyContext } = useLobbyStoreContext();

  const { tournamentID } = useParams();

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

  return (
    <div className="announcements">
      <Card title="Tournament Information">
        <h3>{props.tournamentInfo.name}</h3>
        <h4>Directors: {props.tournamentInfo.directors.join(', ')}</h4>
        <p>{props.tournamentInfo.description}</p>
        <Divider />
        <h3>Recent Games</h3>
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
