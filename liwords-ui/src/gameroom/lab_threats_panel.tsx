import React, { useEffect, useMemo, useRef, useState } from "react";
import { Button, InputNumber, Radio, Tooltip } from "antd";
import { AimOutlined, UnorderedListOutlined } from "@ant-design/icons";
import { machineLetterToRune } from "../constants/alphabets";
import { useTentativeTileContext } from "../store/store";
import { MachineLetter } from "../utils/cwgame/common";
import { phoneLayout } from "../utils/window_room";
import { labCall } from "../wasm/lab";
import { namedTiles } from "../wasm/lab_errors";
import { getLoadableFiles } from "../wasm/loader";
import { wolgesLetterToLiwordsLetter } from "./analyzer";
import {
  ColumnName,
  Inferred,
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
// how long each step of the count weighs racks for, so the answer shows as it
// firms up rather than only when the seconds are spent
const THREATS_STEP_MS = 100;

// A board the threats are counted on: the position's own, or with the tiles
// the player has placed on it, the rest of their rack theirs and the tiles
// they would draw after it still unseen.
type ThreatBoard = {
  letters: Array<MachineLetter>;
  rack: Array<number>;
};

type Counted = {
  // the count held in the lab's worker, let go once the board is not shown
  id?: number;
  answer?: ThreatsAnswer;
  // the opponent's leave as inferred for the count, if it was
  inferred?: Inferred;
  // false while the count goes on, its answer still firming up
  done?: boolean;
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
    "error" in known || seconds === null
      ? null
      : JSON.stringify({
          board: wasmBoard(board.letters, lab.dim),
          lexicon: lab.position.lexicon,
          rules: lab.position.rules,
          rack: board.rack,
          ...(infer ? {} : { known: known.tiles }),
        });
  const keyOf = (board: ThreatBoard | null) => {
    const r = board && request(board);
    return r ? `${round} ${r} ${infer ?? ""}` : null;
  };
  const nowKey = keyOf(now);
  const afterKey = keyOf(after);

  // The plays listed, as many as asked: read when each step is sent, so a new
  // count lists again without counting again.
  const countRef = useRef(count);
  countRef.current = count;

  // The counts so far, read when the boards change: each step's answer
  // changes them, and must not cut short the count it came from.
  const countedRef = useRef(counted);
  countedRef.current = counted;

  // Count each board not yet counted, the position's first; one call at a
  // time, as the worker takes them.
  useEffect(() => {
    const wanted = [
      [nowKey, now],
      [afterKey, after],
    ] as const;
    const todo = wanted.filter(
      ([key]) => key !== null && !countedRef.current.has(key),
    );
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
          // every play listed first, then the racks weighed a step at a
          // time, each step's answer shown, till every rack is counted or
          // the seconds asked are spent
          let id: number | undefined;
          let inferred: Inferred | undefined;
          try {
            ({ id, inferred } = await labCall<{
              id: number;
              inferred?: Inferred;
            }>({
              op: "threats_new",
              args: { request: request(board!)!, infer },
            }));
            const until = Date.now() + seconds! * 1000;
            for (;;) {
              if (!current) break;
              const left = until - Date.now();
              const answer = await labCall<ThreatsAnswer>({
                op: "threats_step",
                args: {
                  id,
                  ms: Math.max(0, Math.min(THREATS_STEP_MS, left)),
                  count: countRef.current ?? DEFAULT_COUNT,
                },
              });
              if (!current) break;
              const done = answer.exact || left <= THREATS_STEP_MS;
              setCounted((m) =>
                new Map(m).set(key!, { id, answer, inferred, done }),
              );
              if (done) break;
            }
          } catch (e) {
            if (current) {
              const error = e instanceof Error ? e.message : String(e);
              setCounted((m) => new Map(m).set(key!, { id, error }));
            }
          }
          // a count cut off by a new board is never shown
          if (!current) {
            if (id !== undefined) {
              void labCall({ op: "threats_free", args: { id } });
            }
            return;
          }
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
  }, [nowKey, afterKey, lab.loadableKey]);

  // Each count held is let go once its board is no longer counted, and all
  // of them when the panel closes.
  const held = useRef(new Set<number>());
  useEffect(() => {
    const live = new Set<number>();
    counted.forEach((c) => c.id !== undefined && live.add(c.id));
    held.current.forEach((id) => {
      if (!live.has(id)) void labCall({ op: "threats_free", args: { id } });
    });
    held.current = live;
  }, [counted]);
  useEffect(
    () => () =>
      held.current.forEach(
        (id) => void labCall({ op: "threats_free", args: { id } }),
      ),
    [],
  );

  const shownKey = afterKey ?? nowKey;
  const shown = shownKey ? counted.get(shownKey) : undefined;
  const shownBoard = afterKey ? after! : now;
  const answer = shown ? shown.answer : undefined;

  // a play picked or pointed at, or a square narrowed to, is of the board it
  // was picked on
  useEffect(() => {
    setPicked(null);
    setPointed(null);
    setSquare(null);
  }, [shownKey]);

  // The plays listed: the count's own, or, through a square or for a count
  // other than the one counted with, as the wasm lists them from the racks
  // already weighed, weighing no more.
  const [listed, setListed] = useState<{
    id: number;
    square: number | null;
    count: number;
    plays: Array<ThreatPlay>;
  } | null>(null);
  const shownId = shown?.id;
  const shownDone = !!shown?.done;
  const countedWith = answer?.plays.length;
  useEffect(() => {
    setListed(null);
    if (shownId === undefined || !shownDone || count === null) return;
    if (square === null && count === countedWith) return;
    let current = true;
    labCall<ThreatsAnswer>({
      op: "threats_step",
      args: {
        id: shownId,
        ms: 0,
        count,
        ...(square === null
          ? {}
          : {
              square: [Math.floor(square / lab.dim), square % lab.dim] as [
                number,
                number,
              ],
            }),
      },
    }).then(
      (a) =>
        current && setListed({ id: shownId, square, count, plays: a.plays }),
      () => {},
    );
    return () => {
      current = false;
    };
    // the plays the count gave are read once it is done, not on each change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [shownId, shownDone, square, count, lab.dim]);
  const plays = useMemo(
    () =>
      listed && listed.id === shownId && listed.square === square
        ? listed.plays
        : // till the wasm lists them, those of the count's own plays
          (answer?.plays ?? [])
            .filter(
              (p) =>
                square === null || playSquares(p, lab.dim).includes(square),
            )
            .slice(0, count ?? 0),
    [listed, shownId, square, answer, count, lab.dim],
  );

  const rows = useMemo(
    () =>
      plays.map((play) => ({
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
    [plays, lab, shownBoard],
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
      // picking goes on, so a click on another square moves the list to it
      pick: setSquare,
    });
  }, [publisher, answer, tint, picked, pointed, square, picking, lab]);
  useEffect(() => () => publishLabThreats(publisher, null), [publisher]);

  const line = (key: string | null, label: string) => {
    if (!key) return null;
    const c = counted.get(key);
    const text = !c
      ? "Counting..."
      : c.error
        ? `Could not count the threats: ${namedTiles(c.error, (t) =>
            machineLetterToRune(
              wolgesLetterToLiwordsLetter(t),
              lab.alphabet,
              false,
              true,
            ),
          )}`
        : threatsSummary(c.answer!, lab.unseen.length, !c.done);
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
              picking
                ? "List every play"
                : "List the plays through a square: click one on the board, another to move, the same again to let it go"
            }
          >
            <Button
              shape="circle"
              size="small"
              icon={<AimOutlined />}
              aria-label="Plays through a square"
              type={picking ? "primary" : "default"}
              disabled={!answer}
              onClick={() => {
                setSquare(null);
                setPicking(!picking);
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
