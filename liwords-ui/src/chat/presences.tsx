import React from 'react';
import { PresenceEntity } from '../store/store';

type Props = {
  players: { [uuid: string]: PresenceEntity };
};

export const Presences = React.memo((props: Props) => {
  const vals = Object.values(props.players);
  vals.sort((a, b) => (a.username < b.username ? -1 : 1));

  /* Use these, once we've marked them as anonymous or not
  const profileLink = (username: string) => (
    <a href={`../profile/${username}`}>{username}</a>
  );
*/
  const profileLink = (username: string) => <>{username}</>;
  const presences = Object.keys(props.players)
    .map<React.ReactNode>((u) => profileLink(props.players[u].username))
    .reduce((prev, curr) => [prev, ', ', curr]);

  return <>{presences}</>;
});
