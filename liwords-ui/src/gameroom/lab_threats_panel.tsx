import React, { useEffect, useMemo, useState } from "react";
import { Button, InputNumber, Radio, Tooltip } from "antd";
import { AimOutlined, UnorderedListOutlined } from "@ant-design/icons";
import { machineLetterToRune } from "../constants/alphabets";
import { useTentativeTileContext } from "../store/store";
import { MachineLetter } from "../utils/cwgame/common";
import { phoneLayout } from "../utils/window_room";
import { labCall } from "../wasm/lab";
import { getLoadableFiles } from "../wasm/loader";
import { wolgesLetterToLiwordsLetter } from "./analyzer";
import {
  ColumnName,
  LabPosition,
  NOTHING_TO_INFER,
  PanelFrame,
  SimSettings,
  inferRequest,
  knownTiles,
  toAnalyzerMove,
  wasmBoard,
} from "./analyzer_lab_panels";
import {
  ThreatPlay,
  ThreatTint,
  ThreatsAnswer,
  leaveAfter,
  percentText,
  playSquares,
  publishLabThreats,
  threatHeat,
  threatsSummary,
} from "./lab_threats";

// How many of the opponent's best plays are listed, and how long the racks
// they may hold are weighed, until the user sets others: the 15 the analyzer
// lists, and a second, as long as Every first move takes.
const DEFAULT_COUNT = 15;
const DEFAULT_SECONDS = 1;

// A board the threats are counted on: the position's own, or with the tiles
// the player has placed on it, the rest of their rack theirs and the tiles
// they would draw after it still unseen.
type ThreatBoard = {
  letters: Array<MachineLetter>;
  rack: Array<number>;
};

type Counted = {
  answer?: ThreatsAnswer;
  error?: string;
};

type ThreatsPanelProps = {
  lab: LabPosition;
  // the sim's settings, whose tiles known on the opponent's rack, or their
  // leave inferred, the threats count with too
  settings: SimSettings;
  onClose: () => void;
};

// The squares a play places a tile on, each with the letter it places there.
const ghostOf = (play: ThreatPlay, lab: LabPosition) => {
  const ghost = new Map<number, string>();
  const at = playSquares(play, lab.dim);
  play.word
    .filter((t) => t !== 0)
    .forEach((t, i) =>
      ghost.set(
        at[i],
        machineLetterToRune(
          wolgesLetterToLiwordsLetter(t),
          lab.alphabet,
          false,
          true,
        ),
      ),
    );
  return ghost;
};

// What the opponent may play next, on the position and, with tiles placed, on
// the board after them: a line for each, the opponent's best plays on the
// last, and each empty square over the board, tinted by its best score or by
// the chance the opponent can score it there.
export const ThreatsPanel = (props: ThreatsPanelProps) => {
  const { lab, settings, onClose } = props;
  const { placedTiles } = useTentativeTileContext();
  const [tint, setTint] = useState<ThreatTint>("score");
  const [count, setCount] = useState<number | null>(DEFAULT_COUNT);
  const [seconds, setSeconds] = useState<number | null>(DEFAULT_SECONDS);
  const [listShown, setListShown] = useState(() => !phoneLayout());
  const [picking, setPicking] = useState(false);
  const [square, setSquare] = useState<number | null>(null);
  const [pointed, setPointed] = useState<ThreatPlay | null>(null);
  const [picked, setPicked] = useState<ThreatPlay | null>(null);
  const [counted, setCounted] = useState(() => new Map<string, Counted>());
  const [counting, setCounting] = useState(false);
  // each count again gives the racks drawn another seed
  const [round, setRound] = useState(0);

  const now: ThreatBoard = useMemo(
    () => ({ letters: lab.letters, rack: lab.rack }),
    [lab],
  );
  const after: ThreatBoard | null = useMemo(() => {
    if (placedTiles.size === 0) return null;
    const letters = [...lab.letters];
    const placed: Array<number> = [];
    for (const t of placedTiles) {
      letters[t.row * lab.dim + t.col] = t.letter;
      placed.push(t.letter & 0x80 ? 0 : t.letter);
    }
    const rack = leaveAfter(lab.rack, placed);
    return rack ? { letters, rack } : null;
  }, [lab, placedTiles]);

  // the opponent's tiles, as the sim's settings have them
  const known = knownTiles(settings.known, lab);
  const infer = settings.infer ? inferRequest(lab, settings) : undefined;
  const note =
    "error" in known
      ? known.error
      : settings.infer && !infer
        ? `Not inferred: ${NOTHING_TO_INFER}.`
        : settings.infer
          ? "Opponent's leave inferred from their last move."
          : known.tiles.length
            ? `Opponent holds ${settings.known.toUpperCase()}.`
            : undefined;
  const request = (board: ThreatBoard) =>
    "error" in known || count === null || seconds === null
      ? null
      : JSON.stringify({
          board: wasmBoard(board.letters, lab.dim),
          lexicon: lab.position.lexicon,
          rules: lab.position.rules,
          rack: board.rack,
          ...(infer ? {} : { known: known.tiles }),
          count,
          ms: seconds * 1000,
        });
  const keyOf = (board: ThreatBoard | null) => {
    const r = board && request(board);
    return r ? `${round} ${r} ${infer ?? ""}` : null;
  };
  const nowKey = keyOf(now);
  const afterKey = keyOf(after);

  // Count each board not yet counted, the position's first; one call at a
  // time, as the worker takes them.
  useEffect(() => {
    const wanted = [
      [nowKey, now],
      [afterKey, after],
    ] as const;
    const todo = wanted.filter(([key]) => key !== null && !counted.has(key));
    if (!todo.length) return;
    let current = true;
    (async () => {
      setCounting(true);
      try {
        await labCall({
          op: "prepare",
          args: { files: getLoadableFiles(lab.loadableKey) },
        });
        for (const [key, board] of todo) {
          let result: Counted;
          try {
            const { answer } = await labCall<{ answer: ThreatsAnswer }>({
              op: "threats",
              args: { request: request(board!)!, infer },
            });
            result = { answer };
          } catch (e) {
            result = { error: e instanceof Error ? e.message : String(e) };
          }
          if (!current) return;
          setCounted((m) => new Map(m).set(key!, result));
        }
      } catch (e) {
        if (!current) return;
        const error = e instanceof Error ? e.message : String(e);
        setCounted((m) => {
          const next = new Map(m);
          for (const [key] of todo) next.set(key!, { error });
          return next;
        });
      } finally {
        if (current) setCounting(false);
      }
    })();
    return () => {
      current = false;
    };
    // the boards and the request follow from the keys
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [nowKey, afterKey, counted, lab.loadableKey]);

  const shown = afterKey
    ? counted.get(afterKey)
    : nowKey && counted.get(nowKey);
  const shownBoard = afterKey ? after! : now;
  const answer = shown ? shown.answer : undefined;

  // a play picked or pointed at, or a square narrowed to, is of the answer
  // it was picked in
  useEffect(() => {
    setPicked(null);
    setPointed(null);
    setSquare(null);
  }, [answer]);

  const rows = useMemo(
    () =>
      (answer?.plays ?? [])
        .filter(
          (p) => square === null || playSquares(p, lab.dim).includes(square),
        )
        .map((play) => ({
          play,
          move: toAnalyzerMove(
            {
              action: "play",
              down: play.down,
              lane: play.lane,
              idx: play.idx,
              word: play.word,
              score: play.score,
            },
            play.score,
            shownBoard.letters,
            play.tiles,
            lab,
          ),
        })),
    [answer, square, lab, shownBoard],
  );

  // the board shows the answer listed
  const [publisher] = useState(() => Symbol("threats"));
  useEffect(() => {
    if (!answer) {
      publishLabThreats(publisher, null);
      return;
    }
    publishLabThreats(publisher, {
      dim: lab.dim,
      heat: threatHeat(answer.squares, tint),
      ghost: picked ? ghostOf(picked, lab) : new Map(),
      outline: new Set(pointed ? playSquares(pointed, lab.dim) : []),
      square,
      picking,
      pick: (s) => {
        setSquare(s);
        setPicking(false);
      },
    });
  }, [publisher, answer, tint, picked, pointed, square, picking, lab]);
  useEffect(() => () => publishLabThreats(publisher, null), [publisher]);

  const line = (key: string | null, label: string) => {
    if (!key) return null;
    const c = counted.get(key);
    const text = !c
      ? "Counting..."
      : c.error
        ? `Could not count the threats: ${c.error}`
        : threatsSummary(c.answer!, lab.unseen.length);
    return (
      <div>
        {label}
        {text}
      </div>
    );
  };
  const status =
    lab.unseen.length === 0 ? (
      "Nothing is unseen: the opponent holds no tiles."
    ) : (
      <React.Fragment>
        {note && <div>{note}</div>}
        {line(nowKey, afterKey ? "Now: " : "")}
        {line(afterKey, "After the tiles placed: ")}
      </React.Fragment>
    );

  return (
    <PanelFrame
      title="Threats"
      status={status}
      running={false}
      canResume={!counting && !!nowKey && lab.unseen.length > 0}
      stop={() => {}}
      resume={() => {
        setCounted(new Map());
        setRound((r) => r + 1);
      }}
      resumeLabel="Count again"
      onClose={onClose}
      buttons={
        <React.Fragment>
          <Tooltip
            title={
              square === null
                ? "List the plays through a square: click one on the board"
                : "List every play"
            }
          >
            <Button
              shape="circle"
              size="small"
              icon={<AimOutlined />}
              aria-label="Plays through a square"
              type={picking || square !== null ? "primary" : "default"}
              disabled={!answer}
              onClick={() => {
                if (square !== null) setSquare(null);
                else setPicking(!picking);
              }}
            />
          </Tooltip>
          <Tooltip title={listShown ? "Hide the plays" : "Show the plays"}>
            <Button
              shape="circle"
              size="small"
              icon={<UnorderedListOutlined />}
              aria-label="Plays"
              type={listShown ? "primary" : "default"}
              onClick={() => setListShown(!listShown)}
            />
          </Tooltip>
        </React.Fragment>
      }
    >
      <div className="lab-threat-settings">
        <Radio.Group
          size="small"
          value={tint}
          onChange={(e) => setTint(e.target.value)}
          options={[
            { label: "Score", value: "score" },
            { label: "Chance", value: "chance" },
          ]}
          optionType="button"
          aria-label="Tint the squares by"
        />
        <label>
          Plays
          <InputNumber
            size="small"
            aria-label="Plays listed"
            min={1}
            precision={0}
            value={count}
            onChange={(v) => setCount(v)}
          />
        </label>
        <label>
          Seconds
          <InputNumber
            size="small"
            aria-label="Seconds"
            min={0}
            value={seconds}
            onChange={(v) => setSeconds(v)}
          />
        </label>
      </div>
      {listShown && answer && (
        <table className="lab-threat-plays">
          <thead>
            <tr>
              <th>
                <ColumnName name="Move" tip="The opponent's play" />
              </th>
              <th>
                <ColumnName name="Score" tip="Its score" />
              </th>
              <th>
                <ColumnName
                  name="Chance"
                  tip="The chance the opponent's rack holds its tiles"
                />
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map(({ play, move }) => (
              <tr
                key={move.jsonKey}
                className={picked === play ? "lab-shown" : undefined}
                onMouseEnter={() => setPointed(play)}
                onMouseLeave={() => setPointed(null)}
                onClick={() => setPicked(picked === play ? null : play)}
              >
                <td className="move">
                  <span className="move-coords">{move.coordinates}</span>{" "}
                  {move.displayMove}
                </td>
                <td className="lab-n">{play.score}</td>
                <td className="lab-n">
                  {answer.exact ? "" : "~"}
                  {percentText(play.chance)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </PanelFrame>
  );
};
