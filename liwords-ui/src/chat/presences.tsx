import React from 'react';
import { Row, Col } from 'antd';
import { PresenceEntity } from '../store/store';

type Props = {
  players: { [uuid: string]: PresenceEntity };
};

export const Presences = React.memo((props: Props) => {
  const vals = Object.values(props.players);
  vals.sort((a, b) => (a.username < b.username ? -1 : 1));

  const presences = Object.keys(props.players).map((p) => (
    <div key={props.players[p].uuid}>{props.players[p].username}</div>
  ));
  return (
    <div className="entities" style={{ marginTop: 20 }}>
      {presences}
    </div>
  );
});
