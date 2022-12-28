// boardwizard is our board editor

import { HomeOutlined } from '@ant-design/icons';
import { Card } from 'antd';
import React, { useEffect, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { ActionType } from '../actions/actions';
import { alphabetFromName } from '../constants/alphabets';
import { Analyzer } from '../gameroom/analyzer';
import { BoardPanel } from '../gameroom/board_panel';
import { defaultGameInfo, GameInfo } from '../gameroom/game_info';
import { PlayerCards } from '../gameroom/player_cards';
import Pool from '../gameroom/pool';
import { ScoreCard } from '../gameroom/scorecard';
import { GameRules } from '../gen/api/proto/ipc/omgwords_pb';
import {
  ClientGameplayEvent,
  GameDocument_MinimalPlayerInfo,
  PlayerInfo as OMGPlayerInfo,
  ChallengeRule as OMGChallengeRule,
} from '../gen/api/proto/ipc/omgwords_pb';
import { GameDocument } from '../gen/api/proto/ipc/omgwords_pb';
import { GameInfoResponse } from '../gen/api/proto/ipc/omgwords_pb';
import { ChallengeRule } from '../gen/api/proto/macondo/macondo_pb';
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

const doNothing = () => {};

export const BoardEditor = () => {
  const { useState } = useMountedState();
  const [gameInfo, setGameInfo] = useState<GameInfoResponse>(defaultGameInfo);

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
      gameID: undefined,
      lexicon: gameInfo.gameRequest?.lexicon ?? '',
      variant: gameInfo.gameRequest?.rules?.variantName,
    });

  const eventClient = useClient(GameEventService);

  // useEffect(() => {
  //   handleExamineStart();
  //   handleExamineGoTo(0);
  // }, [handleExamineGoTo, handleExamineStart]);

  useEffect(() => {
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
          payload: new GameDocument({
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
          }),
        });
        return;
      }
      // Otherwise, fetch the game from the server and try to continue it.
      try {
        const resp = await eventClient.getGameDocument({
          gameId: continuedGame.gameId,
        });
        dispatchGameContext({
          actionType: ActionType.InitFromDocument,
          payload: resp,
        });
      } catch (e) {
        flashError(e);
      }
    };

    initFromDoc();
  }, [dispatchGameContext]);

  const sortedRack = useMemo(() => {
    const rack =
      examinableGameContext.players.find((p) => p.onturn)?.currentRack ?? '';
    return sortTiles(rack);
  }, [examinableGameContext]);

  const alphabet = useMemo(
    () => alphabetFromName(gameInfo.gameRequest?.rules?.letterDistributionName),
    [gameInfo]
  );

  const changeCurrentRack = (rack: string) => {
    dispatchGameContext({
      actionType: ActionType.ChangePlayerRack,
      payload: {
        rack: rack,
      },
    });
  };

  const sendGameplayEvent = (evt: ClientGameplayEvent) => {
    eventClient.createBroadcastGame;
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
        playersInfo: [p1name, p2name].map((v, idx) => {
          const collapsed = v.replaceAll(' ', '');
          return new OMGPlayerInfo({
            nickname: collapsed,
            fullName: v,
            userId: collapsed,
            first: idx === 0,
          });
        }),
        lexicon: lex,
        rules: new GameRules({
          boardLayoutName: 'CrosswordGame', // for now
          letterDistributionName: ld,
          variantName: 'classic', // for now
        }),
        challengeRule: chrule,
        public: false,
      });
      console.log(resp);
    } catch (e) {
      flashError(e);
    }
  };

  const deleteGame = async (gid: string) => {
    try {
      await eventClient.deleteBroadcastGame({ gameId: gid });
    } catch (e) {
      flashError(e);
    }
  };

  return (
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
          <Analyzer
            includeCard
            lexicon={gameInfo.gameRequest?.lexicon ?? ''}
            variant={gameInfo.gameRequest?.rules?.variantName}
          />
          <Card
            title="Editor controls"
            className="editor-control"
            style={{ marginTop: 12 }}
          >
            <EditorControl
              createNewGame={createNewGame}
              gameID={gameContext.gameID}
              deleteGame={deleteGame}
            />
          </Card>
        </div>
        <div className="sticky-player-card-container">
          <PlayerCards
            horizontal
            gameMeta={gameInfo}
            playerMeta={gameInfo.players}
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
            gameID={''} // tbd
            sendSocketMsg={doNothing}
            sendGameplayEvent={(evt) => sendGameplayEvent(evt)}
            gameDone={false} // tbd
            playerMeta={gameInfo.players}
            tournamentID={gameInfo.tournamentId}
            vsBot={false}
            tournamentPairedMode={false}
            // why does my linter keep overwriting this?
            // eslint-disable-next-line max-len
            lexicon={gameInfo.gameRequest?.lexicon ?? ''}
            alphabet={alphabet}
            challengeRule={
              gameInfo.gameRequest?.challengeRule ?? ChallengeRule.VOID
            }
            handleAcceptRematch={doNothing}
            handleAcceptAbort={doNothing}
            handleSetHover={handleSetHover}
            handleUnsetHover={hideDefinitionHover}
            definitionPopover={definitionPopover}
            changeCurrentRack={changeCurrentRack}
          />
        </div>

        <div className="data-area" id="right-sidebar">
          {/* There are two competitor cards, css hides one of them. */}
          {/* There are two player cards, css hides one of them. */}
          <PlayerCards gameMeta={gameInfo} playerMeta={gameInfo.players} />
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
            lexicon={gameInfo.gameRequest?.lexicon ?? ''}
            variant={gameInfo.gameRequest?.rules?.variantName}
            events={examinableGameContext.turns}
            board={examinableGameContext.board}
            playerMeta={gameInfo.players}
            poolFormat={poolFormat}
          />
        </div>
      </div>
    </div>
  );
};
