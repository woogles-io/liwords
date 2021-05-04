import React, { ReactNode, useCallback, useEffect, useMemo } from 'react';
import {
  FriendUser,
  PresenceEntity,
  useFriendsStoreContext,
  useLoginStateStoreContext,
  usePresenceStoreContext,
} from '../store/store';
import { useBriefProfile } from '../utils/brief_profiles';
import { PlayerAvatar } from '../shared/player_avatar';
import { moderateUser } from '../mod/moderate';
import { Form, Input } from 'antd';
import { UsernameWithContext } from '../shared/usernameWithContext';
import './playerList.scss';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { debounce } from '../utils/debounce';
import { useMountedState } from '../utils/mounted';

type Props = {
  defaultChannelType?: string;
  sendMessage?: (uuid: string, username: string) => void;
};

type PlayerProps = {
  className?: string;
  username?: string;
  uuid?: string;
  channel?: string[];
  fromChat?: boolean;
  sendMessage?: (uuid: string, username: string) => void;
};

const Player = React.memo((props: PlayerProps) => {
  const profile = useBriefProfile(props.uuid);

  const online = props.fromChat || (props.channel && props.channel?.length > 0);
  let inGame =
    props.channel && props.channel.some((c) => c.includes('chat.game.'));
  let watching =
    props.channel && props.channel.some((c) => c.includes('chat.gametv.'));
  if (!props.username) {
    return null;
  }
  return (
    <div
      className={`player-display ${!online ? 'offline' : ''} ${
        inGame ? 'ingame' : ''
      } ${props.className ? props.className : ''}`}
      key={props.uuid}
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
          <UsernameWithContext
            username={props.username}
            userID={props.uuid}
            moderate={moderateUser}
            includeFlag
            omitBlock={props.className === 'friends'}
            showModTools
            sendMessage={props.sendMessage}
          />
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

type SearchResponse = {
  users: Array<Partial<FriendUser>>;
};

export const Players = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const { friends } = useFriendsStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { sendMessage, defaultChannelType } = props;
  const { username, loggedIn } = loginState;
  const [maxHeight, setMaxHeight] = useState<number | undefined>(0);
  const [searchResults, setSearchResults] = useState<
    Array<Partial<FriendUser>>
  >([]);
  const [searchText, setSearchText] = useState('');
  const { presences } = usePresenceStoreContext();

  const setHeight = useCallback(() => {
    const tabPaneHeight = document.getElementById('chat')?.clientHeight;
    setMaxHeight(tabPaneHeight ? tabPaneHeight - 117 : undefined);
  }, []);

  useEffect(() => {
    setHeight();
  }, [setHeight]);
  const onPlayerSearch = useCallback(
    (searchText: string) => {
      if (searchText?.length > 0) {
        axios
          .post<SearchResponse>(
            toAPIUrl('user_service.AutocompleteService', 'GetCompletion'),
            {
              prefix: searchText,
            }
          )
          .then((resp) => {
            // Exclude your friends
            const nonfriends = resp.data.users.filter((u) => {
              return u.uuid && !(u.uuid in friends);
            });
            // Exclude yourself
            setSearchResults(
              !searchText
                ? []
                : nonfriends.filter(
                    (u) => u.username?.toLowerCase() !== username.toLowerCase()
                  )
            );
          });
      } else {
        setSearchResults([]);
      }
    },
    [username, friends]
  );
  const searchUsernameDebounced = debounce(onPlayerSearch, 200);

  const handleSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const prefix = e.target.value;
      setSearchText(prefix);
      if (prefix?.length > 0) {
        searchUsernameDebounced(prefix);
      }
    },
    [searchUsernameDebounced]
  );

  const renderPlayerList = useCallback(
    (userList: Partial<FriendUser>[], className: string = ''): ReactNode => {
      return (
        <>
          {userList.map((p) => (
            <Player
              sendMessage={sendMessage}
              className={className}
              key={p.uuid}
              {...p}
            />
          ))}
        </>
      );
    },
    [sendMessage]
  );

  const filterPlayerListBySearch = useCallback(
    (searchTerm: string, list: Partial<FriendUser>[]) => {
      if (searchTerm?.length) {
        return list.filter((u) =>
          u.username?.toLowerCase().startsWith(searchTerm.toLowerCase())
        );
      } else {
        return list;
      }
    },
    []
  );

  const transformAndFilterPresences = useCallback(
    (
      presenceEntities: PresenceEntity[],
      searchTerm: string
    ): Partial<FriendUser>[] => {
      const presencePlayersMap: { [uuid: string]: FriendUser } = {};
      presenceEntities
        .filter((p) => !p.anon)
        .forEach((p) => {
          if (p.uuid in presencePlayersMap) {
            presencePlayersMap[p.uuid] = {
              ...presencePlayersMap[p.uuid],
              channel: presencePlayersMap[p.uuid].channel.concat(p.channel),
            };
          } else {
            presencePlayersMap[p.uuid] = {
              username: p.username,
              uuid: p.uuid,
              channel: [p.channel],
            };
          }
        });
      const presencePlayers = Object.values(presencePlayersMap)
        .sort((a, b) => {
          if (a.username > b.username) {
            return -1;
          }
          if (a.username < b.username) {
            return 1;
          }
          return 0;
        })
        .filter((u) => u.username?.toLowerCase() !== username.toLowerCase());
      return searchTerm?.length
        ? presencePlayers.filter((u) =>
            u.username?.toLowerCase().startsWith(searchTerm.toLowerCase())
          )
        : presencePlayers;
    },
    [username]
  );

  const onlineAlphaComparator = useCallback(
    (a: Partial<FriendUser>, b: Partial<FriendUser>) => {
      const countA = (a.channel || []).length > 0 ? 1 : -1;
      const countB = (b.channel || []).length > 0 ? 1 : -1;
      return (
        countB - countA ||
        ((a.username || '').toLowerCase() || '').localeCompare(
          (b.username || '').toLowerCase()
        )
      );
    },
    []
  );

  const transformedAndFilteredPresences = useMemo(
    () =>
      transformAndFilterPresences(presences, searchText).sort(
        onlineAlphaComparator
      ),
    [transformAndFilterPresences, presences, searchText, onlineAlphaComparator]
  );

  const tournamentPresences = useMemo(() => {
    if (defaultChannelType === 'lobby') {
      return [];
    }
    const tournamentPresences = transformedAndFilteredPresences.filter(
      (p) =>
        p.channel &&
        p.channel.some((c) => {
          return c.startsWith('chat.tournament');
        })
    );
    return tournamentPresences;
  }, [transformedAndFilteredPresences, defaultChannelType]);

  const gamePresence = useMemo(() => {
    if (defaultChannelType === 'lobby') {
      return [];
    }
    const gamePresences = transformedAndFilteredPresences.filter(
      (p) =>
        p.channel &&
        p.channel.some((c) => {
          return c.startsWith('chat.game');
        })
    );
    return gamePresences;
  }, [transformedAndFilteredPresences, defaultChannelType]);

  useEffect(() => {
    window.addEventListener('resize', setHeight);
    return () => {
      window.removeEventListener('resize', setHeight);
    };
  }, [setHeight]);

  const getPresenceLabel = (channelType: string) => {
    switch (channelType) {
      case 'lobby':
        return 'IN LOBBY';
      case 'game':
        return 'OPPONENT';
      case 'gametv':
        return 'OBSERVERS';
      case 'tournament':
        return 'CLUB/TOURNAMENT';
    }
    return 'IN ROOM';
  };

  const friendsValues = useMemo(() => Object.values(friends), [friends]);
  return (
    <div className="player-list">
      <Form name="search-players">
        <Input
          allowClear
          placeholder="Search players"
          name="search-players"
          onChange={handleSearchChange}
          value={searchText}
          autoComplete="off"
        />
      </Form>
      <div
        className={`player-sections p-${
          props.defaultChannelType ? props.defaultChannelType : ''
        }`}
        style={
          maxHeight
            ? {
                maxHeight: maxHeight,
                overflowY: 'auto',
              }
            : undefined
        }
      >
        <section className="friends">
          {loggedIn && <div className="breadcrumb">FRIENDS</div>}
          {loggedIn &&
            renderPlayerList(
              filterPlayerListBySearch(searchText, friendsValues).sort(
                onlineAlphaComparator
              ),
              'friends'
            )}
          {loggedIn && friendsValues.length === 0 && (
            <p className="prompt">
              You haven't added any friends. Add some now to see when they're
              online!
            </p>
          )}
        </section>
        <section className="present">
          {gamePresence.length > 0 && (
            <div className="breadcrumb">
              {getPresenceLabel(defaultChannelType || '')}
            </div>
          )}
          {renderPlayerList(gamePresence)}
          {tournamentPresences.length > 0 && (
            <div className="breadcrumb">{getPresenceLabel('tournament')}</div>
          )}
          {renderPlayerList(tournamentPresences)}
          {!gamePresence.length &&
            !tournamentPresences.length &&
            transformedAndFilteredPresences.length > 0 && (
              <>
                <div className="breadcrumb">
                  {getPresenceLabel(props.defaultChannelType || '')}
                </div>
                {renderPlayerList(transformedAndFilteredPresences)}
              </>
            )}
        </section>
        <section className="search">
          {searchResults?.length > 0 && searchText.length > 0 && (
            <div className="breadcrumb">ALL PLAYERS</div>
          )}
          {searchResults?.length > 0 &&
            searchText.length > 0 &&
            renderPlayerList(searchResults, 'search')}
        </section>
      </div>
    </div>
  );
});
