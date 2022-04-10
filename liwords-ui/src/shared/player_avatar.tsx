import React, { createContext, useContext, useEffect, useMemo } from 'react';
import { useMountedState } from '../utils/mounted';
import { fixedCharAt } from '../utils/cwgame/common';
import './avatar.scss';
import { Tooltip } from 'antd';
import { PlayerMetadata } from '../gameroom/game_info';
import { useBriefProfile } from '../utils/brief_profiles';
// eslint-disable-next-line @typescript-eslint/no-var-requires
const colors = require('../base.scss');

const doNothing = () => {};
export const PettableContext = createContext<{
  isPettable: boolean;
  setPettable: React.Dispatch<React.SetStateAction<boolean>>;
  isPetting: boolean;
  setPetting: React.Dispatch<React.SetStateAction<boolean>>;
}>({
  isPettable: false,
  setPettable: doNothing,
  isPetting: false,
  setPetting: doNothing,
});

export const PettableAvatar = ({ children }: { children: React.ReactNode }) => {
  const { useState } = useMountedState();
  const [isPettable, setPettable] = useState(false);
  const [isPetting, setPetting] = useState(false);
  const value = useMemo(
    () => ({ isPettable, setPettable, isPetting, setPetting }),
    [isPettable, setPettable, isPetting, setPetting]
  );

  return <PettableContext.Provider value={value} children={children} />;
};

type AvatarProps = {
  player: Partial<PlayerMetadata> | undefined;
  username?: string;
  withTooltip?: boolean;
  editable?: boolean;
};

// XXX: AvatarProps should probably not be based on game info's struct.
export const PlayerAvatar = (props: AvatarProps) => {
  const { isPetting, setPettable } = useContext(PettableContext);
  // Do not useBriefProfile if avatar_url is explicitly passed in as "".
  const profile = useBriefProfile(props.player?.user_id);
  const avatarUrl = props.player?.avatar_url ?? profile?.getAvatarUrl();
  const username = props.username ?? profile?.getUsername();

  let canPet = false;

  let avatarStyle = {};

  if (props.player?.first) {
    avatarStyle = {
      transform: 'rotate(-10deg)',
    };
  }

  if (avatarUrl) {
    let avatarCurrentUrl = avatarUrl;
    if (
      avatarUrl === 'https://woogles-prod-assets.s3.amazonaws.com/macondog.png'
    ) {
      canPet = true;
      // TODO: put this in s3
      if (isPetting) avatarCurrentUrl = require('../assets/pet-macondog.gif');
    }
    avatarStyle = {
      backgroundImage: `url(${avatarCurrentUrl})`,
    };
  }

  useEffect(() => {
    if (canPet) {
      setPettable(true);
      return () => {
        setPettable(false);
      };
    }
  }, [canPet, setPettable]);

  const renderAvatar = (
    <div className="player-avatar" style={avatarStyle}>
      {!avatarUrl
        ? fixedCharAt(
            profile?.getFullName() || props.player?.nickname || username || '?',
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
