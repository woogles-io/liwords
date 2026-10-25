import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import { GameInfoResponseSchema } from "../gen/api/proto/ipc/omgwords_pb";
import { GameEvent, hasGameEvent } from "./game_event";

const renderEvent = (fields: Record<string, string>) => {
  const game = create(GameInfoResponseSchema, fields);
  return {
    game,
    ...render(
      <MemoryRouter>
        <GameEvent game={game} />
      </MemoryRouter>,
    ),
  };
};

describe("GameEvent", () => {
  it("names and links a league game's league", () => {
    const { game } = renderEvent({
      leagueSlug: "smoke",
      leagueName: "Smoke League",
    });
    expect(screen.getByRole("link", { name: "Smoke League" })).toHaveAttribute(
      "href",
      "/leagues/smoke",
    );
    expect(hasGameEvent(game)).toBe(true);
  });

  it("names and links a tournament game's tournament or club", () => {
    renderEvent({
      tournamentId: "s1",
      tournamentName: "Philly Mondays 2026-10-19",
      tournamentSlug: "/club/philly/2026-10-19",
    });
    expect(
      screen.getByRole("link", { name: "Philly Mondays 2026-10-19" }),
    ).toHaveAttribute("href", "/club/philly/2026-10-19");
  });

  it("shows nothing for a game in the open, or a tournament gone", () => {
    const open = renderEvent({});
    expect(open.container.textContent).toBe("");
    expect(hasGameEvent(open.game)).toBe(false);
    const gone = renderEvent({ tournamentId: "t0" });
    expect(gone.container.textContent).toBe("");
    expect(hasGameEvent(gone.game)).toBe(false);
  });
});
