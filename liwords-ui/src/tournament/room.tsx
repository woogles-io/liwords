import React from 'react';
import '../App.scss';
import 'antd/dist/antd.css';

import { useCallback, useEffect, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import axios from 'axios';
import { message } from 'antd';
import { clubRedirects } from '../lobby/fixed_seek_controls';
import {
  useLoginStateStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { useMountedState } from '../utils/mounted';
import { TournamentMetadata } from './state';
import { toAPIUrl } from '../api/api';
import { TopBar } from '../topbar/topbar';
import { singularCount } from '../utils/plural';
import { Chat } from '../chat/chat';
import { TournamentInfo } from './tournament_info';
import { sendAccept, sendSeek } from '../lobby/sought_game_interactions';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { ActionsPanel } from './actions_panel';

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
};

export const TournamentRoom = (props: Props) => {
  const { useState } = useMountedState();

  const { partialSlug } = useParams();
  const { loginState } = useLoginStateStoreContext();
  const {
    tournamentContext,
    setTournamentContext,
  } = useTournamentStoreContext();
  const { loggedIn, username } = loginState;
  const { sendSocketMsg } = props;
  const { path } = loginState;
  const [badTournament, setBadTournament] = useState(false);
  const [selectedGameTab, setSelectedGameTab] = useState('GAMES');

  useEffect(() => {
    if (!partialSlug || !path) {
      return;
    }
    // Temporary redirect code:
    if (path.startsWith('/tournament/')) {
      const oldslug = path.substr('/tournament/'.length);
      if (oldslug in clubRedirects) {
        const slug = clubRedirects[oldslug];
        window.location.replace(
          `${window.location.protocol}//${window.location.hostname}${slug}`
        );
      }
    }

    axios
      .post<TournamentMetadata>(
        toAPIUrl(
          'tournament_service.TournamentService',
          'GetTournamentMetadata'
        ),
        {
          slug: path,
        }
      )
      .then((resp) => {
        setTournamentContext({
          metadata: resp.data,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error fetching tournament data',
          duration: 5,
        });
        setBadTournament(true);
      });
  }, [path, partialSlug, setTournamentContext]);

  const tournamentID = useMemo(() => {
    return tournamentContext.metadata.id;
  }, [tournamentContext.metadata]);

  // Should be more like "amdirector"
  const isDirector = useMemo(() => {
    return tournamentContext.metadata.directors.includes(username);
  }, [tournamentContext.metadata]);

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
      <div className="lobby">
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
        />
        <TournamentInfo />
      </div>
    </>
  );
};
