import React from 'react';
import { PresenceEntity } from '../store/store';

type Props = {
  players: { [uuid: string]: PresenceEntity };
};

export const Presences = React.memo((props: Props) => {
  const vals = Object.values(props.players);
  vals.sort((a, b) => (a.username < b.username ? -1 : 1));

  const profileLink = (player: PresenceEntity) =>
    player.anon ? (
      <>{player.username}</>
    ) : (
      <a href={`../profile/${player.username}`}>{player.username}</a>
    );
  const presences = Object.keys(props.players)
    .map<React.ReactNode>((u) => profileLink(props.players[u]))
    .reduce((prev, curr) => [prev, ', ', curr]);

  return <>{presences}</>;
});
