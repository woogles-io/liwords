import React, {
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Button, Tooltip } from "antd";
import {
  CaretRightOutlined,
  CloseOutlined,
  ExperimentOutlined,
  PauseOutlined,
} from "@ant-design/icons";
import { Alphabet } from "../constants/alphabets";
import {
  ChallengeRule,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { useExaminableGameContextStoreContext } from "../store/store";
import { MachineLetter, MachineWord } from "../utils/cwgame/common";
import {
  refitOnPageChanges,
  roomClass,
  roomInWindow,
} from "../utils/window_room";
import { labCall, onLabEvent } from "../wasm/lab";
import { getLoadableFiles } from "../wasm/loader";
import {
  AnalyzerMove,
  parseExaminableGameContext,
  usePlaceMoveCallback,
} from "./analyzer";
import {
  SimCandidate,
  labNow,
  passesAtEnd,
  scorelessTurnsAtEnd,
  simOrder,
  unseenBy,
} from "./analyzer_lab";

export type LabView = "sim";

// A play as the wasm writes one, with no equity.
type WasmPlay =
  | {
      action: "play";
      down: boolean;
      lane: number;
      idx: number;
      word: Array<number>;
      score: number;
    }
  | { action: "exchange"; tiles: Array<number> };

// A sim's answer: its iterations, and its candidates, each with its rollouts'
// plies, the first of them the opponent's reply.
type SimAnswer = {
  iterations: number;
  candidates: Array<
    SimCandidate & { play: WasmPlay; plies?: Array<{ bingo: number }> }
  >;
};

// The examined position as the lab's jobs take it.
export type LabPosition = {
  loadableKey: string;
  dim: number;
  letters: Array<MachineLetter>;
  alphabet: Alphabet;
  rack: MachineWord;
  unseen: MachineWord;
  // the fields the wasm's sim_new takes, less the leave
  position: {
    rack: MachineWord;
    board: Array<Array<number>>;
    lexicon: string | undefined;
    rules: string;
    scores: Array<number>;
    turn: number;
    zero_turns: number;
    pass_turns: number;
  };
  leave: string | undefined;
  // whether the other player has gone out with the play just made, which the
  // game's rules still let the player to move challenge off
  challengeable: boolean;
  // the position, as a string, to tell one from another
  key: string;
};

export const useLabPosition = (
  lexicon: string,
  variant?: string,
  challengeRule?: ChallengeRule,
): LabPosition | null => {
  const { gameContext } = useExaminableGameContextStoreContext();
  return useMemo(() => {
    let parsed;
    try {
      parsed = parseExaminableGameContext(gameContext, lexicon, variant);
    } catch {
      return null;
    }
    const { dim, letters, rackNum, loadableKey, boardObj, alphabet } = parsed;
    const position = {
      rack: rackNum,
      board: boardObj.board,
      lexicon: boardObj.lexicon,
      rules: boardObj.rules,
      scores: gameContext.players.map((p) => p.score),
      turn: gameContext.onturn,
      zero_turns: scorelessTurnsAtEnd(gameContext.turns),
      pass_turns: passesAtEnd(gameContext.turns),
    };
    const unseen = unseenBy(gameContext.pool, rackNum);
    const { turns } = gameContext;
    return {
      loadableKey,
      dim,
      letters,
      alphabet,
      rack: rackNum,
      unseen,
      position,
      leave: boardObj.leave,
      challengeable:
        unseen.length === 0 &&
        turns[turns.length - 1]?.type === GameEvent_Type.TILE_PLACEMENT_MOVE &&
        challengeRule !== undefined &&
        challengeRule !== ChallengeRule.VOID,
      key: JSON.stringify([position, boardObj.leave]),
    };
  }, [gameContext, lexicon, variant, challengeRule]);
};

// Nothing unseen: the other player has gone out. The game is over unless the
// play that went out can still be challenged off.
const goneOutText = (lab: LabPosition) =>
  lab.challengeable
    ? "The other player has gone out, but the play can still be challenged."
    : "The other player has gone out: the game is over.";

const signed = (x: number, digits = 0) =>
  `${x > 0 ? "+" : ""}${x.toFixed(digits)}`;

const percent = (x: number) => `${(x * 100).toFixed(1)}%`;

// The time from one frame of the display to the next, in milliseconds, as
// requestAnimationFrame stamps them: each step of a job takes about as long,
// so that each frame can show the newest answer. Frames dropped can leave two
// callbacks the same stamp, and a step of no time makes no iteration, so it
// waits for a frame stamped later than the last.
const frameMs = () =>
  new Promise<number>((resolve) => {
    const after = (last: number) =>
      requestAnimationFrame((stamp) =>
        stamp > last ? resolve(stamp - last) : after(last),
      );
    requestAnimationFrame(after);
  });

// How long a job has run, in milliseconds, on the page's clock: the runs it
// has finished added up, and the one under way to now. A search's progress
// shows the time it has taken beside what it has done, so while the job runs,
// the time shown goes on with each second, between its answers too.
const useRunMs = (job: number | undefined, running: boolean) => {
  const [runs, setRuns] = useState<{
    job?: number;
    ranMs: number;
    // when the run under way began
    since?: number;
  }>({ ranMs: 0 });
  const [, setTicks] = useState(0);
  const now = labNow();
  let next = runs;
  if (runs.job !== job) {
    next = { job, ranMs: 0, since: running ? now : undefined };
  } else if (running && runs.since === undefined) {
    next = { ...runs, since: now };
  } else if (!running && runs.since !== undefined) {
    next = { job, ranMs: runs.ranMs + now - runs.since };
  }
  // a run begun or ended since the last render, or another job
  if (next !== runs) setRuns(next);
  const ms = next.ranMs + (next.since === undefined ? 0 : now - next.since);
  useEffect(() => {
    if (!running) return;
    const timer = setTimeout(() => setTicks((n) => n + 1), 1000 - (ms % 1000));
    return () => clearTimeout(timer);
  });
  return ms;
};

// A time run, to the second: 8 s, 2 min 5 s, 1 h 0 min 5 s.
const ranText = (ms: number) => {
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  const h = Math.floor(m / 60);
  if (h > 0) return `${h} h ${m % 60} min ${s % 60} s`;
  if (m > 0) return `${m} min ${s % 60} s`;
  return `${s} s`;
};

// A column's name, with what the column holds on hover.
const ColumnName = (props: { name: string; tip: string }) => (
  <Tooltip title={props.tip}>
    <span>{props.name}</span>
  </Tooltip>
);

type PanelFrameProps = {
  title: string;
  status: React.ReactNode;
  running: boolean;
  canResume: boolean;
  stop: () => void;
  resume: () => void;
  // what the resume button says, Resume unless the panel says otherwise
  resumeLabel?: string;
  onClose: () => void;
  // a panel's own buttons, before the frame's
  buttons?: React.ReactNode;
  children: React.ReactNode;
};

const LAB_ROOM = roomClass("lab-room");

// While a panel is open, the analyzer's card is never shorter than its own
// height, and taller as what it shows needs: up to its room in the window
// (window_room.ts), and, but on a phone, up to half the window's height at
// most. The card's body gets the class lab-room, its own height as
// --lab-own, and that room as --lab-room (gameroom.scss), measured as the
// panel opens and again whenever the window or the page changes size, or one
// of the page's affixes pins or lets go. The score card, which holds the
// analyzer on narrower windows, keeps its own height.
export const useLabRoom = (panel: React.RefObject<HTMLElement | null>) => {
  useLayoutEffect(() => {
    const body = panel.current?.closest<HTMLElement>(
      ".analyzer-card .ant-card-body",
    );
    const card = body?.closest<HTMLElement>(".analyzer-card");
    if (!body || !card) {
      return;
    }
    const fit = () => {
      const { own, phone, room } = roomInWindow(body, card);
      const limit = phone ? room : Math.min(window.innerHeight / 2, room);
      body.style.setProperty("--lab-own", `${own}px`);
      body.style.setProperty("--lab-room", `${Math.max(own, limit)}px`);
      body.classList.add(LAB_ROOM);
    };
    fit();
    const stop = refitOnPageChanges(fit);
    return () => {
      stop();
      body.classList.remove(LAB_ROOM);
      body.style.removeProperty("--lab-own");
      body.style.removeProperty("--lab-room");
    };
  }, [panel]);
};

const PanelFrame = (props: PanelFrameProps) => {
  const resumeLabel = props.resumeLabel ?? "Resume";
  const panel = useRef<HTMLDivElement>(null);
  useLabRoom(panel);
  return (
    <div className="lab-panel" ref={panel}>
      <div className="lab-head">
        <span className="lab-title">{props.title}</span>
        <span className="lab-buttons">
          {props.buttons}
          {props.running ? (
            <Tooltip title="Stop">
              <Button
                shape="circle"
                size="small"
                icon={<PauseOutlined />}
                aria-label="Stop"
                onClick={props.stop}
              />
            </Tooltip>
          ) : (
            <Tooltip title={resumeLabel}>
              <Button
                shape="circle"
                size="small"
                icon={<CaretRightOutlined />}
                aria-label={resumeLabel}
                onClick={props.resume}
                disabled={!props.canResume}
              />
            </Tooltip>
          )}
          <Tooltip title="Close">
            <Button
              shape="circle"
              size="small"
              icon={<CloseOutlined />}
              aria-label="Close"
              onClick={props.onClose}
            />
          </Tooltip>
        </span>
      </div>
      <div className="lab-status">{props.status}</div>
      {props.children}
    </div>
  );
};

// A play's identity, written as the analyzer keys its moves.
const playKey = (play: WasmPlay) =>
  JSON.stringify(
    play.action === "play"
      ? {
          action: play.action,
          down: play.down,
          lane: play.lane,
          idx: play.idx,
          word: play.word,
        }
      : { action: play.action, tiles: play.tiles },
  );

// A sim made in the lab's worker: the position it is of, its job there, each
// candidate as the sim holds it (by playKey), and how it stands; done once it
// stopped by itself.
type MadeSim = {
  key: string;
  job: number;
  held: Map<string, AnalyzerMove>;
  answer?: SimAnswer;
  error?: string;
  running: boolean;
  done: boolean;
};

// The plies a sim's rollouts play: liwords's own count, sent with every sim
// made.
const DEFAULT_SIM_PLIES = 4;

const errorText = (e: unknown) => (e instanceof Error ? e.message : String(e));

// Words as a sentence lists them: A; A and B; A, B, and C.
const listWords = (words: Array<string>) =>
  words.length < 3
    ? words.join(" and ")
    : `${words.slice(0, -1).join(", ")}, and ${words[words.length - 1]}`;

// What the sim says of a phony among the moves listed, which it leaves out, as
// the wasm turns down a sim with one: the words the lexicon lacks.
const leftOutText = (m: AnalyzerMove, lexicon: string | undefined) => {
  const words = m.invalid_words ?? [];
  return words.length
    ? `${m.displayMove} is not simmed: ${listWords(words)} ${words.length > 1 ? "are" : "is"} not in ${lexicon ?? "the lexicon"}.`
    : `${m.displayMove} is not simmed: it is a phony.`;
};

export type LabSim = ReturnType<typeof useLabSim>;

// The examined position's sim, which the analyzer keeps while it is open, so
// that closing the panel only stops it: the last sim made keeps its rollouts
// until another job takes the lab's worker. Nothing is simmed until the user
// starts it, with the moves listed then; until then each shows its static
// equity.
export const useLabSim = (
  lab: LabPosition | null,
  // the analyzer's moves, the sim's candidates
  moves: Array<AnalyzerMove> | null | undefined,
  // whether the panel shows: a sim hidden, or left for another position, stops
  shown: boolean,
) => {
  const [sim, setSim] = useState<MadeSim | null>(null);
  const [starting, setStarting] = useState(false);
  const startingRef = useRef(false);
  // what the user last tried that came to nothing, at the position it was
  // tried at, until the next try
  const [note, setNote] = useState<{ key: string; text: string }>();
  const key = lab?.key;
  // The moves listed, the one played among them, which a sim made now holds;
  // a phony is no candidate, as the wasm turns down a call with one.
  const candidates = useMemo(
    () => (moves ?? []).filter((m) => m.valid ?? true),
    [moves],
  );
  // the sim made of this position, while the worker holds it
  const made = sim && sim.key === key ? sim : null;
  // Nothing unseen: the other player has gone out.
  const over = !!lab && lab.unseen.length === 0;

  useEffect(
    () =>
      onLabEvent((e) =>
        setSim((s) => {
          if (!s || e.job !== s.job) return s;
          // another job has taken the worker, and this one is gone
          if (e.freed) return null;
          return {
            ...s,
            answer: (e.answer as SimAnswer) ?? s.answer,
            error: e.error,
            // a step that was under way when a stop was sent leaves it stopped
            running: s.running && !e.done && !e.error,
            done: !!e.done,
          };
        }),
      ),
    [],
  );

  // A sim is freed when another is made, and with the analyzer.
  const job = sim?.job;
  useEffect(
    () => () => {
      if (job !== undefined) {
        labCall({ op: "free", args: { job } }).catch(() => {});
      }
    },
    [job],
  );

  const running = !!sim?.running;
  const ranMs = useRunMs(sim?.job, running);
  const runningKey = sim?.key;
  useEffect(() => {
    if (running && (!shown || runningKey !== key)) {
      labCall({ op: "stop" }).catch(() => {});
      setSim((s) => (s ? { ...s, running: false } : s));
    }
  }, [running, runningKey, shown, key]);

  const stop = () => {
    labCall({ op: "stop" }).catch(() => {});
    setSim((s) => (s ? { ...s, running: false } : s));
  };

  // Run the sim made on, each step as long as a frame.
  const runOn = async (job: number) => {
    const ms = await frameMs();
    setSim((s) => (s?.job === job ? { ...s, running: true, done: false } : s));
    labCall({
      op: "run",
      args: { job, ms },
    }).catch((e) =>
      setSim((s) =>
        s?.job === job ? { ...s, running: false, error: errorText(e) } : s,
      ),
    );
  };

  // Run the sim made of this position on, or make one of the moves listed.
  const run = async () => {
    if (!lab) return;
    if (made) {
      runOn(made.job);
      return;
    }
    if (startingRef.current) return;
    startingRef.current = true;
    setStarting(true);
    setNote(undefined);
    const at = lab.key;
    const plays = candidates;
    try {
      const [ms] = await Promise.all([
        frameMs(),
        labCall({
          op: "prepare",
          args: { files: getLoadableFiles(lab.loadableKey) },
        }),
      ]);
      const { job, added } = await labCall<{
        job: number;
        added: Array<{ play: WasmPlay }>;
      }>({
        op: "sim",
        args: {
          request: JSON.stringify({
            ...lab.position,
            leave: lab.leave,
            plies: DEFAULT_SIM_PLIES,
          }),
          plays: JSON.stringify(plays.map((m) => JSON.parse(m.jsonKey))),
        },
      });
      // A play is held as the sim writes it, which can differ from how it was
      // given: a one-tile play written down is held across.
      setSim({
        key: at,
        job,
        held: new Map(added.map((a, i) => [playKey(a.play), plays[i]])),
        running: true,
        done: false,
      });
      await labCall({
        op: "run",
        args: { job, ms },
      });
    } catch (e) {
      setSim((s) => (s?.key === at ? { ...s, running: false } : s));
      setNote({ key: at, text: `Not started: ${errorText(e)}` });
    } finally {
      startingRef.current = false;
      setStarting(false);
    }
  };

  // The rows shown: the sim's candidates as its last answer has them, those
  // rolled out first, by the chance to win and then equity, and the rest
  // after; or, before a sim, the moves by their static equity.
  const rows = useMemo(() => {
    if (made?.answer) {
      return made.answer.candidates
        .slice()
        .sort((a, b) => Number(a.n === 0) - Number(b.n === 0) || simOrder(a, b))
        .flatMap((c) => {
          const move = made.held.get(playKey(c.play));
          return move ? [{ move, play: c.play, stat: c }] : [];
        });
    }
    return [...candidates]
      .sort((a, b) => b.equity - a.equity)
      .map((move) => ({
        move,
        play: JSON.parse(move.jsonKey) as WasmPlay,
        stat: undefined,
      }));
  }, [made, candidates]);

  const runningCandidates =
    made?.answer?.candidates.filter((c) => c.state === "running").length ?? 0;
  return {
    made,
    candidates,
    rows,
    note: note && note.key === key ? note.text : undefined,
    starting,
    running,
    ranMs,
    canRun: made
      ? !made.error && !made.done && runningCandidates > 0
      : !over && !starting && candidates.length > 1,
    run,
    stop,
  };
};

type SimPanelProps = {
  lab: LabPosition;
  // the analyzer's moves, which are in once it has analyzed the position
  moves: Array<AnalyzerMove> | null | undefined;
  sim: LabSim;
  onClose: () => void;
};

export const SimPanel = (props: SimPanelProps) => {
  const { lab, moves, sim, onClose } = props;
  const placeMove = usePlaceMoveCallback();
  // Nothing unseen: the other player has gone out.
  const over = lab.unseen.length === 0;
  const { made } = sim;

  // While the pointer is in the rows, they keep the order they had when it
  // came, so that what is clicked is the move that was there; once it has
  // left, the rows take the sim's order again.
  const [order, setOrder] = useState<Array<string> | null>(null);
  const rows = useMemo(() => {
    if (!order) return sim.rows;
    const at = new Map(order.map((k, i) => [k, i]));
    return [...sim.rows].sort(
      (a, b) =>
        (at.get(a.move.jsonKey) ?? at.size) -
        (at.get(b.move.jsonKey) ?? at.size),
    );
  }, [sim.rows, order]);
  const holdBy = (on: boolean) =>
    setOrder((o) => (on ? (o ?? rows.map((r) => r.move.jsonKey)) : null));
  // The row a press began in, which its click acts on: on touch the pointer
  // leaves as the finger lifts, before the click comes, and the rows may take
  // the sim's order between the two. A click that no press made (its detail 0,
  // as a screen reader's) acts on the row it lands on.
  const pressed = useRef<LabSim["rows"][number] | null>(null);
  const rowFor = (place: number, e: React.UIEvent) => {
    const row = e.detail > 0 && pressed.current ? pressed.current : rows[place];
    pressed.current = null;
    return row;
  };
  // The phony among the moves listed, which the sim leaves out.
  const phonies = over ? [] : (moves ?? []).filter((m) => !(m.valid ?? true));

  let status: React.ReactNode;
  if (over) {
    status = goneOutText(lab);
  } else if (sim.note) {
    status = <span className="analyzer-error">{sim.note}</span>;
  } else if (made?.error) {
    status = <span className="analyzer-error">{made.error}</span>;
  } else if (made?.answer) {
    const running = made.answer.candidates.filter(
      (c) => c.state === "running",
    ).length;
    status = `${made.answer.iterations} iterations after ${ranText(sim.ranMs)}, ${running} running${made.running ? "" : made.done || running === 0 ? ", done" : ", stopped"}`;
  } else if (made || sim.starting) {
    status = "Starting...";
  } else if (moves == null) {
    status = "Analyzing...";
  } else if (sim.candidates.length < 2) {
    status = "Needs two or more moves to sim.";
  } else {
    status = "Static equity until the sim starts.";
  }

  return (
    <PanelFrame
      title="Sim"
      status={
        <React.Fragment>
          {status}
          {phonies.map((m) => (
            <div key={m.jsonKey}>{leftOutText(m, lab.position.lexicon)}</div>
          ))}
        </React.Fragment>
      }
      running={sim.running}
      canResume={sim.canRun}
      resumeLabel={made ? "Resume" : "Start"}
      stop={sim.stop}
      resume={sim.run}
      onClose={onClose}
    >
      <table
        // A mouse or a pen holds the rows from when it moves over them, not
        // from when the page says it came, which the page says too of a mouse
        // that never moved once what was under it went; a touch holds them
        // while it presses.
        onPointerMove={(e) => {
          if (e.pointerType !== "touch") holdBy(true);
        }}
        onPointerDown={(e) => {
          if (e.pointerType === "touch") holdBy(true);
        }}
        onPointerUp={(e) => {
          if (e.pointerType === "touch") holdBy(false);
        }}
        onPointerCancel={() => holdBy(false)}
        onPointerLeave={() => holdBy(false)}
      >
        {rows.length > 0 && (
          <thead>
            <tr>
              <th className="move-coords" />
              <th className="move" />
              <th className="move-score">
                <ColumnName name="Win" tip="The chance to win" />
              </th>
              <th className="move-equity">
                <ColumnName
                  name="Equity"
                  tip="The spread it gains, leaves counted; before its rollouts, its static equity"
                />
              </th>
              <th className="lab-bingo">
                <ColumnName
                  name="Bingo"
                  tip="How often the opponent's reply is a bingo"
                />
              </th>
              <th className="lab-n">
                <ColumnName name="N" tip="Its rollouts" />
              </th>
            </tr>
          </thead>
        )}
        <tbody>
          {rows.map(({ move: m, stat }, place) => (
            <tr
              key={m.jsonKey}
              className={
                stat && stat.state !== "running" ? "lab-pruned" : undefined
              }
              onPointerDown={() => {
                pressed.current = rows[place];
              }}
              onPointerCancel={() => {
                pressed.current = null;
              }}
              onClick={(e) => placeMove(rowFor(place, e).move)}
            >
              <td className="move-coords">{m.coordinates}</td>
              <td className="move">{m.displayMove}</td>
              <td className="move-score">
                {stat?.n ? percent(stat.win.mean) : ""}
              </td>
              <td className="move-equity">
                {signed(stat?.n ? stat.equity.mean : m.equity, 2)}
              </td>
              <td className="lab-bingo">
                {stat?.n && stat.plies?.[0] ? percent(stat.plies[0].bingo) : ""}
              </td>
              <td className="lab-n">{stat?.n}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </PanelFrame>
  );
};

type LabButtonsProps = {
  view: LabView | null;
  setView: (view: LabView | null) => void;
};

export const LabButtons = (props: LabButtonsProps) => {
  const { view, setView } = props;
  return (
    <span className="lab-controls">
      <Tooltip title="Sim these moves">
        <Button
          shape="circle"
          size="small"
          icon={<ExperimentOutlined />}
          type={view === "sim" ? "primary" : "default"}
          onClick={() => setView(view === "sim" ? null : "sim")}
        />
      </Tooltip>
    </span>
  );
};
