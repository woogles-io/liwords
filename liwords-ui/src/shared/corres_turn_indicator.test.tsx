import React from "react";
import { cleanup, fireEvent, render } from "@testing-library/react";
import {
  CorrespondencePerspective,
  CorrespondenceTurnIndicator,
} from "./corres_turn_indicator";

// Pin both clocks so the time computations and the SimpleTimer display are
// deterministic: Date.now() drives elapsed; performance.now() is fixed, so each
// SimpleTimer renders its anchored value exactly (no extrapolation drift).
const FIXED_NOW = 1_700_000_000_000;
const HOUR = 3600 * 1000;
const MIN = 60 * 1000;

beforeEach(() => {
  vi.spyOn(Date, "now").mockReturnValue(FIXED_NOW);
  vi.spyOn(performance, "now").mockReturnValue(1000);
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function renderIndicator(
  perspective: CorrespondencePerspective,
  opts: { elapsedMs: number; incrementMs: number; bankMs: number },
) {
  return render(
    <CorrespondenceTurnIndicator
      perspective={perspective}
      lastUpdateMs={FIXED_NOW - opts.elapsedMs}
      incrementMs={opts.incrementMs}
      bankMs={opts.bankMs}
    />,
  );
}

// Common anchor: 10h per-turn allowance, 96h bank, 2m elapsed (not bleeding).
// X = 10h + 96h - 2m = 105h58m = 4d 9h 58m; Y = 10h - 2m = 9h58m; Z = 96h.
const notBleeding = {
  elapsedMs: 2 * MIN,
  incrementMs: 10 * HOUR,
  bankMs: 96 * HOUR,
};

it("mine, not bleeding: 'Your turn' + the ticking d:hh:mm:ss deadline inline", () => {
  const { getByText } = renderIndicator({ kind: "mine" }, notBleeding);
  expect(getByText("Your turn")).toBeInTheDocument();
  expect(getByText("4:09:58:00")).toBeInTheDocument();
});

it("mine, not bleeding: tooltip breaks the deadline into free time (Y) and bank (Z)", async () => {
  const { getByText, findByText } = renderIndicator(
    { kind: "mine" },
    notBleeding,
  );
  fireEvent.mouseOver(getByText("Your turn"));
  // Y (free window) = 9h58m -> 09:58:00, Z (bank) = 96h -> 4:00:00:00.
  expect(
    await findByText(/Your free time: 09:58:00, time bank: 4:00:00:00/),
  ).toBeInTheDocument();
});

it("mine, bleeding: deadline counts down the bank; tooltip says it is draining", async () => {
  // 10h05m elapsed: past the per-turn allowance, so the bank is bleeding.
  // X = Z = 96h - 5m = 95h55m = 3d 23h 55m.
  const { getByText, findByText } = renderIndicator(
    { kind: "mine" },
    {
      elapsedMs: 10 * HOUR + 5 * MIN,
      incrementMs: 10 * HOUR,
      bankMs: 96 * HOUR,
    },
  );
  expect(getByText("3:23:55:00")).toBeInTheDocument();
  fireEvent.mouseOver(getByText("Your turn"));
  expect(await findByText(/draining/i)).toBeInTheDocument();
});

it("opponent: greyed 'Their turn' + possessive, bank-free tooltip", async () => {
  // No bank: X = 24h - 2m = 23h58m.
  const { getByText, findByText } = renderIndicator(
    { kind: "opponent", playerName: "Blibble" },
    { elapsedMs: 2 * MIN, incrementMs: 24 * HOUR, bankMs: 0 },
  );
  expect(getByText("Their turn")).toBeInTheDocument();
  expect(getByText("23:58:00")).toBeInTheDocument();
  fireEvent.mouseOver(getByText("Their turn"));
  expect(
    await findByText(/Blibble's clock times out in 23:58:00/),
  ).toBeInTheDocument();
});

it("spectator: labels by the on-turn player's name, no 'Your'/'Their'", () => {
  const { getByText, queryByText } = renderIndicator(
    { kind: "spectator", playerName: "ather" },
    notBleeding,
  );
  expect(getByText("ather")).toBeInTheDocument();
  expect(getByText("4:09:58:00")).toBeInTheDocument();
  expect(queryByText("Your turn")).not.toBeInTheDocument();
  expect(queryByText("Their turn")).not.toBeInTheDocument();
});

it("bare: time only inline (no turn label / name), tooltip still names the player", async () => {
  const { getByText, queryByText, findByText } = renderIndicator(
    { kind: "bare", playerName: "ather" },
    notBleeding,
  );
  // Inline is just the ticking deadline -- no turn label, no name.
  expect(getByText("4:09:58:00")).toBeInTheDocument();
  expect(queryByText("Your turn")).not.toBeInTheDocument();
  expect(queryByText("ather")).not.toBeInTheDocument();
  // The tooltip still names the on-turn player.
  fireEvent.mouseOver(getByText("4:09:58:00"));
  expect(
    await findByText(/ather's free time: 09:58:00, time bank: 4:00:00:00/),
  ).toBeInTheDocument();
});

it("no clock anchor: renders the label only, no time, no crash", () => {
  const { getByText, queryByText } = render(
    <CorrespondenceTurnIndicator perspective={{ kind: "mine" }} />,
  );
  expect(getByText("Your turn")).toBeInTheDocument();
  expect(queryByText(/\d+:\d\d/)).not.toBeInTheDocument();
});
