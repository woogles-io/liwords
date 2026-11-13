import React, { useState } from "react";
import { Button, Input, Tooltip } from "antd";
import {
  CloseCircleOutlined,
  ForwardOutlined,
  PlayCircleOutlined,
  RetweetOutlined,
  UndoOutlined,
} from "@ant-design/icons";
import { runesToMachineWord } from "../constants/alphabets";
import { GameEvent_Type } from "../gen/api/proto/vendored/macondo/macondo_pb";
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useTentativeTileContext,
} from "../store/store";
import { EphemeralTile } from "../utils/cwgame/common";
import {
  exchangeMinBag,
  whatIfDraw,
  whatIfExchange,
  whatIfPass,
  whatIfPlay,
  whatIfSideOf,
} from "./what_if";

// A row of its own, under the analyzer's buttons: made-up moves on top of the
// position examined, for either side in turn:
// the move placed on the board played, scored by the page alone, the tiles
// typed exchanged, or a pass;
// the last one taken back, or all of them. The position examined, and so the
// analyzer and the lab, go on from the last.
type WhatIfBarProps = {
  lexicon: string;
};

export const WhatIfBar = (props: WhatIfBarProps) => {
  const { placedTiles, setPlacedTiles } = useTentativeTileContext();
  const { gameContext } = useExaminableGameContextStoreContext();
  const { isExamining, whatIf, playWhatIf, takeBackWhatIf, clearWhatIf } =
    useExamineStoreContext();
  // the tiles the mover draws after the move, as typed (? a blank)
  const [draws, setDraws] = useState("");
  // the tiles the mover puts back in an exchange, as typed (? a blank)
  const [back, setBack] = useState("");
  const of = whatIfSideOf(gameContext);
  if (!isExamining || !of) return null;
  const { side, unseen: unseenTiles } = of;
  // the moves made up, going out's points no move of their own
  const made = whatIf.filter(
    (e) => e.type !== GameEvent_Type.END_RACK_PTS,
  ).length;
  const me = gameContext.players[side.playerIndex];
  const other = gameContext.players[1 - side.playerIndex];
  // a game gone out has no moves left
  const over = me.currentRack.length === 0 || other.currentRack.length === 0;
  const play = over
    ? null
    : whatIfPlay(placedTiles, gameContext.board, gameContext.alphabet, side);
  const backTiles = runesToMachineWord(
    back.toUpperCase(),
    gameContext.alphabet,
  );
  const minBag = exchangeMinBag(props.lexicon);
  const canExchange = !over && side.bag >= minBag;
  const exchange =
    canExchange && back
      ? whatIfExchange(backTiles, gameContext.alphabet, side, minBag)
      : null;
  // with tiles in the bag, the draw: as many as the move places, the bag
  // allowing, or as many as the exchange puts back, each a tile still unseen
  const drawCount = exchange ? backTiles.length : placedTiles.size;
  const drawing = side.bag > 0 && (!!play || !!exchange);
  const drew = drawing
    ? whatIfDraw(
        runesToMachineWord(draws.toUpperCase(), gameContext.alphabet),
        drawCount,
        me.currentRack,
        unseenTiles,
        side.bag,
      )
    : [];
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
          disabled={!play || !!exchange || drew === null}
          onClick={() => {
            if (!play || exchange || drew === null) return;
            playWhatIf(play, drew);
            setPlacedTiles(new Set<EphemeralTile>());
            setDraws("");
          }}
        />
      </Tooltip>
      {drawing && (
        <Tooltip
          title={`The tiles drawn after the move: ${Math.min(drawCount, side.bag)}, from the tiles unseen (? a blank)`}
        >
          <Input
            size="small"
            className="lab-what-if-draws"
            aria-label="Draws"
            placeholder="draws"
            value={draws}
            status={drew === null && draws ? "error" : undefined}
            onChange={(e) => setDraws(e.target.value)}
          />
        </Tooltip>
      )}
      {canExchange && (
        <Tooltip title="The tiles put back in an exchange, from the rack (? a blank)">
          <Input
            size="small"
            className="lab-what-if-draws"
            aria-label="Exchange"
            placeholder="exchange"
            value={back}
            status={!exchange && back ? "error" : undefined}
            onChange={(e) => setBack(e.target.value)}
          />
        </Tooltip>
      )}
      {canExchange && (
        <Tooltip title="What if: exchange the tiles typed">
          <Button
            shape="circle"
            size="small"
            icon={<RetweetOutlined />}
            aria-label="Exchange the tiles typed"
            disabled={!exchange || drew === null}
            onClick={() => {
              if (!exchange || drew === null) return;
              playWhatIf(exchange, drew);
              setPlacedTiles(new Set<EphemeralTile>());
              setBack("");
              setDraws("");
            }}
          />
        </Tooltip>
      )}
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
