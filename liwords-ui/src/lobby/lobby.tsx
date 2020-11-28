import React, { useCallback, useEffect } from 'react';
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
  ChatMessage,
} from '../gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { ChallengeRuleMap } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { GameLists } from './gameLists';
import { Chat } from '../chat/chat';
import {
  useChatStoreContext,
  useLoginStateStoreContext,
  usePresenceStoreContext,
} from '../store/store';
import { singularCount } from '../utils/plural';
import './lobby.scss';
import { Announcements } from './announcements';
import { toAPIUrl } from '../api/api';
import {
  TournamentInfo,
  TournamentMetadata,
} from '../tournament/tournament_info';

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
  DISCONNECT: () => void;
};

export const Lobby = (props: Props) => {
  const { useState } = useMountedState();
  const { tournamentID } = useParams();
  const { sendSocketMsg } = props;
  const { chat } = useChatStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { presences } = usePresenceStoreContext();
  const { loggedIn, username, userID } = loginState;

  const [tournamentInfo, setTournamentInfo] = useState<TournamentMetadata>({
    name: '',
    description: '',
    directors: [],
  });
  const [selectedGameTab, setSelectedGameTab] = useState(
    loggedIn ? 'PLAY' : 'WATCH'
  );

  useEffect(() => {
    setSelectedGameTab(loggedIn ? 'PLAY' : 'WATCH');
  }, [loggedIn]);

  useEffect(() => {
    if (!tournamentID) {
      return;
    }
    axios
      .post<TournamentMetadata>(
        toAPIUrl(
          'tournament_service.TournamentService',
          'GetTournamentMetadata'
        ),
        {
          id: tournamentID,
        }
      )
      .then((resp) => {
        setTournamentInfo(resp.data);
      })
      .catch((err) => {
        message.error({
          content: 'Tournament does not exist',
          duration: 5,
        });
      });
  }, [tournamentID]);

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

  const sendChat = useCallback(
    (msg: string) => {
      const evt = new ChatMessage();
      evt.setMessage(msg);
      evt.setChannel(
        !tournamentID ? 'chat.lobby' : `chat.tournament.${tournamentID}`
      );
      sendSocketMsg(
        encodeToSocketFmt(MessageType.CHAT_MESSAGE, evt.serializeBinary())
      );
    },
    [sendSocketMsg, tournamentID]
  );
  const peopleOnlineContext = useCallback(
    (n: number) => singularCount(n, 'Player', 'Players'),
    []
  );

  return (
    <>
      <TopBar />
      <div className="lobby">
        <div className="chat-area">
          <Chat
            chatEntities={chat}
            sendChat={sendChat}
            description={tournamentID ? 'Tournament chat' : 'Lobby chat'}
            peopleOnlineContext={peopleOnlineContext}
            presences={presences}
            DISCONNECT={props.DISCONNECT}
            highlight={tournamentInfo.directors}
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
        {tournamentID ? (
          <TournamentInfo
            tournamentID={tournamentID}
            tournamentInfo={tournamentInfo}
          />
        ) : (
          <Announcements />
        )}
      </div>
    </>
  );
};
