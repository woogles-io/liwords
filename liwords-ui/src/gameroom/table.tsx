import React, { useEffect, useState } from 'react';
import { Card, message, notification, Popconfirm } from 'antd';
import { HomeOutlined } from '@ant-design/icons/lib';
import axios from 'axios';

import { useParams } from 'react-router-dom';
import { BoardPanel } from './board_panel';
import { TopBar } from '../topbar/topbar';
import { Chat } from '../chat/chat';
import { useStoreContext } from '../store/store';
import { PlayerCards } from './player_cards';
import Pool from './pool';
import {
  MessageType,
  TimedOut,
  MatchRequest,
  SoughtGameProcessEvent,
  DeclineMatchRequest,
  ChatMessage,
} from '../gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import './scss/gameroom.scss';
import { ScoreCard } from './scorecard';
import { GameInfo, GameMetadata, PlayerMetadata } from './game_info';
import { PlayState } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { BoopSounds } from '../sound/boop';
// import { GameInfoResponse } from '../gen/api/proto/game_service/game_service_pb';
type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  username: string;
  loggedIn: boolean;
  connectedToSocket: boolean;
};

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
    | 'VOID',
  rating_mode: 0, // 0 is rated and 1 is casual; see realtime proto.
  max_overtime_minutes: 0,
  done: false,
};

export const Table = React.memo((props: Props) => {
  const { gameID } = useParams();
  const {
    gameContext,
    chat,
    clearChat,
    pTimedOut,
    poolFormat,
    setPoolFormat,
    setPTimedOut,
    rematchRequest,
    setRematchRequest,
    presences,
  } = useStoreContext();
  const { username, sendSocketMsg } = props;
  // const location = useLocation();
  const [gameInfo, setGameInfo] = useState<GameMetadata>(defaultGameInfo);
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
  });

  useEffect(() => {
    // Request game API to get info about the game at the beginning.
    axios
      .post<GameMetadata>(
        '/twirp/game_service.GameMetadataService/GetMetadata',
        {
          gameId: gameID,
        }
      )
      .then((resp) => {
        setGameInfo(resp.data);
      });
    BoopSounds.startgameSound.play();

    return () => {
      clearChat();
      setGameInfo(defaultGameInfo);
      message.destroy();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameID]);

  useEffect(() => {
    if (pTimedOut === undefined) return;
    // Otherwise, player timed out. This will only send once.
    // Send the time out if we're either of both players that are in the game.
    if (isObserver) return;

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
    if (
      gameContext.playState === PlayState.WAITING_FOR_FINAL_PASS &&
      gameContext.nickToPlayerOrder[props.username] === `p${gameContext.onturn}`
    ) {
      notification.info({
        message: 'Pass or challenge?',
        description:
          'Your opponent has played their final tiles. You must pass or challenge.',
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameContext.playState]);

  useEffect(() => {
    let observer = true;
    gameInfo.players.forEach((p) => {
      if (username === p.nickname) {
        observer = false;
      }
    });
    setIsObserver(observer);
  }, [username, gameInfo]);

  const acceptRematch = (reqID: string) => {
    const evt = new SoughtGameProcessEvent();
    evt.setRequestId(reqID);
    sendSocketMsg(
      encodeToSocketFmt(
        MessageType.SOUGHT_GAME_PROCESS_EVENT,
        evt.serializeBinary()
      )
    );
  };

  const declineRematch = (reqID: string) => {
    const evt = new DeclineMatchRequest();
    evt.setRequestId(reqID);
    sendSocketMsg(
      encodeToSocketFmt(
        MessageType.DECLINE_MATCH_REQUEST,
        evt.serializeBinary()
      )
    );
  };

  const sendChat = (msg: string) => {
    const evt = new ChatMessage();
    evt.setMessage(msg);

    const chan = isObserver ? 'gametv' : 'game';
    // XXX: Backend should figure out channels; also separate game and gameTV channels
    // Right now everyone will get this.
    evt.setChannel(`${chan}.${gameID}`);
    sendSocketMsg(
      encodeToSocketFmt(MessageType.CHAT_MESSAGE, evt.serializeBinary())
    );
  };

  // Figure out what rack we should display.
  // If we are one of the players, display our rack.
  // If we are NOT one of the players (so an observer), display the rack of
  // the player on turn.
  let rack;
  const us = gameInfo.players.find((p) => p.nickname === props.username);
  if (us) {
    rack = gameContext.players.find((p) => p.userID === us.user_id)
      ?.currentRack;
  } else {
    rack = gameContext.players.find((p) => p.onturn)?.currentRack || '';
  }

  // The game "starts" when the GameHistoryRefresher object comes in via the socket.

  return (
    <div className="game-container">
      <TopBar
        username={props.username}
        loggedIn={props.loggedIn}
        connectedToSocket={props.connectedToSocket}
      />
      <div className="game-table">
        <div className="chat-area">
          <Card className="left-menu">
            <a href="/">
              <HomeOutlined />
              Back to lobby
            </a>
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
            username={props.username}
            board={gameContext.board}
            showBonusLabels={false}
            currentRack={rack || ''}
            events={gameContext.turns}
            gameID={gameID}
            sendSocketMsg={props.sendSocketMsg}
            gameDone={gameInfo.done}
            playerMeta={gameInfo.players}
          />
        </div>
        <div className="data-area">
          <Popconfirm
            title={`${rematchRequest
              .getUser()
              ?.getDisplayName()} sent you a rematch request`}
            visible={rematchRequest.getRematchFor() !== ''}
            onConfirm={() => {
              acceptRematch(rematchRequest.getGameRequest()!.getRequestId());
              setRematchRequest(new MatchRequest());
            }}
            onCancel={() => {
              declineRematch(rematchRequest.getGameRequest()!.getRequestId());
              setRematchRequest(new MatchRequest());
            }}
            okText="Accept"
            cancelText="Decline"
          />
          <PlayerCards playerMeta={gameInfo.players} />
          <GameInfo meta={gameInfo} />
          <Pool
            pool={gameContext?.pool}
            currentRack={rack || ''}
            poolFormat={poolFormat}
            setPoolFormat={setPoolFormat}
          />
          <ScoreCard
            username={props.username}
            playing={us !== undefined}
            events={gameContext.turns}
            board={gameContext.board}
            playerMeta={gameInfo.players}
          />
        </div>
      </div>
    </div>
  );
});
