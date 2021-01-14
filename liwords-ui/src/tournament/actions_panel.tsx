import { Button, Card, Select } from 'antd';
import Modal from 'antd/lib/modal/Modal';
import React, { ReactNode, useCallback, useEffect, useMemo } from 'react';
import { LeftOutlined, RightOutlined } from '@ant-design/icons';
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
import { Pairings } from './pairings';
import { isPairedMode } from '../store/constants';

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
  sendReady?: () => void;
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
  const {
    dispatchTournamentContext,
    tournamentContext,
  } = useTournamentStoreContext();
  const { divisions } = tournamentContext;
  const [selectedRound, setSelectedRound] = useState(0);
  const [selectedDivision, setSelectedDivision] = useState('');
  const { lobbyContext } = useLobbyStoreContext();
  const tournamentID = tournamentContext.metadata.id;

  const fetchPrev = useCallback(() => {
    dispatchTournamentContext({
      actionType: ActionType.SetTourneyGamesOffset,
      payload: Math.max(
        tournamentContext.gamesOffset - tournamentContext.gamesPageSize,
        0
      ),
    });
  }, [
    dispatchTournamentContext,
    tournamentContext.gamesOffset,
    tournamentContext.gamesPageSize,
  ]);
  const fetchNext = useCallback(() => {
    dispatchTournamentContext({
      actionType: ActionType.SetTourneyGamesOffset,
      payload: tournamentContext.gamesOffset + tournamentContext.gamesPageSize,
    });
  }, [
    dispatchTournamentContext,
    tournamentContext.gamesOffset,
    tournamentContext.gamesPageSize,
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
          offset: tournamentContext.gamesOffset,
        }
      )
      .then((resp) => {
        dispatchTournamentContext({
          actionType: ActionType.AddTourneyGameResults,
          payload: resp.data.games,
        });
      });
  }, [tournamentID, dispatchTournamentContext, tournamentContext.gamesOffset]);
  const renderDivisionSelector =
    Object.values(divisions).length > 1 ? (
      <Select value={selectedDivision} onChange={setSelectedDivision}>
        {Object.values(divisions).map((d) => {
          return (
            <Select.Option value={d.divisionID} key={d.divisionID}>
              {d.divisionID}
            </Select.Option>
          );
        })}
      </Select>
    ) : null;

  const renderGamesTab = () => {
    if (selectedGameTab === 'GAMES') {
      if (isPairedMode(tournamentContext.metadata.type)) {
        return (
          <div className="pairings-container">
            {renderDivisionSelector}
            <Pairings
              selectedRound={selectedRound}
              selectedDivision={selectedDivision}
              username={username}
              sendReady={props.sendReady}
            />
          </div>
        );
      }
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
            activeGames={tournamentContext?.activeGames}
          />
        </>
      );
    }
    if (selectedGameTab === 'RECENT') {
      return (
        <>
          <h4>Recent Games</h4>
          <RecentTourneyGames
            games={tournamentContext.finishedTourneyGames}
            fetchPrev={
              tournamentContext.gamesOffset > 0 ? fetchPrev : undefined
            }
            fetchNext={
              tournamentContext.finishedTourneyGames.length < pageSize
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
            games={tournamentContext.finishedTourneyGames}
            fetchPrev={
              tournamentContext.gamesOffset > 0 ? fetchPrev : undefined
            }
            fetchNext={
              tournamentContext.finishedTourneyGames.length < pageSize
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
  useEffect(() => {
    const idFromPlayerEntry = (p: string) => p.split(':')[0];
    const divisionArray = Object.values(divisions);
    const foundDivision = userID
      ? divisionArray.find((d) => {
          return d.players.map(idFromPlayerEntry).includes(userID);
        })
      : undefined;
    if (foundDivision) {
      if (!selectedDivision) {
        setSelectedDivision(foundDivision.divisionID);
        setSelectedRound(foundDivision.currentRound);
      }
    } else {
      console.log();
      if (divisionArray.length) {
        if (!selectedDivision) {
          setSelectedDivision(divisionArray[0].divisionID);
        }
      }
    }
  }, [divisions, selectedDivision, userID]);

  const actions = useMemo(() => {
    let matchButtonText = 'Start tournament game';
    if (['CLUB', 'CHILD'].includes(tournamentContext.metadata.type)) {
      matchButtonText = 'Start club game';
    }
    const availableActions = new Array<ReactNode>();
    if (props.loggedIn && !isPairedMode(tournamentContext.metadata.type)) {
      // We are allowing free-form match requests in CLUBHOUSE mode, if desired.
      availableActions.push(
        <div
          className="match"
          onClick={() => {
            setMatchModalVisible(true);
          }}
          key="match-action"
        >
          {matchButtonText}
        </div>
      );
    }
    if (isPairedMode(tournamentContext.metadata.type)) {
      const lastRound =
        tournamentContext.divisions[selectedDivision]?.numRounds - 1 ||
        selectedRound;
      if (selectedRound === 0) {
        availableActions.push(<div className="empty"></div>);
      } else {
        availableActions.push(
          <div
            className="round-change prev"
            onClick={() => {
              setSelectedRound(selectedRound - 1);
            }}
          >
            <LeftOutlined />
            {/* The previous, zero-indexed round converted to 1-indexed */}
            Round {selectedRound}{' '}
          </div>
        );
      }
      availableActions.push(
        <div className="round-label">
          {/* The current zero-indexed round converted to 1-indexed */}
          Round {selectedRound + 1}
        </div>
      );
      if (lastRound === selectedRound) {
        availableActions.push(<div className="empty"></div>);
      } else {
        availableActions.push(
          <div
            className="round-change next"
            onClick={() => {
              setSelectedRound(selectedRound + 1);
            }}
          >
            {/* The next zero-indexed round converted to 1-indexed*/}
            Round {selectedRound + 2}
            <RightOutlined />
          </div>
        );
      }
    }
    return availableActions;
  }, [props.loggedIn, tournamentContext, selectedDivision, selectedRound]);

  return (
    <div className="game-lists">
      <Card
        actions={actions}
        className={
          isPairedMode(tournamentContext.metadata.type)
            ? 'paired-mode'
            : 'free-form'
        }
      >
        <div className="tabs">
          <div
            onClick={() => {
              setSelectedGameTab('GAMES');
            }}
            className={selectedGameTab === 'GAMES' ? 'tab active' : 'tab'}
          >
            Games
          </div>
          {!isPairedMode(tournamentContext.metadata.type) ? (
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
