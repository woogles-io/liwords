import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Card, message, Popconfirm } from 'antd';
import { HomeOutlined } from '@ant-design/icons/lib';
import axios from 'axios';

import { Link, useHistory, useLocation, useParams } from 'react-router-dom';
import { BoardPanel } from './board_panel';
import { TopBar } from '../topbar/topbar';
import { Chat } from '../chat/chat';
import {
  useChatStoreContext,
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useGameEndMessageStoreContext,
  useLoginStateStoreContext,
  usePoolFormatStoreContext,
  usePresenceStoreContext,
  useRematchRequestStoreContext,
  useResetStoreContext,
  useTimerStoreContext,
} from '../store/store';
import { PlayerCards } from './player_cards';
import Pool from './pool';
import {
  MessageType,
  TimedOut,
  MatchRequest,
  SoughtGameProcessEvent,
  DeclineMatchRequest,
  ChatMessage,
  ReadyForGame,
} from '../gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import './scss/gameroom.scss';
import { ScoreCard } from './scorecard';
import {
  GameInfo,
  GameMetadata,
  PlayerMetadata,
  RecentGamesResponse,
} from './game_info';
import { BoopSounds } from '../sound/boop';
import { toAPIUrl } from '../api/api';
import { StreakWidget } from './streak_widget';
import { PlayState } from '../gen/macondo/api/proto/macondo/macondo_pb';
// import { GameInfoResponse } from '../gen/api/proto/game_service/game_service_pb';

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
};

const StreakFetchDelay = 2000;

const defaultGameInfo = {
  players: new Array<PlayerMetadata>(),
  lexicon: '',
  variant: '',
  initial_time_seconds: 0,
  increment_seconds: 0,
  tournament_name: '',
  challenge_rule: 'VOID' as  // wtf typescript? is there a better way?
    | 'FIVE_POINT'
    | 'TEN_POINT'
    | 'SINGLE'
    | 'DOUBLE'
    | 'TRIPLE'
    | 'VOID',
  rating_mode: 'RATED',
  max_overtime_minutes: 0,
  game_end_reason: 'NONE',
  time_control_name: '',
  // done: false,
};

export const Table = React.memo((props: Props) => {
  const stillMountedRef = React.useRef(true);
  React.useEffect(() => () => void (stillMountedRef.current = false), []);

  const { gameID } = useParams();
  const { chat, clearChat } = useChatStoreContext();
  const {
    gameContext: examinableGameContext,
  } = useExaminableGameContextStoreContext();
  const {
    isExamining,
    handleExamineStart,
    handleExamineGoTo,
  } = useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();
  const { gameEndMessage } = useGameEndMessageStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();
  const { presences } = usePresenceStoreContext();
  const { rematchRequest, setRematchRequest } = useRematchRequestStoreContext();
  const { resetStore } = useResetStoreContext();
  const { pTimedOut, setPTimedOut } = useTimerStoreContext();
  const { username, userID } = loginState;

  const { sendSocketMsg } = props;
  // const location = useLocation();
  const [gameInfo, setGameInfo] = useState<GameMetadata>(defaultGameInfo);
  const [streakGameInfo, setStreakGameInfo] = useState<Array<GameMetadata>>([]);
  const [isObserver, setIsObserver] = useState(false);

  useEffect(() => {
    // Prevent backspace unless we're in an input element. We don't want to
    // leave if we're on Firefox.

    const rx = /INPUT|SELECT|TEXTAREA/i;
    const evtHandler = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement;
      if (e.which === 8) {
        if (
          !rx.test(el.tagName) ||
          (el as HTMLInputElement).disabled ||
          (el as HTMLInputElement).readOnly
        ) {
          e.preventDefault();
        }
      }
    };

    document.addEventListener('keydown', evtHandler);
    document.addEventListener('keypress', evtHandler);

    return () => {
      document.removeEventListener('keydown', evtHandler);
      document.removeEventListener('keypress', evtHandler);
    };
  }, []);

  useEffect(() => {
    if (gameContext.playState === PlayState.GAME_OVER || isObserver) {
      return () => {};
    }

    const evtHandler = (evt: BeforeUnloadEvent) => {
      if (gameContext.playState !== PlayState.GAME_OVER && !isObserver) {
        const msg = 'You are currently in a game!';
        // eslint-disable-next-line no-param-reassign
        evt.returnValue = msg;
        return msg;
      }
      return true;
    };
    window.addEventListener('beforeunload', evtHandler);
    return () => {
      window.removeEventListener('beforeunload', evtHandler);
    };
  }, [gameContext.playState, isObserver]);

  useEffect(() => {
    // Request game API to get info about the game at the beginning.
    axios
      .post<GameMetadata>(
        toAPIUrl('game_service.GameMetadataService', 'GetMetadata'),
        {
          gameId: gameID,
        }
      )
      .then((resp) => {
        if (stillMountedRef.current) {
          setGameInfo(resp.data);
          if (localStorage?.getItem('poolFormat')) {
            setPoolFormat(
              parseInt(localStorage.getItem('poolFormat') || '0', 10)
            );
          }
        }
      });
    BoopSounds.playSound('startgameSound');

    return () => {
      clearChat();
      setGameInfo(defaultGameInfo);
      message.destroy();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameID]);

  useEffect(() => {
    // Request streak info only if a few conditions are true.
    // We want to request it as soon as the original request ID comes in,
    // but only if this is an ongoing game. Also, we want to request it
    // as soon as the game ends (so the streak updates without having to go
    // to a new game).

    if (!gameInfo.original_request_id) {
      return;
    }
    if (gameContext.playState === PlayState.GAME_OVER && !gameEndMessage) {
      // if the game has long been over don't request this. Only request it
      // when we are going to play a game (or observe), or when the game just ended.
      return;
    }
    setTimeout(() => {
      axios
        .post<RecentGamesResponse>(
          toAPIUrl('game_service.GameMetadataService', 'GetRematchStreak'),
          {
            original_request_id: gameInfo.original_request_id,
          }
        )
        .then((streakresp) => {
          if (stillMountedRef.current) {
            setStreakGameInfo(streakresp.data.game_info);
          }
        });
      // Put this on a delay. Otherwise the game might not be saved to the
      // db as having finished before the gameEndMessage comes in.
    }, StreakFetchDelay);

    // Call this when a gameEndMessage comes in, so the streak updates
    // at the end of the game.
  }, [gameInfo.original_request_id, gameEndMessage, gameContext.playState]);

  useEffect(() => {
    if (pTimedOut === undefined) return;
    // Otherwise, player timed out. This will only send once.
    // Observers also send the time out, to clean up noticed abandoned games.

    let timedout = '';

    gameInfo.players.forEach((p) => {
      if (gameContext.uidToPlayerOrder[p.user_id] === pTimedOut) {
        timedout = p.user_id;
      }
    });

    const to = new TimedOut();
    to.setGameId(gameID);
    to.setUserId(timedout);
    console.log('sending timeout to socket');
    sendSocketMsg(
      encodeToSocketFmt(MessageType.TIMED_OUT, to.serializeBinary())
    );
    setPTimedOut(undefined);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pTimedOut, gameContext.nickToPlayerOrder, gameID]);

  useEffect(() => {
    let observer = true;
    gameInfo.players.forEach((p) => {
      if (userID === p.user_id) {
        observer = false;
      }
    });
    setIsObserver(observer);

    // If we are not the observer, tell the server we're ready for the game to start.
    if (gameInfo.game_end_reason === 'NONE' && !observer) {
      const evt = new ReadyForGame();
      evt.setGameId(gameID);
      sendSocketMsg(
        encodeToSocketFmt(MessageType.READY_FOR_GAME, evt.serializeBinary())
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userID, gameInfo]);

  const acceptRematch = useCallback(
    (reqID: string) => {
      const evt = new SoughtGameProcessEvent();
      evt.setRequestId(reqID);
      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.SOUGHT_GAME_PROCESS_EVENT,
          evt.serializeBinary()
        )
      );
    },
    [sendSocketMsg]
  );

  const handleAcceptRematch = useCallback(() => {
    acceptRematch(rematchRequest.getGameRequest()!.getRequestId());
    setRematchRequest(new MatchRequest());
  }, [acceptRematch, rematchRequest, setRematchRequest]);

  const declineRematch = useCallback(
    (reqID: string) => {
      const evt = new DeclineMatchRequest();
      evt.setRequestId(reqID);
      sendSocketMsg(
        encodeToSocketFmt(
          MessageType.DECLINE_MATCH_REQUEST,
          evt.serializeBinary()
        )
      );
    },
    [sendSocketMsg]
  );

  const handleDeclineRematch = useCallback(() => {
    declineRematch(rematchRequest.getGameRequest()!.getRequestId());
    setRematchRequest(new MatchRequest());
  }, [declineRematch, rematchRequest, setRematchRequest]);

  const sendChat = useCallback(
    (msg: string) => {
      const evt = new ChatMessage();
      evt.setMessage(msg);

      const chan = isObserver ? 'gametv' : 'game';
      // XXX: Backend should figure out channels; also separate game and gameTV channels
      // Right now everyone will get this.
      evt.setChannel(`${chan}.${gameID}`);
      sendSocketMsg(
        encodeToSocketFmt(MessageType.CHAT_MESSAGE, evt.serializeBinary())
      );
    },
    [gameID, isObserver, sendSocketMsg]
  );

  // Figure out what rack we should display.
  // If we are one of the players, display our rack.
  // If we are NOT one of the players (so an observer), display the rack of
  // the player on turn.
  let rack;
  const gameDone = gameInfo.game_end_reason !== 'NONE';
  const us = useMemo(
    () => gameInfo.players.find((p) => p.user_id === userID),
    [gameInfo.players, userID]
  );
  if (us && !(gameDone && isExamining)) {
    rack = examinableGameContext.players.find((p) => p.userID === us.user_id)
      ?.currentRack;
  } else {
    rack =
      examinableGameContext.players.find((p) => p.onturn)?.currentRack || '';
  }

  // The game "starts" when the GameHistoryRefresher object comes in via the socket.
  // At that point gameID will be filled in.
  const location = useLocation();
  const searchParams = useMemo(() => new URLSearchParams(location.search), [
    location,
  ]);
  const searchedTurn = useMemo(() => searchParams.get('turn'), [searchParams]);
  const turnAsStr = us && !gameDone ? '' : searchedTurn ?? ''; // Do not examine our current games.
  const hasActivatedExamineRef = useRef(false);
  const [autocorrectURL, setAutocorrectURL] = useState(false);
  useEffect(() => {
    if (gameContext.gameID) {
      if (!hasActivatedExamineRef.current) {
        hasActivatedExamineRef.current = true;
        const turnAsInt = parseInt(turnAsStr, 10);
        if (isFinite(turnAsInt) && turnAsStr === String(turnAsInt)) {
          handleExamineStart();
          handleExamineGoTo(turnAsInt - 1); // ?turn= should start from one.
        }
        setAutocorrectURL(true); // Trigger rerender.
      }
    }
  }, [gameContext.gameID, turnAsStr, handleExamineStart, handleExamineGoTo]);

  // Autocorrect the turn on the URL.
  const history = useHistory();
  useEffect(() => {
    if (!autocorrectURL) return; // Too early if examining has not started.
    const turnParamShouldBe = isExamining
      ? String(examinableGameContext.turns.length + 1)
      : null;
    if (turnParamShouldBe !== searchedTurn) {
      if (turnParamShouldBe == null) {
        searchParams.delete('turn');
      } else {
        searchParams.set('turn', turnParamShouldBe);
      }
      history.push({
        ...location,
        search: String(searchParams),
      });
    }
  }, [
    autocorrectURL,
    examinableGameContext.turns.length,
    history,
    isExamining,
    location,
    searchParams,
    searchedTurn,
  ]);

  return (
    <div className="game-container">
      <TopBar />
      <div className="game-table">
        <div className="chat-area" id="left-sidebar">
          <Card className="left-menu">
            <Link
              to="/"
              onClick={() => {
                resetStore();
              }}
            >
              <HomeOutlined />
              Back to lobby
            </Link>
          </Card>
          <Chat
            chatEntities={chat}
            sendChat={sendChat}
            description={isObserver ? 'Observer chat' : 'Game chat'}
            presences={presences}
            peopleOnlineContext={isObserver ? 'Observers' : 'Players'}
          />
        </div>
        {/* we only put the Popconfirm here so that we can physically place it */}

        <div className="play-area">
          <BoardPanel
            username={username}
            board={examinableGameContext.board}
            currentRack={rack || ''}
            events={examinableGameContext.turns}
            gameID={gameID}
            sendSocketMsg={props.sendSocketMsg}
            gameDone={gameDone}
            playerMeta={gameInfo.players}
          />
          <StreakWidget recentGames={streakGameInfo} />
        </div>
        <div className="data-area" id="right-sidebar">
          <PlayerCards gameMeta={gameInfo} playerMeta={gameInfo.players} />
          <GameInfo meta={gameInfo} />
          <Pool
            pool={examinableGameContext?.pool}
            currentRack={rack || ''}
            poolFormat={poolFormat}
            setPoolFormat={setPoolFormat}
          />
          <Popconfirm
            title={`${rematchRequest
              .getUser()
              ?.getDisplayName()} sent you a rematch request`}
            visible={rematchRequest.getRematchFor() !== ''}
            onConfirm={handleAcceptRematch}
            onCancel={handleDeclineRematch}
            okText="Accept"
            cancelText="Decline"
          />
          <ScoreCard
            username={username}
            playing={us !== undefined}
            events={examinableGameContext.turns}
            board={examinableGameContext.board}
            playerMeta={gameInfo.players}
            poolFormat={poolFormat}
          />
        </div>
      </div>
    </div>
  );
});
