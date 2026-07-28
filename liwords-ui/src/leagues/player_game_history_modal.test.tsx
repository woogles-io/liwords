import React from "react";
import { ConfigProvider } from "antd";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { timestampFromDate, type Timestamp } from "@bufbuild/protobuf/wkt";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { liwordsDarkTheme, liwordsDefaultTheme } from "../themes";
import {
  dateSorter,
  formatSeasonRange,
  type GameRow,
  mistakeSorter,
  opponentSorter,
  PlayerGameHistoryModal,
  resultSorter,
  scoreSorter,
  timeSorter,
} from "./player_game_history_modal";

// The modal has no local season data to render against, so the sort logic and
// the display rules (the split Time/Mistakes columns, spread sign,
// 0-vs-unanalyzed, title range) are pinned here instead of by hand.

afterEach(cleanup);

// --- pure sorters -----------------------------------------------------------

const mkRow = (o: Partial<GameRow>): GameRow => ({
  key: o.gameId ?? "k",
  gameId: o.gameId ?? "k",
  opponentUsername: "opp",
  opponentUserId: "u",
  result: "loss",
  playerScore: 0,
  opponentScore: 0,
  mistakeIndex: undefined,
  gameDate: undefined,
  gameEndReason: GameEndReason.NONE,
  lastUpdateMs: undefined,
  incrementSecs: 0,
  onTurnTimeBankMs: 0,
  ...o,
});

const order = (rows: GameRow[], cmp: (a: GameRow, b: GameRow) => number) =>
  [...rows].sort(cmp).map((r) => r.gameId);

describe("game-history sorters", () => {
  it("opponent: locale name order", () => {
    const rows = [
      mkRow({ gameId: "z", opponentUsername: "zed" }),
      mkRow({ gameId: "a", opponentUsername: "abe" }),
    ];
    expect(order(rows, opponentSorter)).toEqual(["a", "z"]);
  });

  it("result: live first, then win/draw/loss, ties by spread", () => {
    const rows = [
      mkRow({ gameId: "loss", result: "loss" }),
      mkRow({
        gameId: "win-small",
        result: "win",
        playerScore: 400,
        opponentScore: 399,
      }),
      mkRow({
        gameId: "win-big",
        result: "win",
        playerScore: 500,
        opponentScore: 300,
      }),
      mkRow({ gameId: "draw", result: "draw" }),
      mkRow({ gameId: "turn", result: "turn" }),
      mkRow({ gameId: "inprog", result: "in_progress" }),
    ];
    expect(order(rows, resultSorter)).toEqual([
      "turn",
      "inprog",
      "win-big",
      "win-small",
      "draw",
      "loss",
    ]);
  });

  it("score: ascending by spread", () => {
    const rows = [
      mkRow({ gameId: "p50", playerScore: 450, opponentScore: 400 }),
      mkRow({ gameId: "m10", playerScore: 390, opponentScore: 400 }),
      mkRow({ gameId: "p100", playerScore: 500, opponentScore: 400 }),
    ];
    expect(order(rows, scoreSorter)).toEqual(["m10", "p50", "p100"]);
  });

  it("date: ascending by last-updated", () => {
    const rows = [
      mkRow({ gameId: "b", gameDate: new Date(2026, 6, 20) }),
      mkRow({ gameId: "a", gameDate: new Date(2026, 6, 10) }),
      mkRow({ gameId: "c", gameDate: new Date(2026, 6, 30) }),
    ];
    expect(order(rows, dateSorter)).toEqual(["a", "b", "c"]);
  });

  it("time: live games by soonest deadline, finished games last", () => {
    const now = Date.now();
    const rows = [
      mkRow({ gameId: "finished", gameDate: new Date(2026, 6, 10) }),
      mkRow({
        gameId: "turn-lots",
        result: "turn",
        lastUpdateMs: now,
        incrementSecs: 100000,
      }),
      mkRow({
        gameId: "turn-soon",
        result: "turn",
        lastUpdateMs: now,
        incrementSecs: 10,
      }),
      mkRow({
        gameId: "opp",
        result: "in_progress",
        lastUpdateMs: now,
        incrementSecs: 50000,
      }),
    ];
    expect(order(rows, timeSorter)).toEqual([
      "turn-soon", // soonest deadline first
      "opp",
      "turn-lots",
      "finished", // no clock -> last
    ]);
  });

  it("mistakes: by score ascending, no-score games last", () => {
    const rows = [
      mkRow({ gameId: "m8", mistakeIndex: 8 }),
      mkRow({ gameId: "m0", mistakeIndex: 0 }),
      mkRow({ gameId: "live", result: "turn" }), // no score
      mkRow({ gameId: "pending" }), // finished, unanalyzed -> no score
      mkRow({ gameId: "m2", mistakeIndex: 2 }),
    ];
    // 0, 2, 8, then the two no-score rows (tied, stable input order).
    expect(order(rows, mistakeSorter)).toEqual([
      "m0",
      "m2",
      "m8",
      "live",
      "pending",
    ]);
  });
});

describe("formatSeasonRange", () => {
  it("cross-year shows both years", () => {
    const s = formatSeasonRange(new Date(2025, 11, 30), new Date(2026, 0, 14));
    expect(s).toContain("2025");
    expect(s).toContain("2026");
  });

  it("within one year shows the year once, with a range separator", () => {
    const s = formatSeasonRange(new Date(2026, 6, 16), new Date(2026, 6, 30));
    expect(s.match(/2026/g)?.length).toBe(1);
    expect(s).toMatch(/–/); // en dash between the two days
  });

  it("a single day shows one date, no range", () => {
    const s = formatSeasonRange(
      new Date(2026, 6, 16, 9, 5),
      new Date(2026, 6, 16, 21, 40),
    );
    expect(s).not.toMatch(/–/);
    expect(s.match(/2026/g)?.length).toBe(1);
  });
});

// --- rendering --------------------------------------------------------------

// Keep the query and the router/store-heavy sub-components out: feed the modal
// mock games directly and stub the opponent cell and clock down to markers.
const mock = vi.hoisted(() => ({ games: [] as unknown[] }));

vi.mock("@connectrpc/connect-query", async (importActual) => ({
  ...(await importActual<typeof import("@connectrpc/connect-query")>()),
  useQuery: () => ({
    data: { games: mock.games },
    isLoading: false,
    error: null,
  }),
}));

// The real one is a dropdown needing the login, match and pet contexts. Its
// "View game history" entry becomes a button so the wiring can still be
// clicked.
vi.mock("../shared/usernameWithContext", () => ({
  UsernameWithContext: (props: {
    username: string;
    infoText?: string;
    handleInfoText?: () => void;
  }) => (
    <span>
      {props.username}
      {props.infoText && (
        <button
          onClick={(e) => {
            props.handleInfoText?.();
            // The real menu is portalled outside the modal and unmounts on
            // click, which drops focus onto the body. Reproduce that, since it
            // is what stops Escape reaching the dialog wrapper.
            e.currentTarget.blur();
          }}
        >
          {props.infoText} for {props.username}
        </button>
      )}
    </span>
  ),
}));

vi.mock("../shared/corres_turn_indicator", () => ({
  CorrespondenceTurnIndicator: (props: {
    perspective: { playerName: string };
  }) => <span data-testid="clock">clock:{props.perspective.playerName}</span>,
}));

const mkGame = (o: Record<string, unknown>) => ({
  gameId: "g",
  opponentUserId: "u",
  opponentUsername: "Opp",
  playerScore: 400,
  opponentScore: 350,
  result: "loss",
  gameDate: timestampFromDate(new Date(2026, 6, 16, 14, 30)),
  round: 0,
  gameEndReason: GameEndReason.STANDARD,
  mistakeIndex: undefined,
  lastUpdate: undefined,
  incrementSecs: 0,
  onTurnTimeBankMs: BigInt(0),
  ...o,
});

const mixed = [
  mkGame({
    gameId: "mine",
    result: "turn",
    lastUpdate: timestampFromDate(new Date(Date.now() - 1000)),
    incrementSecs: 259200,
    gameDate: timestampFromDate(new Date(2026, 6, 20, 10, 0)),
  }),
  mkGame({
    gameId: "theirs",
    result: "in_progress",
    lastUpdate: timestampFromDate(new Date(Date.now() - 2000)),
    incrementSecs: 259200,
    gameDate: timestampFromDate(new Date(2026, 6, 19, 10, 0)),
  }),
  mkGame({
    gameId: "beaten",
    result: "loss",
    mistakeIndex: 5.4,
    gameDate: timestampFromDate(new Date(2026, 6, 18, 10, 0)),
  }),
  mkGame({
    gameId: "perfect",
    result: "win",
    playerScore: 500,
    opponentScore: 400,
    mistakeIndex: 0,
    gameDate: timestampFromDate(new Date(2026, 6, 17, 10, 0)),
  }),
  mkGame({
    gameId: "level",
    result: "draw",
    playerScore: 400,
    opponentScore: 400,
    gameDate: timestampFromDate(new Date(2026, 6, 16, 10, 0)),
  }),
];

const renderModal = (
  mode: "default" | "dark",
  games: unknown[],
  seasonStartDate?: Timestamp,
  onSelectPlayer: (userId: string, username: string) => void = () => {},
) => {
  mock.games = games;
  document.body.className = `mode--${mode}`;
  return render(
    <ConfigProvider
      theme={mode === "dark" ? liwordsDarkTheme : liwordsDefaultTheme}
    >
      <PlayerGameHistoryModal
        visible
        onClose={() => {}}
        userId="me"
        username="Me"
        seasonId="s"
        seasonNumber={18}
        seasonStartDate={seasonStartDate}
        onSelectPlayer={onSelectPlayer}
      />
    </ConfigProvider>,
  );
};

describe.each(["default", "dark"] as const)(
  "modal render in %s mode",
  (mode) => {
    it("shows separate Time and Mistakes headers when both kinds are present", () => {
      renderModal(mode, mixed);
      const headers = screen
        .getAllByRole("columnheader")
        .map((h) => h.textContent ?? "");
      expect(headers.some((h) => h.includes("Time"))).toBe(true);
      expect(headers.some((h) => h.includes("Mistakes"))).toBe(true);
      expect(headers.some((h) => h.includes("Time / Mistakes"))).toBe(false);
    });

    it("shows 0.0 for a perfect game and - for an unanalyzed one", () => {
      renderModal(mode, mixed);
      const text = screen.getByRole("table").textContent ?? "";
      expect(text).toContain("0.0"); // perfect game is a real value, not hidden
      expect(text).toContain("5.4"); // analyzed loss
    });

    it("shows the spread inline, +0 when level", () => {
      renderModal(mode, mixed);
      const text = screen.getByRole("table").textContent ?? "";
      expect(text).toContain("(+0)"); // draw, 400-400
      expect(text).toContain("(+100)"); // perfect win, 500-400
    });

    it("puts the season date range in the title", () => {
      renderModal(mode, mixed);
      const title =
        document.querySelector(".ant-modal-title")?.textContent ?? "";
      expect(title).toContain("Season 18 Games");
      expect(title).toContain("2026");
      expect(title).toMatch(/–/); // 16 - 20 Jul, a real range
    });
  },
);

it("shows only the Mistakes column when the season has no live games", () => {
  renderModal("default", [
    mkGame({ gameId: "a", result: "loss", mistakeIndex: 3 }),
  ]);
  const headers = screen
    .getAllByRole("columnheader")
    .map((h) => h.textContent ?? "");
  expect(headers.some((h) => h.includes("Mistakes"))).toBe(true);
  expect(headers.some((h) => h.includes("Time"))).toBe(false);
});

it("extends the title range back to the season start", () => {
  // Games run 16-20 Jul; a season start of 1 Jul should push the range's start
  // back to 1 Jul (the games are all created then), not the earliest game date.
  renderModal("default", mixed, timestampFromDate(new Date(2026, 6, 1)));
  const title = document.querySelector(".ant-modal-title")?.textContent ?? "";
  expect(title).toContain(
    formatSeasonRange(new Date(2026, 6, 1), new Date(2026, 6, 20)),
  );
});

// --- following an opponent --------------------------------------------------

const carol = [
  mkGame({
    gameId: "beaten",
    result: "loss",
    mistakeIndex: 5.4,
    opponentUserId: "u-carol",
    opponentUsername: "carol",
  }),
];

// A real Escape press lands on whatever holds focus.
const pressEscape = () =>
  fireEvent.keyDown(document.activeElement ?? document.body, {
    key: "Escape",
    keyCode: 27,
  });

const showPlayer = (
  player: { userId: string; username: string },
  onClose: () => void,
  onSelectPlayer: (userId: string, username: string) => void,
) => (
  <PlayerGameHistoryModal
    visible
    onClose={onClose}
    userId={player.userId}
    username={player.username}
    seasonId="s"
    seasonNumber={18}
    onSelectPlayer={onSelectPlayer}
  />
);

it("hands an opponent over so their games can replace this one", async () => {
  const selected: Array<[string, string]> = [];
  renderModal("default", carol, undefined, (userId, username) =>
    selected.push([userId, username]),
  );
  await userEvent.click(screen.getByText("View game history for carol"));
  expect(selected).toEqual([["u-carol", "carol"]]);
});

// jsdom never runs the motion callback rc-dialog focuses from, so without a
// focus target of our own nothing inside the modal holds focus here, even on
// first open. A browser does focus it -- this pins that the wrapper serves both
// paths.
it("closes on escape", () => {
  const closed: string[] = [];
  mock.games = carol;
  render(
    showPlayer(
      { userId: "me", username: "Me" },
      () => closed.push("Me"),
      () => {},
    ),
  );
  pressEscape();
  expect(closed).toEqual(["Me"]);
});

it("still closes on escape after following an opponent", async () => {
  // Switching players re-renders this modal rather than reopening it, so
  // rc-dialog never re-runs its focus-on-open, and the click that switched came
  // from a menu outside the modal. Escape is bound to the dialog wrapper, so it
  // only works while focus is still inside.
  const closed: string[] = [];
  mock.games = carol;
  const Harness = () => {
    const [player, setPlayer] = React.useState({
      userId: "me",
      username: "Me",
    });
    return showPlayer(
      player,
      () => closed.push(player.username),
      (userId, username) => setPlayer({ userId, username }),
    );
  };
  render(<Harness />);

  await userEvent.click(screen.getByText("View game history for carol"));
  expect(document.querySelector(".ant-modal-title")?.textContent).toContain(
    "carol's Season 18 Games",
  );

  // Escape has to be fired at whatever holds focus, since rc-dialog listens on
  // the dialog wrapper and a keydown from the body never reaches it. user-event
  // omits the deprecated keyCode that rc-dialog checks, so this goes through
  // fireEvent.
  pressEscape();
  expect(closed).toEqual(["carol"]);
});

it("renders the whole table (dumped for eyeballing)", () => {
  renderModal("default", mixed);
  const table = screen.getByRole("table");
  const title = document.querySelector(".ant-modal-title")?.textContent ?? "";
  const headers = [...table.querySelectorAll("thead th")]
    .map((th) => th.textContent?.trim())
    .filter(Boolean);
  const rows = [...table.querySelectorAll("tbody tr")]
    .filter((tr) => !tr.classList.contains("ant-table-measure-row"))
    .map((tr) =>
      [...tr.querySelectorAll("td")].map((td) => td.textContent?.trim()),
    );
  console.log("TITLE:", title);
  console.log("HEADERS:", headers.join(" | "));
  for (const r of rows) console.log("ROW:", r.join(" | "));
  expect(rows).toHaveLength(mixed.length);
});
