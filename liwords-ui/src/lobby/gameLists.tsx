import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Card, Button } from 'antd';
import { Modal } from '../utils/focus_modal';
import { SoughtGames } from './sought_games';
import { ActiveGames } from './active_games';
import { SeekForm } from './seek_form';
import { useContextMatchContext, useLobbyStoreContext } from '../store/store';
import { ActiveGame, SoughtGame } from '../store/reducers/lobby_reducer';
import { ActionType } from '../actions/actions';
import './seek_form.scss';
import '../shared/gameLists.scss';

type Props = {
  loggedIn: boolean;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  selectedGameTab: string;
  setSelectedGameTab: (tab: string) => void;
  onSeekSubmit: (g: SoughtGame) => void;
};

export const GameLists = React.memo((props: Props) => {
  const navigate = useNavigate();

  const {
    loggedIn,
    userID,
    username,
    newGame,
    selectedGameTab,
    setSelectedGameTab,
    onSeekSubmit,
  } = props;
  const { lobbyContext, dispatchLobbyContext } = useLobbyStoreContext();
  const [formDisabled, setFormDisabled] = useState(false);
  const [seekModalVisible, setSeekModalVisible] = useState(false);
  const [matchModalVisible, setMatchModalVisible] = useState(false);
  const [botModalVisible, setBotModalVisible] = useState(false);

  const { addHandleContextMatch, removeHandleContextMatch } =
    useContextMatchContext();
  const friendRef = useRef('');
  const handleContextMatch = useCallback((s: string) => {
    friendRef.current = s;
    setMatchModalVisible(true);
  }, []);
  useEffect(() => {
    if (!(seekModalVisible || matchModalVisible || botModalVisible)) {
      addHandleContextMatch(handleContextMatch);
      return () => {
        removeHandleContextMatch(handleContextMatch);
      };
    }
  }, [
    seekModalVisible,
    matchModalVisible,
    botModalVisible,
    handleContextMatch,
    addHandleContextMatch,
    removeHandleContextMatch,
  ]);

  const [simultaneousModeEnabled, setSimultaneousModeEnabled] = useState(false);
  const handleEnableSimultaneousMode = React.useCallback(
    (evt: { preventDefault: () => void }) => {
      evt.preventDefault();
      setSimultaneousModeEnabled(true);
    },
    []
  );
  const myCurrentGames = React.useMemo(
    () =>
      lobbyContext.activeGames.filter((ag) =>
        ag.players.some((p) => p.displayName === username)
      ),
    [lobbyContext.activeGames, username]
  );
  const simultaneousModeEffectivelyEnabled =
    simultaneousModeEnabled || myCurrentGames.length !== 1;
  const currentGame: ActiveGame | null = myCurrentGames[0] ?? null;
  const opponent = currentGame?.players.find(
    (p) => p.displayName !== username
  )?.displayName;

  const enableVariants = React.useMemo(
    () => localStorage.getItem('enableVariants') === 'true',
    []
  );
  const unsanitizedSoughtGames = lobbyContext.soughtGames;
  const sanitizedSoughtGames = React.useMemo(
    () =>
      (unsanitizedSoughtGames || []).filter((soughtGame) => {
        if (!enableVariants && (soughtGame.variant || 'classic') !== 'classic')
          return false;
        return true;
      }),

    [enableVariants, unsanitizedSoughtGames]
  );

  const matchButtonText = 'Match a friend';

  const renderGames = () => {
    if (loggedIn && userID && username && selectedGameTab === 'PLAY') {
      return (
        <>
          {simultaneousModeEffectivelyEnabled && myCurrentGames.length > 0 && (
            <ActiveGames
              type="RESUME"
              username={username}
              activeGames={myCurrentGames}
            />
          )}

          {lobbyContext?.matchRequests.length ? (
            <SoughtGames
              isMatch={true}
              userID={userID}
              username={username}
              newGame={newGame}
              requests={lobbyContext?.matchRequests}
            />
          ) : null}

          <SoughtGames
            isMatch={false}
            userID={userID}
            username={username}
            newGame={newGame}
            requests={sanitizedSoughtGames}
            ratings={lobbyContext?.profile?.ratings}
          />
        </>
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
          activeGames={lobbyContext?.activeGames}
        />
      </>
    );
  };
  const resetLobbyFilter = (gameLexicon: string) => {
    const lobbyFilter = localStorage.getItem('lobbyFilterByLexicon');
    if (lobbyFilter && lobbyFilter !== gameLexicon) {
      localStorage.removeItem('lobbyFilterByLexicon');
      dispatchLobbyContext({
        actionType: ActionType.setLobbyFilterByLexicon,
        payload: null,
      });
    }
  };
  const onFormSubmit = (sg: SoughtGame) => {
    setSeekModalVisible(false);
    setMatchModalVisible(false);
    setBotModalVisible(false);
    setFormDisabled(true);
    if (!formDisabled) {
      onSeekSubmit(sg);
      setTimeout(() => {
        setFormDisabled(false);
      }, 500);
    }
    resetLobbyFilter(sg.lexicon);
  };
  const seekModal = (
    <Modal
      title="Create a game"
      className="seek-modal"
      open={seekModalVisible}
      destroyOnClose
      onCancel={() => {
        setSeekModalVisible(false);
        setFormDisabled(false);
      }}
      footer={[
        <Button
          key="back"
          onClick={() => {
            setSeekModalVisible(false);
          }}
        >
          Cancel
        </Button>,
        <button
          className="primary"
          key="submit"
          form="open-seek"
          type="submit"
          disabled={formDisabled}
        >
          Create game
        </button>,
      ]}
    >
      <SeekForm
        id="open-seek"
        onFormSubmit={onFormSubmit}
        loggedIn={props.loggedIn}
        showFriendInput={false}
      />
    </Modal>
  );
  const matchModal = (
    <Modal
      className="seek-modal"
      title="Match a friend"
      open={matchModalVisible}
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
        showFriendInput={true}
        friendRef={friendRef}
        id="match-seek"
      />
    </Modal>
  );
  const botModal = (
    <Modal
      title="Play a computer"
      open={botModalVisible}
      className="seek-modal"
      destroyOnClose
      onCancel={() => {
        setBotModalVisible(false);
        setFormDisabled(false);
      }}
      footer={[
        <Button
          key="back"
          onClick={() => {
            setBotModalVisible(false);
          }}
        >
          Cancel
        </Button>,
        <button
          className="primary"
          key="submit"
          form="bot-seek"
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
        showFriendInput={false}
        vsBot={true}
        id="bot-seek"
      />
    </Modal>
  );
  let showingResumeButton = false;
  const actions = [];
  // if no existing game
  if (loggedIn) {
    if (currentGame && !simultaneousModeEffectivelyEnabled) {
      showingResumeButton = true;
      actions.push(
        <div
          className="resume"
          onClick={() => {
            navigate(`/game/${encodeURIComponent(currentGame.gameID)}`);
            console.log('redirecting to', currentGame.gameID);
          }}
        >
          Resume your game with {opponent}
        </div>
      );
    } else {
      actions.push(
        <div
          className="bot"
          onClick={() => {
            setBotModalVisible(true);
          }}
        >
          Play a computer
        </div>
      );
      actions.push(
        <div
          className="match"
          onClick={() => {
            setMatchModalVisible(true);
          }}
        >
          {matchButtonText}
        </div>
      );

      actions.push(
        <div
          className="seek"
          onClick={() => {
            setSeekModalVisible(true);
          }}
        >
          Create a game
        </div>
      );
    }
  }
  return (
    <div className="game-lists">
      <Card actions={actions}>
        <div className="tabs">
          {loggedIn ? (
            <div
              onClick={() => {
                setSelectedGameTab('PLAY');
              }}
              className={selectedGameTab === 'PLAY' ? 'tab active' : 'tab'}
            >
              Play
            </div>
          ) : null}
          <div
            onClick={() => {
              setSelectedGameTab('WATCH');
            }}
            className={
              selectedGameTab === 'WATCH' || !loggedIn ? 'tab active' : 'tab'
            }
          >
            Watch
          </div>
        </div>
        <div className="main-content">
          {renderGames()}
          {seekModal}
          {matchModal}
          {botModal}
        </div>
        {showingResumeButton && (
          <div className="enable-simultaneous-ignore-link">
            <Link to="/" onClick={handleEnableSimultaneousMode}>
              Ignore
            </Link>
          </div>
        )}
      </Card>
    </div>
  );
});
