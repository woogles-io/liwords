import React, { useEffect, useState } from "react";
import { Button, Input, Tooltip } from "antd";
import {
  DoubleLeftOutlined,
  LeftOutlined,
  RightOutlined,
} from "@ant-design/icons";
import { machineWordToRunes, runesToMachineWord } from "../constants/alphabets";
import {
  GameEvent,
  GameEvent_Direction,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { GameState } from "../store/reducers/game_reducer";
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useTentativeTileContext,
} from "../store/store";
import { EphemeralTile } from "../utils/cwgame/common";
import { getWolges } from "../wasm/loader";
import { parseExaminableGameContext, wolgesLabelsToLetter } from "./analyzer";
import {
  bagTilesOf,
  exchangeMinBag,
  randomDraw,
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
  variant?: string;
  // switched on by the lab's what-if button; shown anyway while moves are
  // made up, as going into a pre-endgame's play makes them
  on: boolean;
};

// The words a play placed makes that the lexicon lacks, as the analyzer
// marks a move's (play_score): shown, never refused, as a made-up phony is a
// move to try. Null until known, or with no play.
const usePhonies = (
  play: GameEvent | undefined,
  gs: GameState,
  lexicon: string,
  variant?: string,
) => {
  const [phonies, setPhonies] = useState<{
    key: string;
    words: Array<string>;
  } | null>(null);
  const key = play
    ? `${play.row},${play.column},${play.direction},${play.playedTiles}`
    : "";
  useEffect(() => {
    if (!play) return;
    let live = true;
    (async () => {
      const { loadableKey, boardObj, alphabet } = parseExaminableGameContext(
        gs,
        lexicon,
        variant,
      );
      const down = play.direction === GameEvent_Direction.VERTICAL;
      const wasmPlay = {
        action: "play",
        down,
        lane: down ? play.column : play.row,
        idx: down ? play.row : play.column,
        word: wolgesLabelsToLetter(play.playedTiles, alphabet),
        score: play.score,
      };
      const wolges = await getWolges(loadableKey);
      const scored = JSON.parse(
        await wolges.play_score(
          JSON.stringify({ ...boardObj, plays: [wasmPlay] }),
        ),
      )[0];
      if (!live || scored?.result !== "scored") return;
      setPhonies({
        key,
        words: (scored.invalid_words ?? []).map((t: Array<number>) =>
          machineWordToRunes(t, alphabet),
        ),
      });
    })().catch((e) => console.error("what-if play_score", e));
    return () => {
      live = false;
    };
    // the play's own fields are its key
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, gs.board, lexicon, variant]);
  return phonies?.key === key ? phonies.words : null;
};

// how long a single click's "(pass)" shows
const PASS_HINT_MS = 1500;

export const WhatIfBar = (props: WhatIfBarProps) => {
  const { placedTiles, setPlacedTiles } = useTentativeTileContext();
  const { gameContext } = useExaminableGameContextStoreContext();
  const {
    isExamining,
    whatIf,
    whatIfSteps,
    playWhatIf,
    takeBackWhatIf,
    clearWhatIf,
  } = useExamineStoreContext();
  // the tiles the mover puts back in an exchange, as typed (? a blank)
  const [back, setBack] = useState("");
  // a single click with nothing to play or exchange says, in the exchange
  // box, that a double click passes, for a moment
  const [passHint, setPassHint] = useState(false);
  useEffect(() => {
    if (!passHint) return;
    const t = setTimeout(() => setPassHint(false), PASS_HINT_MS);
    return () => clearTimeout(t);
  }, [passHint]);
  const of = whatIfSideOf(gameContext);
  const shown = isExamining && !!of && (props.on || !!whatIfSteps);
  const placedPlay =
    shown && of
      ? whatIfPlay(
          placedTiles,
          gameContext.board,
          gameContext.alphabet,
          of.side,
        )
      : null;
  const phonies = usePhonies(
    placedPlay?.[0],
    gameContext,
    props.lexicon,
    props.variant,
  );
  if (!shown || !of) return null;
  const { side } = of;
  // the moves made up, going out's points no move of their own
  const made = whatIf.filter(
    (e) => e.type !== GameEvent_Type.END_RACK_PTS,
  ).length;
  const me = gameContext.players[side.playerIndex];
  const other = gameContext.players[1 - side.playerIndex];
  // a game gone out has no moves left: the bag empty and a rack emptied (a
  // rack not known, with tiles in the bag, is no rack gone out)
  const over =
    side.bag === 0 &&
    (me.currentRack.length === 0 || other.currentRack.length === 0);
  const play = over
    ? null
    : whatIfPlay(placedTiles, gameContext.board, gameContext.alphabet, side);
  const backTiles = runesToMachineWord(
    back.toUpperCase(),
    gameContext.alphabet,
  );
  const minBag = exchangeMinBag(props.lexicon);
  const canExchange = !over && side.bag >= minBag;
  // tiles placed on the board make the move a play; with none, the tiles
  // typed to exchange make it an exchange
  const swapping = canExchange && placedTiles.size === 0;
  const exchange =
    swapping && back
      ? whatIfExchange(backTiles, gameContext.alphabet, side, minBag)
      : null;
  // with tiles in the bag, the draw: as many as the play places, the bag
  // allowing, or as many as the exchange puts back, each a tile still unseen
  const drawCount = placedTiles.size || backTiles.length;
  const drawing = side.bag > 0 && (!!play || !!exchange);
  // with tiles in the bag, drawn at random as the move is made; the rack
  // under the board sets them otherwise, after
  const drawn = () =>
    drawing ? randomDraw(Math.min(drawCount, side.bag), bagTilesOf(side)) : [];
  // one button makes the move: the play placed on the board, if any; else
  // the exchange typed, if any; else, on a double click, a pass
  const mode = placedTiles.size ? "play" : back ? "exchange" : "pass";
  const moveReady =
    mode === "play" ? !!play : mode === "exchange" ? !!exchange : !over;
  const moveTitle =
    mode === "play"
      ? phonies?.length
        ? `What if: play the move placed, phony: ${phonies.join(", ")}`
        : "What if: play the move placed"
      : mode === "exchange"
        ? "What if: exchange the tiles typed"
        : "What if: pass, on a double click (or place a play, or type tiles to exchange)";
  // a line that keeps its place, whatever applies, and a play's phonies
  // under it
  return (
    <div className="lab-what-if">
      <div className="lab-what-if-line">
        <span className="lab-what-if-label">What if</span>
        <Tooltip title="Back to the game's own position">
          <Button
            shape="circle"
            size="small"
            icon={<DoubleLeftOutlined />}
            aria-label="Clear the moves made up"
            disabled={!whatIfSteps}
            onClick={clearWhatIf}
          />
        </Tooltip>
        <Tooltip
          title={
            made
              ? `Take back the last move made up (${made} made up)`
              : "Take back the last move made up"
          }
        >
          <Button
            shape="circle"
            size="small"
            icon={<LeftOutlined />}
            aria-label="Take back"
            disabled={!whatIfSteps}
            onClick={takeBackWhatIf}
          />
        </Tooltip>
        <Tooltip title="The tiles put back in an exchange, from the rack (? a blank), with no tiles placed on the board">
          <Input
            size="small"
            className="lab-what-if-draws"
            aria-label="Exchange"
            placeholder={passHint ? "(pass)" : "exchange"}
            disabled={!swapping}
            value={back}
            status={swapping && !exchange && back ? "error" : undefined}
            onChange={(e) => setBack(e.target.value)}
          />
        </Tooltip>
        <Tooltip title={moveTitle}>
          <Button
            shape="circle"
            size="small"
            icon={<RightOutlined />}
            aria-label={moveTitle}
            disabled={!moveReady}
            onClick={() => {
              if (mode === "pass") {
                // a single click only says what a double click does
                setPassHint(true);
                return;
              }
              const events = mode === "play" ? play : exchange;
              if (!events) return;
              playWhatIf(events, drawn());
              setPlacedTiles(new Set<EphemeralTile>());
              setBack("");
            }}
            onDoubleClick={() => {
              if (mode !== "pass" || over) return;
              playWhatIf(whatIfPass(gameContext.alphabet, side));
              setPassHint(false);
            }}
          />
        </Tooltip>
      </div>
      {!!phonies?.length && (
        <div className="lab-what-if-line lab-what-if-phony">
          {phonies.map((w) => `${w}*`).join(", ")}
        </div>
      )}
    </div>
  );
};
