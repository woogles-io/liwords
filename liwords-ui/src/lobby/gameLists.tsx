import React from 'react';
import { SoughtGames } from './sought_games';
import { Card } from 'antd';
import { ActiveGames } from './active_games';

type Props = {
  loggedIn: boolean;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  selectedGameTab: string;
  setSelectedGameTab: (tab: string) => void;
};

export const GameLists = React.memo((props: Props) => {
  const {
    loggedIn,
    userID,
    username,
    newGame,
    selectedGameTab,
    setSelectedGameTab,
  } = props;
  const renderGames = () => {
    if (loggedIn && userID && username && selectedGameTab === 'PLAY') {
      return (
        <SoughtGames userID={userID} username={username} newGame={newGame} />
      );
    }
    return <ActiveGames username={username} />;
  };
  return (
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
    </Card>
  );
});
