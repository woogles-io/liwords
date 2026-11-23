import React from "react";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { PlayerSeasonGameSchema } from "../gen/api/proto/league_service/league_service_pb";
import { LeagueCorrespondenceGames } from "./league_correspondence_games";

// The viewer's games this season, as getPlayerSeasonGames returns them.
const season = vi.hoisted(() => ({ games: [] as Array<unknown> }));
// the season games asked for again, and the lobby's live games
const refetch = vi.hoisted(() => vi.fn());
const lobby = vi.hoisted(() => ({ games: [] as Array<unknown> }));

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: (_method: unknown, input: { seasonId?: string }) => ({
    data: input.seasonId === "season" ? { games: season.games } : undefined,
    refetch,
  }),
}));

vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));

vi.mock("../store/store", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../store/store")>()),
  useLoginStateStoreContext: () => ({ loginState: { userID: "me" } }),
  useLobbyStoreContext: () => ({
    lobbyContext: { correspondenceGames: lobby.games },
  }),
}));

afterEach(() => {
  cleanup();
  refetch.mockReset();
  lobby.games = [];
});

const ended = (
  opponent: string,
  mine?: number,
  theirs?: number,
  hoursAgo = 2,
) =>
  create(PlayerSeasonGameSchema, {
    gameId: `${opponent}-game`,
    opponentUserId: `${opponent}-id`,
    opponentUsername: opponent,
    playerScore: 450,
    opponentScore: 380,
    result: "win",
    gameEndReason: GameEndReason.STANDARD,
    gameDate: timestampFromDate(
      new Date(Date.now() - hoursAgo * 60 * 60 * 1000),
    ),
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

  it("lets a game go once a day has passed, as the page stays open", () => {
    vi.useFakeTimers({ toFake: ["Date"] });
    try {
      season.games = [ended("late", undefined, undefined, 23)];
      const { rerender } = render(
        <LeagueCorrespondenceGames leagueSlug="league" seasonId="season" />,
      );
      expect(rowLines("late")[1]).toBe("Won 23h agoMI -");
      // two hours on, the next render (a turn clock's tick) drops it
      vi.setSystemTime(Date.now() + 2 * 60 * 60 * 1000);
      rerender(
        <LeagueCorrespondenceGames leagueSlug="league" seasonId="season" />,
      );
      expect(screen.queryByText("vs late")).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  it("redraws itself when its last game reaches a day, leaving the empty box", () => {
    vi.useFakeTimers({ toFake: ["Date", "setTimeout", "clearTimeout"] });
    try {
      season.games = [ended("late", undefined, undefined, 23)];
      render(
        <LeagueCorrespondenceGames leagueSlug="league" seasonId="season" />,
      );
      expect(screen.getByText("vs late")).toBeInTheDocument();
      // nothing else redraws the box: its own timer must
      act(() => {
        vi.advanceTimersByTime(60 * 60 * 1000 + 1000);
      });
      expect(screen.queryByText("vs late")).toBeNull();
      expect(screen.getByText("No active league games")).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it("asks for the season's games again when a live game ends", () => {
    season.games = [];
    lobby.games = [
      { gameID: "live-1", leagueSlug: "league", players: [], lastUpdate: 0 },
    ];
    const { rerender } = render(
      <LeagueCorrespondenceGames leagueSlug="league" seasonId="season" />,
    );
    expect(refetch).not.toHaveBeenCalled();
    // the socket takes the ended game off the lobby's list
    lobby.games = [];
    rerender(
      <LeagueCorrespondenceGames leagueSlug="league" seasonId="season" />,
    );
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
