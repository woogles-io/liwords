import React, { ReactNode, useCallback } from 'react';
import { FriendUser, useFriendsStoreContext } from '../store/store';
import { useBriefProfile } from '../utils/brief_profiles';
import { PlayerAvatar } from '../shared/player_avatar';
import { DisplayFlag } from '../shared/display_flag';

type Props = {
  sendMessage?: (uuid: string, username: string) => void;
};

type PlayerList = {
  [uuid: string]: Partial<FriendUser>;
};

type PlayerProps = {
  username?: string;
  uuid?: string;
  channel?: string[];
  fromChat?: boolean;
};

const Player = React.memo((props: PlayerProps) => {
  const profile = useBriefProfile(props.uuid);
  const online = props.fromChat || (props.channel && props.channel?.length > 0);
  let inGame =
    props.channel &&
    props.channel.filter((c) => c.includes('chat.game.')).length > 0;
  let watching =
    props.channel &&
    props.channel.filter((c) => c.includes('chat.gametv.')).length > 0;
  return (
    <div
      className={`player-display ${!online ? 'offline' : ''} ${
        inGame ? 'ingame' : ''
      }`}
    >
      <PlayerAvatar
        player={{
          avatar_url: profile?.getAvatarUrl(),
          user_id: props.uuid,
          nickname: props.username,
        }}
      />
      <div>
        <p className="player-name">
          {props.username}{' '}
          <DisplayFlag countryCode={profile?.getCountryCode()} />
        </p>

        {inGame || watching ? (
          <p className="player-activity">
            {inGame ? 'Playing OMGWords' : 'Watching OMGWords'}
          </p>
        ) : null}
      </div>
    </div>
  );
});

export const Players = React.memo((props: Props) => {
  const { friends } = useFriendsStoreContext();

  const renderPlayerList = useCallback(
    (userList: Partial<FriendUser>[]): ReactNode => {
      return (
        <>
          {userList.map((p) => (
            <Player {...p} />
          ))}
        </>
      );
    },
    []
  );
  return (
    <div className="player-list">
      {renderPlayerList(Object.values(friends))}
    </div>
  );
});
