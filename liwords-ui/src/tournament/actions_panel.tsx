import { Card } from 'antd';
import React from 'react';
import { useMountedState } from '../utils/mounted';
import { DirectorTools } from './director_tools';

type Props = {
  // newGame: (seekID: string) => void;
  // For logged-in users:
  userID?: string;
  username?: string;
  selectedGameTab: string;
  setSelectedGameTab: (tab: string) => void;
  tournamentID: string;
  isDirector: boolean;
};

export const ActionsPanel = React.memo((props: Props) => {
  //const { useState } = useMountedState();
  const { selectedGameTab, setSelectedGameTab, isDirector } = props;
  const renderDirectorTools = () => {
    return <DirectorTools tournamentID={props.tournamentID} />;
  };
  return (
    <div className="game-lists">
      <Card>
        <div className="tabs">
          <div
            onClick={() => {
              setSelectedGameTab('GAMES');
            }}
            className={selectedGameTab === 'GAMES' ? 'tab active' : 'tab'}
          >
            Games
          </div>
          <div
            onClick={() => {
              setSelectedGameTab('STANDINGS');
            }}
            className={selectedGameTab === 'STANDINGS' ? 'tab active' : 'tab'}
          >
            Standings
          </div>
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
      </Card>
    </div>
  );
});
