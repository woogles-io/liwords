import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { useEffect } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { Store, useTentativeTileContext } from "../store/store";
import { EphemeralTile } from "../utils/cwgame/common";
import { LabPosition, SimSettings } from "./analyzer_lab_panels";
import { ThreatsAnswer, useLabThreats } from "./lab_threats";
import { ThreatsPanel } from "./lab_threats_panel";

// The lab's worker, faked: each threats call answers what the test sets.
const lab = vi.hoisted(() => ({
  calls: [] as Array<{ op: string; args?: { request?: string } }>,
  answers: [] as Array<object>,
  last: undefined as object | undefined,
}));
vi.mock("../wasm/lab", () => ({
  labCall: async (request: { op: string; args?: { request?: string } }) => {
    lab.calls.push(request);
    if (request.op === "threats_new") return { id: lab.calls.length };
    // each step takes the next answer set, the last one again once none is
    // left
    if (request.op === "threats_step") {
      lab.last = lab.answers.shift() ?? lab.last;
      return lab.last;
    }
    return undefined;
  },
  labParts: () => 1,
  setLabParts: () => {},
  onLabEvent: () => () => {},
}));

const A = 1;
const B = 2;
const E = 5;
const S = 19;

const position = (more: Partial<LabPosition> = {}): LabPosition => ({
  loadableKey: "CSW24",
  dim: 15,
  letters: new Array(225).fill(0),
  alphabet: StandardEnglishAlphabet,
  rack: [A, B, E],
  unseen: [A, E, S, S],
  position: {
    rack: [A, B, E],
    board: [],
    lexicon: "CSW24",
    rules: "CrosswordGame",
    scores: [300, 290],
    turn: 0,
    zero_turns: 0,
    pass_turns: 0,
  },
  leave: "CSW24",
  challengeable: false,
  challengeBonus: 5,
  exchanges: [0, 0],
  lastMove: null,
  names: ["andy", "wolges"],
  key: "k",
  ...more,
});

const settings = (more: Partial<SimSettings> = {}): SimSettings => ({
  plies: 4,
  known: "",
  unpruned: false,
  opponentsPass: false,
  infer: false,
  fuzz: 3.105,
  list: 0,
  alpha: 0.05,
  workers: 1,
  ...more,
});

// SEA at 8H across, scoring 12, and AS at H8 down, scoring 9
const answer = (more: Partial<ThreatsAnswer> = {}): ThreatsAnswer => {
  const squares: ThreatsAnswer["squares"] = Array.from({ length: 15 }, () =>
    Array.from({ length: 15 }, () => ({ best: 0, chance: 0, expected: 0 })),
  );
  squares[7][7] = { best: 12, chance: 0.5, expected: 6 };
  squares[7][8] = { best: 12, chance: 0.5, expected: 6 };
  squares[7][9] = { best: 12, chance: 0.5, expected: 6 };
  squares[8][7] = { best: 9, chance: 0.75, expected: 5 };
  return {
    racks: 4,
    count: 2,
    expected_best: 10.5,
    bingo: null,
    covered: 1,
    samples: 0,
    list_ms: 1,
    exact: true,
    plays: [
      {
        action: "play",
        down: false,
        lane: 7,
        idx: 7,
        word: [S, E, A],
        score: 12,
        tiles: [A, E, S],
        chance: 0.5,
      },
      {
        action: "play",
        down: true,
        lane: 7,
        idx: 7,
        word: [A, S],
        score: 9,
        tiles: [A, S],
        chance: 0.75,
      },
    ],
    squares,
    ...more,
  };
};

// What the board would draw: the overlay's store, read as the board reads it.
let shown: ReturnType<typeof useLabThreats> = null;
const Board = () => {
  const threats = useLabThreats();
  useEffect(() => {
    shown = threats;
  }, [threats]);
  return null;
};

// Tiles placed on the board, as a click on an analyzer's row places them.
let place: (tiles: Set<EphemeralTile>) => void = () => {};
const Placer = () => {
  const { setPlacedTiles } = useTentativeTileContext();
  useEffect(() => {
    place = (tiles) => act(() => setPlacedTiles(tiles));
  }, [setPlacedTiles]);
  return null;
};

const threatsRequests = () =>
  lab.calls
    .filter((c) => c.op === "threats_new")
    .map((c) => JSON.parse(c.args!.request!));

const mount = (more: Partial<SimSettings> = {}, at = position()) =>
  render(
    <Store>
      <Placer />
      <Board />
      <ThreatsPanel lab={at} settings={settings(more)} onClose={() => {}} />
    </Store>,
  );

beforeEach(() => {
  lab.calls.length = 0;
  lab.answers.length = 0;
  lab.last = undefined;
  shown = null;
});

afterEach(cleanup);

describe("ThreatsPanel", () => {
  it("counts the position's threats and lists the opponent's plays", async () => {
    lab.answers.push(answer());
    mount();
    await screen.findByText("Opponent's best 10.5 on average, from 4 unseen");
    expect(threatsRequests()).toEqual([
      {
        board: expect.any(Array),
        lexicon: "CSW24",
        rules: "CrosswordGame",
        rack: [A, B, E],
        known: [],
      },
    ]);
    expect(screen.getByText("SEA")).toBeTruthy();
    expect(screen.getByText("50.0%")).toBeTruthy();
    // the board tinted by the best score, the hottest square fully
    expect(shown?.heat[7 * 15 + 7]).toEqual({ level: 1, label: "12" });
    expect(shown?.heat[8 * 15 + 7]).toEqual({ level: 0.75, label: "9" });
    expect(shown?.heat[0]).toBeUndefined();
  });

  it("tints by chance when asked", async () => {
    lab.answers.push(answer());
    mount();
    await screen.findByText("SEA");
    fireEvent.click(screen.getByRole("radio", { name: "Chance" }));
    expect(shown?.heat[8 * 15 + 7]).toEqual({ level: 0.75, label: "75" });
  });

  it("outlines a play pointed at, and shows one picked as the opponent's tiles", async () => {
    lab.answers.push(answer());
    mount();
    const row = (await screen.findByText("SEA")).closest("tr")!;
    fireEvent.mouseEnter(row);
    expect([...shown!.outline]).toEqual([112, 113, 114]);
    fireEvent.click(row);
    expect([...shown!.ghost]).toEqual([
      [112, "S"],
      [113, "E"],
      [114, "A"],
    ]);
    // a click on it again takes it off
    fireEvent.click(row);
    expect(shown!.ghost.size).toBe(0);
  });

  it("lists only the plays through a square picked on the board", async () => {
    lab.answers.push(answer());
    mount();
    await screen.findByText("SEA");
    fireEvent.click(screen.getByLabelText("Plays through a square"));
    expect(shown?.picking).toBe(true);
    act(() => shown!.pick(8 * 15 + 7));
    expect(screen.queryByText("SEA")).toBeNull();
    expect(screen.getByText("AS")).toBeTruthy();
    expect(shown?.picking).toBe(false);
    // the button widens the list back
    fireEvent.click(screen.getByLabelText("Plays through a square"));
    expect(screen.getByText("SEA")).toBeTruthy();
  });

  it("lists a square's plays however far down they rank, cut to the count after", async () => {
    lab.answers.push(answer());
    mount();
    await screen.findByText("SEA");
    fireEvent.change(screen.getByLabelText("Plays listed"), {
      target: { value: "1" },
    });
    await waitFor(() => expect(screen.queryByText("AS")).toBeNull());
    // the count lists without counting again
    expect(threatsRequests()).toHaveLength(1);
    fireEvent.click(screen.getByLabelText("Plays through a square"));
    act(() => shown!.pick(8 * 15 + 7));
    expect(screen.getByText("AS")).toBeTruthy();
  });

  it("counts the board after the tiles placed too, beside now", async () => {
    lab.answers.push(answer());
    mount();
    await screen.findByText("SEA");
    lab.answers.push(answer({ expected_best: 4 }));
    place(new Set([{ row: 0, col: 0, letter: B }]));
    await screen.findByText(
      "After the tiles placed: Opponent's best 4.0 on average, from 4 unseen",
    );
    screen.getByText("Now: Opponent's best 10.5 on average, from 4 unseen");
    const after = threatsRequests()[1];
    // the tile placed off the rack; the tiles drawn after it stay unseen
    expect(after.rack).toEqual([A, E]);
    expect(after.board[0][0]).toBe(B);
  });

  it("sends the tiles known on the opponent's rack", async () => {
    lab.answers.push(answer());
    mount({ known: "s" });
    await screen.findByText("Opponent holds S.");
    await waitFor(() => expect(threatsRequests()[0]?.known).toEqual([S]));
  });

  it("says when there is nothing to infer from, and counts without", async () => {
    lab.answers.push(answer());
    mount({ infer: true });
    await screen.findByText(/^Not inferred:/);
    await waitFor(() => expect(threatsRequests()).toHaveLength(1));
    const made = lab.calls.find((c) => c.op === "threats_new")!;
    expect((made.args as { infer?: string }).infer).toBeUndefined();
  });

  it("infers the opponent's leave from their last move when asked", async () => {
    lab.answers.push(answer());
    mount(
      { infer: true },
      position({
        lastMove: {
          board: [[0]],
          play: {
            action: "play",
            down: false,
            lane: 7,
            idx: 7,
            word: [B, E],
            score: 8,
          },
        },
      }),
    );
    await screen.findByText("Opponent's leave inferred from their last move.");
    await waitFor(() => expect(threatsRequests()).toHaveLength(1));
    const args = lab.calls.find((c) => c.op === "threats_new")!.args as {
      request: string;
      infer: string;
    };
    expect(JSON.parse(args.infer).play.word).toEqual([B, E]);
    expect(JSON.parse(args.request).known).toBeUndefined();
  });

  it("marks the chances of an answer cut short", async () => {
    lab.answers.push(answer({ exact: false, covered: 0.5, samples: 20 }));
    mount();
    // once the seconds are spent
    await screen.findByText(/time ran out: 50.0% of racks counted/, undefined, {
      timeout: 3000,
    });
    expect(screen.getByText("~50.0%")).toBeTruthy();
  });

  it("counts again on asking, and clears the board on closing", async () => {
    lab.answers.push(answer(), answer({ expected_best: 7 }));
    const { rerender } = mount();
    await screen.findByText("SEA");
    fireEvent.click(screen.getByLabelText("Count again"));
    await screen.findByText("Opponent's best 7.0 on average, from 4 unseen");
    expect(threatsRequests()).toHaveLength(2);
    // the panel closed, the board stays
    rerender(
      <Store>
        <Placer />
        <Board />
      </Store>,
    );
    expect(shown).toBeNull();
  });
});
