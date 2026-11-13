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
  lastId: 0,
  made: [] as Array<number>,
  // the inference a count made with one answers with
  inferred: undefined as object | undefined,
  // what a step of ms 0 lists, as the wasm would from the racks weighed
  listing: undefined as object | undefined,
}));
vi.mock("../wasm/lab", () => ({
  labCall: async (request: { op: string; args?: { request?: string } }) => {
    lab.calls.push(request);
    // ids never given twice, as the worker's, so a test's count is never
    // taken for the last test's, let go late
    if (request.op === "threats_new") {
      lab.made.push(++lab.lastId);
      return {
        id: lab.lastId,
        ...(lab.inferred ? { inferred: lab.inferred } : {}),
      };
    }
    // each step takes the next answer set, the last one again once none is
    // left
    if (request.op === "threats_step") {
      // the answer taken when the step is asked, then given a while after,
      // as the worker's is, letting the page render between steps
      const given =
        (request.args as { ms: number }).ms === 0 && lab.listing
          ? lab.listing
          : (lab.last = lab.answers.shift() ?? lab.last);
      await new Promise((resolve) => setTimeout(resolve, 5));
      return given;
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
  lab.listing = undefined;
  lab.made.length = 0;
  lab.inferred = undefined;
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

  it("lists the plays through a square picked on the board, as the wasm lists them", async () => {
    lab.answers.push(answer());
    mount();
    await screen.findByText("SEA");
    lab.listing = answer({ plays: [answer().plays[1]] });
    fireEvent.click(screen.getByLabelText("Plays through a square"));
    expect(shown?.picking).toBe(true);
    act(() => shown!.pick(8 * 15 + 7));
    // still picking, so another square moves the list to it
    expect(shown?.picking).toBe(true);
    await waitFor(() => expect(screen.queryByText("SEA")).toBeNull());
    expect(screen.getByText("AS")).toBeTruthy();
    // asked of the count held, weighing no more
    expect(lab.calls.at(-1)).toEqual({
      op: "threats_step",
      args: { id: expect.any(Number), ms: 0, count: 15, square: [8, 7] },
    });
    // the button widens the list back
    fireEvent.click(screen.getByLabelText("Plays through a square"));
    expect(screen.getByText("SEA")).toBeTruthy();
  });

  it("lists another count without counting again", async () => {
    lab.answers.push(answer());
    mount();
    await screen.findByText("SEA");
    lab.listing = answer({ plays: [answer().plays[0]] });
    fireEvent.change(screen.getByLabelText("Plays listed"), {
      target: { value: "1" },
    });
    await waitFor(() => expect(screen.queryByText("AS")).toBeNull());
    expect(threatsRequests()).toHaveLength(1);
    expect(lab.calls.at(-1)).toEqual({
      op: "threats_step",
      args: { id: expect.any(Number), ms: 0, count: 1 },
    });
  });

  it("lets each count go once its board is gone, and all on closing", async () => {
    lab.answers.push(answer());
    const { unmount } = mount();
    await screen.findByText("SEA");
    place(new Set([{ row: 0, col: 0, letter: B }]));
    await waitFor(() => expect(threatsRequests()).toHaveLength(2));
    unmount();
    const freed = () =>
      lab.calls
        .filter((c) => c.op === "threats_free")
        .map((c) => (c.args as unknown as { id: number }).id)
        .sort();
    await waitFor(() => expect(freed()).toEqual([...lab.made].sort()));
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
    const tiles = Array.from({ length: 27 }, () => ({ held: 0.2, prior: 0.2 }));
    tiles[17] = { held: 0.62, prior: 0.18 };
    lab.inferred = { count: 3, leaves: [], tiles };
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
    // what the inference says of the tiles, once the count has it
    await screen.findByText(
      "Against nothing inferred, likelier to hold Q 62.0% (18.0%).",
    );
    await waitFor(() => expect(threatsRequests()).toHaveLength(1));
    const args = lab.calls.find((c) => c.op === "threats_new")!.args as {
      request: string;
      infer: string;
    };
    expect(JSON.parse(args.infer).play.word).toEqual([B, E]);
    expect(JSON.parse(args.request).known).toBeUndefined();
  });

  it("steps the count on till it is exact, holding it after", async () => {
    lab.answers.push(
      answer({ exact: false, covered: 0.5, samples: 20 }),
      answer({ expected_best: 9 }),
    );
    mount();
    await screen.findByText("Opponent's best 9.0 on average, from 4 unseen");
    expect(
      lab.calls.filter(
        (c) => c.op === "threats_step" && (c.args as { ms: number }).ms > 0,
      ),
    ).toHaveLength(2);
    // this count is held, whatever the last test lets go late
    expect(
      lab.calls.some(
        (c) =>
          c.op === "threats_free" &&
          lab.made.includes((c.args as unknown as { id: number }).id),
      ),
    ).toBe(false);
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
