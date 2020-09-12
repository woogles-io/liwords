import React from 'react';
import { fixedCharAt } from '../utils/cwgame/common';
import './avatar.scss';
import { PlayerMetadata } from '../gameroom/game_info';

type AvatarProps = {
  player: Partial<PlayerMetadata> | undefined;
};

export const PlayerAvatar = (props: AvatarProps) => {
  let avatarStyle = {};

  // XXX temporary code!
  if (props.player?.is_bot) {
    // eslint-disable-next-line no-param-reassign
    props.player.avatar_url =
      'https://woogles-prod-assets.s3.amazonaws.com/macondog.png';
  }

  if (props.player?.avatar_url) {
    avatarStyle = {
      backgroundImage: `url(${props.player?.avatar_url})`,
    };
  }

  return (
    <div className="player-avatar" style={avatarStyle}>
      {!props.player?.avatar_url
        ? fixedCharAt(
            props.player?.full_name || props.player?.nickname || '?',
            0,
            1
          )
        : ''}
    </div>
  );
};
