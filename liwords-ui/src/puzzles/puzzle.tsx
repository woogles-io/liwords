import { HomeOutlined } from '@ant-design/icons';
import { Button, Card, Form, message, Modal } from 'antd';
import React, { useCallback, useEffect, useMemo } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { LiwordsAPIError, postProto } from '../api/api';
import { Chat } from '../chat/chat';
import { alphabetFromName } from '../constants/alphabets';
import { TopBar } from '../navigation/topbar';
import {
  useExaminableGameContextStoreContext,
  useGameContextStoreContext,
  useLoginStateStoreContext,
  usePoolFormatStoreContext,
  useTentativeTileContext,
} from '../store/store';
import { BoardPanel } from '../gameroom/board_panel';
import {
  ChallengeRule,
  protoChallengeRuleConvert,
} from '../gameroom/game_info';
import { calculatePuzzleScore, PuzzleScore, renderStars } from './puzzle_score';
import Pool from '../gameroom/pool';
import './puzzles.scss';
import { PuzzleInfo as PuzzleInfoWidget } from './puzzle_info';
import { ActionType } from '../actions/actions';
import {
  PuzzleRequest,
  PuzzleResponse,
  PuzzleStatus,
  NextPuzzleIdRequest,
  NextPuzzleIdResponse,
  SubmissionRequest,
  SubmissionResponse,
  StartPuzzleIdRequest,
  StartPuzzleIdResponse,
} from '../gen/api/proto/puzzle_service/puzzle_service_pb';
import { sortTiles } from '../store/constants';
import { Notepad, NotepadContextProvider } from '../gameroom/notepad';
import { StaticPlayerCards } from './static_player_cards';

import {
  GameEvent,
  GameHistory,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import { excludedLexica, LexiconFormItem } from '../shared/lexicon_display';
import { Store } from 'antd/lib/form/interface';

import {
  ClientGameplayEvent,
  GameInfoResponse,
  RatingMode,
} from '../gen/api/proto/ipc/omgwords_pb';
import { computeLeave } from '../utils/cwgame/game_event';
import { EmptySpace, EphemeralTile } from '../utils/cwgame/common';
import { AnalyzerMove } from '../gameroom/analyzer';
import { useMountedState } from '../utils/mounted';
import { BoopSounds } from '../sound/boop';
import { GameInfoRequest } from '../gen/api/proto/game_service/game_service_pb';
type Props = {
  sendChat: (msg: string, chan: string) => void;
};

type PuzzleInfo = {
  // puzzle parameters:
  attempts: number;
  dateSolved?: Date;
  lexicon: string;
  variantName: string;
  solved: number;
  // game parameters:
  challengeRule?: ChallengeRule;
  ratingMode?: string;
  gameDate?: Date;
  initialTimeSeconds?: number;
  incrementSeconds?: number;
  maxOvertimeMinutes?: number;
  gameUrl?: string;
  player1?: {
    nickname: string;
  };
  player2?: {
    nickname: string;
  };
};

const defaultPuzzleInfo = {
  attempts: 0,
  dateSolved: undefined,
  lexicon: '',
  variantName: '',
  solved: PuzzleStatus.UNANSWERED,
};

export const SinglePuzzle = (props: Props) => {
  const { useState } = useMountedState();
  const { puzzleID } = useParams();
  // const [gameInfo, setGameInfo] = useState<GameMetadata>(defaultGameInfo);
  const [puzzleInfo, setPuzzleInfo] = useState<PuzzleInfo>(defaultPuzzleInfo);
  const [userLexicon, setUserLexicon] = useState<string | undefined>(
    localStorage?.getItem('puzzleLexicon') || undefined
  );
  const [pendingSolution, setPendingSolution] = useState(false);
  const [gameHistory, setGameHistory] = useState<GameHistory | null>(null);
  const [showResponseModalWrong, setShowResponseModalWrong] = useState(false);
  const [showResponseModalCorrect, setShowResponseModalCorrect] =
    useState(false);
  const [showLexiconModal, setShowLexiconModal] = useState(false);
  const { loginState } = useLoginStateStoreContext();
  const { username, loggedIn } = loginState;
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { setDisplayedRack, setPlacedTiles, setPlacedTilesTempScore } =
    useTentativeTileContext();

  const navigate = useNavigate();

  useEffect(() => {
    if (!puzzleID) {
      setShowLexiconModal(true);
    }
  }, [puzzleID]);

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

  // add definitions stuff here. We should make common library instead of
  // copy-pasting from table.tsx

  // Figure out what rack we should display
  const rack = gameContext.players.find((p) => p.onturn)?.currentRack ?? '';
  const sortedRack = useMemo(() => sortTiles(rack), [rack]);
  const userIDOnTurn = useMemo(
    () => gameContext.players.find((p) => p.onturn)?.userID,
    [gameContext]
  );
  // Play sound here.

  const alphabet = useMemo(() => {
    if (gameHistory) {
      return alphabetFromName(
        gameHistory?.getLetterDistribution().toLowerCase()
      );
    }
    return undefined;
  }, [gameHistory]);

  const loadNewPuzzle = useCallback(
    async (firstLoad?: boolean) => {
      if (!userLexicon) {
        return;
      }
      let req, respType, method;
      if (firstLoad === true) {
        req = new StartPuzzleIdRequest();
        respType = StartPuzzleIdResponse;
        method = 'GetStartPuzzleId';
      } else {
        req = new NextPuzzleIdRequest();
        respType = NextPuzzleIdResponse;
        method = 'GetNextPuzzleId';
      }
      req.setLexicon(userLexicon);
      try {
        const resp = await postProto(
          respType,
          'puzzle_service.PuzzleService',
          method,
          req
        );
        console.log('got resp', resp.toObject());
        navigate(`/puzzle/${encodeURIComponent(resp.getPuzzleId())}`);
      } catch (err) {
        message.error({
          content: (err as LiwordsAPIError).message,
          duration: 5,
        });
      }
    },
    [userLexicon, navigate]
  );

  // XXX: This is copied from analyzer.tsx. When we add the analyzer
  // to the puzzle page we should figure out another solution.
  const placeMove = useCallback(
    (move: AnalyzerMove) => {
      const {
        board: { dim, letters },
      } = examinableGameContext;
      const newPlacedTiles = new Set<EphemeralTile>();
      let row = move.row;
      let col = move.col;
      let vertical = move.vertical;
      if (move.isExchange) {
        row = 0;
        col = 0;
        vertical = false;
      }
      for (const t of move.tiles) {
        if (move.isExchange) {
          while (letters[row * dim + col] !== EmptySpace) {
            ++col;
            if (col >= dim) {
              ++row;
              if (row >= dim) {
                // Cannot happen with the standard number of tiles and squares.
                row = dim - 1;
                col = dim - 1;
                break;
              }
              col = 0;
            }
          }
        }
        if (t !== '.') {
          newPlacedTiles.add({
            row,
            col,
            letter: t,
          });
        }
        if (vertical) ++row;
        else ++col;
      }
      setDisplayedRack(move.leaveWithGaps);
      setPlacedTiles(newPlacedTiles);
      setPlacedTilesTempScore(move.score);
    },
    [
      examinableGameContext,
      setDisplayedRack,
      setPlacedTiles,
      setPlacedTilesTempScore,
    ]
  );

  const placeGameEvt = useCallback(
    (evt: GameEvent) => {
      const m = {
        jsonKey: '',
        displayMove: '',
        coordinates: '',
        vertical: evt.getDirection() === GameEvent.Direction.VERTICAL,
        col: evt.getColumn(),
        row: evt.getRow(),
        score: evt.getScore(),
        equity: 0.0, // not shown yet
        tiles: evt.getPlayedTiles() || evt.getExchanged(),
        isExchange: evt.getType() === GameEvent.Type.EXCHANGE,
        leave: '',
        leaveWithGaps: computeLeave(evt.getPlayedTiles(), sortedRack),
      };
      placeMove(m);
    },
    [placeMove, sortedRack]
  );

  const setGameInfo = useCallback(async (gid: string, turnNumber: number) => {
    const req = new GameInfoRequest();
    req.setGameId(gid);
    try {
      const resp = await postProto(
        GameInfoResponse,
        'game_service.GameMetadataService',
        'GetMetadata',
        req
      );
      console.log('got game info', resp.toObject());
      const gameRequest = resp.getGameRequest();
      if (gameRequest) {
        setPuzzleInfo((x) => ({
          ...x,
          challengeRule: protoChallengeRuleConvert(
            gameRequest.getChallengeRule()
          ),
          ratingMode:
            gameRequest?.getRatingMode() === RatingMode.RATED
              ? 'Rated'
              : 'Casual',
          gameDate: resp.getCreatedAt()?.toDate(),
          initialTimeSeconds: gameRequest?.getInitialTimeSeconds(),
          incrementSeconds: gameRequest?.getIncrementSeconds(),
          maxOvertimeMinutes: gameRequest?.getMaxOvertimeMinutes(),
          gameUrl: `/game/${gid}?turn=${turnNumber + 1}`,
          player1: { nickname: resp.getPlayersList()[0].getNickname() },
          player2: { nickname: resp.getPlayersList()[1].getNickname() },
        }));
      }
    } catch (err) {
      message.error({
        content: (err as LiwordsAPIError).message,
        duration: 5,
      });
    }
  }, []);

  const showSolution = useCallback(async () => {
    if (!puzzleID) {
      return;
    }
    const req = new SubmissionRequest();
    req.setShowSolution(true);
    req.setPuzzleId(puzzleID);
    BoopSounds.playSound('puzzleWrongSound');
    console.log('showing solution?', userIDOnTurn, gameContext.players);
    try {
      const resp = await postProto(
        SubmissionResponse,
        'puzzle_service.PuzzleService',
        'SubmitAnswer',
        req
      );
      console.log('got resp', resp.toObject());
      const solution = resp.getCorrectAnswer();
      setPuzzleInfo((x) => ({
        ...x,
        attempts: resp.getAttempts(),
        solved: PuzzleStatus.INCORRECT,
      }));
      // Place the tiles from the event.
      if (solution) {
        placeGameEvt(solution);
      }
      // Also get the game metadata.
      setGameInfo(resp.getGameId(), resp.getTurnNumber());
    } catch (err) {
      message.error({
        content: (err as LiwordsAPIError).message,
        duration: 5,
      });
    }
  }, [puzzleID, userIDOnTurn, gameContext.players, placeGameEvt, setGameInfo]);

  const attemptPuzzle = useCallback(
    async (evt: ClientGameplayEvent) => {
      if (!puzzleID) {
        return;
      }
      const req = new SubmissionRequest();
      req.setAnswer(evt);
      req.setPuzzleId(puzzleID);
      try {
        const resp = await postProto(
          SubmissionResponse,
          'puzzle_service.PuzzleService',
          'SubmitAnswer',
          req
        );
        console.log('got resp', resp.toObject());
        if (resp.getUserIsCorrect()) {
          BoopSounds.playSound('puzzleCorrectSound');
          setGameInfo(resp.getGameId(), resp.getTurnNumber());
          setShowResponseModalCorrect(true);
        } else {
          // Wrong answer
          BoopSounds.playSound('puzzleWrongSound');
          setShowResponseModalWrong(true);
        }
        setPuzzleInfo((x) => ({
          ...x,
          dateSolved:
            resp.getStatus() === PuzzleStatus.CORRECT
              ? resp.getLastAttemptTime()?.toDate()
              : undefined,
          attempts: resp.getAttempts(),
          solved: resp.getStatus(),
        }));
      } catch (err) {
        message.error({
          content: (err as LiwordsAPIError).message,
          duration: 5,
        });
      }
    },
    [puzzleID, setGameInfo]
  );

  useEffect(() => {
    // Request Puzzle API to get info about the puzzle on load if we have an id.
    async function fetchPuzzleData() {
      if (!puzzleID) {
        return;
      }
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
        const gh = resp.getHistory();
        if (gh === null || gh === undefined) {
          throw new Error('Did not receive a valid puzzle position!');
        }
        dispatchGameContext({
          actionType: ActionType.SetupStaticPosition,
          payload: gh,
        });
        setGameHistory(gh);
        console.log('got game history', gh.toObject());
        BoopSounds.playSound('puzzleStartSound');
        setPuzzleInfo({
          attempts: resp.getAttempts(),
          // XXX: add dateSolved to backend, in the meantime...
          dateSolved:
            resp.getStatus() === PuzzleStatus.CORRECT
              ? resp.getLastAttemptTime()?.toDate()
              : undefined,
          lexicon: gh.getLexicon(),
          variantName: gh.getVariant(),
          solved: resp.getStatus(),
        });
        setPendingSolution(true);
      } catch (err) {
        message.error({
          content: (err as LiwordsAPIError).message,
          duration: 5,
        });
      }
    }
    if (puzzleID) {
      console.log('fetching puzzle info');
      dispatchGameContext({
        actionType: ActionType.ClearHistory,
        payload: 'noclock',
      });

      fetchPuzzleData();
    }
  }, [dispatchGameContext, puzzleID, setPoolFormat]);

  useEffect(() => {
    if (userLexicon && !puzzleID) {
      loadNewPuzzle(true);
    }
  }, [loadNewPuzzle, userLexicon, puzzleID]);

  useEffect(() => {
    if (pendingSolution) {
      //TODO: placeGameEvt(??);
    }
    setPendingSolution(false);
  }, [puzzleInfo.solved, pendingSolution, showSolution]);

  // This is displayed if there is no puzzle id and no preferred puzzle lexicon saved in local storage
  const lexiconModal = useMemo(() => {
    if (!puzzleID && !userLexicon) {
      return (
        <Modal
          className="puzzle-lexicon-modal"
          closable={false}
          destroyOnClose
          visible={showLexiconModal}
          title="Welcome to puzzle mode!"
          footer={[
            <button
              disabled={false}
              className="primary"
              form="chooseLexicon"
              key="ok"
              type="submit"
            >
              Start
            </button>,
          ]}
        >
          <Form
            name="chooseLexicon"
            onFinish={(val: Store) => {
              localStorage?.setItem('puzzleLexicon', val.lexicon);
              setUserLexicon(val.lexicon);
            }}
          >
            <LexiconFormItem excludedLexica={excludedLexica(false, false)} />
          </Form>
        </Modal>
      );
    }
    return null;
  }, [puzzleID, showLexiconModal, userLexicon]);

  const responseModalWrong = useMemo(() => {
    return (
      <Modal
        className="response-modal"
        destroyOnClose
        visible={showResponseModalWrong}
        title="Try again!"
        onCancel={() => {
          setShowResponseModalWrong(false);
        }}
        footer={[
          <Button
            key="giveup"
            onClick={() => {
              showSolution();
              setShowResponseModalWrong(false);
            }}
          >
            Give up
          </Button>,
          <Button
            key="ok"
            type="primary"
            onClick={() => {
              setShowResponseModalWrong(false);
            }}
          >
            Keep trying
          </Button>,
        ]}
      >
        <p>
          Sorry, that’s not the correct solution. You have made{' '}
          {puzzleInfo.attempts}{' '}
          {puzzleInfo.attempts === 1 ? 'attempt' : 'attempts'}.
        </p>
      </Modal>
    );
  }, [showResponseModalWrong, puzzleInfo, showSolution]);

  const responseModalCorrect = useMemo(() => {
    //TODO: different title for different scores
    let correctTitle = 'Awesome!';
    switch (puzzleInfo.attempts) {
      case 0:
        correctTitle = 'Awesome!';
        break;
      case 1:
        correctTitle = 'Great job!';
        break;
      case 2:
      default:
        correctTitle = 'Nicely done.';
    }
    const stars = calculatePuzzleScore(true, puzzleInfo.attempts);
    return (
      <Modal
        className="response-modal"
        destroyOnClose
        visible={showResponseModalCorrect}
        title={correctTitle}
        onCancel={() => {
          setShowResponseModalCorrect(false);
        }}
        footer={[
          <button
            disabled={false}
            className="primary"
            key="ok"
            onClick={() => {
              loadNewPuzzle();
            }}
          >
            Next
          </button>,
        ]}
      >
        {renderStars(stars)}
        <p>
          You solved the puzzle in {puzzleInfo.attempts}{' '}
          {puzzleInfo.attempts === 1 ? 'attempt' : 'attempts'}.
        </p>
      </Modal>
    );
  }, [showResponseModalCorrect, puzzleInfo, loadNewPuzzle]);

  const doNothing = useCallback(() => {}, []);
  let ret = (
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
            suppressDefault
          />
          <React.Fragment key="not-examining">
            <Notepad includeCard />
          </React.Fragment>
        </div>
        <div className="play-area">
          {lexiconModal}
          {responseModalWrong}
          {responseModalCorrect}
          {gameHistory?.getLexicon() && alphabet && (
            <BoardPanel
              anonymousViewer={!loggedIn}
              username={username}
              board={gameContext.board}
              currentRack={sortedRack}
              events={gameContext.turns}
              gameID={''} /* no game id for a puzzle */
              sendSocketMsg={doNothing}
              sendGameplayEvent={attemptPuzzle}
              gameDone={false}
              playerMeta={[]}
              vsBot={false} /* doesn't matter */
              lexicon={gameHistory?.getLexicon()}
              alphabet={alphabet}
              challengeRule={'SINGLE' as ChallengeRule} /* doesn't matter */
              handleAcceptRematch={doNothing}
              handleAcceptAbort={doNothing}
              puzzleMode
              puzzleSolved={puzzleInfo.solved}
              // handleSetHover={handleSetHover}   // fix later with definitions.
              // handleUnsetHover={hideDefinitionHover}
              // definitionPopover={definitionPopover}
            />
          )}
        </div>

        <div className="data-area" id="right-sidebar">
          <PuzzleScore
            attempts={puzzleInfo.attempts}
            dateSolved={puzzleInfo.dateSolved}
            loadNewPuzzle={loadNewPuzzle}
            showSolution={showSolution}
            solved={puzzleInfo.solved}
          />
          <PuzzleInfoWidget
            solved={puzzleInfo.solved}
            gameDate={puzzleInfo.gameDate}
            gameUrl={puzzleInfo.gameUrl}
            lexicon={puzzleInfo.lexicon}
            variantName={puzzleInfo.variantName}
            player1={puzzleInfo.player1}
            player2={puzzleInfo.player2}
            ratingMode={puzzleInfo.ratingMode}
            challengeRule={puzzleInfo.challengeRule}
            initial_time_seconds={puzzleInfo.initialTimeSeconds}
            increment_seconds={puzzleInfo.incrementSeconds}
            max_overtime_minutes={puzzleInfo.maxOvertimeMinutes}
          />
          {alphabet && (
            <Pool
              pool={gameContext.pool}
              currentRack={sortedRack}
              poolFormat={poolFormat}
              setPoolFormat={setPoolFormat}
              alphabet={alphabet}
            />
          )}
          <StaticPlayerCards
            playerOnTurn={gameContext.onturn}
            p0Score={gameContext?.players[0]?.score || 0}
            p1Score={gameContext?.players[1]?.score || 0}
          />
        </div>
      </div>
    </div>
  );
  ret = <NotepadContextProvider children={ret} />;
  return ret;
};
