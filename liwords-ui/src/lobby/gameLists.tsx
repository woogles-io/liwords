import React from 'react';
import { SoughtGames } from './sought_games';
import { Button, Card, Modal } from 'antd';
import { ActiveGames } from './active_games';
import { SeekForm, seekPropVals } from './seek_form';
import { useStoreContext } from '../store/store';

type Props = {
  loggedIn: boolean;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  selectedGameTab: string;
  setSelectedGameTab: (tab: string) => void;
  matchModalVisible: boolean;
  showMatchModal: () => void;
  showSeekModal: () => void;
  seekModalVisible: boolean;
  handleMatchModalCancel: () => void;
  handleMatchModalOk: () => void;
  handleSeekModalCancel: () => void;
  handleSeekModalOk: () => void;
  seekSettings: seekPropVals;
  setSeekSettings: (seekProps: seekPropVals) => void;
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
    matchModalVisible,
    seekModalVisible,
    handleMatchModalCancel,
    handleMatchModalOk,
    handleSeekModalCancel,
    handleSeekModalOk,
    seekSettings,
    setSeekSettings,
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
              onOk={handleSeekModalOk}
              onCancel={handleSeekModalCancel}
            >
              <SeekForm
                vals={seekSettings}
                onChange={setSeekSettings}
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
              onOk={handleMatchModalOk}
              onCancel={handleMatchModalCancel}
            >
              <SeekForm
                vals={seekSettings}
                onChange={setSeekSettings}
                loggedIn={props.loggedIn}
                showFriendInput={true}
              />
            </Modal>
          </div>
        ) : null}
      </Card>
    </div>
  );
});
