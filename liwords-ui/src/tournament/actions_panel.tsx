import { Button, Card } from 'antd';
import Modal from 'antd/lib/modal/Modal';
import React, { useCallback, useEffect } from 'react';
import { ActiveGames } from '../lobby/active_games';
import { SeekForm } from '../lobby/seek_form';
import { SoughtGames } from '../lobby/sought_games';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import {
  useLobbyStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { useMountedState } from '../utils/mounted';
import { RecentTourneyGames } from './recent_games';
import { pageSize, RecentGame } from './recent_game';
import { ActionType } from '../actions/actions';
import axios from 'axios';
import { toAPIUrl } from '../api/api';

export type RecentTournamentGames = {
  games: Array<RecentGame>;
};

type Props = {
  newGame: (seekID: string) => void;
  loggedIn: boolean;
  // For logged-in users:
  userID?: string;
  username?: string;
  selectedGameTab: string;
  setSelectedGameTab: (tab: string) => void;
  tournamentID: string;
  isDirector: boolean;
  onSeekSubmit: (g: SoughtGame) => void;
};

export const ActionsPanel = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const [matchModalVisible, setMatchModalVisible] = useState(false);
  const [formDisabled, setFormDisabled] = useState(false);

  const {
    selectedGameTab,
    setSelectedGameTab,
    isDirector,
    onSeekSubmit,
    newGame,
    userID,
    username,
  } = props;
  const renderDirectorTools = () => {
    // return <DirectorTools tournamentID={props.tournamentID} />;
    return <div>Coming soon!</div>;
  };
  const { tournamentContext } = useTournamentStoreContext();
  const tournamentID = tournamentContext.metadata.id;
  const { lobbyContext, dispatchLobbyContext } = useLobbyStoreContext();

  let matchButtonText;
  if (['CLUB', 'CLUBSESSION'].includes(tournamentContext.metadata.type)) {
    matchButtonText = 'Start club game';
  } else if (tournamentContext.metadata.type === 'STANDARD') {
    matchButtonText = 'Start tournament game';
  }
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
  const onFormSubmit = (sg: SoughtGame) => {
    setMatchModalVisible(false);
    setFormDisabled(true);
    if (!formDisabled) {
      onSeekSubmit(sg);
      setTimeout(() => {
        setFormDisabled(false);
      }, 500);
    }
  };

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
          actionType: ActionType.AddTourneyGameResults,
          payload: resp.data.games,
        });
      });
  }, [tournamentID, dispatchLobbyContext, lobbyContext.gamesOffset]);

  const renderGamesTab = () => {
    if (selectedGameTab === 'GAMES') {
      return (
        <>
          {lobbyContext?.matchRequests.length ? (
            <SoughtGames
              isMatch={true}
              userID={userID}
              username={username}
              newGame={newGame}
              requests={lobbyContext?.matchRequests}
            />
          ) : null}
          <ActiveGames
            username={username}
            activeGames={lobbyContext?.activeGames}
          />
        </>
      );
    }
    if (selectedGameTab === 'RECENT') {
      return (
        <>
          <h4>Recent Games</h4>
          <RecentTourneyGames
            games={lobbyContext.tourneyGames}
            fetchPrev={lobbyContext.gamesOffset > 0 ? fetchPrev : undefined}
            fetchNext={
              lobbyContext.tourneyGames.length < pageSize
                ? undefined
                : fetchNext
            }
          />
        </>
      );
    }
    if (selectedGameTab === 'STANDINGS') {
      return (
        <>
          <h4>Recent Games</h4>
          <RecentTourneyGames
            games={lobbyContext.tourneyGames}
            fetchPrev={lobbyContext.gamesOffset > 0 ? fetchPrev : undefined}
            fetchNext={
              lobbyContext.tourneyGames.length < pageSize
                ? undefined
                : fetchNext
            }
          />
        </>
      );
    }
    return null;
  };

  const matchModal = (
    <Modal
      className="seek-modal"
      title="Send Match Request"
      visible={matchModalVisible}
      destroyOnClose
      onCancel={() => {
        setMatchModalVisible(false);
        setFormDisabled(false);
      }}
      footer={[
        <Button
          key="back"
          onClick={() => {
            setMatchModalVisible(false);
          }}
        >
          Cancel
        </Button>,
        <button
          className="primary"
          key="submit"
          form="match-seek"
          type="submit"
          disabled={formDisabled}
        >
          Create Game
        </button>,
      ]}
    >
      <SeekForm
        onFormSubmit={onFormSubmit}
        loggedIn={props.loggedIn}
        showFriendInput={true}
        id="match-seek"
        tournamentID={props.tournamentID}
      />
    </Modal>
  );

  const actions = [];
  if (props.loggedIn && (true || tournamentContext.metadata.type === 'CLUB')) {
    // We are allowing free-form match requests in CLUBHOUSE mode, if desired.
    actions.push([
      <div
        className="match"
        onClick={() => {
          setMatchModalVisible(true);
        }}
        key="match-action"
      >
        {matchButtonText}
      </div>,
    ]);
  }
  return (
    <div className="game-lists">
      <Card actions={actions}>
        <div className="tabs">
          <div
            onClick={() => {
              setSelectedGameTab('GAMES');
            }}
            className={selectedGameTab === 'GAMES' ? 'tab active' : 'tab'}
          >
            Games
          </div>
          {/* {tournamentContext.metadata.type === 'CLUB' ? ( */}
          {true ? (
            <div
              onClick={() => {
                setSelectedGameTab('RECENT');
              }}
              className={selectedGameTab === 'RECENT' ? 'tab active' : 'tab'}
            >
              Recent Games
            </div>
          ) : (
            <div
              onClick={() => {
                setSelectedGameTab('STANDINGS');
              }}
              className={selectedGameTab === 'STANDINGS' ? 'tab active' : 'tab'}
            >
              Standings
            </div>
          )}
          {isDirector && (
            <div
              onClick={() => {
                setSelectedGameTab('DIRECTOR TOOLS');
              }}
              className={
                selectedGameTab === 'DIRECTOR TOOLS' ? 'tab active' : 'tab'
              }
            >
              Director Tools
            </div>
          )}
        </div>
        {isDirector &&
          selectedGameTab === 'DIRECTOR TOOLS' &&
          renderDirectorTools()}
        {matchModal}
        {renderGamesTab()}
      </Card>
    </div>
  );
});
