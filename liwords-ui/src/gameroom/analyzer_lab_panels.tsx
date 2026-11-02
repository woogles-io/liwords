import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  Button,
  Checkbox,
  Input,
  InputNumber,
  Popconfirm,
  Popover,
  Radio,
  Tooltip,
} from "antd";
import {
  CaretRightOutlined,
  CloseOutlined,
  ExperimentOutlined,
  FlagOutlined,
  OrderedListOutlined,
  PauseOutlined,
  PlusOutlined,
  ProfileOutlined,
  SettingOutlined,
  SwapOutlined,
} from "@ant-design/icons";
import {
  Alphabet,
  machineLetterToRune,
  machineWordToRunes,
  runesToMachineWord,
} from "../constants/alphabets";
import {
  ChallengeRule,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import {
  useExaminableGameContextStoreContext,
  useTentativeTileContext,
} from "../store/store";
import { Board } from "../utils/cwgame/board";
import {
  Direction,
  EphemeralTile,
  MachineLetter,
  MachineWord,
} from "../utils/cwgame/common";
import { contiguousTilesFromTileSet } from "../utils/cwgame/scoring";
import { Modal } from "../utils/focus_modal";
import {
  refitOnPageChanges,
  roomClass,
  roomInWindow,
} from "../utils/window_room";
import { labCall, onLabEvent } from "../wasm/lab";
import { getLoadableFiles, getWolges } from "../wasm/loader";
import {
  AnalyzerMove,
  JsonMove,
  analyzerMoveFromJsonMove,
  liwordsLetterToWolgesLetter,
  parseExaminableGameContext,
  usePlaceMoveCallback,
  wolgesLetterToLiwordsLetter,
} from "./analyzer";
import {
  SimCandidate,
  SimStat,
  endgameBag,
  endgameMemoryMb,
  labNow,
  passesAtEnd,
  scorelessTurnsAtEnd,
  simOrder,
  unseenBy,
} from "./analyzer_lab";

export type LabView = "sim" | "endgame";

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

type EndgameAnswer = {
  steps: number;
  exact: boolean;
  play: WasmPlay | null;
  value: number;
  spread: number;
  depth: number;
} & (
  | { bag: 0; line: Array<{ play: WasmPlay; value: number }> }
  | {
      bag: 1;
      win: number;
      tiles: Array<{ tile: number; count: number; value: number }>;
    }
);

// A ply of a candidate's rollouts: how many reached it, the score made on it,
// and the share of them whose play took a whole rack.
type SimPly = { n: number; score: SimStat; bingo: number };

// A sim's answer: its iterations, and its candidates, each with its rollouts'
// plies, the first of them the opponent's reply.
type SimAnswer = {
  iterations: number;
  candidates: Array<SimCandidate & { play: WasmPlay; plies?: Array<SimPly> }>;
};

// A play as the wasm's play_score answers for it: checked, its score
// recounted, and its static equity; or why it is no play here.
type ScoredPlay =
  | (WasmPlay & {
      result: "scored";
      valid: boolean;
      invalid_words: Array<Array<number>>;
      equity: number;
    })
  | { result: "error"; error: string };

// The examined position as the lab's jobs take it.
export type LabPosition = {
  loadableKey: string;
  dim: number;
  letters: Array<MachineLetter>;
  alphabet: Alphabet;
  rack: MachineWord;
  unseen: MachineWord;
  // the fields the wasm's sim_new and endgame_new share, less the leave
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
  // each player's name, by seat
  names: Array<string>;
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
    const names = ["", ""];
    for (const [name, seat] of Object.entries(gameContext.nickToPlayerOrder)) {
      names[seat === "p1" ? 1 : 0] = name;
    }
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
      names,
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

const toAnalyzerMove = (
  play: WasmPlay,
  value: number,
  letters: Array<MachineLetter>,
  rack: MachineWord,
  lab: LabPosition,
) =>
  analyzerMoveFromJsonMove(
    { ...play, equity: value } as JsonMove,
    lab.dim,
    letters,
    rack,
    lab.alphabet,
  );

// The plays of an endgame's line, each made on the board the plays before it
// leave, from the rack its player then holds: the player to move's, then the
// other's, which with an empty bag is every unseen tile.
const lineMoves = (
  line: Array<{ play: WasmPlay; value: number }>,
  lab: LabPosition,
) => {
  const letters = [...lab.letters];
  const racks = [lab.rack, lab.unseen];
  return line.map(({ play, value }, ply) => {
    const move = toAnalyzerMove(play, value, letters, racks[ply % 2], lab);
    if (!move.isExchange) {
      let row = move.row;
      let col = move.col;
      for (const t of move.tiles) {
        if (t !== 0)
          letters[row * lab.dim + col] = wolgesLetterToLiwordsLetter(t);
        if (move.vertical) ++row;
        else ++col;
      }
    }
    racks[ply % 2] = move.leave;
    return move;
  });
};

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

type JobState<T> = {
  job?: number;
  answer?: T;
  error?: string;
  running: boolean;
};

// A job in the lab's worker for as long as the panel shows it: made when the
// position changes, run in steps as long as a frame, and freed when the panel
// goes.
const useLabJob = <T,>(start: (() => Promise<number>) | null, key: string) => {
  const [state, setState] = useState<JobState<T>>({ running: false });
  useEffect(() => {
    setState({ running: false });
    if (!start) return;
    let current = true;
    let job: number | undefined;
    const stopListening = onLabEvent((e) => {
      if (!current || e.job !== job) return;
      setState((s) => ({
        ...s,
        answer: (e.answer as T) ?? s.answer,
        error: e.error,
        // a step that was under way when a stop was sent leaves it stopped
        running: s.running && !e.done && !e.error,
      }));
    });
    (async () => {
      try {
        const made = await start();
        if (!current) {
          labCall({ op: "free", args: { job: made } }).catch(() => {});
          return;
        }
        job = made;
        const ms = await frameMs();
        if (!current) return;
        setState({ job, running: true });
        await labCall({ op: "run", args: { job, ms } });
      } catch (e) {
        if (current) {
          setState({
            running: false,
            error: e instanceof Error ? e.message : String(e),
          });
        }
      }
    })();
    return () => {
      current = false;
      stopListening();
      if (job !== undefined) {
        labCall({ op: "free", args: { job } }).catch(() => {});
      }
    };
    // start is made anew on each render; key says when it differs
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);
  const stop = useCallback(() => {
    labCall({ op: "stop" }).catch(() => {});
    setState((s) => ({ ...s, running: false }));
  }, []);
  const resume = useCallback(async () => {
    const { job } = state;
    if (job === undefined) return;
    const ms = await frameMs();
    setState((s) => ({ ...s, running: true }));
    labCall({ op: "run", args: { job, ms } }).catch((e) =>
      setState((s) => ({ ...s, running: false, error: String(e) })),
    );
  }, [state]);
  return { ...state, stop, resume };
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

type EndgamePanelProps = {
  lab: LabPosition;
  onClose: () => void;
};

export const EndgamePanel = (props: EndgamePanelProps) => {
  const { lab, onClose } = props;
  const placeMove = usePlaceMoveCallback();
  // Nothing unseen: the other player has gone out.
  const over = lab.unseen.length === 0;
  const bag = endgameBag(lab.unseen.length, lab.rack.length);
  const solvable = !over && bag !== null && bag <= 1;
  const job = useLabJob<EndgameAnswer>(
    solvable
      ? async () => {
          // An endgame makes its own word graph from the lexicon's, and reads
          // no leaves.
          await labCall({
            op: "prepare",
            args: {
              files: getLoadableFiles(lab.loadableKey).filter((f) =>
                f.cacheKey.startsWith("kwg/"),
              ),
            },
          });
          return labCall<number>({
            op: "endgame",
            args: {
              request: JSON.stringify({
                ...lab.position,
                max_memory_mb: endgameMemoryMb(),
              }),
            },
          });
        }
      : null,
    lab.key,
  );
  const { answer } = job;
  // the answer from its first depth on: before that, it has no value or win
  // to give, which the wasm leaves out or gives as 0
  const solved = answer && answer.depth > 0 ? answer : undefined;
  const ranMs = useRunMs(job.job, job.running);
  const line = useMemo(
    () => (solved?.bag === 0 ? lineMoves(solved.line, lab) : []),
    [solved, lab],
  );
  const best = useMemo(
    () =>
      solved?.bag === 1 && solved.play
        ? toAnalyzerMove(solved.play, solved.value, lab.letters, lab.rack, lab)
        : null,
    [solved, lab],
  );

  // the players by who moves first in the line: the player to move, then the
  // other
  const names = [lab.position.turn, 1 - lab.position.turn].map(
    (seat) => lab.names[seat] || `Player ${seat + 1}`,
  );
  const runes = (tiles: MachineWord) =>
    machineWordToRunes(tiles, lab.alphabet, false, true);

  let status: React.ReactNode;
  if (over) {
    status = goneOutText(lab);
  } else if (!solvable) {
    status =
      bag === null
        ? "Needs the full rack of the player to move."
        : `The bag holds ${bag} tiles.`;
  } else if (job.error) {
    status = <span className="analyzer-error">{job.error}</span>;
  } else if (!solved || !solved.play) {
    status = "Solving...";
  } else {
    const depth = `depth ${solved.depth} after ${ranText(ranMs)}${solved.exact ? ", exact" : job.running ? ", solving" : ", stopped"}`;
    status =
      solved.bag === 0
        ? `Value ${signed(solved.value)}, spread ${signed(solved.spread)}, ${depth}`
        : `Expected value ${signed(solved.value, 2)}, wins ${percent(solved.win)}, ${depth}`;
  }

  return (
    <PanelFrame
      title={bag === 1 ? "Endgame, one tile in the bag" : "Endgame"}
      status={status}
      running={job.running}
      canResume={job.job !== undefined && !answer?.exact && !job.error}
      stop={job.stop}
      resume={job.resume}
      onClose={onClose}
    >
      {solvable && (
        // what each player holds: with an empty bag the other player holds
        // every tile unseen, and with one in the bag all but that one
        <div className="lab-racks">
          <div>
            {names[0]} to move: {runes(lab.rack)}
          </div>
          <div>
            {bag === 0 ? names[1] : "Unseen"}: {runes(lab.unseen)}
          </div>
        </div>
      )}
      <table>
        {(line.length > 0 || best) && (
          <thead>
            <tr>
              <th className="lab-who" />
              <th className="move-coords" />
              <th className="move" />
              <th className="move-score">Score</th>
              <th className="move-leave">Leave</th>
              <th className="move-equity">
                <ColumnName
                  name="Value"
                  tip="What the player gains from here to the game's end"
                />
              </th>
            </tr>
          </thead>
        )}
        <tbody>
          {line.map((m, ply) => (
            <tr
              key={ply}
              className={ply % 2 ? "lab-other" : undefined}
              onClick={() => placeMove(m, line.slice(0, ply))}
            >
              <td className="lab-who">{names[ply % 2]}</td>
              <td className="move-coords">{m.coordinates}</td>
              <td className="move">{m.displayMove}</td>
              <td className="move-score">{m.score}</td>
              <td className="move-leave">{runes(m.leave)}</td>
              <td className="move-equity">{signed(m.equity)}</td>
            </tr>
          ))}
          {best && (
            <tr onClick={() => placeMove(best)}>
              <td className="lab-who">{names[0]}</td>
              <td className="move-coords">{best.coordinates}</td>
              <td className="move">{best.displayMove}</td>
              <td className="move-score">{best.score}</td>
              <td className="move-leave">{runes(best.leave)}</td>
              <td className="move-equity">{signed(best.equity, 2)}</td>
            </tr>
          )}
          {solved?.bag === 1 &&
            solved.tiles.map((t) => (
              <tr key={t.tile} className="lab-tile">
                <td className="lab-who" />
                <td className="move-coords" />
                <td className="move" colSpan={3}>
                  {machineLetterToRune(t.tile, lab.alphabet)} in the bag
                  {t.count > 1 ? ` (${t.count})` : ""}
                </td>
                <td className="move-equity">{signed(t.value)}</td>
              </tr>
            ))}
        </tbody>
      </table>
    </PanelFrame>
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

// The move placed on the board, as the wasm writes a play (its score 0, which
// play_score counts again), or null if the tiles placed make none.
const placedPlay = (
  placed: Set<EphemeralTile>,
  board: Board,
): WasmPlay | null => {
  const run = placed.size ? contiguousTilesFromTileSet(placed, board) : null;
  if (!run) return null;
  const [tiles, direction] = run;
  const down = direction === Direction.Vertical;
  return {
    action: "play",
    down,
    lane: down ? tiles[0].col : tiles[0].row,
    idx: down ? tiles[0].row : tiles[0].col,
    word: tiles.map((t) =>
      t.fresh ? liwordsLetterToWolgesLetter(t.letter) : 0,
    ),
    score: 0,
  };
};

// The user's own changes to a position's candidates: the moves added, and the
// plays taken out (by playKey).
type SimEdits = {
  added: Array<AnalyzerMove>;
  removed: Array<string>;
};

const noEdits: SimEdits = { added: [], removed: [] };

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

// The plies a sim's rollouts play until the user sets others: liwords's own
// count, sent with every sim made.
const DEFAULT_SIM_PLIES = 4;

// What a sim is made with, as the user sets it: the plies each rollout plays,
// and the tiles known to be on the opponent's rack, written as tiles are, a
// blank as ?; and whether it prunes none of its candidates, rolling each out
// until stopped.
export type SimSettings = { plies: number; known: string; unpruned: boolean };

const defaultSimSettings: SimSettings = {
  plies: DEFAULT_SIM_PLIES,
  known: "",
  unpruned: false,
};

// The tiles written as known on the opponent's rack, or why they cannot be
// there: each one a tile of the game, and all of them among the tiles unseen.
const knownTiles = (
  written: string,
  lab: LabPosition,
): { tiles: MachineWord } | { error: string } => {
  const text = written.replace(/\s/g, "").toUpperCase();
  let tiles: MachineWord;
  try {
    tiles = [
      ...Array.from(text.matchAll(/\?/g), () => 0),
      ...runesToMachineWord(text.replace(/\?/g, ""), lab.alphabet),
    ];
  } catch {
    return { error: `${written.trim()} is not tiles of this game` };
  }
  const left = [...lab.unseen];
  for (const t of tiles) {
    const at = left.indexOf(t);
    if (at < 0) return { error: `${text} are not all among the tiles unseen` };
    left.splice(at, 1);
  }
  return { tiles };
};

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

// A play checked and scored on the page's own wasm, as the analyzer checks and
// scores the move played.
const scorePlay = async (lab: LabPosition, play: WasmPlay) => {
  const wolges = await getWolges(lab.loadableKey);
  const [scored]: Array<ScoredPlay> = JSON.parse(
    await wolges.play_score(
      JSON.stringify({ ...lab.position, leave: lab.leave, plays: [play] }),
    ),
  );
  return scored;
};

export type LabSim = ReturnType<typeof useLabSim>;

// The examined position's sim, which the analyzer keeps while it is open, so
// that closing the panel only stops it. Each position keeps the moves the user
// added and took out, and the last sim made keeps its rollouts until another
// job takes the lab's worker. Nothing is simmed until the user starts it, with
// the moves listed then; until then each shows its static equity.
export const useLabSim = (
  lab: LabPosition | null,
  // the analyzer's moves, the sim's candidates
  moves: Array<AnalyzerMove> | null | undefined,
  // whether the panel shows: a sim hidden, or left for another position, stops
  shown: boolean,
) => {
  const [allEdits, setAllEdits] = useState(() => new Map<string, SimEdits>());
  const [sim, setSim] = useState<MadeSim | null>(null);
  const [settings, setSettings] = useState(defaultSimSettings);
  const [starting, setStarting] = useState(false);
  const startingRef = useRef(false);
  // what the user last tried that came to nothing, at the position it was
  // tried at, until the next try
  const [note, setNote] = useState<{ key: string; text: string }>();
  const key = lab?.key;
  const edits = (key !== undefined && allEdits.get(key)) || noEdits;
  const changeEdits = (at: string, change: (e: SimEdits) => SimEdits) =>
    setAllEdits((all) => new Map(all).set(at, change(all.get(at) ?? noEdits)));
  // The moves listed, the one played among them; a phony is no candidate, as
  // the wasm turns down a call with one.
  const listed = useMemo(
    () => (moves ?? []).filter((m) => m.valid ?? true),
    [moves],
  );
  // what a sim made now holds: the moves listed, with the user's changes
  const candidates = useMemo(
    () => [
      ...listed.filter(
        (m) => !edits.removed.includes(playKey(JSON.parse(m.jsonKey))),
      ),
      ...edits.added.filter(
        (a) => !listed.some((m) => m.jsonKey === a.jsonKey),
      ),
    ],
    [listed, edits],
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
  const runOn = async (job: number, unpruned: boolean) => {
    const ms = await frameMs();
    setSim((s) => (s?.job === job ? { ...s, running: true, done: false } : s));
    labCall({
      op: "run",
      args: unpruned ? { job, ms, prune: false } : { job, ms },
    }).catch((e) =>
      setSim((s) =>
        s?.job === job ? { ...s, running: false, error: errorText(e) } : s,
      ),
    );
  };

  // New plies or tiles are for the next sim made; the sim made goes, as its
  // rollouts were made with the old ones. Whether it prunes applies to the
  // sim made from its next step on.
  const changeSettings = (next: SimSettings) => {
    setSettings(next);
    setNote(undefined);
    if (next.plies !== settings.plies || next.known !== settings.known) {
      setSim(null);
    } else if (made && next.unpruned !== settings.unpruned) {
      if (made.running) runOn(made.job, next.unpruned);
      // done as it was pruning, it may have more to do now
      else setSim((s) => (s?.job === made.job ? { ...s, done: false } : s));
    }
  };

  // Run the sim made of this position on, or make one of the moves listed.
  const run = async () => {
    if (!lab) return;
    if (made) {
      runOn(made.job, settings.unpruned);
      return;
    }
    if (startingRef.current) return;
    const at = lab.key;
    const known = knownTiles(settings.known, lab);
    if ("error" in known) {
      setNote({ key: at, text: `Not started: ${known.error}.` });
      return;
    }
    startingRef.current = true;
    setStarting(true);
    setNote(undefined);
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
            plies: settings.plies,
            known: known.tiles,
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
        args: settings.unpruned ? { job, ms, prune: false } : { job, ms },
      });
    } catch (e) {
      setSim((s) => (s?.key === at ? { ...s, running: false } : s));
      setNote({ key: at, text: `Not started: ${errorText(e)}` });
    } finally {
      startingRef.current = false;
      setStarting(false);
    }
  };

  // Take a candidate out; a sim made of the position forgets it and its
  // rollouts.
  const remove = (play: WasmPlay) => {
    if (!lab) return;
    const at = lab.key;
    const removed = playKey(play);
    setNote(undefined);
    changeEdits(at, (e) => ({
      added: e.added.filter((m) => playKey(JSON.parse(m.jsonKey)) !== removed),
      removed: [...e.removed, removed],
    }));
    if (!made) return;
    const { job } = made;
    labCall({ op: "remove", args: { job, plays: JSON.stringify([play]) } })
      .then(() => labCall({ op: "read", args: { job } }))
      .catch((e) =>
        setNote({ key: at, text: `Not taken out: ${errorText(e)}` }),
      );
  };

  // Pause a candidate, keeping its rollouts, or roll a paused or pruned one out
  // again from the sim's next iteration on.
  const hold = (play: WasmPlay, paused: boolean) => {
    if (!lab || !made) return;
    const at = lab.key;
    const { job } = made;
    setNote(undefined);
    labCall({
      op: paused ? "pause" : "resume",
      args: { job, plays: JSON.stringify([play]) },
    })
      .then(() => labCall({ op: "read", args: { job } }))
      .catch((e) =>
        setNote({
          key: at,
          text: `${paused ? "Not paused" : "Not resumed"}: ${errorText(e)}`,
        }),
      );
  };

  // Add a move, checked and scored as the analyzer checks and scores the move
  // played, so that it shows at once with its static equity; a sim made of the
  // position rolls it out from its next iteration on.
  const add = async (play: WasmPlay) => {
    if (!lab) return;
    const at = lab.key;
    const refuse = (text: string) =>
      setNote({ key: at, text: `Not added: ${text}` });
    setNote(undefined);
    let scored: ScoredPlay;
    try {
      scored = await scorePlay(lab, play);
    } catch (e) {
      refuse(errorText(e));
      return;
    }
    if (scored.result === "error") {
      refuse(scored.error);
      return;
    }
    if (!scored.valid) {
      const words = scored.invalid_words.map((w) =>
        machineWordToRunes(w, lab.alphabet, false, true),
      );
      refuse(
        `${listWords(words)} ${words.length > 1 ? "are" : "is"} not in ${lab.position.lexicon ?? "the lexicon"}.`,
      );
      return;
    }
    const added = playKey(scored);
    if (candidates.some((m) => playKey(JSON.parse(m.jsonKey)) === added)) {
      setNote({ key: at, text: "That move is listed already." });
      return;
    }
    const move = toAnalyzerMove(
      scored,
      scored.equity,
      lab.letters,
      lab.rack,
      lab,
    );
    changeEdits(at, (e) => ({
      added: [...e.added, move],
      removed: e.removed.filter((r) => r !== added),
    }));
    if (!made) return;
    const { job } = made;
    try {
      const [joined] = await labCall<Array<{ play: WasmPlay }>>({
        op: "add",
        args: { job, plays: JSON.stringify([JSON.parse(added)]) },
      });
      setSim((s) =>
        s?.job === job
          ? { ...s, held: new Map(s.held).set(playKey(joined.play), move) }
          : s,
      );
      await labCall({ op: "read", args: { job } });
    } catch (e) {
      refuse(errorText(e));
    }
  };

  // Add the position's top moves, as many as asked, to those listed: the lab's
  // worker finds them, each with its static equity, and a sim made of the
  // position rolls out those it lacks from its next iteration on.
  const addTop = async (count: number) => {
    if (!lab) return;
    const at = lab.key;
    const refuse = (text: string) =>
      setNote({ key: at, text: `Not added: ${text}` });
    setNote(undefined);
    let plays: Array<WasmPlay & { equity: number }>;
    try {
      await labCall({
        op: "prepare",
        args: { files: getLoadableFiles(lab.loadableKey) },
      });
      plays = await labCall({
        op: "analyze",
        args: {
          request: JSON.stringify({ ...lab.position, leave: lab.leave, count }),
        },
      });
    } catch (e) {
      refuse(errorText(e));
      return;
    }
    const have = new Set(candidates.map((m) => playKey(JSON.parse(m.jsonKey))));
    const fresh = plays.filter((p) => !have.has(playKey(p)));
    if (fresh.length === 0) {
      setNote({ key: at, text: "Those moves are listed already." });
      return;
    }
    const keys = fresh.map(playKey);
    const moves = fresh.map((p) =>
      toAnalyzerMove(p, p.equity, lab.letters, lab.rack, lab),
    );
    changeEdits(at, (e) => ({
      added: [...e.added, ...moves],
      removed: e.removed.filter((r) => !keys.includes(r)),
    }));
    if (!made) return;
    const { job } = made;
    try {
      const joined = await labCall<Array<{ play: WasmPlay }>>({
        op: "add",
        args: { job, plays: JSON.stringify(keys.map((k) => JSON.parse(k))) },
      });
      setSim((s) => {
        if (s?.job !== job) return s;
        const held = new Map(s.held);
        joined.forEach((j, i) => held.set(playKey(j.play), moves[i]));
        return { ...s, held };
      });
      await labCall({ op: "read", args: { job } });
    } catch (e) {
      refuse(errorText(e));
    }
  };

  // The rows shown: the sim's candidates as its last answer has them, those
  // rolled out first, by the chance to win and then equity, and the rest
  // after; or, before a sim, the moves by their static equity.
  const rows = useMemo(() => {
    if (made?.answer) {
      return made.answer.candidates
        .filter((c) => !edits.removed.includes(playKey(c.play)))
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
  }, [made, edits.removed, candidates]);

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
      : !over && !starting && candidates.length > (settings.unpruned ? 0 : 1),
    run,
    stop,
    add,
    remove,
    hold,
    addTop,
    settings,
    changeSettings,
  };
};

// What a tap on a row's button asks before it acts, until it is answered.
type Asking = {
  question: string;
  okText: string;
  confirm: () => void;
  cancel: () => void;
};

type RowButtonProps = {
  label: string;
  tip: string;
  icon: typeof CloseOutlined;
  onClick: (e: React.UIEvent) => void;
  asking?: Asking;
};

// An icon in a candidate's row that acts on the candidate, clicked or keyed,
// without also placing its move as a click on the row does. A press on it
// leaves the keyboard where it was, so that a click holds the rows' order no
// longer than the pointer stays; a tap asks first, by the icon, in place of
// its tooltip. The tooltip holds the same element whichever icon it shows, as
// a tooltip given another sets itself up again.
const RowButton = (props: RowButtonProps) => {
  const { label, tip, icon: Icon, onClick, asking } = props;
  const act = (e: React.UIEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onClick(e);
  };
  const button = (
    <span
      role="button"
      aria-label={label}
      tabIndex={0}
      onMouseDown={(e) => e.preventDefault()}
      onClick={act}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") act(e);
      }}
    >
      <Icon />
    </span>
  );
  return asking ? (
    <Popconfirm
      open
      title={asking.question}
      okText={asking.okText}
      cancelText="Cancel"
      onConfirm={asking.confirm}
      onCancel={asking.cancel}
      onOpenChange={(open) => {
        if (!open) asking.cancel();
      }}
    >
      {button}
    </Popconfirm>
  ) : (
    <Tooltip title={tip}>{button}</Tooltip>
  );
};

// A move as a question names it: where it goes, then the move.
const moveName = (m: AnalyzerMove) =>
  [m.coordinates, m.displayMove].filter(Boolean).join(" ");

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
  const { placedTiles } = useTentativeTileContext();
  const { gameContext } = useExaminableGameContextStoreContext();
  const placed = useMemo(
    () => placedPlay(placedTiles, gameContext.board),
    [placedTiles, gameContext.board],
  );
  // Nothing unseen: the other player has gone out.
  const over = lab.unseen.length === 0;
  const { made } = sim;
  // While a sim starts, the moves it is made of stay as they are.
  const fixed = sim.starting;
  const [details, setDetails] = useState(false);

  // While the pointer or the keyboard is in the rows, or a tap's question is
  // asked, they keep the order they had when it came, so that what is clicked,
  // keyed, or tapped is the move that was there; once all have gone, the rows
  // take the sim's order again.
  const [hold, setHold] = useState<{
    pointer: boolean;
    focus: boolean;
    ask: boolean;
    order: Array<string> | null;
  }>({ pointer: false, focus: false, ask: false, order: null });
  const order = hold.order;
  const rows = useMemo(() => {
    if (!order) return sim.rows;
    const at = new Map(order.map((k, i) => [k, i]));
    return [...sim.rows].sort(
      (a, b) =>
        (at.get(a.move.jsonKey) ?? at.size) -
        (at.get(b.move.jsonKey) ?? at.size),
    );
  }, [sim.rows, order]);
  // a hold begun by a tap's question keeps the order the tap landed on
  const holdBy = (
    by: "pointer" | "focus" | "ask",
    on: boolean,
    from?: Array<string>,
  ) =>
    setHold((h) => {
      // the same state when nothing changes, so that a mouse moving over the
      // rows renders nothing more
      if (h[by] === on) return h;
      const next = { ...h, [by]: on };
      return {
        ...next,
        order:
          next.pointer || next.focus || next.ask
            ? (h.order ?? from ?? rows.map((r) => r.move.jsonKey))
            : null,
      };
    });
  // The row a press began in, which its click acts on: on touch the pointer
  // leaves as the finger lifts, before the click comes, and the rows may take
  // the sim's order between the two. A click that no press made (its detail 0,
  // as a screen reader's) acts on the row it lands on.
  const pressed = useRef<{
    row: LabSim["rows"][number];
    touch: boolean;
    order: Array<string>;
  } | null>(null);
  const pressAt = (place: number, e: React.UIEvent) => {
    const at = e.detail > 0 ? pressed.current : null;
    pressed.current = null;
    return at ?? { row: rows[place], touch: false, order: undefined };
  };
  const rowFor = (place: number, e: React.UIEvent) => pressAt(place, e).row;

  // On a touch screen nothing holds the rows while a finger aims, so a tap on
  // a row's button asks first, naming the move it landed on: a remove or a
  // hold, at a place, at the position it was asked.
  const [asking, setAsking] = useState<{
    key: string;
    place: number;
    button: "remove" | "hold";
    question: string;
    okText: string;
    act: () => void;
  } | null>(null);
  const stopAsking = () => {
    setAsking(null);
    holdBy("ask", false);
  };
  // A click or a key acts on its row at once; a tap asks.
  const actOn = (
    place: number,
    button: "remove" | "hold",
    e: React.UIEvent,
    verb: (row: LabSim["rows"][number]) => string,
    act: (row: LabSim["rows"][number]) => void,
  ) => {
    const { row, touch, order: landed } = pressAt(place, e);
    if (!touch) {
      act(row);
      return;
    }
    holdBy("ask", true, landed);
    setAsking({
      key: lab.key,
      place,
      button,
      question: `${verb(row)} ${moveName(row.move)}?`,
      okText: verb(row),
      act: () => act(row),
    });
  };
  const askingAt = (
    place: number,
    button: "remove" | "hold",
  ): Asking | undefined =>
    asking &&
    asking.key === lab.key &&
    asking.place === place &&
    asking.button === button
      ? {
          question: asking.question,
          okText: asking.okText,
          confirm: () => {
            stopAsking();
            asking.act();
          },
          cancel: stopAsking,
        }
      : undefined;

  // A row taken out with the button the keyboard was on takes the focus with
  // it, and the table hears no blur: the rows let go then too; and a question
  // asked at another position goes.
  const table = useRef<HTMLTableElement>(null);
  useEffect(() => {
    if (hold.focus && !table.current?.contains(document.activeElement)) {
      holdBy("focus", false);
    }
    if (asking && asking.key !== lab.key) stopAsking();
  });
  // Pauses the candidate in a row, or resumes it, as its button showed when
  // the press began.
  const holdRow = (place: number, e: React.UIEvent) =>
    actOn(
      place,
      "hold",
      e,
      (row) => (row.stat?.state === "running" ? "Pause" : "Resume"),
      (row) => sim.hold(row.play, row.stat?.state === "running"),
    );
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
    status = `${made.answer.iterations} iterations after ${ranText(sim.ranMs)}, ${running} running${made.running ? "" : made.done || running === 0 ? ", done" : ", stopped"}${settingsText(sim.settings, made.answer.candidates[0]?.plies?.length)}`;
  } else if (made || sim.starting) {
    status = "Starting...";
  } else if (moves == null) {
    status = "Analyzing...";
  } else if (sim.settings.unpruned && sim.candidates.length === 0) {
    status = "Needs a move to sim.";
  } else if (!sim.settings.unpruned && sim.candidates.length < 2) {
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
      buttons={
        <React.Fragment>
          <Tooltip title="Add the move placed on the board">
            <Button
              shape="circle"
              size="small"
              icon={<PlusOutlined />}
              aria-label="Add"
              disabled={!placed || over || fixed}
              onClick={() => placed && sim.add(placed)}
            />
          </Tooltip>
          <ExchangeButton
            rack={lab.rack}
            alphabet={lab.alphabet}
            disabled={over || lab.rack.length === 0 || fixed}
            onAdd={sim.add}
          />
          <TopMovesButton
            disabled={over || lab.rack.length === 0 || fixed}
            onAdd={sim.addTop}
          />
          <SimSettingsButton
            settings={sim.settings}
            disabled={fixed}
            onChange={sim.changeSettings}
          />
          <Tooltip title="Details">
            <Button
              shape="circle"
              size="small"
              icon={<ProfileOutlined />}
              aria-label="Details"
              disabled={!made?.answer}
              onClick={() => setDetails(true)}
            />
          </Tooltip>
        </React.Fragment>
      }
    >
      {made?.answer && (
        <SimDetails
          open={details}
          onClose={() => setDetails(false)}
          lab={lab}
          iterations={made.answer.iterations}
          settings={sim.settings}
          rows={sim.rows}
        />
      )}
      <table
        // A mouse or a pen holds the rows from when it moves over them, not
        // from when the page says it came, which the page says too of a mouse
        // that never moved once what was under it went; a touch holds them
        // while it presses.
        ref={table}
        onPointerMove={(e) => {
          if (e.pointerType !== "touch") holdBy("pointer", true);
        }}
        onPointerDown={(e) => {
          if (e.pointerType === "touch") holdBy("pointer", true);
        }}
        onPointerUp={(e) => {
          if (e.pointerType === "touch") holdBy("pointer", false);
        }}
        onPointerCancel={() => holdBy("pointer", false)}
        onPointerLeave={() => holdBy("pointer", false)}
        onFocus={() => holdBy("focus", true)}
        onBlur={(e) => {
          if (!e.currentTarget.contains(e.relatedTarget)) {
            holdBy("focus", false);
          }
        }}
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
              <th className="lab-hold" />
              <th className="lab-remove" />
            </tr>
          </thead>
        )}
        <tbody>
          {rows.map(({ move: m, stat }, place) => (
            <tr
              // keyed by place, not by move: a re-sort then changes what each
              // row holds and moves none, as a tooltip moved sets itself up
              // again
              key={place}
              className={
                stat && stat.state !== "running" ? "lab-pruned" : undefined
              }
              onPointerDown={(e) => {
                pressed.current = {
                  row: rows[place],
                  touch: e.pointerType === "touch",
                  order: rows.map((r) => r.move.jsonKey),
                };
              }}
              onPointerCancel={() => {
                pressed.current = null;
              }}
              // a click on what the row shows in front of the page, as a tap's
              // question, comes here too, along React's tree, and is none of
              // the row's
              onClick={(e) => {
                if (e.currentTarget.contains(e.target as Node)) {
                  placeMove(rowFor(place, e).move);
                }
              }}
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
              <td className="lab-hold">
                {stat &&
                  (stat.state === "running" ? (
                    <RowButton
                      label="Pause candidate"
                      tip="Pause"
                      icon={PauseOutlined}
                      onClick={(e) => holdRow(place, e)}
                      asking={askingAt(place, "hold")}
                    />
                  ) : (
                    <RowButton
                      label="Resume candidate"
                      tip="Resume"
                      icon={CaretRightOutlined}
                      onClick={(e) => holdRow(place, e)}
                      asking={askingAt(place, "hold")}
                    />
                  ))}
              </td>
              <td className="lab-remove">
                {!fixed && (
                  <RowButton
                    label="Take out"
                    tip="Take out"
                    icon={CloseOutlined}
                    onClick={(e) =>
                      actOn(
                        place,
                        "remove",
                        e,
                        () => "Take out",
                        (row) => sim.remove(row.play),
                      )
                    }
                    asking={askingAt(place, "remove")}
                  />
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </PanelFrame>
  );
};

// What a ply of a sim's rollouts is called, as Quackle calls them: the
// opponent's next turn and the player's own, then each one's turns by number.
const plyName = (ply: number) => {
  const turn = Math.floor(ply / 2) + 1;
  const whose = ply % 2 ? "Your" : "Opponent's";
  return turn === 1 ? `${whose} next turn` : `${whose} turn ${turn}`;
};

const sd = (stat: SimStat) => Math.sqrt(stat.variance);

type SimDetailsProps = {
  open: boolean;
  onClose: () => void;
  lab: LabPosition;
  iterations: number;
  settings: SimSettings;
  rows: LabSim["rows"];
};

// A sim's numbers in full: each candidate's chance to win and equity with
// their standard deviations, its score, and its static equity; then each
// ply's score, its standard deviation, and how often it is a bingo. They can
// be saved to a file, as the wasm gives them.
const SimDetails = (props: SimDetailsProps) => {
  const { open, onClose, lab, iterations, settings, rows } = props;
  const stats = rows.flatMap(({ move, stat }) =>
    stat ? [{ move, stat }] : [],
  );
  const plies = Math.max(
    0,
    ...stats.map(({ stat }) => stat.plies?.length ?? 0),
  );
  const name = (m: AnalyzerMove) => `${m.coordinates} ${m.displayMove}`.trim();
  // The numbers by ply, a table each, or by move, a block each, as Quackle
  // gives them; a move's row clicked in a table shows its block.
  const [view, setView] = useState<"ply" | "move">("ply");
  const [shown, setShown] = useState<string | null>(null);
  const blocks = useRef(new Map<string, HTMLElement>());
  useEffect(() => {
    if (view === "move" && shown !== null) {
      blocks.current.get(shown)?.scrollIntoView({ block: "nearest" });
    }
  }, [view, shown]);
  const showBlock = (key: string) => {
    setShown(key);
    setView("move");
  };
  const rack = machineWordToRunes(lab.rack, lab.alphabet, false, true);
  const save = () => {
    const saved = {
      rack,
      lexicon: lab.position.lexicon,
      rules: lab.position.rules,
      iterations,
      plies,
      known: settings.known.trim().toUpperCase(),
      candidates: stats.map(({ move: m, stat }) => ({
        move: name(m),
        score: m.score,
        static_equity: m.equity,
        state: stat.state,
        n: stat.n,
        win: stat.win,
        equity: stat.equity,
        plies: stat.plies,
      })),
    };
    const url = URL.createObjectURL(
      new Blob([JSON.stringify(saved, null, 2)], {
        type: "application/json",
      }),
    );
    const link = document.createElement("a");
    link.href = url;
    link.download = `sim-${rack}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };
  return (
    <Modal
      className="lab-details"
      open={open}
      onCancel={onClose}
      footer={<Button onClick={save}>Save</Button>}
      title={`${iterations} iterations of ${rack}`}
      width={640}
    >
      <Radio.Group
        className="lab-details-view"
        optionType="button"
        size="small"
        value={view}
        onChange={(e) => {
          setShown(null);
          setView(e.target.value);
        }}
        options={[
          { label: "By ply", value: "ply" },
          { label: "By move", value: "move" },
        ]}
      />
      {view === "move" ? (
        stats.map(({ move: m, stat }) => (
          <div
            key={m.jsonKey}
            className="lab-details-block"
            ref={(el) => {
              if (el) {
                blocks.current.set(m.jsonKey, el);
              } else {
                blocks.current.delete(m.jsonKey);
              }
            }}
          >
            <h4>{name(m)}</h4>
            <ul>
              {stat.n ? (
                <React.Fragment>
                  <li>{`Win: ${percent(stat.win.mean)} (SD ${percent(sd(stat.win))})`}</li>
                  <li>{`Equity: ${signed(stat.equity.mean, 2)} (SD ${sd(stat.equity).toFixed(1)})`}</li>
                </React.Fragment>
              ) : (
                <li>No rollouts</li>
              )}
              <li>{`Score: ${m.score}, static ${signed(m.equity, 2)}`}</li>
              <li>{`N: ${stat.n}`}</li>
              {(stat.plies ?? []).map((p, ply) => {
                if (!p.n) {
                  return null;
                }
                const score = `${p.score.mean.toFixed(1)} (SD ${sd(p.score).toFixed(1)})`;
                return (
                  <li key={ply}>
                    {`${plyName(ply)}: ${score}, bingo ${percent(p.bingo)}`}
                  </li>
                );
              })}
            </ul>
          </div>
        ))
      ) : (
        <React.Fragment>
          <table>
            <thead>
              <tr>
                <th />
                <th>Win</th>
                <th>SD</th>
                <th>Equity</th>
                <th>SD</th>
                <th>Score</th>
                <th>Static</th>
                <th>N</th>
              </tr>
            </thead>
            <tbody>
              {stats.map(({ move: m, stat }) => (
                <tr key={m.jsonKey} onClick={() => showBlock(m.jsonKey)}>
                  <td className="lab-details-move">{name(m)}</td>
                  <td>{stat.n ? percent(stat.win.mean) : ""}</td>
                  <td>{stat.n ? percent(sd(stat.win)) : ""}</td>
                  <td>{stat.n ? signed(stat.equity.mean, 2) : ""}</td>
                  <td>{stat.n ? sd(stat.equity).toFixed(1) : ""}</td>
                  <td>{m.score}</td>
                  <td>{signed(m.equity, 2)}</td>
                  <td>{stat.n}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {Array.from({ length: plies }, (_, ply) => (
            <React.Fragment key={ply}>
              <h4>{plyName(ply)}</h4>
              <table>
                <thead>
                  <tr>
                    <th />
                    <th>Score</th>
                    <th>SD</th>
                    <th>Bingo</th>
                    <th>N</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.map(({ move: m, stat }) => {
                    const p = stat.plies?.[ply];
                    return (
                      <tr key={m.jsonKey} onClick={() => showBlock(m.jsonKey)}>
                        <td className="lab-details-move">{name(m)}</td>
                        <td>{p?.n ? p.score.mean.toFixed(1) : ""}</td>
                        <td>{p?.n ? sd(p.score).toFixed(1) : ""}</td>
                        <td>{p?.n ? percent(p.bingo) : ""}</td>
                        <td>{p?.n}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </React.Fragment>
          ))}
        </React.Fragment>
      )}
    </Modal>
  );
};

// What a sim was made with: the plies its rollouts play, as its answer has
// them; then the other settings the user set.
const settingsText = (settings: SimSettings, plies: number | undefined) =>
  `${plies !== undefined ? `; ${plies} ${plies === 1 ? "ply" : "plies"}` : ""}${settings.known.trim() ? `; the opponent holds ${settings.known.trim().toUpperCase()}` : ""}${settings.unpruned ? "; unpruned" : ""}`;

type SimSettingsButtonProps = {
  settings: SimSettings;
  disabled: boolean;
  onChange: (settings: SimSettings) => void;
};

// The plies a sim's rollouts play, never left unset, and the tiles known to
// be on the opponent's rack, applied to the next sim made; and whether it
// prunes, applied to the sim made as well.
const SimSettingsButton = (props: SimSettingsButtonProps) => {
  const { settings, disabled, onChange } = props;
  const [open, setOpen] = useState(false);
  const [plies, setPlies] = useState<number | null>(settings.plies);
  const [known, setKnown] = useState(settings.known);
  const [unpruned, setUnpruned] = useState(settings.unpruned);
  const apply = () => {
    if (plies === null) return;
    setOpen(false);
    if (
      plies !== settings.plies ||
      known !== settings.known ||
      unpruned !== settings.unpruned
    ) {
      onChange({ plies, known, unpruned });
    }
  };
  return (
    <Popover
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) {
          setPlies(settings.plies);
          setKnown(settings.known);
          setUnpruned(settings.unpruned);
        }
      }}
      trigger="click"
      content={
        <div className="lab-settings">
          <label>
            Plies
            <InputNumber
              size="small"
              aria-label="Plies"
              min={1}
              precision={0}
              status={plies === null ? "error" : undefined}
              value={plies}
              onChange={(v) => setPlies(v)}
            />
          </label>
          <label>
            The opponent holds
            <Input
              size="small"
              value={known}
              placeholder="tiles, ? a blank"
              onChange={(e) => setKnown(e.target.value)}
              onPressEnter={apply}
            />
          </label>
          <Checkbox
            checked={unpruned}
            onChange={(e) => setUnpruned(e.target.checked)}
          >
            Unpruned
            <span className="lab-settings-note">
              Each candidate rolls out until the sim is stopped.
            </span>
          </Checkbox>
          <Button
            size="small"
            type="primary"
            disabled={plies === null}
            onClick={apply}
          >
            Apply
          </Button>
        </div>
      }
    >
      <Tooltip title="Sim settings">
        <Button
          shape="circle"
          size="small"
          icon={<SettingOutlined />}
          aria-label="Sim settings"
          disabled={disabled}
        />
      </Tooltip>
    </Popover>
  );
};

type ExchangeButtonProps = {
  rack: MachineWord;
  alphabet: Alphabet;
  disabled: boolean;
  onAdd: (play: WasmPlay) => void;
};

// Tiles picked from the rack, to add their exchange; or a pass, an exchange of
// no tiles.
const ExchangeButton = (props: ExchangeButtonProps) => {
  const { rack, alphabet, disabled, onAdd } = props;
  const [open, setOpen] = useState(false);
  // the rack's tiles picked, by index
  const [picked, setPicked] = useState(() => new Set<number>());
  const toggle = (i: number) =>
    setPicked((p) => {
      const next = new Set(p);
      if (!next.delete(i)) next.add(i);
      return next;
    });
  const add = (tiles: MachineWord) => {
    setOpen(false);
    setPicked(new Set());
    onAdd({ action: "exchange", tiles });
  };
  return (
    <Popover
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (!o) setPicked(new Set());
      }}
      trigger="click"
      content={
        <div className="lab-exchange">
          <div className="lab-exchange-tiles">
            {rack.map((t, i) => (
              <Button
                key={i}
                size="small"
                type={picked.has(i) ? "primary" : "default"}
                aria-pressed={picked.has(i)}
                onClick={() => toggle(i)}
              >
                {machineLetterToRune(t, alphabet, false, true)}
              </Button>
            ))}
          </div>
          <div className="lab-exchange-actions">
            <Button
              size="small"
              disabled={picked.size === 0}
              onClick={() =>
                add([...picked].sort((a, b) => a - b).map((i) => rack[i]))
              }
            >
              Exchange
            </Button>
            <Button size="small" onClick={() => add([])}>
              Pass
            </Button>
          </div>
        </div>
      }
    >
      <Tooltip title="Add an exchange or a pass">
        <Button
          shape="circle"
          size="small"
          icon={<SwapOutlined />}
          aria-label="Exchange or pass"
          disabled={disabled}
        />
      </Tooltip>
    </Popover>
  );
};

type TopMovesButtonProps = {
  disabled: boolean;
  onAdd: (count: number) => void;
};

// A count of the position's top moves to add, as many as asked, starting at
// the 15 the analyzer asks for.
const TopMovesButton = (props: TopMovesButtonProps) => {
  const { disabled, onAdd } = props;
  const [open, setOpen] = useState(false);
  const [count, setCount] = useState<number | null>(null);
  const add = () => {
    if (count === null || count < 1) return;
    setOpen(false);
    onAdd(count);
  };
  return (
    <Popover
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) setCount(15);
      }}
      trigger="click"
      content={
        <div className="lab-top">
          <label>
            Top moves
            <InputNumber
              size="small"
              aria-label="Top moves"
              min={1}
              precision={0}
              value={count}
              onChange={(v) => setCount(v)}
              onPressEnter={add}
            />
          </label>
          <Button
            size="small"
            type="primary"
            aria-label="Add the top moves"
            disabled={count === null || count < 1}
            onClick={add}
          >
            Add
          </Button>
        </div>
      }
    >
      <Tooltip title="Add the position's top moves">
        <Button
          shape="circle"
          size="small"
          icon={<OrderedListOutlined />}
          aria-label="Top moves"
          disabled={disabled}
        />
      </Tooltip>
    </Popover>
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
      <Tooltip title="Solve the endgame">
        <Button
          shape="circle"
          size="small"
          icon={<FlagOutlined />}
          type={view === "endgame" ? "primary" : "default"}
          onClick={() => setView(view === "endgame" ? null : "endgame")}
        />
      </Tooltip>
    </span>
  );
};
