import { useEffect } from 'react';
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
  const { loggedIn, username, userID } = loginState;
  const { path } = loginState;
  const [badTournament, setBadTournament] = useState(false);
  const [selectedGameTab, setSelectedGameTab] = useState(
    loggedIn ? 'PLAY' : 'WATCH'
  );

  useEffect(() => {
    setSelectedGameTab(loggedIn ? 'PLAY' : 'WATCH');
  }, [loggedIn]);

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
};
