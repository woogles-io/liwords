import React, { useEffect, useState } from 'react';
import { Row, Col } from 'antd';
import axios from 'axios';

import { useParams } from 'react-router-dom';
import { BoardPanel } from './board_panel';
import { TopBar } from '../topbar/topbar';
import { Chat } from './chat';
import { useStoreContext } from '../store/store';
import { PlayerCards } from './player_cards';
import Pool from './pool';
import { MessageType, TimedOut } from '../gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from '../utils/protobuf';
import './scss/gameroom.scss';
import { ScoreCard } from './scorecard';
import { GameInfo, GameMetadata } from './game_info';
// import { GameInfoResponse } from '../gen/api/proto/game_service/game_service_pb';

const gutter = 16;
const boardspan = 12;

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  username: string;
  loggedIn: boolean;
};

export const Table = (props: Props) => {
  const { gameID } = useParams();
  const {
    setRedirGame,
    gameContext,
    chat,
    clearChat,
    pTimedOut,
    poolFormat,
    setPoolFormat,
    setPTimedOut,
  } = useStoreContext();
  const { username, sendSocketMsg } = props;
  // const location = useLocation();
  const [gameInfo, setGameInfo] = useState<GameMetadata>({
    players: [],
    lexicon: '',
    variant: '',
    time_control: '',
    tournament_name: '',
    challenge_rule: 'VOID',
    rating_mode: 0, // 0 is rated and 1 is casual; see realtime proto.
    done: false,
  });
  useEffect(() => {
    // Avoid react-router hijacking the back button.
    // If setRedirGame is not defined, then we're SOL I guess.
    setRedirGame ? setRedirGame('') : (() => {})();
  }, [setRedirGame]);

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
    return () => {
      clearChat();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameID]);

  useEffect(() => {
    if (pTimedOut === undefined) return;
    // Otherwise, player timed out. This will only send once.
    // Send the time out if we're either of both players that are in the game.
    let send = false;
    let timedout = '';

    gameInfo.players.forEach((p) => {
      if (gameContext.uidToPlayerOrder[p.user_id] === pTimedOut) {
        timedout = p.user_id;
      }
      if (username === p.nickname) {
        send = true;
      }
    });

    if (!send) return;

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

  return (
    <div>
      <Row>
        <TopBar username={props.username} loggedIn={props.loggedIn} />
      </Row>
      <Row gutter={gutter} className="game-table">
        <Col span={6} className="chat-area">
          <Chat chatEntities={chat} />
        </Col>
        <Col span={boardspan} className="play-area">
          <BoardPanel
            username={props.username}
            board={gameContext.board}
            showBonusLabels={false}
            currentRack={rack || ''}
            lastPlayedLetters={{}}
            gameID={gameID}
            sendSocketMsg={props.sendSocketMsg}
          />
        </Col>
        <Col span={6} className="data-area">
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
            turns={gameContext.turns}
            currentTurn={gameContext.currentTurn}
            board={gameContext.board}
            playerMeta={gameInfo.players}
          />
        </Col>
      </Row>
    </div>
  );
};
