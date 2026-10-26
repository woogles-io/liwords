import { Tooltip } from "antd";
import React from "react";
import { Link } from "react-router";
import { GameInfoResponse } from "../gen/api/proto/ipc/omgwords_pb";

// Where a game was played, if not in the open: its league, or its tournament,
// club or club session, named and linked. The game list carries the names.
// Hovering gives the name whole, where a narrow card cuts it.
export const GameEvent = (props: { game: GameInfoResponse }) => {
  const { game } = props;
  if (game.leagueSlug) {
    const name = game.leagueName || game.leagueSlug;
    return (
      <Tooltip title={name}>
        <Link to={`/leagues/${encodeURIComponent(game.leagueSlug)}`}>
          {name}
        </Link>
      </Tooltip>
    );
  }
  if (game.tournamentSlug) {
    const name = game.tournamentName || game.tournamentSlug;
    return (
      <Tooltip title={name}>
        <Link to={game.tournamentSlug}>{name}</Link>
      </Tooltip>
    );
  }
  return null;
};

// Whether a game has an event to show.
export const hasGameEvent = (game: GameInfoResponse): boolean =>
  !!(game.leagueSlug || game.tournamentSlug);
