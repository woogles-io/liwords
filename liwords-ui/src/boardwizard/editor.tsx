// boardwizard is our board editor

import { HomeOutlined } from '@ant-design/icons';
import { Button, Card } from 'antd';
import React, { useCallback, useEffect, useMemo } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ActionType } from '../actions/actions';
import { alphabetFromName, runesToUint8Array } from '../constants/alphabets';
import { Analyzer, AnalyzerContextProvider } from '../gameroom/analyzer';
import { BoardPanel } from '../gameroom/board_panel';
import { PlayerCards } from '../gameroom/player_cards';
import Pool from '../gameroom/pool';
import { ScoreCard } from '../gameroom/scorecard';
import { GameRules } from '../gen/api/proto/ipc/omgwords_pb';
import { GameRequest } from '../gen/api/proto/ipc/omgwords_pb';
import { RatingMode } from '../gen/api/proto/ipc/omgwords_pb';
import { PlayerInfo } from '../gen/api/proto/ipc/omgwords_pb';
import { GameInfo } from '../gameroom/game_info';

import {
  ClientGameplayEvent,
  GameDocument_MinimalPlayerInfo,
  PlayerInfo as OMGPlayerInfo,
  ChallengeRule as OMGChallengeRule,
} from '../gen/api/proto/ipc/omgwords_pb';
import { GameDocument } from '../gen/api/proto/ipc/omgwords_pb';
import { GameInfoResponse } from '../gen/api/proto/ipc/omgwords_pb';
import { ChallengeRule as MacondoChallengeRule } from '../gen/api/proto/macondo/macondo_pb';
import { GameEventService } from '../gen/api/proto/omgwords_service/omgwords_connectweb';
import { defaultLetterDistribution } from '../lobby/sought_game_interactions';
import { TopBar } from '../navigation/topbar';
import { sortTiles } from '../store/constants';
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  usePoolFormatStoreContext,
} from '../store/store';
import { useClient, flashError } from '../utils/hooks/connect';
import { useDefinitionAndPhonyChecker } from '../utils/hooks/definitions';
import { useMountedState } from '../utils/mounted';
import { EditorControl } from './editor_control';
import { Notepad, NotepadContextProvider } from '../gameroom/notepad';

const doNothing = () => {};

const blankGamePayload = new GameDocument({
  players: [
    new GameDocument_MinimalPlayerInfo({
      nickname: 'player1',
      userId: 'player1',
    }),
    new GameDocument_MinimalPlayerInfo({
      nickname: 'player2',
      userId: 'player2',
    }),
  ],
});

export const BoardEditor = () => {
  const { useState } = useMountedState();
  const { gameID } = useParams();
  const navigate = useNavigate();

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();

  const { isExamining, handleExamineStart, handleExamineGoTo } =
    useExamineStoreContext();
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { poolFormat, setPoolFormat } = usePoolFormatStoreContext();

  const { handleSetHover, hideDefinitionHover, definitionPopover } =
    useDefinitionAndPhonyChecker({
      addChat: doNothing,
      enableHoverDefine: true,
      gameContext,
      gameDone: true,
      gameID: gameID,
      lexicon: '',
      variant: '',
    });

  const eventClient = useClient(GameEventService);

  // useEffect(() => {
  //   handleExamineStart();
  //   handleExamineGoTo(0);
  // }, [handleExamineGoTo, handleExamineStart]);

  const fetchAndDispatchDocument = useCallback(
    async (gid: string, redirect: boolean) => {
      try {
        const resp = await eventClient.getGameDocument({
          gameId: gid,
        });
        console.log('got a game document, dispatching, redirect is', redirect);
        dispatchGameContext({
          actionType: ActionType.InitFromDocument,
          payload: resp,
        });
        if (redirect) {
          // Also, redirect the URL so we can subscribe to the right channel
          // on the socket.
          navigate(`/editor/${encodeURIComponent(gid)}`, { replace: true });
        }
      } catch (e) {
        flashError(e);
      }
    },
    [dispatchGameContext, eventClient]
  );

  // Initialize on mount with unfinished game, new game, or existing game:
  useEffect(() => {
    if (gameID) {
      fetchAndDispatchDocument(gameID, false);
      return;
    }
    const initFromDoc = async () => {
      let continuedGame;

      try {
        const resp = await eventClient.getMyUnfinishedGames({});
        if (resp.games.length > 0) {
          continuedGame = resp.games[resp.games.length - 1];
        }
      } catch (e) {
        flashError(e);
      }

      if (!continuedGame) {
        // Just dispatch a blank game.
        dispatchGameContext({
          actionType: ActionType.InitFromDocument,
          payload: blankGamePayload,
        });
        return;
      }
      // Otherwise, fetch the game from the server and try to continue it.
      fetchAndDispatchDocument(continuedGame.gameId, true);
    };

    initFromDoc();
  }, [gameID]);

  const sortedRack = useMemo(() => {
    const rack =
      examinableGameContext.players.find((p) => p.onturn)?.currentRack ?? '';
    return sortTiles(rack);
  }, [examinableGameContext]);

  const alphabet = useMemo(
    () => alphabetFromName(gameContext.gameDocument.letterDistribution),
    [gameContext.gameDocument.letterDistribution]
  );

  const changeCurrentRack = async (rack: string, evtIdx: number) => {
    if (evtIdx !== gameContext.turns.length) {
      // We're trying to edit an old event's rack.
      // not onturn here
    }

    const onturn = gameContext.onturn;
    const racks: [Uint8Array, Uint8Array] = [
      new Uint8Array(),
      new Uint8Array(),
    ];
    racks[onturn] = Uint8Array.from(
      runesToUint8Array(rack, gameContext.alphabet)
    );

    try {
      await eventClient.setRacks({
        gameId: gameContext.gameID,
        racks: racks,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const sendGameplayEvent = async (evt: ClientGameplayEvent) => {
    console.log('sendGameplayEvent', evt);
    try {
      await eventClient.sendGameEvent({
        event: evt,
        userId: gameContext.players[gameContext.onturn].userID,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const omgPlayerInfo = (pname: string, idx: number) => {
    const collapsed = pname.replaceAll(' ', '');
    return new OMGPlayerInfo({
      nickname: collapsed,
      fullName: pname,
      userId: collapsed,
      first: idx === 0,
    });
  };

  const createNewGame = async (
    p1name: string,
    p2name: string,
    lex: string,
    chrule: OMGChallengeRule
  ) => {
    // the lexicon and letter distribution are tied together.
    const ld = defaultLetterDistribution(lex);
    try {
      const resp = await eventClient.createBroadcastGame({
        playersInfo: [p1name, p2name].map(omgPlayerInfo),
        lexicon: lex,
        rules: new GameRules({
          boardLayoutName: 'CrosswordGame', // for now
          letterDistributionName: ld,
          variantName: 'classic', // for now
        }),
        challengeRule: chrule,
        public: false,
      });
      fetchAndDispatchDocument(resp.gameId, true);
    } catch (e) {
      flashError(e);
    }
  };

  const editGame = async (
    p1name: string,
    p2name: string,
    description: string
  ) => {
    try {
      await eventClient.patchGameDocument({
        document: new GameDocument({
          players: [p1name, p2name].map((p, idx) => {
            const collapsed = p.replaceAll(' ', '');
            return new GameDocument_MinimalPlayerInfo({
              nickname: collapsed,
              realName: p,
              userId: collapsed,
              quit: gameContext.gameDocument.players[idx].quit,
            });
          }),
          description: description,
          uid: gameContext.gameID,
        }),
      });
    } catch (e) {
      flashError(e);
    }
  };

  const deleteGame = async (gid: string) => {
    try {
      await eventClient.deleteBroadcastGame({ gameId: gid });
      dispatchGameContext({
        actionType: ActionType.InitFromDocument,
        payload: blankGamePayload,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const macChallengeRule = useMemo(
    () => gameContext.gameDocument.challengeRule.valueOf(),
    [gameContext.gameDocument.challengeRule]
  );
  // Create a GameInfoResponse for the purposes of rendering a few of our widgets.
  console.log('macChallengeRule', macChallengeRule);
  const gameInfo = useMemo(() => {
    const d = gameContext.gameDocument;

    return new GameInfoResponse({
      players: d.players.map(
        (p) =>
          new PlayerInfo({
            userId: p.userId,
            nickname: p.nickname,
            fullName: p.realName,
          })
      ),
      timeControlName: 'Annotated',
      tournamentId: '', // maybe can populate from a description later
      gameEndReason: d.endReason,
      scores: d.currentScores,
      winner: d.winner,
      createdAt: d.createdAt,
      gameId: d.uid,
      // no last update
      type: d.type,
      gameRequest: new GameRequest({
        lexicon: d.lexicon,
        rules: new GameRules({
          boardLayoutName: d.boardLayout,
          letterDistributionName: d.letterDistribution,
          variantName: d.variant,
        }),
        incrementSeconds: d.timers?.incrementSeconds,
        challengeRule: macChallengeRule,
        ratingMode: RatingMode.CASUAL,
        requestId: 'none',
        maxOvertimeMinutes: d.timers?.maxOvertime,
        originalRequestId: 'none',
      }),
    });
  }, [gameContext.gameDocument]);

  let ret = (
    <div className="game-container">
      <TopBar />
      <div className="game-table">
        <div className="chat-area" id="left-sidebar">
          <Card className="left-menu">
            <Link to="/">
              <HomeOutlined />
              Back to lobby
            </Link>
          </Card>
          {/* <Chat
            sendChat={props.sendChat}
            defaultChannel={`chat.${isObserver ? 'gametv' : 'game'}.${gameID}`}
            defaultDescription={getChatTitle(playerNames, username, isObserver)}
            tournamentID={gameInfo.tournamentId}
          /> */}
          <Card></Card>
          {isExamining ? (
            <Analyzer
              includeCard
              lexicon={gameContext.gameDocument.lexicon}
              variant={gameContext.gameDocument.variant}
            />
          ) : (
            <React.Fragment key="not-examining">
              <Notepad includeCard />
            </React.Fragment>
          )}

          <Card
            title="Editor controls"
            className="editor-control"
            style={{ marginTop: 12 }}
          >
            <EditorControl
              createNewGame={createNewGame}
              gameID={gameContext.gameID}
              deleteGame={deleteGame}
              editGame={editGame}
            />
          </Card>
        </div>
        <div className="sticky-player-card-container">
          <PlayerCards
            horizontal
            gameMeta={gameInfo}
            playerMeta={gameInfo.players}
            hideProfileLink
          />
        </div>

        <div className="play-area">
          <BoardPanel
            boardEditingMode={true}
            anonymousViewer={false} // tbd
            username={''} // shouldn't matter, but it might have to be some large random string
            board={examinableGameContext.board}
            currentRack={sortedRack}
            events={examinableGameContext.turns}
            gameID={gameContext.gameID}
            sendSocketMsg={doNothing}
            sendGameplayEvent={(evt) => sendGameplayEvent(evt)}
            gameDone={false} // tbd
            playerMeta={gameInfo.players}
            tournamentID={''}
            vsBot={false}
            tournamentPairedMode={false}
            // why does my linter keep overwriting this?
            // eslint-disable-next-line max-len
            lexicon={gameContext.gameDocument.lexicon}
            alphabet={alphabet}
            challengeRule={macChallengeRule}
            handleAcceptRematch={doNothing}
            handleAcceptAbort={doNothing}
            handleSetHover={handleSetHover}
            handleUnsetHover={hideDefinitionHover}
            definitionPopover={definitionPopover}
            changeCurrentRack={changeCurrentRack}
          />
        </div>

        <div className="data-area" id="right-sidebar">
          {/* There are two player cards, css hides one of them. */}
          <PlayerCards
            gameMeta={gameInfo}
            playerMeta={gameInfo.players}
            hideProfileLink
          />
          <GameInfo meta={gameInfo} tournamentName={''} />
          <Pool
            pool={examinableGameContext?.pool}
            currentRack={sortedRack}
            poolFormat={poolFormat}
            setPoolFormat={setPoolFormat}
            alphabet={alphabet}
          />
          <ScoreCard
            isExamining={isExamining}
            lexicon={gameContext.gameDocument.lexicon}
            variant={gameContext.gameDocument.variant}
            events={examinableGameContext.turns}
            board={examinableGameContext.board}
            playerMeta={gameInfo.players}
            poolFormat={poolFormat}
          />
        </div>
      </div>
    </div>
  );
  ret = <NotepadContextProvider children={ret} feRackInfo />;
  ret = <AnalyzerContextProvider children={ret} />;
  return ret;
};
