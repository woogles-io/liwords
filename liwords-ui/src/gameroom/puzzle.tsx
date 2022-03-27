import { HomeOutlined } from '@ant-design/icons';
import { Card } from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { postJsonObj } from '../api/api';
import { Chat } from '../chat/chat';
import { alphabetFromName } from '../constants/alphabets';
import { TopBar } from '../navigation/topbar';
import {
  useExaminableGameContextStoreContext,
  useLoginStateStoreContext,
  usePoolFormatStoreContext,
} from '../store/store';
import { BoardPanel } from './board_panel';
import { defaultGameInfo, GameInfo, GameMetadata } from './game_info';
import { PlayerCards } from './player_cards';
import Pool from './pool';

type Props = {
  sendChat: (msg: string, chan: string) => void;
};

export const SinglePuzzle = (props: Props) => {
  const { puzzleID } = useParams();
  const [gameInfo, setGameInfo] = useState<GameMetadata>(defaultGameInfo);
  const {
    gameContext: examinableGameContext,
  } = useExaminableGameContextStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { username, userID, loggedIn } = loginState;
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();

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
    // Request Puzzle API to get info about the puzzle on load.
    console.log('fetching puzzle info');

    postJsonObj(
      'puzzle_service.PuzzleService',
      'GetPuzzle',
      {
        puzzle_id: puzzleID,
      },
      (resp) => {
        console.log('got response', resp);
        if (localStorage?.getItem('poolFormat')) {
          setPoolFormat(
            parseInt(localStorage.getItem('poolFormat') || '0', 10)
          );
        }
      }
    );
  }, [puzzleID, setPoolFormat]);

  // add definitions stuff here. We should make common library instead of
  // copy-pasting from table.tsx

  // Figure out what rack we should display
  const sortedRack = 'ABCDEFG';
  // Play sound here.

  const alphabet = useMemo(
    () =>
      alphabetFromName(gameInfo.game_request.rules.letter_distribution_name),
    [gameInfo]
  );

  const ret = (
    <div className="game-container puzzle-container">
      <TopBar />
      <div className="game-table board-- tile--">
        <div className="chat-area" id="left-sidebar">
          <Card className="left-menu">
            <Link to="/">
              <HomeOutlined />
              Back to lobby
            </Link>
          </Card>
          <Chat
            sendChat={props.sendChat}
            defaultChannel="lobby"
            defaultDescription=""
            supressDefault
          />
        </div>
        <div className="play-area">
          <BoardPanel
            anonymousViewer={!loggedIn}
            username={username}
            board={examinableGameContext.board}
            currentRack={sortedRack}
            events={examinableGameContext.turns}
            gameID={''} /* no game id for a puzzle */
            sendSocketMsg={() => {}} // have to overwrite this
            gameDone={false}
            playerMeta={gameInfo.players}
            vsBot={false} /* doesn't matter */
            lexicon={gameInfo.game_request.lexicon}
            alphabet={alphabet}
            challengeRule={
              gameInfo.game_request.challenge_rule
            } /* doesn't matter */
            handleAcceptRematch={() => {}}
            handleAcceptAbort={() => {}}
            // handleSetHover={handleSetHover}   // fix later with definitions.
            // handleUnsetHover={hideDefinitionHover}
            // definitionPopover={definitionPopover}
          />
        </div>

        <div className="data-area" id="right-sidebar">
          <PlayerCards gameMeta={gameInfo} playerMeta={gameInfo.players} />
          <GameInfo meta={gameInfo} tournamentName="" />
          <Pool
            pool={examinableGameContext?.pool}
            currentRack={sortedRack}
            poolFormat={poolFormat}
            setPoolFormat={setPoolFormat}
            alphabet={alphabet}
          />
        </div>
      </div>
    </div>
  );
  return ret;
};
