import React from "react";
import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import {
  DivisionSchema,
  LeaguePlayerStandingSchema,
} from "../gen/api/proto/ipc/league_pb";
import { DivisionStandings } from "./standings";

// The real ones need the login, match, and profile contexts.
vi.mock("../shared/usernameWithContext", () => ({
  UsernameWithContext: (props: { username: string }) => (
    <span>{props.username}</span>
  ),
}));

vi.mock("../shared/display_title", () => ({
  DisplayUserTitle: () => null,
}));

afterEach(cleanup);

// The first has played all its games and can no longer finish first; the
// second and the third still play, and either may; the fourth is settled.
const standing = (
  rank: number,
  username: string,
  bestRank: number,
  worstRank: number,
  wins: number,
  gamesRemaining: number,
) =>
  create(LeaguePlayerStandingSchema, {
    userId: username,
    username,
    rank,
    bestRank,
    worstRank,
    wins,
    gamesPlayed: 3 - gamesRemaining,
    gamesRemaining,
  });

const division = create(DivisionSchema, {
  divisionNumber: 1,
  standings: [
    standing(1, "alice", 2, 3, 2, 0),
    standing(2, "bob", 1, 3, 2, 1),
    standing(3, "carol", 1, 3, 1, 1),
    standing(4, "dave", 4, 4, 0, 1),
  ],
});

const rankCells = (container: HTMLElement) =>
  [...container.querySelectorAll("tr.ant-table-row")].map(
    (tr) => tr.querySelector("td")!,
  );

const marked = (cells: Array<HTMLElement>) =>
  cells.map((td) => td.classList.contains("can-reach"));

const show = () =>
  render(
    <DivisionStandings
      division={division}
      totalDivisions={1}
      seasonId="s"
      seasonNumber={1}
    />,
  );

it("marks the rows that can still finish at the rank whose range shows", async () => {
  const { container } = show();
  const cells = rankCells(container);
  expect(marked(cells)).toEqual([false, false, false, false]);
  // the first's range, 2-3: it can no longer finish first; the second and
  // the third can
  fireEvent.mouseEnter(cells[0].firstElementChild!);
  await waitFor(() =>
    expect(marked(cells)).toEqual([false, true, true, false]),
  );
  fireEvent.mouseLeave(cells[0].firstElementChild!);
  await waitFor(() =>
    expect(marked(cells)).toEqual([false, false, false, false]),
  );
  // the third's: all three can finish third, the third itself among them
  fireEvent.mouseEnter(cells[2].firstElementChild!);
  await waitFor(() => expect(marked(cells)).toEqual([true, true, true, false]));
});

it("marks nothing for a rank already known, which shows no range", async () => {
  const { container } = show();
  const cells = rankCells(container);
  fireEvent.mouseEnter(cells[3].firstElementChild!);
  await new Promise((r) => setTimeout(r, 300));
  expect(marked(cells)).toEqual([false, false, false, false]);
});
