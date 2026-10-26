import { create } from "@bufbuild/protobuf";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import {
  TournamentMetadataSchema,
  TType,
} from "../gen/api/proto/tournament_service/tournament_service_pb";
import {
  directedRows,
  UserDirectedTournaments,
} from "./user_directed_tournaments";

const query = vi.hoisted(() => ({
  input: undefined as unknown,
  data: undefined as unknown,
}));

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: (_method: unknown, input: unknown) => {
    query.input = input;
    return { data: query.data, error: null, isLoading: false };
  },
}));

const club = create(TournamentMetadataSchema, {
  id: "c1",
  name: "Philly Mondays",
  slug: "/club/philly",
  type: TType.CLUB,
});
const session = create(TournamentMetadataSchema, {
  id: "s1",
  name: "Philly Mondays 2026-10-19",
  slug: "/club/philly/2026-10-19",
  type: TType.CHILD,
});
const unnamed = create(TournamentMetadataSchema, {
  id: "t1",
  type: TType.STANDARD,
});
const irl = create(TournamentMetadataSchema, {
  id: "t2",
  name: "Fall Open",
  slug: "/tournament/fall-open",
  type: TType.STANDARD,
  irlMode: true,
});

describe("directedRows", () => {
  it("names each kind and links each page", () => {
    expect(
      directedRows([club, session, unnamed, irl]).map((r) => [
        r.name,
        r.kind,
        r.link,
      ]),
    ).toEqual([
      ["Philly Mondays", "Club", "/club/philly"],
      ["Philly Mondays 2026-10-19", "Session", "/club/philly/2026-10-19"],
      ["t1", "Tournament", "/tournament/t1"],
      ["Fall Open", "Tournament (in person)", "/tournament/fall-open"],
    ]);
  });
});

describe("UserDirectedTournaments", () => {
  it("asks for the user's tournaments and links them", () => {
    query.data = { tournaments: [club] };
    render(
      <MemoryRouter>
        <UserDirectedTournaments username="cesar" />
      </MemoryRouter>,
    );
    expect(query.input).toEqual({ username: "cesar" });
    const link = screen.getByRole("link", { name: "Philly Mondays" });
    expect(link).toHaveAttribute("href", "/club/philly");
    expect(within(link.closest("tr")!).getByText("Club")).toBeTruthy();
  });
});
