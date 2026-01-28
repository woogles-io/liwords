import React, { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router";
import {
  ChallengeRule,
  PlayState,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
} from "../store/store";
import { defaultGameInfo } from "../gameroom/game_info";
import { BoardPanel } from "../gameroom/board_panel";
import { sortTiles } from "../store/constants";
import { alphabetFromName } from "../constants/alphabets";
import { ActionType } from "../actions/actions";
import {
  GameHistoryRefresherSchema,
  GameInfoResponse,
} from "../gen/api/proto/ipc/omgwords_pb";
import { PlayerCards } from "../gameroom/player_cards";
import { useDefinitionAndPhonyChecker } from "../utils/hooks/definitions";
import { flashError, useClient } from "../utils/hooks/connect";
import { GameMetadataService } from "../gen/api/proto/game_service/game_service_pb";
import { MachineLetter } from "../utils/cwgame/common";
import { create } from "@bufbuild/protobuf";

const doNothing = () => {};

export const Embed = () => {
  const [gameInfo, setGameInfo] = useState<GameInfoResponse>(defaultGameInfo);
  const { gameID } = useParams();
  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();

  const {
    handleExamineStart,
    handleExamineGoTo,
    handleExamineDisableShortcuts,
  } = useExamineStoreContext();

  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const { handleSetHover, hideDefinitionHover, definitionPopover } =
    useDefinitionAndPhonyChecker({
      addChat: doNothing,
      enableHoverDefine: true,
      gameContext,
      gameDone: true,
      gameID,
      lexicon: gameInfo.gameRequest?.lexicon ?? "",
      variant: gameInfo.gameRequest?.rules?.variantName,
    });

  const gameDone =
    gameContext.playState === PlayState.GAME_OVER && !!gameContext.gameID;

  const gameMetadataClient = useClient(GameMetadataService);

  useEffect(() => {
    if (!gameID) {
      return;
    }
    // Request game API to get info about the game at the beginning.
    console.log("gonna fetch metadata, game id is", gameID);

    (async () => {
      try {
        const resp = await gameMetadataClient.getMetadata({ gameId: gameID });
        setGameInfo(resp);
      } catch (e) {
        flashError(e);
      }
    })();

    (async () => {
      try {
        const resp = await gameMetadataClient.getGameHistory({
          gameId: gameID,
        });
        const ghr = create(GameHistoryRefresherSchema, {
          history: resp.history,
        });
        dispatchGameContext({
          actionType: ActionType.RefreshHistory,
          payload: ghr,
        });
      } catch (e) {
        flashError(e);
      }
    })();

    return () => {
      setGameInfo(defaultGameInfo);
    };
  }, [gameID, dispatchGameContext, gameMetadataClient]);

  useEffect(() => {
    if (gameContext.turns.length > 0) {
      handleExamineStart();
      handleExamineGoTo(0);
      handleExamineDisableShortcuts();
    }
  }, [
    gameContext.turns.length,
    handleExamineGoTo,
    handleExamineStart,
    handleExamineDisableShortcuts,
  ]);

  //   if (!gameHistory) {
  //     return <div>Could not load game history</div>;
  //   }
  //   if (gameHistory.getFinalScoresList().length === 0) {
  //     return <div>This game is not over.</div>;
  //   }

  const sortedRack = useMemo(() => {
    const rack =
      examinableGameContext.players.find((p) => p.onturn)?.currentRack ??
      new Array<MachineLetter>();

    return sortTiles(rack, examinableGameContext.alphabet);
  }, [examinableGameContext.alphabet, examinableGameContext.players]);
  const alphabet = useMemo(
    () => alphabetFromName(gameInfo.gameRequest?.rules?.letterDistributionName),
    [gameInfo],
  );

  if (!gameID) {
    return <div>Invalid game ID</div>;
  }

  return (
    <div className="game-container board-embed">
      <div className="game-table">
        <div className="sticky-player-card-container">
          <PlayerCards
            horizontal
            hideProfileLink
            gameMeta={gameInfo}
            playerMeta={gameInfo.players}
          />
        </div>
        <div className="play-area ">
          <BoardPanel
            anonymousViewer={true}
            username={""}
            board={examinableGameContext.board}
            currentRack={sortedRack}
            events={examinableGameContext.turns}
            gameID={gameID}
            sendSocketMsg={doNothing}
            sendGameplayEvent={doNothing}
            gameDone={gameDone}
            playerMeta={gameInfo.players}
            tournamentID={gameInfo.tournamentId}
            vsBot={gameInfo.gameRequest?.playerVsBot ?? false}
            lexicon={gameInfo.gameRequest?.lexicon ?? ""}
            alphabet={alphabet}
            challengeRule={
              gameInfo.gameRequest?.challengeRule ?? ChallengeRule.VOID
            }
            handleSetHover={handleSetHover}
            handleUnsetHover={hideDefinitionHover}
            definitionPopover={definitionPopover}
            handleAcceptAbort={doNothing}
            handleAcceptRematch={doNothing}
            exitableExaminer={false}
          />
        </div>
        {/* <div className="data-area" id="right-sidebar">
          <ScoreCard
            username={''}
            playing={false}
            lexicon={gameInfo.game_request.lexicon}
            variant={gameInfo.game_request.rules.variant_name}
            events={examinableGameContext.turns}
            board={examinableGameContext.board}
            playerMeta={gameInfo.players}
            poolFormat={PoolFormatType.Alphabet} // not used, I think.
            gameEpilog={<></>}
            hideExtraInteractions
          />
        </div> */}
      </div>
    </div>
  );
};
