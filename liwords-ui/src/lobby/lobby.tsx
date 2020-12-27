import React, { useCallback, useEffect, useMemo } from 'react';
import { message } from 'antd';
import { useParams } from 'react-router-dom';
import axios from 'axios';
import { useMountedState } from '../utils/mounted';

import { TopBar } from '../topbar/topbar';
import {
  SeekRequest,
  GameRequest,
  MessageType,
  GameRules,
  RatingMode,
  MatchRequest,
  SoughtGameProcessEvent,
} from '../gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { ChallengeRuleMap } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { GameLists } from './gameLists';
import { Chat } from '../chat/chat';
import {
  useLoginStateStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { singularCount } from '../utils/plural';
import './lobby.scss';
import { Announcements } from './announcements';
import { toAPIUrl } from '../api/api';
import { TournamentInfo } from '../tournament/tournament_info';
import { TournamentMetadata } from '../tournament/state';

const sendSeek = (
  game: SoughtGame,
  sendSocketMsg: (msg: Uint8Array) => void
) => {
  const sr = new SeekRequest();
  const mr = new MatchRequest();
  const gr = new GameRequest();
  const rules = new GameRules();
  rules.setBoardLayoutName('CrosswordGame');
  rules.setLetterDistributionName('english');

  gr.setChallengeRule(
    game.challengeRule as ChallengeRuleMap[keyof ChallengeRuleMap]
  );
  gr.setLexicon(game.lexicon);
  gr.setInitialTimeSeconds(game.initialTimeSecs);
  gr.setMaxOvertimeMinutes(game.maxOvertimeMinutes);
  gr.setIncrementSeconds(game.incrementSecs);
  gr.setRules(rules);
  gr.setRatingMode(game.rated ? RatingMode.RATED : RatingMode.CASUAL);
  gr.setPlayerVsBot(game.playerVsBot);

  if (game.receiver.getDisplayName() === '' && game.playerVsBot === false) {
    sr.setGameRequest(gr);

    sendSocketMsg(
      encodeToSocketFmt(MessageType.SEEK_REQUEST, sr.serializeBinary())
    );
  } else {
    // We make it a match request if the receiver is non-empty, or if playerVsBot.
    mr.setGameRequest(gr);
    mr.setReceivingUser(game.receiver);
    mr.setTournamentId(game.tournamentID);
    sendSocketMsg(
      encodeToSocketFmt(MessageType.MATCH_REQUEST, mr.serializeBinary())
    );
  }
};

const sendAccept = (
  seekID: string,
  sendSocketMsg: (msg: Uint8Array) => void
) => {
  // Eventually use the ID.
  const sa = new SoughtGameProcessEvent();
  sa.setRequestId(seekID);
  sendSocketMsg(
    encodeToSocketFmt(
      MessageType.SOUGHT_GAME_PROCESS_EVENT,
      sa.serializeBinary()
    )
  );
};

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
  DISCONNECT: () => void;
};

export const Lobby = (props: Props) => {
  const { useState } = useMountedState();
  const { partialSlug } = useParams();
  console.log('partialSlug is', partialSlug);
  const { sendSocketMsg } = props;
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

  return (
    <>
      <TopBar />
      <div className="lobby">
        <div className="chat-area">
          <Chat
            sendChat={props.sendChat}
            defaultChannel={
              !tournamentID ? 'chat.lobby' : `chat.tournament.${tournamentID}`
            }
            defaultDescription={
              tournamentID ? tournamentContext.metadata.name : 'Lobby'
            }
            peopleOnlineContext={peopleOnlineContext}
            DISCONNECT={props.DISCONNECT}
            highlight={tournamentContext.metadata.directors}
            tournamentID={tournamentID}
          />
        </div>
        <GameLists
          loggedIn={loggedIn}
          userID={userID}
          username={username}
          newGame={handleNewGame}
          selectedGameTab={selectedGameTab}
          setSelectedGameTab={setSelectedGameTab}
          onSeekSubmit={onSeekSubmit}
          tournamentID={tournamentID}
        />
        {tournamentID ? <TournamentInfo /> : <Announcements />}
      </div>
    </>
  );
};
