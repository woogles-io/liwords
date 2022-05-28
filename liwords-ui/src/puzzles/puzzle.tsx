import { HomeOutlined } from '@ant-design/icons';
import { Card, Form, message, Modal, Select } from 'antd';
import React, { useCallback, useEffect, useMemo } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { LiwordsAPIError, postProto, toAPIUrl } from '../api/api';
import { Chat } from '../chat/chat';
import { alphabetFromName } from '../constants/alphabets';
import { TopBar } from '../navigation/topbar';
import {
  useGameContextStoreContext,
  useLoginStateStoreContext,
  // usePoolFormatStoreContext,
  useTentativeTileContext,
} from '../store/store';
import { BoardPanel } from '../gameroom/board_panel';
import {
  ChallengeRule,
  DefineWordsResponse,
  protoChallengeRuleConvert,
} from '../gameroom/game_info';
import { calculatePuzzleScore, renderStars } from './puzzle_info';
// import Pool from '../gameroom/pool';
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
// Put the player cards back when we have strategy puzzles.
// import { StaticPlayerCards } from './static_player_cards';

import {
  GameEvent,
  GameHistory,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import { MatchLexiconDisplay, puzzleLexica } from '../shared/lexicon_display';
import { Store } from 'antd/lib/form/interface';

import {
  ClientGameplayEvent,
  GameInfoResponse,
  RatingMode,
} from '../gen/api/proto/ipc/omgwords_pb';
import { computeLeave } from '../utils/cwgame/game_event';
import { EphemeralTile } from '../utils/cwgame/common';
import { usePlaceMoveCallback } from '../gameroom/analyzer';
import { useFirefoxPatch } from '../utils/hooks';
import { useMountedState } from '../utils/mounted';
import { BoopSounds } from '../sound/boop';
import { GameInfoRequest } from '../gen/api/proto/game_service/game_service_pb';
import { isLegalPlay } from '../utils/cwgame/scoring';
import { singularCount } from '../utils/plural';
import { getWordsFormed } from '../utils/cwgame/tile_placement';
import axios from 'axios';
import { LearnContextProvider } from '../learn/learn_overlay';
import { PuzzleShareButton } from './puzzle_share';

const doNothing = () => {};

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
  gameId?: string;
  initialTimeSeconds?: number;
  incrementSeconds?: number;
  maxOvertimeMinutes?: number;
  solution?: GameEvent;
  turn?: number;
  puzzleRating?: number;
  userRating?: number;
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
  const [puzzleInfo, setPuzzleInfo] = useState<PuzzleInfo>(defaultPuzzleInfo);
  const [userLexicon, setUserLexicon] = useState<string | undefined>(
    localStorage?.getItem('puzzleLexicon') || undefined
  );
  const [pendingSolution, setPendingSolution] = useState(false);
  const [gameHistory, setGameHistory] = useState<GameHistory | null>(null);
  const [showResponseModalWrong, setShowResponseModalWrong] = useState(false);
  const [checkWordsPending, setCheckWordsPending] = useState(false);
  const [showResponseModalCorrect, setShowResponseModalCorrect] =
    useState(false);
  const [showLexiconModal, setShowLexiconModal] = useState(false);
  const [phoniesPlayed, setPhoniesPlayed] = useState<string[]>([]);
  const [nextPending, setNextPending] = useState(false);
  const { loginState } = useLoginStateStoreContext();
  const { username, loggedIn } = loginState;
  // const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const {
    setDisplayedRack,
    setPlacedTiles,
    setPlacedTilesTempScore,
    placedTiles,
  } = useTentativeTileContext();

  const navigate = useNavigate();

  useEffect(() => {
    if (!puzzleID) {
      setShowLexiconModal(true);
    }
  }, [puzzleID]);

  useFirefoxPatch();

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
        setShowLexiconModal(true);
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
        navigate(`/puzzle/${encodeURIComponent(resp.getPuzzleId())}`, {
          replace: !!firstLoad,
        });
      } catch (err) {
        message.error({
          content: (err as LiwordsAPIError).message,
          duration: 5,
        });
      }
    },
    [userLexicon, navigate]
  );

  useEffect(() => {
    if (nextPending) {
      loadNewPuzzle();
      setNextPending(false);
    }
  }, [loadNewPuzzle, nextPending]);

  // XXX: This is copied from analyzer.tsx. When we add the analyzer
  // to the puzzle page we should figure out another solution.
  const placeMove = usePlaceMoveCallback();

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
        leaveWithGaps: computeLeave(
          evt.getPlayedTiles() || evt.getExchanged(),
          sortedRack
        ),
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
      const answerResponse = resp.getAnswer();
      if (!answerResponse) {
        throw new Error('Did not have an answer!');
      }
      const solution = answerResponse.getCorrectAnswer();
      setPuzzleInfo((x) => ({
        ...x,
        attempts: answerResponse.getAttempts(),
        solved: PuzzleStatus.INCORRECT,
        solution: solution,
        gameId: answerResponse.getGameId(),
        turn: answerResponse.getTurnNumber(),
        puzzleRating: answerResponse.getNewPuzzleRating(),
        userRating: answerResponse.getNewUserRating(),
      }));
      // Place the tiles from the event.
      if (solution) {
        setPendingSolution(true);
      }
      // Also get the game metadata.
    } catch (err) {
      message.error({
        content: (err as LiwordsAPIError).message,
        duration: 5,
      });
    }
  }, [puzzleID, userIDOnTurn, gameContext.players]);

  useEffect(() => {
    if (puzzleInfo.gameId) {
      setGameInfo(puzzleInfo.gameId, puzzleInfo.turn || 0);
    }
  }, [puzzleInfo.gameId, puzzleInfo.turn, setGameInfo]);

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
        const answerResponse = resp.getAnswer();
        if (!answerResponse) {
          throw new Error('Did not have an answer!');
        }
        if (resp.getUserIsCorrect()) {
          BoopSounds.playSound('puzzleCorrectSound');
          setGameInfo(
            answerResponse.getGameId(),
            answerResponse.getTurnNumber()
          );
          setPuzzleInfo((x) => ({
            ...x,
            turn: answerResponse.getTurnNumber(),
            gameId: answerResponse.getGameId(),
            dateSolved:
              answerResponse.getStatus() === PuzzleStatus.CORRECT
                ? answerResponse.getLastAttemptTime()?.toDate()
                : undefined,
            attempts: answerResponse.getAttempts(),
            solved: answerResponse.getStatus(),
            puzzleRating: answerResponse.getNewPuzzleRating(),
            userRating: answerResponse.getNewUserRating(),
          }));
          setShowResponseModalCorrect(true);
        } else {
          // Wrong answer
          BoopSounds.playSound('puzzleWrongSound');
          setShowResponseModalWrong(true);
          setCheckWordsPending(true);
          setPuzzleInfo((x) => ({
            ...x,
            turn: answerResponse.getTurnNumber(),
            gameId: answerResponse.getGameId(),
            dateSolved:
              answerResponse.getStatus() === PuzzleStatus.CORRECT
                ? answerResponse.getLastAttemptTime()?.toDate()
                : undefined,
            attempts: answerResponse.getAttempts(),
            solved: answerResponse.getStatus(),
            puzzleRating: answerResponse.getNewPuzzleRating(),
            userRating: answerResponse.getNewUserRating(),
          }));
        }
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
        /*if (localStorage?.getItem('poolFormat')) {
          setPoolFormat(
            parseInt(localStorage.getItem('poolFormat') || '0', 10)
          );
        }*/
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
        const answerResponse = resp.getAnswer();
        if (!answerResponse) {
          throw new Error('Fetch puzzle returned a null response!');
        }
        if (answerResponse.getStatus() === PuzzleStatus.UNANSWERED) {
          BoopSounds.playSound('puzzleStartSound');
        }
        setPuzzleInfo({
          attempts: answerResponse.getAttempts(),
          // XXX: add dateSolved to backend, in the meantime...
          dateSolved:
            answerResponse.getStatus() === PuzzleStatus.CORRECT
              ? answerResponse.getLastAttemptTime()?.toDate()
              : undefined,
          lexicon: gh.getLexicon(),
          variantName: gh.getVariant(),
          solved: answerResponse.getStatus(),
          solution: answerResponse.getCorrectAnswer(),
          gameId: answerResponse.getGameId(),
          turn: answerResponse.getTurnNumber(),
          puzzleRating: answerResponse.getNewPuzzleRating(),
          userRating: answerResponse.getNewUserRating(),
        });
        setPendingSolution(
          answerResponse.getStatus() !== PuzzleStatus.UNANSWERED
        );
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
  }, [dispatchGameContext, puzzleID]);

  useEffect(() => {
    if (userLexicon && !puzzleID) {
      loadNewPuzzle(true);
    }
  }, [loadNewPuzzle, userLexicon, puzzleID]);

  useEffect(() => {
    if (puzzleInfo.solution && pendingSolution) {
      placeGameEvt(puzzleInfo.solution);
    }
    setPendingSolution(false);
  }, [puzzleInfo.solution, pendingSolution, placeGameEvt]);

  // This is displayed if there is no puzzle id and no preferred puzzle lexicon saved in local storage
  const lexiconModal = useMemo(() => {
    if (!userLexicon) {
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
              if (puzzleID) {
                //This loaded because user tried to go to next with no lexicon. Try again now.
                setNextPending(true);
              }
            }}
          >
            <Form.Item
              label="Dictionary"
              name="lexicon"
              rules={[
                {
                  required: true,
                },
              ]}
            >
              <Select className="puzzle-lexicon-selection" size="large">
                {puzzleLexica.map((k) => (
                  <Select.Option key={k} value={k}>
                    <MatchLexiconDisplay lexiconCode={k} useShortDescription />
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
          </Form>

          <p>More languages are coming soon! Watch for an announcement.</p>
        </Modal>
      );
    }
    return null;
  }, [puzzleID, showLexiconModal, userLexicon]);

  const responseModalWrong = useMemo(() => {
    const reset = () => {
      setDisplayedRack(sortedRack);
      setPlacedTiles(new Set<EphemeralTile>());
      setPlacedTilesTempScore(undefined);
      setPhoniesPlayed([]);
      document.getElementById('board-container')?.focus();
    };
    return (
      <Modal
        className="response-modal"
        destroyOnClose
        visible={showResponseModalWrong}
        title="Try again!"
        onCancel={() => {
          setShowResponseModalWrong(false);
          reset();
        }}
        footer={[
          <button
            key="ok"
            type="submit"
            className="ant-button primary"
            autoFocus
            onClick={() => {
              setShowResponseModalWrong(false);
              reset();
            }}
          >
            Keep trying
          </button>,
        ]}
      >
        <p>
          Sorry, that’s not the correct solution. You have made{' '}
          {singularCount(puzzleInfo.attempts, 'attempt', 'attempts')}.
        </p>
        {phoniesPlayed?.length > 0 && (
          <p className={'invalid-plays'}>{`Invalid words played: ${phoniesPlayed
            .map((x) => `${x}*`)
            .join(', ')}`}</p>
        )}
        {!!puzzleInfo.puzzleRating && !!puzzleInfo.userRating && (
          <>
            <p>The puzzle is now rated {puzzleInfo.puzzleRating}.</p>
            <p>Your puzzle rating is now {puzzleInfo.userRating}.</p>
          </>
        )}
      </Modal>
    );
  }, [
    showResponseModalWrong,
    phoniesPlayed,
    puzzleInfo,
    sortedRack,
    setDisplayedRack,
    setPlacedTiles,
    setPlacedTilesTempScore,
  ]);

  useEffect(() => {
    if (checkWordsPending) {
      const wordsFormed = getWordsFormed(gameContext.board, placedTiles).map(
        (w) => w.toUpperCase()
      );
      setCheckWordsPending(false);
      //Todo: Now run them by the endpoint
      axios
        .post<DefineWordsResponse>(
          toAPIUrl('word_service.WordService', 'DefineWords'),
          {
            lexicon: puzzleInfo.lexicon,
            words: wordsFormed,
            definitions: false,
            anagrams: false,
          }
        )
        .then((resp) => {
          const wordsChecked = resp.data.results;
          const phonies = Object.keys(wordsChecked).filter(
            (w) => !wordsChecked[w].v
          );
          console.log('Phonies played: ', phonies);
          setPhoniesPlayed(phonies);
        });
    }
  }, [checkWordsPending, placedTiles, gameContext.board, puzzleInfo.lexicon]);

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
          <PuzzleShareButton
            key="share"
            puzzleID={puzzleID}
            attempts={puzzleInfo.attempts}
            solved={PuzzleStatus.CORRECT}
          />,
          <button
            autoFocus
            disabled={false}
            className="btn ant-btn primary"
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
          You solved the puzzle in{' '}
          {singularCount(puzzleInfo.attempts, 'attempt', 'attempts')}.
        </p>
        {!!puzzleInfo.puzzleRating && !!puzzleInfo.userRating && (
          <>
            <p>The puzzle is now rated {puzzleInfo.puzzleRating}.</p>
            <p>Your puzzle rating is now {puzzleInfo.userRating}.</p>
          </>
        )}
      </Modal>
    );
  }, [showResponseModalCorrect, puzzleInfo, loadNewPuzzle, puzzleID]);

  const allowAttempt = useMemo(() => {
    return (
      isLegalPlay(Array.from(placedTiles.values()), gameContext.board) &&
      loggedIn &&
      puzzleInfo.solved === PuzzleStatus.UNANSWERED
    );
  }, [placedTiles, gameContext.board, loggedIn, puzzleInfo.solved]);

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
            channelTypeOverride="puzzle"
            suppressDefault
          />
          <React.Fragment key="not-examining">
            <Notepad includeCard />
          </React.Fragment>
        </div>
        <div className="play-area puzzle-area">
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
              sendGameplayEvent={allowAttempt ? attemptPuzzle : doNothing}
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
            attempts={puzzleInfo.attempts}
            dateSolved={puzzleInfo.dateSolved}
            loadNewPuzzle={loadNewPuzzle}
            puzzleID={puzzleID}
            showSolution={showSolution}
            userRating={puzzleInfo.userRating}
            puzzleRating={puzzleInfo.puzzleRating}
          />
          {/* alphabet && (
            <Pool
              pool={gameContext.pool}
              currentRack={sortedRack}
              poolFormat={poolFormat}
              setPoolFormat={setPoolFormat}
              alphabet={alphabet}
            />
          ) */}
          <Notepad includeCard />
          {/*<StaticPlayerCards
            playerOnTurn={gameContext.onturn}
            p0Score={gameContext?.players[0]?.score || 0}
            p1Score={gameContext?.players[1]?.score || 0}
          />*/}
        </div>
      </div>
    </div>
  );
  ret = <NotepadContextProvider children={ret} />;
  ret = <LearnContextProvider children={ret} />;
  return ret;
};
