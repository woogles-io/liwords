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
import { GameHistoryRefresher } from '../gen/api/proto/ipc/omgwords_pb';
import { GameInfoResponse } from '../gen/api/proto/ipc/omgwords_pb';
import { GameHistory } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { PlayerInfo } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { TopBar } from '../navigation/topbar';
import { sortTiles } from '../store/constants';
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  usePoolFormatStoreContext,
} from '../store/store';
import { useDefinitionAndPhonyChecker } from '../utils/hooks/definitions';
import { useMountedState } from '../utils/mounted';

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

  // useEffect(() => {
  //   handleExamineStart();
  //   handleExamineGoTo(0);
  // }, [handleExamineGoTo, handleExamineStart]);

  useEffect(() => {
    dispatchGameContext({
      actionType: ActionType.RefreshHistory,
      payload: new GameHistoryRefresher({
        history: new GameHistory({
          players: [
            new PlayerInfo({ nickname: 'player1', userId: 'player1' }),
            new PlayerInfo({ nickname: 'player2', userId: 'player2' }),
          ],
          lastKnownRacks: ['EGLOOSW', 'EGGHEAD'],
          uid: 'uid',
        }),
      }),
    });
  }, []);

  const rack =
    examinableGameContext.players.find((p) => p.onturn)?.currentRack ?? '';
  const sortedRack = useMemo(() => sortTiles(rack), [rack]);

  const alphabet = useMemo(
    () => alphabetFromName(gameInfo.gameRequest?.rules?.letterDistributionName),
    [gameInfo]
  );

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
            sendGameplayEvent={(evt) => console.log(evt)}
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
