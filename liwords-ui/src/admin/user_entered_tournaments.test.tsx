import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import {
  EnteredTournamentSchema,
  TType,
} from "../gen/api/proto/tournament_service/tournament_service_pb";
import {
  enteredRows,
  UserEnteredTournaments,
} from "./user_entered_tournaments";

const query = vi.hoisted(() => ({
  input: undefined as unknown,
  options: undefined as unknown,
  data: undefined as unknown,
  refetch: vi.fn(),
}));

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: (_method: unknown, input: unknown, options: unknown) => {
    query.input = input;
    query.options = options;
    return {
      data: query.data,
      error: null,
      isFetching: false,
      refetch: query.refetch,
    };
  },
}));

const played = create(EnteredTournamentSchema, {
  id: "s1",
  name: "Philly Mondays 2026-10-19",
  slug: "/club/philly/2026-10-19",
  type: TType.CHILD,
  games: 4,
  lastPlayed: timestampFromMs(Date.UTC(2026, 9, 19, 23, 0, 0)),
  registered: true,
});
const registeredOnly = create(EnteredTournamentSchema, {
  id: "t2",
  name: "Fall Open",
  slug: "/tournament/fall-open",
  type: TType.STANDARD,
  irlMode: true,
  registered: true,
});

describe("enteredRows", () => {
  it("says how many games, when the last was, and whether registered", () => {
    expect(
      enteredRows([played, registeredOnly]).map((r) => [
        r.name,
        r.kind,
        r.games,
        r.registered,
      ]),
    ).toEqual([
      ["Philly Mondays 2026-10-19", "Session", 4, "yes"],
      ["Fall Open", "Tournament (in person)", 0, "yes"],
    ]);
    expect(enteredRows([played])[0].lastPlayed).toMatch(/^2026-10-(19|20)$/);
  });
});

describe("UserEnteredTournaments", () => {
  it("lists them only when asked, then links each", () => {
    query.data = undefined;
    const { rerender } = render(
      <MemoryRouter>
        <UserEnteredTournaments username="cesar" />
      </MemoryRouter>,
    );
    expect(query.input).toEqual({ username: "cesar" });
    expect(query.options).toMatchObject({ enabled: false });
    expect(screen.queryByRole("table")).toBeNull();
    fireEvent.click(
      screen.getByRole("button", { name: "List their tournaments" }),
    );
    expect(query.refetch).toHaveBeenCalledTimes(1);

    query.data = { tournaments: [registeredOnly] };
    rerender(
      <MemoryRouter>
        <UserEnteredTournaments username="cesar" />
      </MemoryRouter>,
    );
    expect(screen.getByRole("link", { name: "Fall Open" })).toHaveAttribute(
      "href",
      "/tournament/fall-open",
    );
  });
});
