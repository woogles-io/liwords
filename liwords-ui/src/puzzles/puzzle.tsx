import { HomeOutlined } from '@ant-design/icons';
import { Card, message } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { postJsonObj, postProto } from '../api/api';
import { Chat } from '../chat/chat';
import { alphabetFromName } from '../constants/alphabets';
import { TopBar } from '../navigation/topbar';
import {
  useExaminableGameContextStoreContext,
  useGameContextStoreContext,
  useLoginStateStoreContext,
  usePoolFormatStoreContext,
} from '../store/store';
import { BoardPanel } from '../gameroom/board_panel';
import {
  ChallengeRule,
  defaultGameInfo,
  GameInfo,
  GameMetadata,
} from '../gameroom/game_info';
import { PuzzleScore } from './puzzle_score';
import Pool from '../gameroom/pool';
import './puzzles.scss';
import { PuzzleInfo } from './puzzle_info';
import { ActionType } from '../actions/actions';
import {
  PuzzleRequest,
  PuzzleResponse,
} from '../gen/api/proto/puzzle_service/puzzle_service_pb';
import { sortTiles } from '../store/constants';
import { Notepad } from '../gameroom/notepad';

type Props = {
  sendChat: (msg: string, chan: string) => void;
};
// TODO: Delete this after you hook everything up, César
const mockData = {
  attempts: 2,
  // dateSolved: new Date('2022-03-20 00:01:00'),
  dateSolved: undefined,
  challengeRule: 'VOID' as ChallengeRule,
  ratingMode: 'RATED',
  gameDate: new Date('2021-01-01 00:01:00'),
  initial_time_seconds: 3000,
  increment_seconds: 0,
  max_overtime_minutes: 0,
  gameUrl: '/game/abcde',
  lexicon: 'CSW21',
  variantName: 'classic',
  // players aren't needed until after solution is shown
  player1: {
    nickname: 'magrathean',
  },
  player2: {
    nickname: 'RightBehindYou',
  },
};

export const SinglePuzzle = (props: Props) => {
  const { puzzleID } = useParams();
  const [gameInfo, setGameInfo] = useState<GameMetadata>(defaultGameInfo);
  const { loginState } = useLoginStateStoreContext();
  const { username, userID, loggedIn } = loginState;
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
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
    dispatchGameContext({
      actionType: ActionType.ClearHistory,
      payload: 'noclock',
    });
    async function fetchPuzzleData() {
      const req = new PuzzleRequest();
      req.setPuzzleId(puzzleID);
      try {
        const resp = await postProto(
          PuzzleResponse,
          'puzzle_service.PuzzleService',
          'GetPuzzle',
          req
        );
        if (localStorage?.getItem('poolFormat')) {
          setPoolFormat(
            parseInt(localStorage.getItem('poolFormat') || '0', 10)
          );
        }
        dispatchGameContext({
          actionType: ActionType.SetupStaticPosition,
          payload: resp.getHistory(),
        });
      } catch (err) {
        console.log(err, 'is of type', typeof err);
        message.error({
          content: err.message,
          duration: 5,
        });
      }
    }
    fetchPuzzleData();
  }, [dispatchGameContext, puzzleID, setPoolFormat]);

  // add definitions stuff here. We should make common library instead of
  // copy-pasting from table.tsx

  // Figure out what rack we should display
  console.log('gamecontextplayers', gameContext.players);
  const rack = gameContext.players.find((p) => p.onturn)?.currentRack ?? '';
  const sortedRack = useMemo(() => sortTiles(rack), [rack]);
  // Play sound here.

  const alphabet = useMemo(
    () =>
      alphabetFromName(gameInfo.game_request.rules.letter_distribution_name),
    [gameInfo]
  );

  const loadNewPuzzle = useCallback(() => {
    // TODO: César, when I grow up I want to be a callback that loads a new puzzle...
  }, []);

  const showSolution = useCallback(() => {
    // TODO: César, when I grow up I want to be a callback that shows the solution and
    // tells the backend I gave up. Josh said sending show solution endpoint with no attempts
    // should do all that.
  }, []);

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
          <React.Fragment key="not-examining">
            <Notepad includeCard />
          </React.Fragment>
        </div>
        <div className="play-area">
          <BoardPanel
            anonymousViewer={!loggedIn}
            username={username}
            board={gameContext.board}
            currentRack={sortedRack}
            events={gameContext.turns}
            gameID={''} /* no game id for a puzzle */
            sendSocketMsg={() => {}}
            sendGameplayEvent={(evt) => console.log(evt.toObject())}
            gameDone={false}
            playerMeta={gameInfo.players}
            vsBot={false} /* doesn't matter */
            lexicon={gameInfo.game_request.lexicon}
            alphabet={alphabet}
            challengeRule={'SINGLE' as ChallengeRule} /* doesn't matter */
            handleAcceptRematch={() => {}}
            handleAcceptAbort={() => {}}
            puzzleMode
            puzzleSolved={!!mockData.dateSolved}
            // handleSetHover={handleSetHover}   // fix later with definitions.
            // handleUnsetHover={hideDefinitionHover}
            // definitionPopover={definitionPopover}
          />
        </div>

        <div className="data-area" id="right-sidebar">
          <PuzzleScore
            attempts={mockData.attempts}
            dateSolved={mockData.dateSolved}
            loadNewPuzzle={loadNewPuzzle}
            showSolution={showSolution}
          />
          <PuzzleInfo
            solved={!!mockData.dateSolved}
            gameDate={mockData.gameDate}
            gameUrl={mockData.gameUrl}
            lexicon={mockData.lexicon}
            variantName={mockData.variantName}
            player1={mockData.player1}
            player2={mockData.player2}
            ratingMode={mockData.ratingMode}
            challengeRule={mockData.challengeRule}
            initial_time_seconds={mockData.initial_time_seconds}
            increment_seconds={mockData.increment_seconds}
            max_overtime_minutes={mockData.max_overtime_minutes}
          />
          <Pool
            pool={gameContext.pool}
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
