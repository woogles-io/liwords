import React from 'react';
import { fixedCharAt } from '../utils/cwgame/common';
import './avatar.scss';
import { Tooltip } from 'antd';
import { PlayerMetadata } from '../gameroom/game_info';
const colors = require('../base.scss');

type AvatarProps = {
  player: Partial<PlayerMetadata> | undefined;
  withTooltip?: boolean;
};

export const PlayerAvatar = (props: AvatarProps) => {
  let avatarStyle = {};

  if (props.player?.first) {
    avatarStyle = {
      transform: 'rotate(-10deg)',
    };
  }

  if (props.player?.avatar_url) {
    avatarStyle = {
      backgroundImage: `url(${props.player?.avatar_url})`,
    };
  }
  const renderAvatar = (
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
  if (!props.withTooltip) {
    return renderAvatar;
  }
  return (
    <Tooltip
      title={props.player?.nickname}
      placement="left"
      mouseEnterDelay={0.1}
      mouseLeaveDelay={0.01}
      color={colors.colorPrimary}
    >
      {renderAvatar}
    </Tooltip>
  );
};
