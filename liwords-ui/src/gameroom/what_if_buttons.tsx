import React from "react";
import { Button, Tooltip } from "antd";
import {
  CloseCircleOutlined,
  ForwardOutlined,
  PlayCircleOutlined,
  UndoOutlined,
} from "@ant-design/icons";
import { GameEvent_Type } from "../gen/api/proto/vendored/macondo/macondo_pb";
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useTentativeTileContext,
} from "../store/store";
import { EphemeralTile } from "../utils/cwgame/common";
import { WhatIfSide, whatIfPass, whatIfPlay } from "./what_if";

// A row of its own, under the analyzer's buttons: made-up moves on top of the
// position examined, for either side in turn:
// the move placed on the board played, scored by the page alone, or a pass;
// the last one taken back, or all of them. The position examined, and so the
// analyzer and the lab, go on from the last.
export const WhatIfBar = () => {
  const { placedTiles, setPlacedTiles } = useTentativeTileContext();
  const { gameContext } = useExaminableGameContextStoreContext();
  const { isExamining, whatIf, playWhatIf, takeBackWhatIf, clearWhatIf } =
    useExamineStoreContext();
  if (!isExamining || gameContext.players.length !== 2) return null;
  // the moves made up, going out's points no move of their own
  const made = whatIf.filter(
    (e) => e.type !== GameEvent_Type.END_RACK_PTS,
  ).length;
  const mover = gameContext.onturn;
  const me = gameContext.players[mover];
  const other = gameContext.players[1 - mover];
  const unseen = Object.values(gameContext.pool).reduce((a, b) => a + b, 0);
  const side: WhatIfSide = {
    playerIndex: mover,
    score: me.score,
    rack: me.currentRack,
    otherScore: other.score,
    otherRack: other.currentRack,
    bag: Math.max(0, unseen - me.currentRack.length - other.currentRack.length),
  };
  // a game gone out has no moves left
  const over = me.currentRack.length === 0 || other.currentRack.length === 0;
  const play = over
    ? null
    : whatIfPlay(placedTiles, gameContext.board, gameContext.alphabet, side);
  return (
    <div className="lab-what-if">
      <span className="lab-what-if-label">
        {made
          ? `What if: ${made} ${made === 1 ? "move" : "moves"} made up`
          : "What if"}
      </span>
      <Tooltip title="What if: play the move placed on the board">
        <Button
          shape="circle"
          size="small"
          icon={<PlayCircleOutlined />}
          aria-label="Play the move placed"
          disabled={!play}
          onClick={() => {
            if (!play) return;
            playWhatIf(play);
            setPlacedTiles(new Set<EphemeralTile>());
          }}
        />
      </Tooltip>
      <Tooltip title="What if: pass">
        <Button
          shape="circle"
          size="small"
          icon={<ForwardOutlined />}
          aria-label="Pass"
          disabled={over}
          onClick={() => {
            playWhatIf(whatIfPass(gameContext.alphabet, side));
            setPlacedTiles(new Set<EphemeralTile>());
          }}
        />
      </Tooltip>
      <Tooltip title="Take back the last move made up">
        <Button
          shape="circle"
          size="small"
          icon={<UndoOutlined />}
          aria-label="Take back"
          disabled={!whatIf.length}
          onClick={takeBackWhatIf}
        />
      </Tooltip>
      <Tooltip title="Back to the game's own position">
        <Button
          shape="circle"
          size="small"
          icon={<CloseCircleOutlined />}
          aria-label="Clear the moves made up"
          disabled={!whatIf.length}
          onClick={clearWhatIf}
        />
      </Tooltip>
    </div>
  );
};
