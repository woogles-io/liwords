import React from "react";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { PlayerSeasonGameSchema } from "../gen/api/proto/league_service/league_service_pb";
import { LeagueCorrespondenceGames } from "./league_correspondence_games";

// The viewer's games this season, as getPlayerSeasonGames returns them.
const season = vi.hoisted(() => ({ games: [] as Array<unknown> }));

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: (_method: unknown, input: { seasonId?: string }) => ({
    data: input.seasonId === "season" ? { games: season.games } : undefined,
  }),
}));

vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));

vi.mock("../store/store", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../store/store")>()),
  useLoginStateStoreContext: () => ({ loginState: { userID: "me" } }),
  useLobbyStoreContext: () => ({ lobbyContext: { correspondenceGames: [] } }),
}));

afterEach(cleanup);

const ended = (opponent: string, mine?: number, theirs?: number) =>
  create(PlayerSeasonGameSchema, {
    gameId: `${opponent}-game`,
    opponentUserId: `${opponent}-id`,
    opponentUsername: opponent,
    playerScore: 450,
    opponentScore: 380,
    result: "win",
    gameEndReason: GameEndReason.STANDARD,
    gameDate: timestampFromDate(new Date(Date.now() - 2 * 60 * 60 * 1000)),
    mistakeIndex: mine,
    opponentMistakeIndex: theirs,
  });

// A row's two lines: the opponent and the score, then how it ended.
const rowLines = (opponent: string) => {
  const row = screen.getByText(`vs ${opponent}`).closest(".league-game-item")!;
  return [
    row.querySelector(".opponent-name-compact")!.textContent,
    row.querySelector(".turn-indicator-compact")!.textContent,
  ];
};

describe("the Just ended list", () => {
  it("gives each game's mistakes on its second line, under the score", () => {
    season.games = [ended("analyzed", 3.2, 5.1), ended("waiting")];
    render(<LeagueCorrespondenceGames leagueSlug="league" seasonId="season" />);
    expect(rowLines("analyzed")).toEqual([
      "vs analyzed450-380",
      "Won 2h agoMI 3.2-5.1",
    ]);
    expect(rowLines("waiting")).toEqual([
      "vs waiting450-380",
      "Won 2h agoMI -",
    ]);
  });
});
