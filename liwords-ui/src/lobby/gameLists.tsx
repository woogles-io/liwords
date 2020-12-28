import React from 'react';
import { useHistory } from 'react-router-dom';
import { Card, Modal, Button } from 'antd';
import { useMountedState } from '../utils/mounted';
import { SoughtGames } from './sought_games';
import { ActiveGames } from './active_games';
import { SeekForm } from './seek_form';
import {
  useLobbyStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { ActiveGame, SoughtGame } from '../store/reducers/lobby_reducer';
import './seek_form.scss';

type Props = {
  loggedIn: boolean;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  selectedGameTab: string;
  setSelectedGameTab: (tab: string) => void;
  onSeekSubmit: (g: SoughtGame) => void;
  tournamentID?: string;
};

export const GameLists = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const history = useHistory();

  const {
    loggedIn,
    userID,
    username,
    newGame,
    selectedGameTab,
    setSelectedGameTab,
    onSeekSubmit,
    tournamentID,
  } = props;
  const { lobbyContext } = useLobbyStoreContext();
  const { tournamentContext } = useTournamentStoreContext();
  const [formDisabled, setFormDisabled] = useState(false);
  const [seekModalVisible, setSeekModalVisible] = useState(false);
  const [matchModalVisible, setMatchModalVisible] = useState(false);
  const [botModalVisible, setBotModalVisible] = useState(false);
  const currentGame: ActiveGame | null =
    lobbyContext.activeGames.find((ag) =>
      ag.players.some((p) => p.displayName === username)
    ) || null;
  const opponent = currentGame?.players.find((p) => p.displayName !== username)
    ?.displayName;

  let matchButtonText = 'Match a friend';
  if (tournamentID) {
    console.log('tid', tournamentID, tournamentContext.metadata);
    if (['CLUB', 'CLUBSESSION'].includes(tournamentContext.metadata.type)) {
      matchButtonText = 'Start Club Game';
    } else if (tournamentContext.metadata.type === 'STANDARD') {
      matchButtonText = 'Start Tournament Game';
    }
  }
  const renderGames = () => {
    if (loggedIn && userID && username && selectedGameTab === 'PLAY') {
      return (
        <>
          {lobbyContext?.matchRequests.length ? (
            <SoughtGames
              isMatch={true}
              userID={userID}
              username={username}
              newGame={newGame}
              tournamentID={tournamentID}
              requests={lobbyContext?.matchRequests}
            />
          ) : null}
          {!tournamentID ? (
            <SoughtGames
              isMatch={false}
              userID={userID}
              username={username}
              newGame={newGame}
              requests={lobbyContext?.soughtGames}
            />
          ) : null}
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
  };
  const seekModal = (
    <Modal
      title="Create a Game"
      className="seek-modal"
      visible={seekModalVisible}
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
          Create Game
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
      title={!tournamentID ? 'Match a Friend' : 'Send Match Request'}
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
        tournamentID={tournamentID}
      />
    </Modal>
  );
  const botModal = (
    <Modal
      title="Play a Computer"
      visible={botModalVisible}
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
          Create Game
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
  const actions = [];
  // if no existing game
  if (loggedIn) {
    if (currentGame) {
      actions.push(
        <div
          className="resume"
          onClick={() => {
            history.replace(`/game/${encodeURIComponent(currentGame.gameID)}`);
            console.log('redirecting to', currentGame.gameID);
          }}
        >
          Resume your game with {opponent}
        </div>
      );
    } else {
      // If this is a tournament, only show match modal.
      !tournamentID &&
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
      !tournamentID &&
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
        {renderGames()}
        {seekModal}
        {matchModal}
        {botModal}
      </Card>
    </div>
  );
});
