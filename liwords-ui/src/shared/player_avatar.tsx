import React from 'react';
import { fixedCharAt } from '../utils/cwgame/common';
import './avatar.scss';
import { Tooltip } from 'antd';
import { PlayerMetadata } from '../gameroom/game_info';
import { useBriefProfile } from '../utils/brief_profiles';
const colors = require('../base.scss');

type AvatarProps = {
  player: Partial<PlayerMetadata> | undefined;
  username?: string;
  withTooltip?: boolean;
  editable?: boolean;
};

export const PlayerAvatar = (props: AvatarProps) => {
  // Do not useBriefProfile if avatar_url is explicitly passed in as "".
  const profile = useBriefProfile(props.player?.user_id);
  const avatarUrl = props.player?.avatar_url ?? profile?.getAvatarUrl();

  let avatarStyle = {};

  if (props.player?.first) {
    avatarStyle = {
      transform: 'rotate(-10deg)',
    };
  }

  if (avatarUrl) {
    avatarStyle = {
      backgroundImage: `url(${avatarUrl})`,
    };
  }

  const renderAvatar = (
    <div className="player-avatar" style={avatarStyle}>
      {!avatarUrl
        ? fixedCharAt(
            profile?.getFullName() ||
              props.player?.nickname ||
              props.username ||
              '?',
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
