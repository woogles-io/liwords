import React from 'react';

import { Button, Input } from 'antd';
import { PlayerAvatar } from '../shared/player_avatar';
import { PlayerMetadata } from '../gameroom/game_info';

import './personal_info.scss';

type Props = {
  player: Partial<PlayerMetadata> | undefined;
};

export const PersonalInfo = React.memo((props: Props) => {
  return (
    <>
      <h3>Personal info</h3>
      <div className="avatar-section">
        <PlayerAvatar player={props.player} />
        <Button>Change</Button>
        <Button>Remove</Button>
      </div>
      <div>Player bio</div>
      <div>(the big bio box)</div>
      <div>Account details</div>
      <div>Email</div>
      <Input />
      <div>First name</div>
      <Input />
      <div>Last name</div>
      <Input />
      <div>Country</div>
      <Input />
      <Button>Close my account</Button>
    </>
  );
});
