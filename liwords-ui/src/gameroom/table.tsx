import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import { Card, message, Popconfirm } from 'antd';
import { HomeOutlined } from '@ant-design/icons';
import axios from 'axios';

import { Link, useSearchParams, useParams } from 'react-router-dom';
import { useFirefoxPatch } from '../utils/hooks/firefox';
import { useDefinitionAndPhonyChecker } from '../utils/hooks/definitions';
import { useMountedState } from '../utils/mounted';
import { BoardPanel } from './board_panel';
import { TopBar } from '../navigation/topbar';
import { Chat } from '../chat/chat';
import {
  useChatStoreContext,
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useGameEndMessageStoreContext,
  useLoginStateStoreContext,
  usePoolFormatStoreContext,
  useRematchRequestStoreContext,
  useTimerStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { PlayerCards } from './player_cards';
import Pool from './pool';
import { encodeToSocketFmt } from '../utils/protobuf';
import './scss/gameroom.scss';
import { ScoreCard } from './scorecard';
import {
  defaultGameInfo,
  GameInfo,
  GameMetadata,
  StreakInfoResponse,
} from './game_info';
import { BoopSounds } from '../sound/boop';
import { toAPIUrl } from '../api/api';
import { StreakWidget } from './streak_widget';
import { PlayState } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { endGameMessageFromGameInfo } from '../store/end_of_game';
import { Notepad, NotepadContextProvider } from './notepad';
import { Analyzer, AnalyzerContextProvider } from './analyzer';
import { isClubType, isPairedMode, sortTiles } from '../store/constants';
import { readyForTournamentGame } from '../store/reducers/tournament_reducer';
import { CompetitorStatus } from '../tournament/competitor_status';
import { MetaEventControl } from './meta_event_control';
import { useTourneyMetadata } from '../tournament/utils';
import { Disclaimer } from './disclaimer';
import { alphabetFromName } from '../constants/alphabets';
import {
  GameEndReason,
  ReadyForGame,
  TimedOut,
} from '../gen/api/proto/ipc/omgwords_pb';
import { MessageType } from '../gen/api/proto/ipc/ipc_pb';
import {
  DeclineSeekRequest,
  SeekRequest,
  SoughtGameProcessEvent,
} from '../gen/api/proto/ipc/omgseeks_pb';

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
};

const StreakFetchDelay = 2000;

const DEFAULT_TITLE = 'Woogles.io';

const ManageWindowTitleAndTurnSound = () => {
  const { gameContext } = useGameContextStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { userID } = loginState;

  const userIDToNick = useMemo(() => {
    const ret: { [key: string]: string } = {};
    for (const userID in gameContext.uidToPlayerOrder) {
      const playerOrder = gameContext.uidToPlayerOrder[userID];
      for (const nick in gameContext.nickToPlayerOrder) {
        if (playerOrder === gameContext.nickToPlayerOrder[nick]) {
          ret[userID] = nick;
          break;
        }
      }
    }
    return ret;
  }, [gameContext.uidToPlayerOrder, gameContext.nickToPlayerOrder]);

  const playerNicks = useMemo(() => {
    return gameContext.players.map((player) => userIDToNick[player.userID]);
  }, [gameContext.players, userIDToNick]);

  const myId = useMemo(() => {
    const myPlayerOrder = gameContext.uidToPlayerOrder[userID];
    // eslint-disable-next-line no-nested-ternary
    return myPlayerOrder === 'p0' ? 0 : myPlayerOrder === 'p1' ? 1 : null;
  }, [gameContext.uidToPlayerOrder, userID]);

  const gameDone =
    gameContext.playState === PlayState.GAME_OVER && !!gameContext.gameID;

  // do not play sound when game ends (e.g. resign) or has not loaded
  const canPlaySound = !gameDone && gameContext.gameID;
  const soundUnlocked = useRef(false);
  useEffect(() => {
    if (canPlaySound) {
      if (!soundUnlocked.current) {
        // ignore first sound
        soundUnlocked.current = true;
        return;
      }

      if (myId === gameContext.onturn) {
        BoopSounds.playSound('oppMoveSound');
      } else {
        BoopSounds.playSound('makeMoveSound');
      }
    } else {
      soundUnlocked.current = false;
    }
  }, [canPlaySound, myId, gameContext.onturn]);

  const desiredTitle = useMemo(() => {
    let title = '';
    if (!gameDone && myId === gameContext.onturn) {
      title += '*';
    }
    let first = true;
    for (let i = 0; i < gameContext.players.length; ++i) {
      // eslint-disable-next-line no-continue
      if (gameContext.players[i].userID === userID) continue;
      if (first) {
        first = false;
      } else {
        title += ' vs ';
      }
      title += playerNicks[i] ?? '?';
      if (!gameDone && myId == null && i === gameContext.onturn) {
        title += '*';
      }
    }
    if (title.length > 0) title += ' - ';
    title += DEFAULT_TITLE;
    return title;
  }, [
    gameContext.onturn,
    gameContext.players,
    gameDone,
    myId,
    playerNicks,
    userID,
  ]);

  useEffect(() => {
    document.title = desiredTitle;
  }, [desiredTitle]);

  useEffect(() => {
    return () => {
      document.title = DEFAULT_TITLE;
    };
  }, []);

  return null;
};

const getChatTitle = (
  playerNames: Array<string> | undefined,
  username: string,
  isObserver: boolean
): string => {
  if (!playerNames) {
    return '';
  }
  if (isObserver) {
    return playerNames.join(' versus ');
  }
  return playerNames.filter((n) => n !== username).shift() || '';
};

export const Table = React.memo((props: Props) => {
  const { useState } = useMountedState();

  const { gameID } = useParams();
  const { addChat } = useChatStoreContext();
  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { isExamining, handleExamineStart, handleExamineGoTo } =
    useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();
  const { gameEndMessage, setGameEndMessage } = useGameEndMessageStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();
  const { rematchRequest, setRematchRequest } = useRematchRequestStoreContext();
  const { pTimedOut, setPTimedOut } = useTimerStoreContext();
  const { username, userID, loggedIn } = loginState;
  const { tournamentContext, dispatchTournamentContext } =
    useTournamentStoreContext();
  const competitorState = tournamentContext.competitorState;
  const isRegistered = competitorState.isRegistered;
  const [playerNames, setPlayerNames] = useState(new Array<string>());
  const { sendSocketMsg } = props;
  const [gameInfo, setGameInfo] = useState<GameMetadata>(defaultGameInfo);
  const [streakGameInfo, setStreakGameInfo] = useState<StreakInfoResponse>({
    streak: [],
    playersInfo: [],
  });
  const [isObserver, setIsObserver] = useState(false);
  const tournamentNonDirectorObserver = useMemo(() => {
    return (
      isObserver &&
      !tournamentContext.directors?.includes(username) &&
      !loginState.perms.includes('adm')
    );
  }, [isObserver, loginState.perms, username, tournamentContext.directors]);
  useFirefoxPatch();

  const gameDone =
    gameContext.playState === PlayState.GAME_OVER && !!gameContext.gameID;

  useEffect(() => {
    if (gameDone || isObserver) {
      return () => {};
    }

    const evtHandler = (evt: BeforeUnloadEvent) => {
      if (!gameDone && !isObserver) {
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
  }, [gameDone, isObserver]);

  useEffect(() => {
    // Request game API to get info about the game at the beginning.
    console.log('gonna fetch metadata, game id is', gameID);
    axios
      .post<GameMetadata>(
        toAPIUrl('game_service.GameMetadataService', 'GetMetadata'),
        {
          gameId: gameID,
        }
      )
      .then((resp) => {
        setGameInfo(resp.data);
        if (localStorage?.getItem('poolFormat')) {
          setPoolFormat(
            parseInt(localStorage.getItem('poolFormat') || '0', 10)
          );
        }
        if (resp.data.game_end_reason !== GameEndReason.NONE) {
          // Basically if we are here, we've reloaded the page after the game
          // ended. We want to synthesize a new GameEnd message
          setGameEndMessage(endGameMessageFromGameInfo(resp.data));
        }
      })
      .catch((err) => {
        message.error({
          content: `Failed to fetch game information; please refresh. (Error: ${err.message})`,
          duration: 10,
        });
      });

    return () => {
      setGameInfo(defaultGameInfo);
      message.destroy('board-messages');
    };
    // React Hook useEffect has missing dependencies: 'setGameEndMessage' and 'setPoolFormat'.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameID]);

  useTourneyMetadata(
    '',
    gameInfo.tournament_id,
    dispatchTournamentContext,
    loginState,
    undefined
  );

  useEffect(() => {
    // Request streak info only if a few conditions are true.
    // We want to request it as soon as the original request ID comes in,
    // but only if this is an ongoing game. Also, we want to request it
    // as soon as the game ends (so the streak updates without having to go
    // to a new game).

    if (!gameInfo.game_request.originalRequestId) {
      return;
    }
    if (gameDone && !gameEndMessage) {
      // if the game has long been over don't request this. Only request it
      // when we are going to play a game (or observe), or when the game just ended.
      return;
    }
    setTimeout(() => {
      axios
        .post<StreakInfoResponse>(
          toAPIUrl('game_service.GameMetadataService', 'GetRematchStreak'),
          {
            original_request_id: gameInfo.game_request.originalRequestId,
          }
        )
        .then((streakresp) => {
          setStreakGameInfo(streakresp.data);
        });
      // Put this on a delay. Otherwise the game might not be saved to the
      // db as having finished before the gameEndMessage comes in.
    }, StreakFetchDelay);

    // Call this when a gameEndMessage comes in, so the streak updates
    // at the end of the game.
  }, [gameInfo.game_request.originalRequestId, gameEndMessage, gameDone]);

  useEffect(() => {
    if (pTimedOut === undefined) return;
    // Otherwise, player timed out. This will only send once.
    // Send the time out if we're either of both players that are in the game.
    if (isObserver) return;
    if (!gameID) return;

    let timedout = '';

    gameInfo.players.forEach((p) => {
      if (gameContext.uidToPlayerOrder[p.user_id] === pTimedOut) {
        timedout = p.user_id;
      }
    });

    const to = new TimedOut();
    to.gameId = gameID;
    to.userId = timedout;
    sendSocketMsg(encodeToSocketFmt(MessageType.TIMED_OUT, to.toBinary()));
    setPTimedOut(undefined);
    // React Hook useEffect has missing dependencies: 'gameContext.uidToPlayerOrder', 'gameInfo.players', 'isObserver', 'sendSocketMsg', and 'setPTimedOut'.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pTimedOut, gameContext.nickToPlayerOrder, gameID]);

  useEffect(() => {
    if (!gameID) return;
    let observer = true;
    gameInfo.players.forEach((p) => {
      if (userID === p.user_id) {
        observer = false;
      }
    });
    setIsObserver(observer);
    setPlayerNames(gameInfo.players.map((p) => p.nickname));
    // If we are not the observer, tell the server we're ready for the game to start.
    if (gameInfo.game_end_reason === GameEndReason.NONE && !observer) {
      const evt = new ReadyForGame();
      evt.gameId = gameID;
      sendSocketMsg(
        encodeToSocketFmt(MessageType.READY_FOR_GAME, evt.toBinary())
      );
    }
    // React Hook useEffect has missing dependencies: 'gameID' and 'sendSocketMsg'.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userID, gameInfo]);

  const enableHoverDefine = gameDone || isObserver;
  const { handleSetHover, hideDefinitionHover, definitionPopover } =
    useDefinitionAndPhonyChecker({
      addChat,
      enableHoverDefine,
      gameContext,
      gameDone,
      gameID,
      lexicon: gameInfo.game_request.lexicon,
      variant: gameInfo.game_request.rules?.variantName,
    });

  const acceptRematch = useCallback(
    (reqID: string) => {
      const evt = new SoughtGameProcessEvent();
      evt.requestId = reqID;
      sendSocketMsg(
        encodeToSocketFmt(MessageType.SOUGHT_GAME_PROCESS_EVENT, evt.toBinary())
      );
    },
    [sendSocketMsg]
  );

  const handleAcceptRematch = useCallback(() => {
    const gr = rematchRequest.gameRequest;
    if (gr) {
      acceptRematch(gr.requestId);
      setRematchRequest(new SeekRequest());
    }
  }, [acceptRematch, rematchRequest, setRematchRequest]);

  const declineRematch = useCallback(
    (reqID: string) => {
      const evt = new DeclineSeekRequest({ requestId: reqID });
      sendSocketMsg(
        encodeToSocketFmt(MessageType.DECLINE_SEEK_REQUEST, evt.toBinary())
      );
    },
    [sendSocketMsg]
  );

  const handleDeclineRematch = useCallback(() => {
    const gr = rematchRequest.gameRequest;
    if (gr) {
      declineRematch(gr.requestId);
      setRematchRequest(new SeekRequest());
    }
  }, [declineRematch, rematchRequest, setRematchRequest]);

  // Figure out what rack we should display.
  // If we are one of the players, display our rack.
  // If we are NOT one of the players (so an observer), display the rack of
  // the player on turn.
  let rack: string;
  const us = useMemo(
    () => gameInfo.players.find((p) => p.user_id === userID),
    [gameInfo.players, userID]
  );
  if (us && !(gameDone && isExamining)) {
    rack =
      examinableGameContext.players.find((p) => p.userID === us.user_id)
        ?.currentRack ?? '';
  } else {
    rack =
      examinableGameContext.players.find((p) => p.onturn)?.currentRack ?? '';
  }
  const sortedRack = useMemo(() => sortTiles(rack), [rack]);

  // The game "starts" when the GameHistoryRefresher object comes in via the socket.
  // At that point gameID will be filled in.

  useEffect(() => {
    // Don't play when loading from history
    if (!gameDone) {
      BoopSounds.playSound('startgameSound');
    }
  }, [gameID, gameDone]);

  const [searchParams, setSearchParams] = useSearchParams();
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
  // Do not autocorrect when NEW_GAME_EVENT redirects to a rematch.
  const canAutocorrectURL = autocorrectURL && gameID === gameContext.gameID;
  useEffect(() => {
    if (!canAutocorrectURL) return; // Too early if examining has not started.
    const turnParamShouldBe = isExamining
      ? String(examinableGameContext.turns.length + 1)
      : null;
    if (turnParamShouldBe !== searchedTurn) {
      if (turnParamShouldBe == null) {
        setSearchParams({}, { replace: true });
      } else {
        setSearchParams({ turn: turnParamShouldBe }, { replace: true });
      }
    }
  }, [
    canAutocorrectURL,
    examinableGameContext.turns.length,
    isExamining,
    searchParams,
    searchedTurn,
    setSearchParams,
  ]);
  const boardTheme = 'board--' + tournamentContext.metadata.boardStyle || '';
  const tileTheme = 'tile--' + tournamentContext.metadata.tileStyle || '';
  const alphabet = useMemo(
    () => alphabetFromName(gameInfo.game_request.rules?.letterDistributionName),
    [gameInfo]
  );
  const showingFinalTurn =
    gameContext.turns.length === examinableGameContext.turns.length;

  const feRackInfo = useMemo(() => {
    // Enable rack info to be available to all widgets all the time,
    // except in some private situations.
    if (gameDone) {
      // If the game is done, it's fine to always allow rack info
      return true;
    }
    // If we are not a director, but are observing, and private analysis is off:
    // if (
    //   tournamentNonDirectorObserver &&
    //   tournamentContext.metadata?.getPrivateAnalysis()
    // ) {
    //   return false;
    // }
    // If we are an anonymous observer, and this is a tournament, don't
    // allow rack info.
    if (!loggedIn && gameInfo.tournament_id) {
      return false;
    }
    return true;
  }, [gameDone, gameInfo.tournament_id, loggedIn]);
  const gameEpilog = useMemo(() => {
    // XXX: this doesn't get updated when game ends, only when refresh?

    return (
      <React.Fragment>
        {showingFinalTurn && (
          <React.Fragment>
            {gameInfo.game_end_reason === GameEndReason.FORCE_FORFEIT && (
              <React.Fragment>
                Game ended in forfeit.{/* XXX: How to get winners? */}
              </React.Fragment>
            )}
            {gameInfo.game_end_reason === GameEndReason.ABORTED && (
              <React.Fragment>
                The game was cancelled. Rating and statistics were not affected.
              </React.Fragment>
            )}
          </React.Fragment>
        )}
      </React.Fragment>
    );
  }, [gameInfo.game_end_reason, showingFinalTurn]);

  if (!gameID) {
    return (
      <div className="game-container">
        These are not the games you are looking for.
      </div>
    );
  }
  let ret = (
    <div className={`game-container${isRegistered ? ' competitor' : ''}`}>
      <ManageWindowTitleAndTurnSound />
      <TopBar tournamentID={gameInfo.tournament_id} />
      <div className={`game-table ${boardTheme} ${tileTheme}`}>
        <div
          className={`chat-area ${
            !isExamining && tournamentContext.metadata.disclaimer
              ? 'has-disclaimer'
              : ''
          }`}
          id="left-sidebar"
        >
          <Card className="left-menu">
            {gameInfo.tournament_id ? (
              <Link to={tournamentContext.metadata?.slug}>
                <HomeOutlined />
                Back to
                {isClubType(tournamentContext.metadata?.type)
                  ? ' Club'
                  : ' Tournament'}
              </Link>
            ) : (
              <Link to="/">
                <HomeOutlined />
                Back to lobby
              </Link>
            )}
          </Card>
          {playerNames.length > 1 ? (
            <Chat
              sendChat={props.sendChat}
              highlight={tournamentContext.directors}
              highlightText="Director"
              defaultChannel={`chat.${
                isObserver ? 'gametv' : 'game'
              }.${gameID}`}
              defaultDescription={getChatTitle(
                playerNames,
                username,
                isObserver
              )}
              tournamentID={gameInfo.tournament_id}
            />
          ) : null}
          {isExamining ? (
            <Analyzer
              includeCard
              lexicon={gameInfo.game_request.lexicon}
              variant={gameInfo.game_request.rules?.variantName}
            />
          ) : (
            <React.Fragment key="not-examining">
              <Notepad includeCard />
              {tournamentContext.metadata.disclaimer && (
                <Disclaimer
                  disclaimer={tournamentContext.metadata.disclaimer}
                  logoUrl={tournamentContext.metadata.logo}
                />
              )}
            </React.Fragment>
          )}
          {isRegistered && (
            <CompetitorStatus
              sendReady={() =>
                readyForTournamentGame(
                  sendSocketMsg,
                  tournamentContext.metadata?.id,
                  competitorState
                )
              }
            />
          )}
        </div>
        {/* There are two player cards, css hides one of them. */}
        <div className="sticky-player-card-container">
          <PlayerCards
            horizontal
            gameMeta={gameInfo}
            playerMeta={gameInfo.players}
          />
        </div>
        <div className="play-area">
          <BoardPanel
            anonymousViewer={!loggedIn}
            username={username}
            board={examinableGameContext.board}
            currentRack={sortedRack}
            events={examinableGameContext.turns}
            gameID={gameID}
            sendSocketMsg={props.sendSocketMsg}
            sendGameplayEvent={(evt) =>
              props.sendSocketMsg(
                encodeToSocketFmt(
                  MessageType.CLIENT_GAMEPLAY_EVENT,
                  evt.toBinary()
                )
              )
            }
            gameDone={gameDone}
            playerMeta={gameInfo.players}
            tournamentID={gameInfo.tournament_id}
            vsBot={gameInfo.game_request.playerVsBot}
            tournamentSlug={tournamentContext.metadata?.slug}
            tournamentPairedMode={isPairedMode(
              tournamentContext.metadata?.type
            )}
            tournamentNonDirectorObserver={tournamentNonDirectorObserver}
            // why does my linter keep overwriting this?
            // eslint-disable-next-line max-len
            tournamentPrivateAnalysis={
              tournamentContext.metadata?.privateAnalysis
            }
            lexicon={gameInfo.game_request.lexicon}
            alphabet={alphabet}
            challengeRule={gameInfo.game_request.challengeRule}
            handleAcceptRematch={
              rematchRequest.rematchFor === gameID ? handleAcceptRematch : null
            }
            handleAcceptAbort={() => {}}
            handleSetHover={handleSetHover}
            handleUnsetHover={hideDefinitionHover}
            definitionPopover={definitionPopover}
          />
          {!gameDone && (
            <MetaEventControl
              sendSocketMsg={props.sendSocketMsg}
              gameID={gameID}
            />
          )}
          <StreakWidget streakInfo={streakGameInfo} />
        </div>
        <div className="data-area" id="right-sidebar">
          {/* There are two competitor cards, css hides one of them. */}
          {isRegistered && (
            <CompetitorStatus
              sendReady={() =>
                readyForTournamentGame(
                  sendSocketMsg,
                  tournamentContext.metadata?.id,
                  competitorState
                )
              }
            />
          )}
          {/* There are two player cards, css hides one of them. */}
          <PlayerCards gameMeta={gameInfo} playerMeta={gameInfo.players} />
          <GameInfo
            meta={gameInfo}
            tournamentName={tournamentContext.metadata?.name}
            colorOverride={tournamentContext.metadata?.color}
            logoUrl={tournamentContext.metadata?.logo}
          />
          <Pool
            pool={examinableGameContext?.pool}
            currentRack={sortedRack}
            poolFormat={poolFormat}
            setPoolFormat={setPoolFormat}
            alphabet={alphabet}
          />
          <Popconfirm
            title={`${rematchRequest.user?.displayName} sent you a rematch request`}
            visible={rematchRequest.rematchFor !== ''}
            onConfirm={handleAcceptRematch}
            onCancel={handleDeclineRematch}
            okText="Accept"
            cancelText="Decline"
          />
          <ScoreCard
            isExamining={isExamining}
            username={username}
            playing={us !== undefined}
            lexicon={gameInfo.game_request.lexicon}
            variant={gameInfo.game_request.rules?.variantName}
            events={examinableGameContext.turns}
            board={examinableGameContext.board}
            playerMeta={gameInfo.players}
            poolFormat={poolFormat}
            gameEpilog={gameEpilog}
          />
        </div>
      </div>
    </div>
  );
  ret = <NotepadContextProvider children={ret} feRackInfo={feRackInfo} />;
  ret = <AnalyzerContextProvider children={ret} />;
  return ret;
});
