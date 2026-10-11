import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import {
  GameInfoResponseSchema,
  GameMode,
  GameRequestSchema,
  PlayerInfoSchema,
} from "../gen/api/proto/ipc/omgwords_pb";
import {
  correspondenceRows,
  UserCorrespondenceGames,
} from "./user_corres_games";

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

const HOUR = 3600 * 1000;
const NOW = Date.UTC(2026, 9, 11, 12, 0, 0);

// cesar against someone; two days a turn, no time bank used yet.
const game = (
  gameId: string,
  opponent: string,
  playerOnTurn: number,
  lastUpdateMs: number,
) =>
  create(GameInfoResponseSchema, {
    gameId,
    players: [
      create(PlayerInfoSchema, { nickname: "cesar", userId: "cesar-id" }),
      create(PlayerInfoSchema, {
        nickname: opponent,
        userId: `${opponent}-id`,
      }),
    ],
    gameRequest: create(GameRequestSchema, {
      lexicon: "CSW24",
      incrementSeconds: 2 * 86400,
      gameMode: GameMode.CORRESPONDENCE,
    }),
    playerOnTurn,
    lastUpdate: timestampFromMs(lastUpdateMs),
    timeBank: [BigInt(0), BigInt(0)],
  });

describe("correspondenceRows", () => {
  it("puts the game closest to a forfeit first and names the opponent", () => {
    const rows = correspondenceRows(
      [
        game("calm", "andy", 0, NOW - 12 * HOUR),
        game("urgent", "Josh", 1, NOW - 40 * HOUR),
      ],
      "Cesar",
      NOW,
    );
    expect(rows.map((r) => [r.gameID, r.opponent])).toEqual([
      ["urgent", "Josh"],
      ["calm", "andy"],
    ]);
  });
});

describe("UserCorrespondenceGames", () => {
  it("asks for the user's games and lists whom they play and whose turn it is", () => {
    query.data = { gameInfo: [game("g1", "andy", 1, Date.now() - HOUR)] };
    render(
      <MemoryRouter>
        <UserCorrespondenceGames username="cesar" />
      </MemoryRouter>,
    );
    expect(query.input).toEqual({ username: "cesar" });
    const row = screen.getByRole("link", { name: "andy" }).closest("tr")!;
    expect(screen.getByRole("link", { name: "andy" })).toHaveAttribute(
      "href",
      "/profile/andy",
    );
    expect(within(row).getByRole("link", { name: "Open" })).toHaveAttribute(
      "href",
      "/game/g1",
    );
    // andy is on turn (player 1): the clock column names him.
    expect(within(row).getAllByText("andy")).toHaveLength(2);
  });
});
