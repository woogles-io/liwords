import React from 'react';
import { teamTourneys } from '../lobby/fixed_seek_controls';

const playerTag = (
  username: string,
  players: Array<string>,
  tournamentSlug: string
) => {
  // Look up the player in players to figure out its index.
  if (!(tournamentSlug in teamTourneys)) {
    return '';
  }
  const idx = players.findIndex((el) => {
    if (username === el.split(':')[1]) {
      return true;
    }
    return false;
  });

  console.log('found idx', idx, players, username);

  if (idx === -1) {
    return '';
  }
  if (idx % 2 === 1) {
    return teamTourneys[tournamentSlug].evens;
  }
  return teamTourneys[tournamentSlug].odds;
};

type Props = {
  username: string;
  players: Array<string>;
  tournamentSlug: string;
};

// memoize this:
export const PlayerTag = React.memo((props: Props) => {
  return <>{playerTag(props.username, props.players, props.tournamentSlug)}</>;
});
