import { useQuery } from "@connectrpc/connect-query";
import { Table } from "antd";
import React from "react";
import { Link } from "react-router";
import { getActiveCorrespondenceGames } from "../gen/api/proto/game_service/game_service-GameMetadataService_connectquery";
import { GameInfoResponse } from "../gen/api/proto/ipc/omgwords_pb";
import { CorrespondenceTurnIndicator } from "../shared/corres_turn_indicator";
import {
  ActiveGame,
  GameInfoResponseToActiveGame,
} from "../store/reducers/lobby_reducer";

type Row = {
  key: string;
  gameID: string;
  opponent: string;
  game: ActiveGame;
};

// Milliseconds until the on-turn player's clock forfeits (per-turn allowance +
// time bank - elapsed); a game without a clock sorts last.
export const msToForfeit = (ag: ActiveGame, now: number): number => {
  if (ag.playerOnTurn === undefined || ag.lastUpdate === undefined) {
    return Number.MAX_SAFE_INTEGER;
  }
  return (
    ag.incrementSecs * 1000 +
    (ag.timeBank?.[ag.playerOnTurn] ?? 0) -
    (now - ag.lastUpdate)
  );
};

// One row per game the user is playing, the closest to a forfeit first.
export const correspondenceRows = (
  games: GameInfoResponse[],
  username: string,
  now: number,
): Row[] =>
  games
    .map((gi) => GameInfoResponseToActiveGame(gi))
    .filter((ag): ag is ActiveGame => ag !== null)
    .sort((a, b) => msToForfeit(a, now) - msToForfeit(b, now))
    .map((ag) => ({
      key: ag.gameID,
      gameID: ag.gameID,
      opponent:
        ag.players.find(
          (p) => p.displayName.toLowerCase() !== username.toLowerCase(),
        )?.displayName ?? "?",
      game: ag,
    }));

// A user's correspondence games in progress, for the admin user lookup: whom
// they play, whose turn it is, and that player's clock, ticking.
export const UserCorrespondenceGames = (props: { username: string }) => {
  const { data, error, isLoading } = useQuery(
    getActiveCorrespondenceGames,
    { username: props.username },
    { retry: false },
  );
  const rows = correspondenceRows(
    data?.gameInfo ?? [],
    props.username,
    Date.now(),
  );

  return (
    <>
      <h3>Ongoing correspondence games</h3>
      <Table
        dataSource={rows}
        size="small"
        pagination={false}
        loading={isLoading}
        locale={{ emptyText: error ? "Error: " + error.message : "None" }}
        columns={[
          {
            title: "Opponent",
            dataIndex: "opponent",
            key: "opponent",
            render: (opponent: string) => (
              <Link
                to={`/profile/${encodeURIComponent(opponent)}`}
                target="_blank"
              >
                {opponent}
              </Link>
            ),
          },
          {
            title: "On turn",
            key: "turn",
            render: (_: unknown, row: Row) => {
              const ag = row.game;
              if (ag.playerOnTurn === undefined) return null;
              return (
                <CorrespondenceTurnIndicator
                  perspective={{
                    kind: "spectator",
                    playerName: ag.players[ag.playerOnTurn]?.displayName || "?",
                  }}
                  lastUpdateMs={ag.lastUpdate || undefined}
                  incrementMs={ag.incrementSecs * 1000}
                  bankMs={ag.timeBank?.[ag.playerOnTurn] ?? 0}
                />
              );
            },
          },
          {
            title: "Game",
            dataIndex: "gameID",
            key: "gameID",
            render: (gameID: string) => (
              <Link to={`/game/${encodeURIComponent(gameID)}`} target="_blank">
                Open
              </Link>
            ),
          },
        ]}
      />
    </>
  );
};
