import { Button, Card, message, Select } from 'antd';
import { Modal } from '../utils/focus_modal';
import React, {
  ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from 'react';
import { LeftOutlined, RightOutlined } from '@ant-design/icons';
import { ActiveGames } from '../lobby/active_games';
import { SeekForm } from '../lobby/seek_form';
import { SoughtGames } from '../lobby/sought_games';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import {
  useContextMatchContext,
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
import { isPairedMode, isClubType } from '../store/constants';
import { Standings } from './standings';
import { DirectorTools } from './director_tools';
// import { CheckIn } from './check_in';

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
  isAdmin: boolean;
  onSeekSubmit: (g: SoughtGame) => void;
  sendReady: () => void;
};

export const ActionsPanel = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const [matchModalVisible, setMatchModalVisible] = useState(false);
  const [formDisabled, setFormDisabled] = useState(false);
  const {
    selectedGameTab,
    setSelectedGameTab,
    isDirector,
    isAdmin,
    onSeekSubmit,
    newGame,
    userID,
    username,
  } = props;
  const renderDirectorTools = () => {
    return <DirectorTools tournamentID={props.tournamentID} />;
  };
  const {
    dispatchTournamentContext,
    tournamentContext,
  } = useTournamentStoreContext();
  const itIsPairedMode = useMemo(
    () => isPairedMode(tournamentContext.metadata?.getType()),
    [tournamentContext]
  );
  const { divisions } = tournamentContext;
  const [competitorStatusLoaded, setCompetitorStatusLoaded] = useState(
    tournamentContext.competitorState.isRegistered
  );
  let initialRound = 0;
  let initialDivision = '';
  if (tournamentContext.competitorState.division) {
    initialDivision = tournamentContext.competitorState.division;
    initialRound = tournamentContext.competitorState.currentRound;
  }
  const [selectedRound, setSelectedRound] = useState(initialRound);
  const [selectedDivision, setSelectedDivision] = useState(initialDivision);
  const { lobbyContext } = useLobbyStoreContext();
  const tournamentID = tournamentContext.metadata?.getId();

  const {
    addHandleContextMatch,
    removeHandleContextMatch,
  } = useContextMatchContext();
  const friendRef = useRef('');
  const handleContextMatch = useCallback((s: string) => {
    friendRef.current = s;
    setMatchModalVisible(true);
  }, []);
  useEffect(() => {
    if (!itIsPairedMode && !matchModalVisible) {
      addHandleContextMatch(handleContextMatch);
      return () => {
        removeHandleContextMatch(handleContextMatch);
      };
    }
  }, [
    itIsPairedMode,
    matchModalVisible,
    handleContextMatch,
    addHandleContextMatch,
    removeHandleContextMatch,
  ]);

  const lobbyContextMatchRequests = lobbyContext?.matchRequests;
  const thisTournamentMatchRequests = useMemo(
    () =>
      lobbyContextMatchRequests?.filter(
        (matchRequest) => matchRequest.tournamentID === tournamentID
      ),
    [lobbyContextMatchRequests, tournamentID]
  );

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
  const renderStartRoundButton = () => {
    const division = tournamentContext.divisions[selectedDivision];
    if (!division) {
      return null;
    }
    const { currentRound } = division;
    let roundToStart: null | number = null;
    if (division) {
      roundToStart = currentRound + 1;
    }
    if (
      !isDirector ||
      !(typeof roundToStart === 'number') ||
      !(roundToStart === selectedRound) ||
      roundToStart === 0 // (handle this with Start Tournament)
    ) {
      return null;
    }
    const startRound = () => {
      axios
        .post(
          toAPIUrl(
            'tournament_service.TournamentService',
            'StartRoundCountdown'
          ),
          {
            id: tournamentID,
            division: division.divisionID,
            round: roundToStart,
          },
          { withCredentials: true }
        )
        .catch((err) => {
          message.error({
            content:
              'Round cannot be started yet. The error message was: ' +
              err.response?.data?.msg,
            duration: 8,
          });
          console.log('Error starting round: ' + err.response?.data?.msg);
        });
    };
    return (
      <Button className="primary open-round" onClick={startRound}>
        Open Round {roundToStart! + 1}
      </Button>
    );
  };
  const renderGamesTab = () => {
    if (selectedGameTab === 'GAMES') {
      if (itIsPairedMode) {
        return (
          <div className="pairings-container">
            {/* <CheckIn /> */}
            <div className="round-options">
              {renderDivisionSelector}
              {renderStartRoundButton()}
            </div>
            <Pairings
              selectedRound={selectedRound}
              selectedDivision={selectedDivision}
              username={username}
              sendReady={props.sendReady}
              isDirector={isDirector}
            />
          </div>
        );
      }
      return (
        <>
          {thisTournamentMatchRequests?.length ? (
            <SoughtGames
              isMatch={true}
              userID={userID}
              username={username}
              newGame={newGame}
              requests={thisTournamentMatchRequests}
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
          <h4>Recent games</h4>
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
            isDirector={isDirector}
          />
        </>
      );
    }
    if (selectedGameTab === 'STANDINGS') {
      return (
        <div className="standings-container">
          <div className="round-options">{renderDivisionSelector}</div>
          <Standings selectedDivision={selectedDivision} />
        </div>
      );
    }
    return null;
  };

  const matchModal = (
    <Modal
      className="seek-modal"
      title="Send match request"
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
          Create game
        </button>,
      ]}
    >
      <SeekForm
        onFormSubmit={onFormSubmit}
        loggedIn={props.loggedIn}
        username={props.username}
        showFriendInput={true}
        friendRef={friendRef}
        id="match-seek"
        tournamentID={props.tournamentID}
      />
    </Modal>
  );

  const idFromPlayerEntry = useCallback((p: string) => p.split(':')[0], []);
  useEffect(() => {
    const divisionArray = Object.values(divisions);
    const foundDivision = userID
      ? divisionArray.find((d) => {
          return d.players
            .map((v) => v.getId())
            .map(idFromPlayerEntry)
            .includes(userID);
        })
      : undefined;
    // look for ourselves in the division.
    if (foundDivision) {
      if (!competitorStatusLoaded) {
        setCompetitorStatusLoaded(true);
        setSelectedDivision(foundDivision.divisionID);
        setSelectedRound(foundDivision.currentRound);
      } else if (!selectedDivision) {
        setSelectedDivision(foundDivision.divisionID);
        setSelectedRound(foundDivision.currentRound);
      } else if (selectedRound === -1) {
        // If we are directing _and_ playing, this, combined with the code
        // in pairings.tsx to hide initial pairings, will show preview
        // pairings for the director.
        setSelectedRound(isDirector ? 0 : foundDivision.currentRound);
      }
    } else {
      // we are an observer
      if (divisionArray.length) {
        if (!selectedDivision) {
          setSelectedDivision(divisionArray[0].divisionID);
          setSelectedRound(
            divisionArray[0].currentRound > -1
              ? divisionArray[0].currentRound
              : 0
          );
        }
      }
    }
  }, [
    divisions,
    idFromPlayerEntry,
    isDirector,
    selectedDivision,
    competitorStatusLoaded,
    selectedRound,
    userID,
  ]);

  const actions = useMemo(() => {
    if (selectedGameTab === 'STANDINGS') {
      return [];
    }
    let matchButtonText = 'Start tournament game';
    if (isClubType(tournamentContext.metadata?.getType())) {
      matchButtonText = 'Start club game';
    }
    const availableActions = new Array<ReactNode>();
    if (props.loggedIn && !itIsPairedMode) {
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
    if (
      itIsPairedMode &&
      selectedRound > -1 &&
      tournamentContext.divisions[selectedDivision]
    ) {
      const lastRound =
        tournamentContext.divisions[selectedDivision]?.numRounds - 1 ||
        selectedRound;
      if (selectedRound < 1) {
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
  }, [
    itIsPairedMode,
    props.loggedIn,
    tournamentContext,
    selectedDivision,
    selectedRound,
    selectedGameTab,
  ]);
  return (
    <div className="game-lists">
      <Card
        actions={actions}
        className={itIsPairedMode ? 'paired-mode' : 'free-form'}
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
          {!itIsPairedMode ? (
            <div
              onClick={() => {
                setSelectedGameTab('RECENT');
              }}
              className={selectedGameTab === 'RECENT' ? 'tab active' : 'tab'}
            >
              Recent games
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
          {(isDirector || isAdmin) && (
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
        <div className="main-content">
          {(isDirector || isAdmin) &&
            selectedGameTab === 'DIRECTOR TOOLS' &&
            renderDirectorTools()}
          {matchModal}
          {renderGamesTab()}
        </div>
      </Card>
    </div>
  );
});
