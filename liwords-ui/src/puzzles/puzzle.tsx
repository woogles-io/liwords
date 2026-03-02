import { HomeOutlined } from "@ant-design/icons";
import { Card, Form, Modal, Select } from "antd";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { Chat } from "../chat/chat";
import {
  alphabetFromName,
  machineWordToRunes,
  runesToMachineWord,
} from "../constants/alphabets";
import { TopBar } from "../navigation/topbar";
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useLoginStateStoreContext,
  // usePoolFormatStoreContext,
  useTentativeTileContext,
} from "../store/store";
import { BoardPanel } from "../gameroom/board_panel";
import { calculatePuzzleScore, renderStars } from "./puzzle_info";
// import Pool from '../gameroom/pool';
import "./puzzles.scss";
import { PuzzleInfo as PuzzleInfoWidget } from "./puzzle_info";
import { ActionType } from "../actions/actions";
import {
  PuzzleStatus,
  NextClosestRatingPuzzleIdRequestSchema,
  StartPuzzleIdRequestSchema,
  SubmissionRequestSchema,
  PuzzleRequestSchema,
} from "../gen/api/proto/puzzle_service/puzzle_service_pb";
import { sortTiles } from "../store/constants";
import { Notepad, NotepadContextProvider } from "../gameroom/notepad";
import {
  Analyzer,
  AnalyzerContextProvider,
  usePlaceMoveCallback,
} from "../gameroom/analyzer";
// Put the player cards back when we have strategy puzzles.
// import { StaticPlayerCards } from './static_player_cards';

import {
  ChallengeRule,
  GameEvent,
  GameHistory,
  GameEvent_Direction,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { MatchLexiconDisplay, puzzleLexica } from "../shared/lexicon_display";
import { Store } from "antd/lib/form/interface";

import {
  ClientGameplayEvent,
  ClientGameplayEvent_EventType,
  RatingMode,
} from "../gen/api/proto/ipc/omgwords_pb";
import { computeLeaveWithGaps } from "../utils/cwgame/game_event";
import { EphemeralTile, MachineLetter } from "../utils/cwgame/common";
import { useFirefoxPatch } from "../utils/hooks/firefox";
import { useDefinitionAndPhonyChecker } from "../utils/hooks/definitions";
import { BoopSounds } from "../sound/boop";
import { GameInfoRequestSchema } from "../gen/api/proto/game_service/game_service_pb";
import { isLegalPlay } from "../utils/cwgame/scoring";
import { singularCount } from "../utils/plural";
import { getWordsFormed } from "../utils/cwgame/tile_placement";
import { LearnContextProvider } from "../learn/learn_overlay";
import { PuzzleShareButton } from "./puzzle_share";
import { RatingsCard } from "./ratings";
import { flashError, useClient } from "../utils/hooks/connect";
import { WordService } from "../gen/api/proto/word_service/word_service_pb";
import { PuzzleService } from "../gen/api/proto/puzzle_service/puzzle_service_pb";
import { GameMetadataService } from "../gen/api/proto/game_service/game_service_pb";
import { create } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";

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
  lexicon: "",
  variantName: "",
  solved: PuzzleStatus.UNANSWERED,
};

export const SinglePuzzle = (props: Props) => {
  const { puzzleID } = useParams();
  const [puzzleInfo, setPuzzleInfo] = useState<PuzzleInfo>(defaultPuzzleInfo);
  const [initialUserRating, setInitialUserRating] = useState<
    number | undefined
  >(undefined);
  const [userLexicon, setUserLexicon] = useState<string | undefined>(
    localStorage?.getItem("puzzleLexicon") || undefined,
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
  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { isExamining } = useExamineStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const {
    setDisplayedRack,
    setPlacedTiles,
    setPlacedTilesTempScore,
    placedTiles,
  } = useTentativeTileContext();

  const navigate = useNavigate();
  const puzzleClient = useClient(PuzzleService);
  const gameMetadataClient = useClient(GameMetadataService);
  useEffect(() => {
    if (!puzzleID) {
      setShowLexiconModal(true);
    }
  }, [puzzleID]);

  useFirefoxPatch();

  // add definitions stuff here.
  const { handleSetHover, hideDefinitionHover, definitionPopover } =
    useDefinitionAndPhonyChecker({
      addChat: doNothing,
      enableHoverDefine: puzzleInfo.solved !== PuzzleStatus.UNANSWERED,
      gameContext, // the final gameContext, not examinableGameContext
      gameDone: false,
      gameID: puzzleInfo.gameId,
      lexicon: puzzleInfo.lexicon,
      variant: puzzleInfo.variantName,
    });

  // Figure out what rack we should display

  const sortedRack = useMemo(() => {
    const rack =
      examinableGameContext.players.find((p) => p.onturn)?.currentRack ??
      new Array<MachineLetter>();
    return sortTiles(rack, examinableGameContext.alphabet);
  }, [examinableGameContext.alphabet, examinableGameContext.players]);
  const userIDOnTurn = useMemo(
    () => examinableGameContext.players.find((p) => p.onturn)?.userID,
    [examinableGameContext],
  );
  // Play sound here.

  const alphabet = useMemo(() => {
    if (gameHistory) {
      return alphabetFromName(gameHistory?.letterDistribution.toLowerCase());
    }
    return undefined;
  }, [gameHistory]);

  const loadNewPuzzle = useCallback(
    async (firstLoad?: boolean) => {
      if (!userLexicon) {
        setShowLexiconModal(true);
        return;
      }

      try {
        let req, resp;
        if (firstLoad === true) {
          req = create(StartPuzzleIdRequestSchema, { lexicon: userLexicon });
          resp = await puzzleClient.getStartPuzzleId(req);
        } else {
          req = create(NextClosestRatingPuzzleIdRequestSchema, {
            lexicon: userLexicon,
          });
          resp = await puzzleClient.getNextClosestRatingPuzzleId(req);
        }

        navigate(`/puzzle/${encodeURIComponent(resp.puzzleId)}`, {
          replace: !!firstLoad,
        });
      } catch (err) {
        flashError(err);
      }
    },
    [userLexicon, navigate, puzzleClient],
  );

  useEffect(() => {
    if (nextPending) {
      loadNewPuzzle();
      setNextPending(false);
    }
  }, [loadNewPuzzle, nextPending]);

  // XXX: tis is all broken.
  // XXX: This is copied from analyzer.tsx. When we add the analyzer
  // to the puzzle page we should figure out another solution.
  const placeMove = usePlaceMoveCallback();

  const placeGameEvt = useCallback(
    (evt: GameEvent) => {
      const m = {
        jsonKey: "",
        displayMove: "",
        coordinates: "",
        vertical: evt.direction === GameEvent_Direction.VERTICAL,
        col: evt.column,
        row: evt.row,
        score: evt.score,
        equity: 0.0, // not shown yet
        tiles: runesToMachineWord(
          evt.playedTiles || evt.exchanged,
          examinableGameContext.alphabet,
        ),
        isExchange: evt.type === GameEvent_Type.EXCHANGE,
        leave: new Array<MachineLetter>(),
        leaveWithGaps: runesToMachineWord(
          computeLeaveWithGaps(
            evt.playedTiles || evt.exchanged,
            machineWordToRunes(sortedRack, examinableGameContext.alphabet),
          ),
          examinableGameContext.alphabet,
        ),
      };
      placeMove(m);
    },
    [placeMove, sortedRack, examinableGameContext.alphabet],
  );

  const setGameInfo = useCallback(
    async (gid: string, turnNumber: number) => {
      const req = create(GameInfoRequestSchema, { gameId: gid });
      try {
        const resp = await gameMetadataClient.getMetadata(req);
        const gameRequest = resp.gameRequest;
        if (gameRequest) {
          setPuzzleInfo((x) => ({
            ...x,
            challengeRule: gameRequest.challengeRule,
            ratingMode:
              gameRequest?.ratingMode === RatingMode.RATED ? "Rated" : "Casual",
            gameDate: resp.createdAt
              ? timestampDate(resp.createdAt)
              : new Date(),
            initialTimeSeconds: gameRequest?.initialTimeSeconds,
            incrementSeconds: gameRequest?.incrementSeconds,
            maxOvertimeMinutes: gameRequest?.maxOvertimeMinutes,
            gameUrl: `/game/${gid}?turn=${turnNumber + 1}`,
            player1: { nickname: resp.players[0].nickname },
            player2: { nickname: resp.players[1].nickname },
          }));
        }
      } catch (err) {
        flashError(err);
      }
    },
    [gameMetadataClient],
  );

  const showSolution = useCallback(async () => {
    if (!puzzleID) {
      return;
    }
    const req = create(SubmissionRequestSchema);
    req.showSolution = true;
    req.puzzleId = puzzleID;
    BoopSounds.playSound("puzzleWrongSound");
    console.log(
      "showing solution?",
      userIDOnTurn,
      examinableGameContext.players,
    );
    try {
      const resp = await puzzleClient.submitAnswer(req);
      const answerResponse = resp.answer;
      if (!answerResponse) {
        throw new Error("Did not have an answer!");
      }
      const solution = answerResponse.correctAnswer;
      setPuzzleInfo((x) => ({
        ...x,
        attempts: answerResponse.attempts,
        solved: PuzzleStatus.INCORRECT,
        solution: solution,
        gameId: answerResponse.gameId,
        turn: answerResponse.turnNumber,
        puzzleRating: answerResponse.newPuzzleRating,
        userRating: answerResponse.newUserRating,
      }));
      // Place the tiles from the event.
      if (solution) {
        setPendingSolution(true);
      }
      // Also get the game metadata.
    } catch (err) {
      flashError(err);
    }
  }, [puzzleID, userIDOnTurn, examinableGameContext.players, puzzleClient]);

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
      if (
        evt.type === ClientGameplayEvent_EventType.TILE_PLACEMENT &&
        !isLegalPlay(
          Array.from(placedTiles.values()),
          examinableGameContext.board,
        )
      ) {
        return;
      }
      const req = create(SubmissionRequestSchema, {
        answer: evt,
        puzzleId: puzzleID,
      });

      try {
        const resp = await puzzleClient.submitAnswer(req);
        const answerResponse = resp.answer;
        if (!answerResponse) {
          throw new Error("Did not have an answer!");
        }
        if (resp.userIsCorrect) {
          BoopSounds.playSound("puzzleCorrectSound");
          setGameInfo(answerResponse.gameId, answerResponse.turnNumber);
          setPuzzleInfo((x) => ({
            ...x,
            turn: answerResponse.turnNumber,
            gameId: answerResponse.gameId,
            dateSolved:
              answerResponse.status === PuzzleStatus.CORRECT &&
              answerResponse.lastAttemptTime
                ? timestampDate(answerResponse.lastAttemptTime)
                : undefined,
            attempts: answerResponse.attempts,
            solved: answerResponse.status,
            puzzleRating: answerResponse.newPuzzleRating,
            userRating: answerResponse.newUserRating,
          }));
          setShowResponseModalCorrect(true);
        } else {
          // Wrong answer
          BoopSounds.playSound("puzzleWrongSound");
          setShowResponseModalWrong(true);
          setCheckWordsPending(true);
          setPuzzleInfo((x) => ({
            ...x,
            turn: answerResponse.turnNumber,
            gameId: answerResponse.gameId,
            dateSolved:
              answerResponse.status === PuzzleStatus.CORRECT &&
              answerResponse.lastAttemptTime
                ? timestampDate(answerResponse.lastAttemptTime)
                : undefined,
            attempts: answerResponse.attempts,
            solved: answerResponse.status,
            puzzleRating: answerResponse.newPuzzleRating,
            userRating: answerResponse.newUserRating,
          }));
        }
      } catch (err) {
        flashError(err);
      }
    },
    [
      placedTiles,
      examinableGameContext.board,
      puzzleID,
      puzzleClient,
      setGameInfo,
    ],
  );

  useEffect(() => {
    // Request Puzzle API to get info about the puzzle on load if we have an id.
    async function fetchPuzzleData() {
      if (!puzzleID) {
        return;
      }
      const req = create(PuzzleRequestSchema, { puzzleId: puzzleID });
      try {
        const resp = await puzzleClient.getPuzzle(req);

        /*if (localStorage?.getItem('poolFormat')) {
          setPoolFormat(
            parseInt(localStorage.getItem('poolFormat') || '0', 10)
          );
        }*/
        const gh = resp.history;
        if (gh === null || gh === undefined) {
          throw new Error("Did not receive a valid puzzle position!");
        }
        dispatchGameContext({
          actionType: ActionType.SetupStaticPosition,
          payload: gh,
        });
        setGameHistory(gh);
        const answerResponse = resp.answer;
        if (!answerResponse) {
          throw new Error("Fetch puzzle returned a null response!");
        }
        if (answerResponse.status === PuzzleStatus.UNANSWERED) {
          BoopSounds.playSound("puzzleStartSound");
        }
        setPuzzleInfo({
          attempts: answerResponse.attempts,
          // XXX: add dateSolved to backend, in the meantime...
          dateSolved:
            answerResponse.status === PuzzleStatus.CORRECT &&
            answerResponse.lastAttemptTime
              ? timestampDate(answerResponse.lastAttemptTime)
              : undefined,
          lexicon: gh.lexicon,
          variantName: gh.variant,
          solved: answerResponse.status,
          solution: answerResponse.correctAnswer,
          gameId: answerResponse.gameId,
          turn: answerResponse.turnNumber,
          puzzleRating: answerResponse.newPuzzleRating,
          userRating: answerResponse.newUserRating,
        });
        setInitialUserRating(answerResponse.newUserRating);
        setPendingSolution(answerResponse.status !== PuzzleStatus.UNANSWERED);
      } catch (err) {
        flashError(err);
      }
    }
    if (puzzleID) {
      dispatchGameContext({
        actionType: ActionType.ClearHistory,
        payload: "noclock",
      });

      fetchPuzzleData();
    }
  }, [dispatchGameContext, puzzleID, puzzleClient]);

  useEffect(() => {
    if (userLexicon && !puzzleID) {
      loadNewPuzzle(true);
    }
  }, [loadNewPuzzle, userLexicon, puzzleID]);

  useEffect(() => {
    const puzzleInfoSolution = puzzleInfo.solution;
    if (puzzleInfoSolution && pendingSolution) {
      setPendingSolution(() => {
        placeGameEvt(puzzleInfoSolution);
        return false;
      });
    }
  }, [puzzleInfo.solution, pendingSolution, placeGameEvt]);

  // This is displayed if there is no puzzle id and no preferred puzzle lexicon saved in local storage
  const lexiconModal = useMemo(() => {
    if (!userLexicon) {
      return (
        <Modal
          className="puzzle-lexicon-modal"
          closable={false}
          destroyOnClose
          open={showLexiconModal}
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
              localStorage?.setItem("puzzleLexicon", val.lexicon);
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
      document.getElementById("board-container")?.focus();
    };
    return (
      <Modal
        className="response-modal"
        destroyOnClose
        open={showResponseModalWrong}
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
          Sorry, that’s not the correct solution. You have made{" "}
          {singularCount(puzzleInfo.attempts, "attempt", "attempts")}.
        </p>
        {phoniesPlayed?.length > 0 && (
          <p className={"invalid-plays"}>{`Invalid words played: ${phoniesPlayed
            .map((x) => `${x}*`)
            .join(", ")}`}</p>
        )}
        {!!puzzleInfo.puzzleRating && !!puzzleInfo.userRating && (
          <>
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

  const wordClient = useClient(WordService);
  useEffect(() => {
    if (checkWordsPending) {
      const wordsFormed = getWordsFormed(
        examinableGameContext.board,
        placedTiles,
        examinableGameContext.alphabet,
      ).map((w) => w.toUpperCase());
      setCheckWordsPending(false);
      //Todo: Now run them by the endpoint

      (async () => {
        const resp = await wordClient.defineWords({
          lexicon: puzzleInfo.lexicon,
          words: wordsFormed,
          definitions: false,
          anagrams: false,
        });
        const wordsChecked = resp.results;
        const phonies = Object.keys(wordsChecked).filter(
          (w) => !wordsChecked[w].v,
        );
        console.log("Phonies played: ", phonies);
        setPhoniesPlayed(phonies);
      })();
    }
  }, [
    checkWordsPending,
    placedTiles,
    examinableGameContext.board,
    examinableGameContext.alphabet,
    puzzleInfo.lexicon,
    wordClient,
  ]);

  const responseModalCorrect = useMemo(() => {
    //TODO: different title for different scores
    let correctTitle = "Awesome!";
    switch (puzzleInfo.attempts) {
      case 0:
        correctTitle = "Awesome!";
        break;
      case 1:
        correctTitle = "Great job!";
        break;
      case 2:
      default:
        correctTitle = "Nicely done.";
    }
    const stars = calculatePuzzleScore(true, puzzleInfo.attempts);
    return (
      <Modal
        className="response-modal"
        destroyOnClose
        open={showResponseModalCorrect}
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
          You solved the puzzle in{" "}
          {singularCount(puzzleInfo.attempts, "attempt", "attempts")}.
        </p>
        {!!puzzleInfo.puzzleRating && !!puzzleInfo.userRating && (
          <>
            <p>Your puzzle rating is now {puzzleInfo.userRating}.</p>
          </>
        )}
      </Modal>
    );
  }, [showResponseModalCorrect, puzzleInfo, loadNewPuzzle, puzzleID]);

  const allowAttempt = useMemo(() => {
    return loggedIn && puzzleInfo.solved === PuzzleStatus.UNANSWERED;
  }, [loggedIn, puzzleInfo.solved]);

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
          {isExamining ? (
            <Analyzer includeCard />
          ) : (
            <React.Fragment key="not-examining">
              <Notepad includeCard />
            </React.Fragment>
          )}
        </div>
        <div className="play-area puzzle-area">
          {lexiconModal}
          {showResponseModalWrong && responseModalWrong}
          {showResponseModalCorrect && responseModalCorrect}
          {gameHistory?.lexicon && alphabet && (
            <BoardPanel
              anonymousViewer={!loggedIn}
              username={username}
              board={examinableGameContext.board}
              currentRack={sortedRack}
              events={examinableGameContext.turns}
              gameID={""} /* no game id for a puzzle */
              sendSocketMsg={doNothing}
              sendGameplayEvent={allowAttempt ? attemptPuzzle : doNothing}
              gameDone={false}
              playerMeta={[]}
              vsBot={false} /* doesn't matter */
              lexicon={gameHistory?.lexicon}
              alphabet={alphabet}
              challengeRule={ChallengeRule.SINGLE} /* doesn't matter */
              handleAcceptRematch={doNothing}
              handleAcceptAbort={doNothing}
              puzzleMode
              puzzleSolved={puzzleInfo.solved}
              handleSetHover={handleSetHover}
              handleUnsetHover={hideDefinitionHover}
              definitionPopover={definitionPopover}
            />
          )}
        </div>

        <div className="data-area" id="right-sidebar">
          <RatingsCard
            userRating={puzzleInfo.userRating || initialUserRating}
            puzzleRating={puzzleInfo.puzzleRating}
            initialUserRating={initialUserRating}
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
            initialTimeSeconds={puzzleInfo.initialTimeSeconds}
            incrementSeconds={puzzleInfo.incrementSeconds}
            maxOvertimeMinutes={puzzleInfo.maxOvertimeMinutes}
            attempts={puzzleInfo.attempts}
            dateSolved={puzzleInfo.dateSolved}
            loadNewPuzzle={loadNewPuzzle}
            puzzleID={puzzleID}
            showSolution={showSolution}
          />
          {/* alphabet && (
            <Pool
              pool={examinableGameContext.pool}
              currentRack={sortedRack}
              poolFormat={poolFormat}
              setPoolFormat={setPoolFormat}
              alphabet={alphabet}
            />
          ) */}
          <Notepad includeCard />
          {/*<StaticPlayerCards
            playerOnTurn={examinableGameContext.onturn}
            p0Score={examinableGameContext?.players[0]?.score || 0}
            p1Score={examinableGameContext?.players[1]?.score || 0}
          />*/}
        </div>
      </div>
    </div>
  );
  ret = <NotepadContextProvider children={ret} feRackInfo />;
  ret = (
    <AnalyzerContextProvider
      children={ret}
      lexicon={puzzleInfo.lexicon}
      variant={puzzleInfo.variantName}
    />
  );
  ret = <LearnContextProvider children={ret} />;
  return ret;
};
