import React from 'react';

import { useCallback, useMemo } from 'react';

import { useParams } from 'react-router-dom';

import {
  useLoginStateStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { useMountedState } from '../utils/mounted';
import { TopBar } from '../topbar/topbar';
import { singularCount } from '../utils/plural';
import { Chat } from '../chat/chat';
import { TournamentInfo } from './tournament_info';
import { sendAccept, sendSeek } from '../lobby/sought_game_interactions';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { ActionsPanel } from './actions_panel';
import { CompetitorStatus } from './competitor_status';
import { readyForTournamentGame } from '../store/reducers/tournament_reducer';
import './room.scss';
import { useTourneyMetadata } from './utils';

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
};

export const TournamentRoom = (props: Props) => {
  const { useState } = useMountedState();

  const { loginState } = useLoginStateStoreContext();
  const {
    tournamentContext,
    dispatchTournamentContext,
  } = useTournamentStoreContext();
  const { loggedIn, username, userID } = loginState;
  const { competitorState: competitorContext } = tournamentContext;
  const { isRegistered } = competitorContext;
  const { sendSocketMsg } = props;
  const { path } = loginState;
  const [badTournament, setBadTournament] = useState(false);
  const [selectedGameTab, setSelectedGameTab] = useState('GAMES');

  useTourneyMetadata(
    path,
    '',
    dispatchTournamentContext,
    loginState,
    setBadTournament
  );

  const tournamentID = useMemo(() => {
    return tournamentContext.metadata.id;
  }, [tournamentContext.metadata]);

  // Should be more like "amdirector"
  const isDirector = useMemo(() => {
    return tournamentContext.metadata.directors.includes(username);
  }, [tournamentContext.metadata, username]);

  const handleNewGame = useCallback(
    (seekID: string) => {
      sendAccept(seekID, sendSocketMsg);
    },
    [sendSocketMsg]
  );
  const onSeekSubmit = useCallback(
    (g: SoughtGame) => {
      sendSeek(g, sendSocketMsg);
    },
    [sendSocketMsg]
  );

  const peopleOnlineContext = useCallback(
    (n: number) => singularCount(n, 'Player', 'Players'),
    []
  );

  if (badTournament) {
    return (
      <>
        <TopBar />
        <div className="lobby">
          <h3>You tried to access a non-existing page.</h3>
        </div>
      </>
    );
  }

  if (!tournamentID) {
    return (
      <>
        <TopBar />
      </>
    );
  }

  return (
    <>
      <TopBar />
      <div className={`lobby room ${isRegistered ? ' competitor' : ''}`}>
        <div className="chat-area">
          <Chat
            sendChat={props.sendChat}
            defaultChannel={`chat.tournament.${tournamentID}`}
            defaultDescription={tournamentContext.metadata.name}
            peopleOnlineContext={peopleOnlineContext}
            highlight={tournamentContext.metadata.directors}
            highlightText="Director"
            tournamentID={tournamentID}
          />
          {isRegistered && (
            <CompetitorStatus
              sendReady={() =>
                readyForTournamentGame(
                  sendSocketMsg,
                  tournamentContext.metadata.id,
                  competitorContext
                )
              }
            />
          )}
        </div>
        <ActionsPanel
          selectedGameTab={selectedGameTab}
          setSelectedGameTab={setSelectedGameTab}
          isDirector={isDirector}
          tournamentID={tournamentID}
          onSeekSubmit={onSeekSubmit}
          loggedIn={loggedIn}
          newGame={handleNewGame}
          username={username}
          userID={userID}
          sendReady={() =>
            readyForTournamentGame(
              sendSocketMsg,
              tournamentContext.metadata.id,
              competitorContext
            )
          }
        />
        <TournamentInfo
          setSelectedGameTab={setSelectedGameTab}
          sendSocketMsg={sendSocketMsg}
        />
      </div>
    </>
  );
};
