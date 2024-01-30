import React, { useState } from 'react';

import { useCallback, useMemo } from 'react';

import {
  useLoginStateStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { TopBar } from '../navigation/topbar';
import { Chat } from '../chat/chat';
import { TournamentInfo } from './tournament_info';
import { sendAccept, sendSeek } from '../lobby/sought_game_interactions';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { ActionsPanel } from './actions_panel';
import { CompetitorStatus } from './competitor_status';
import { readyForTournamentGame } from '../store/reducers/tournament_reducer';
import './room.scss';
import { useTourneyMetadata } from './utils';
import { useSearchParams } from 'react-router-dom';
import { OwnScoreEnterer } from './enter_own_scores';

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
};

export const TournamentRoom = (props: Props) => {
  const [searchParams] = useSearchParams();

  const { loginState } = useLoginStateStoreContext();
  const { tournamentContext, dispatchTournamentContext } =
    useTournamentStoreContext();
  const { loggedIn, username, userID, perms } = loginState;
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
    return tournamentContext.directors.includes(username);
  }, [tournamentContext.directors, username]);

  const isAdmin = useMemo(() => {
    return perms.includes('adm');
  }, [perms]);

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

  if (searchParams.get('es') != null) {
    return (
      <>
        <OwnScoreEnterer truncatedID={searchParams.get('es') ?? ''} />
        <div style={{ marginTop: 400 }}></div>
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
            highlight={tournamentContext.directors}
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
          isAdmin={isAdmin}
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
