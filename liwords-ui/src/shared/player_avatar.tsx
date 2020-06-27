import React from 'react';
import { ReducedPlayerInfo } from '../store/reducers/game_reducer';
import { fixedCharAt} from '../utils/cwgame/common'
import './avatar.scss';

type AvatarProps = {
  player: ReducedPlayerInfo,
};

export const PlayerAvatar = (props: AvatarProps) => {
  let avatarStyle = {};
  if (props.player.avatarUrl) {
    avatarStyle = {
      backgroundImage: `url(${props.player?.avatarUrl})`,
    };
  }

  return (
    <div
      className="player-avatar"
      style={avatarStyle}
    >
      {!props.player?.avatarUrl ?
        fixedCharAt(props.player.fullName || props.player.nickname || '?', 0, 1) : ''}
    </div>
  );
}
