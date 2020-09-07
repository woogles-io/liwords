import React from 'react';
import { Button, Card, Modal } from 'antd';
import { SoughtGames } from './sought_games';
import { ActiveGames } from './active_games';
import { SeekForm } from './seek_form';
import { useStoreContext } from '../store/store';
import { SoughtGame } from '../store/reducers/lobby_reducer';

type Props = {
  loggedIn: boolean;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  selectedGameTab: string;
  setSelectedGameTab: (tab: string) => void;
  showMatchModal: () => void;
  showSeekModal: () => void;
  showBotModal: () => void;
  matchModalVisible: boolean;
  seekModalVisible: boolean;
  botModalVisible: boolean;
  handleMatchModalCancel: () => void;
  handleSeekModalCancel: () => void;
  handleBotModalCancel: () => void;
  onSeekSubmit: (g: SoughtGame) => void;
};

export const GameLists = React.memo((props: Props) => {
  const {
    loggedIn,
    userID,
    username,
    newGame,
    selectedGameTab,
    setSelectedGameTab,
    showMatchModal,
    showSeekModal,
    showBotModal,
    matchModalVisible,
    seekModalVisible,
    botModalVisible,
    handleMatchModalCancel,
    handleSeekModalCancel,
    handleBotModalCancel,
    onSeekSubmit,
  } = props;
  const { lobbyContext } = useStoreContext();
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
              requests={lobbyContext?.matchRequests}
            />
          ) : null}
          <SoughtGames
            isMatch={false}
            userID={userID}
            username={username}
            newGame={newGame}
            requests={lobbyContext?.soughtGames}
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

  return (
    <div className="game-lists">
      <Card>
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
        {loggedIn && selectedGameTab === 'PLAY' ? (
          <div className="requests">
            <Button type="primary" onClick={showSeekModal}>
              New Game
            </Button>
            <Modal
              title="Seek New Game"
              visible={seekModalVisible}
              onCancel={handleSeekModalCancel}
              footer={[
                <Button key="back" onClick={handleSeekModalCancel}>
                  Cancel
                </Button>,
              ]}
            >
              <SeekForm
                onFormSubmit={onSeekSubmit}
                loggedIn={props.loggedIn}
                showFriendInput={false}
              />
            </Modal>

            <Button type="primary" onClick={showMatchModal}>
              Match a Friend
            </Button>
            <Modal
              title="Match a Friend"
              visible={matchModalVisible}
              onCancel={handleMatchModalCancel}
              footer={[
                <Button key="back" onClick={handleMatchModalCancel}>
                  Cancel
                </Button>,
              ]}
            >
              <SeekForm
                onFormSubmit={onSeekSubmit}
                loggedIn={props.loggedIn}
                showFriendInput={true}
              />
            </Modal>

            <Button type="primary" onClick={showBotModal}>
              Play a Bot
            </Button>
            <Modal
              title="Play a Bot"
              visible={botModalVisible}
              onCancel={handleBotModalCancel}
              footer={[
                <Button key="back" onClick={handleBotModalCancel}>
                  Cancel
                </Button>,
              ]}
            >
              <SeekForm
                onFormSubmit={onSeekSubmit}
                loggedIn={props.loggedIn}
                showFriendInput={false}
                vsBot={true}
              />
            </Modal>
          </div>
        ) : null}
      </Card>
    </div>
  );
});
