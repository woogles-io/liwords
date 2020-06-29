import React from 'react';
import { fixedCharAt } from '../utils/cwgame/common';
import './avatar.scss';
import { PlayerMetadata } from '../gameroom/game_info';

type AvatarProps = {
  player: Partial<PlayerMetadata>;
};

export const PlayerAvatar = (props: AvatarProps) => {
  let avatarStyle = {};
  if (props.player.avatar_url) {
    avatarStyle = {
      backgroundImage: `url(${props.player?.avatar_url})`,
    };
  }

  return (
    <div className="player-avatar" style={avatarStyle}>
      {!props.player?.avatar_url
        ? fixedCharAt(
            props.player.full_name || props.player.nickname || '?',
            0,
            1
          )
        : ''}
    </div>
  );
};
