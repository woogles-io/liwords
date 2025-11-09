import React, {
  ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  FriendUser,
  useFriendsStoreContext,
  useLoginStateStoreContext,
  usePresenceStoreContext,
  useExcludedPlayersStoreContext,
} from "../store/store";
import { PresenceEntity } from "../store/constants";
import { PettableAvatar, PlayerAvatar } from "../shared/player_avatar";
import { moderateUser } from "../mod/moderate";
import { Form, Input } from "antd";
import { UsernameWithContext } from "../shared/usernameWithContext";
import "./playerList.scss";
import { useDebounce } from "../utils/debounce";
import { flashError, useClient } from "../utils/hooks/connect";
import { AutocompleteService } from "../gen/api/proto/user_service/user_service_pb";

type Props = {
  defaultChannelType: string;
  sendMessage: (uuid: string, username: string) => void;
};

type MyIntersectionObserver = {
  observe: (domElt: HTMLElement, uuid: string) => void;
  unobserve: (domElt: HTMLElement, uuid: string) => void;
};

type PlayerProps = {
  className: string;
  username?: string;
  uuid?: string;
  channel?: string[];
  fromChat?: boolean; // XXX: this doesn't seem to be used?
  sendMessage: (uuid: string, username: string) => void;
  myio: MyIntersectionObserver;
};

const activityToText = (
  inGame: boolean,
  watching: boolean,
  editing: boolean,
  watchingAnno: boolean,
) => {
  if (inGame) {
    return "Playing OMGWords";
  }
  if (watching) {
    return "Watching OMGWords";
  }
  if (editing) {
    return "Annotating an OMGWords game";
  }
  if (watchingAnno) {
    return "Watching an OMGWords annotation";
  }
};

const Player = React.memo((props: PlayerProps) => {
  let online = props.fromChat;
  let puzzling = false;
  const games = new Map<string, Set<string>>();
  if (props.channel) {
    let numChannels = props.channel.length;
    const re = /^(activegame:|chat\.game\.|chat\.gametv\.)(.*)$/;
    for (const c of props.channel) {
      const m = c.match(re);
      if (m) {
        const [, groupName, gameID] = m;
        if (!games.has(gameID)) {
          games.set(gameID, new Set());
        }
        games.get(gameID)!.add(groupName);
        if (groupName === "activegame:") {
          --numChannels;
        }
      }
    }
    if (numChannels > 0) {
      // if a user is offline but still in a game this condition would not be entered.
      online = true;
    }
  }
  const currentActiveGames: Array<string> = [];
  const currentWatchedGames: Array<string> = [];
  const currentEditingGames: Array<string> = [];
  const currentWatchedAnnoGames: Array<string> = [];
  games.forEach((groupNames, gameId) => {
    if (groupNames.has("activegame:") && groupNames.has("chat.game.")) {
      currentActiveGames.push(gameId);
    } else if (groupNames.has("chat.gametv.")) {
      if (gameId.includes(".")) {
        // editor or anno
        const [t, actualGameId] = gameId.split(".");
        if (t === "editor") {
          currentEditingGames.push(actualGameId);
        } else if (t === "anno") {
          currentWatchedAnnoGames.push(actualGameId);
        }
      } else {
        currentWatchedGames.push(gameId);
      }
    }
  });
  if (props.channel?.includes("chat.puzzlelobby")) {
    puzzling = true;
    // later can use specific puzzle chats and follow people to specific puzzles.
  }

  const inGame = currentActiveGames.length > 0;
  const watching = currentWatchedGames.length > 0;
  const editing = currentEditingGames.length > 0;
  const watchingAnno = currentWatchedAnnoGames.length > 0;

  const [domElt, setDomElt] = useState<HTMLElement | null>();
  useEffect(() => {
    if (domElt && props.myio && props.uuid) {
      const uuid = props.uuid;
      props.myio.observe(domElt, uuid);
      return () => {
        props.myio.unobserve(domElt, uuid);
      };
    }
  }, [domElt, props.myio, props.uuid]);

  if (!props.username) {
    return null;
  }

  return (
    <div
      className={`player-display ${!online ? "offline" : ""} ${
        inGame ? "ingame" : ""
      } ${props.className ? props.className : ""} ${
        puzzling && !inGame ? "puzzling" : ""
      }`}
      ref={setDomElt}
      key={props.uuid}
    >
      <PettableAvatar>
        <PlayerAvatar
          player={{
            userId: props.uuid,
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
              omitBlock={props.className === "friends"}
              showModTools
              sendMessage={props.sendMessage}
              currentActiveGames={currentActiveGames}
              currentWatchedGames={currentWatchedGames}
              currentEditingGames={currentEditingGames}
              currentWatchedAnnoGames={currentWatchedAnnoGames}
              currentlyPuzzling={puzzling}
            />
          </p>
          {inGame || watching || editing || watchingAnno ? (
            <p className="player-activity">
              {activityToText(inGame, watching, editing, watchingAnno)}
            </p>
          ) : puzzling ? (
            <p className="player-activity">Solving puzzles</p>
          ) : null}
        </div>
      </PettableAvatar>
    </div>
  );
});

type PlayerListProps = {
  userList: Partial<FriendUser>[];
  className: string;
  sendMessage: (uuid: string, username: string) => void;
};

const PlayerList = (props: PlayerListProps) => {
  const uuidToIndex = useMemo(
    () =>
      props.userList.reduce(
        (ret, { uuid }, idx) => {
          if (uuid != null) ret[uuid] = idx;
          return ret;
        },
        {} as { [key: string]: number },
      ),
    [props.userList],
  );

  // 48px height + 18px margin = 66px for each entry.
  // initially we show just as many items as needed
  // so it does not immediately trigger another load.
  // assume 1080px monitor at full height.
  const threshold = 0; // can increase this to load earlier.
  const [numShown, setNumShown] = useState(
    Math.ceil(1080 + 18) / 66 + (threshold + 1),
  );

  const domEltToUuid = useRef(new Map());
  const visibleDomElts = useRef(new Set());

  const [intersectionObserver, setIntersectionObserver] =
    useState<IntersectionObserver>();
  useEffect(() => {
    const callback = (entries: IntersectionObserverEntry[]) => {
      entries.forEach((entry) => {
        const uuid = domEltToUuid.current.get(entry.target);
        if (uuid) {
          const visible = entry.isIntersecting;
          if (visible) {
            visibleDomElts.current.add(entry.target);

            // expand in one direction only for simplicity.
            // (because not rendering the earlier elements would affect the scroll position.)

            // one-based index corresponds to the minimum number of items
            // that must be shown for this item to be shown.
            const oneBasedIndex = (uuidToIndex[uuid] ?? 0) + 1;
            // pageSize is approximate, depends on timing.
            const pageSize = Math.max(visibleDomElts.current.size, 1);
            const newMinNumShown = oneBasedIndex + pageSize;
            setNumShown((numShown) => {
              if (oneBasedIndex >= numShown - threshold) {
                // it is near the end, time to load more.
                return Math.max(numShown, newMinNumShown);
              } else {
                // it is not near the end, do nothing for now.
                return numShown;
              }
            });
          } else {
            visibleDomElts.current.delete(entry.target);
          }
        }
      });
    };
    const intersectionObserver = new IntersectionObserver(callback);
    setIntersectionObserver(intersectionObserver);
    return () => {
      intersectionObserver.disconnect();
    };
  }, [uuidToIndex]);

  const myio = useMemo(() => {
    return {
      observe: (domElt: HTMLElement, uuid: string): void => {
        if (!domEltToUuid.current.has(domElt)) {
          domEltToUuid.current.set(domElt, uuid);
        }
        intersectionObserver?.observe(domElt);
      },
      unobserve: (domElt: HTMLElement, uuid: string): void => {
        intersectionObserver?.unobserve(domElt);
        domEltToUuid.current.delete(domElt);
        visibleDomElts.current.delete(domElt);
      },
    };
  }, [intersectionObserver]);

  return (
    <>
      {props.userList.reduce((ret, p, idx) => {
        if (idx < numShown)
          ret.push(
            <Player
              sendMessage={props.sendMessage}
              className={props.className}
              key={p.uuid}
              myio={myio}
              {...p}
            />,
          );
        return ret;
      }, [] as ReactNode[])}
    </>
  );
};

export const Players = React.memo((props: Props) => {
  const { friends } = useFriendsStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const { sendMessage, defaultChannelType } = props;
  const { userID, loggedIn } = loginState;
  const [maxHeight, setMaxHeight] = useState<number | undefined>(0);
  const [searchResults, setSearchResults] = useState<
    Array<Partial<FriendUser>>
  >([]);
  const [searchText, setSearchText] = useState("");
  const { presences } = usePresenceStoreContext();
  const { excludedPlayers } = useExcludedPlayersStoreContext();

  const setHeight = useCallback(() => {
    const tabPaneHeight = document.getElementById("chat")?.clientHeight;
    setMaxHeight(tabPaneHeight ? tabPaneHeight - 117 : undefined);
  }, []);

  useEffect(() => {
    setHeight();
  }, [setHeight]);

  const acClient = useClient(AutocompleteService);

  const onlineAlphaComparator = useCallback(
    (a: Partial<FriendUser>, b: Partial<FriendUser>) => {
      const countA = (a.channel || []).length > 0 ? 1 : -1;
      const countB = (b.channel || []).length > 0 ? 1 : -1;
      return (
        countB - countA || (a.username ?? "").localeCompare(b.username ?? "")
      );
    },
    [],
  );

  const lastSearchedText = useRef("");
  const onPlayerSearch = useCallback(
    async (searchText: string) => {
      if (lastSearchedText.current === searchText) {
        // do not trigger a search if user types something and undoes it,
        // because we would already have the same search results in that case.
        // (does not apply if user clears the text box.)
        return;
      }
      lastSearchedText.current = searchText;
      if (searchText) {
        try {
          const resp = await acClient.getCompletion({ prefix: searchText });
          setSearchResults(
            resp.users
              .filter(
                (u) => u.uuid && u.uuid !== userID && !(u.uuid in friends),
              )
              .sort(onlineAlphaComparator),
          );
        } catch (e) {
          flashError(e);
        }
      } else {
        setSearchResults([]);
      }
    },
    [userID, friends, onlineAlphaComparator, acClient],
  );
  const searchUsernameDebounced = useDebounce(onPlayerSearch, 200);

  const handleSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const prefix = e.target.value;
      setSearchText(prefix);
      // when clearing the text box, cancel the now unwanted search.
      // so, always call this.
      searchUsernameDebounced(prefix);
      if (!prefix) {
        // but clear the results immediately.
        lastSearchedText.current = "";
        setSearchResults([]);
      }
    },
    [searchUsernameDebounced],
  );

  const renderPlayerList = useCallback(
    (userList: Partial<FriendUser>[], className = ""): ReactNode => {
      const nonExcludedUsers = userList.filter((p) => {
        if (p.uuid) {
          return !excludedPlayers.has(p.uuid);
        }
      });

      return (
        <PlayerList
          userList={nonExcludedUsers}
          className={className}
          sendMessage={sendMessage}
        />
      );
    },
    [sendMessage, excludedPlayers],
  );

  const filterPlayerListBySearch = useCallback(
    (searchTerm: string, list: Partial<FriendUser>[]) => {
      if (searchTerm.length) {
        const lowercasedSearchTerm = searchTerm.toLowerCase();
        return list.filter((u) =>
          u.username?.toLowerCase().startsWith(lowercasedSearchTerm),
        );
      } else {
        return list;
      }
    },
    [],
  );

  const transformAndFilterPresences = useCallback(
    (
      presenceEntities: PresenceEntity[],
      searchTerm: string,
    ): Partial<FriendUser>[] => {
      const presencePlayersMap: { [uuid: string]: FriendUser } = {};
      presenceEntities.forEach((p) => {
        if (p.uuid === userID) {
          // ignore self
        } else if (p.anon) {
          // ignore anonymous
        } else if (p.uuid in presencePlayersMap) {
          // mutating this in-place is safe, it has not been shared
          presencePlayersMap[p.uuid].channel.push(p.channel);
        } else {
          presencePlayersMap[p.uuid] = {
            username: p.username,
            uuid: p.uuid,
            channel: [p.channel],
          };
        }
      });
      const presencePlayers = Object.values(presencePlayersMap).sort(
        onlineAlphaComparator,
      );
      return filterPlayerListBySearch(searchTerm, presencePlayers);
    },
    [userID, onlineAlphaComparator, filterPlayerListBySearch],
  );

  const transformedAndFilteredPresences = useMemo(
    () => transformAndFilterPresences(presences, searchText),
    [transformAndFilterPresences, presences, searchText],
  );

  const tournamentPresences = useMemo(() => {
    if (defaultChannelType === "lobby") {
      return [];
    }
    const tournamentPresences = transformedAndFilteredPresences.filter(
      (p) =>
        p.channel &&
        p.channel.some((c) => {
          return c.startsWith("chat.tournament");
        }),
    );
    return tournamentPresences;
  }, [transformedAndFilteredPresences, defaultChannelType]);

  const leaguePresences = useMemo(() => {
    if (defaultChannelType === "lobby") {
      return [];
    }
    const leaguePresences = transformedAndFilteredPresences.filter(
      (p) =>
        p.channel &&
        p.channel.some((c) => {
          return c.startsWith("chat.league");
        }),
    );
    return leaguePresences;
  }, [transformedAndFilteredPresences, defaultChannelType]);

  const gamePresence = useMemo(() => {
    if (defaultChannelType === "lobby") {
      return [];
    }
    const gamePresences = transformedAndFilteredPresences.filter(
      (p) =>
        p.channel &&
        p.channel.some((c) => {
          return c.startsWith("chat.game");
        }),
    );
    return gamePresences;
  }, [transformedAndFilteredPresences, defaultChannelType]);

  useEffect(() => {
    window.addEventListener("resize", setHeight);
    return () => {
      window.removeEventListener("resize", setHeight);
    };
  }, [setHeight]);

  const getPresenceLabel = (channelType: string) => {
    switch (channelType) {
      case "lobby":
        return "IN LOBBY";
      case "game":
        return "OPPONENT";
      case "gametv":
        return "OBSERVERS";
      case "tournament":
        return "CLUB/TOURNAMENT";
      case "league":
        return "LEAGUE";
      case "puzzle":
        return "SOLVING PUZZLES";
    }
    return "IN ROOM";
  };

  const friendsValues = useMemo(
    () => Object.values(friends).sort(onlineAlphaComparator),
    [friends, onlineAlphaComparator],
  );
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
          props.defaultChannelType ? props.defaultChannelType : ""
        }`}
        style={
          maxHeight
            ? {
                maxHeight: maxHeight,
                overflowY: "auto",
              }
            : undefined
        }
      >
        <section className="friends">
          {loggedIn && <div className="breadcrumb">FRIENDS</div>}
          {loggedIn &&
            renderPlayerList(
              filterPlayerListBySearch(searchText, friendsValues),
              "friends",
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
              {getPresenceLabel(defaultChannelType || "")}
            </div>
          )}
          {renderPlayerList(gamePresence)}
          {tournamentPresences.length > 0 && (
            <div className="breadcrumb">{getPresenceLabel("tournament")}</div>
          )}
          {renderPlayerList(tournamentPresences)}
          {leaguePresences.length > 0 && (
            <div className="breadcrumb">{getPresenceLabel("league")}</div>
          )}
          {renderPlayerList(leaguePresences)}
          {!gamePresence.length &&
            !tournamentPresences.length &&
            !leaguePresences.length &&
            transformedAndFilteredPresences.length > 0 && (
              <>
                <div className="breadcrumb">
                  {getPresenceLabel(props.defaultChannelType || "")}
                </div>
                {renderPlayerList(transformedAndFilteredPresences)}
              </>
            )}
        </section>
        <section className="search">
          {searchResults.length > 0 && (
            <>
              <div className="breadcrumb">ALL PLAYERS</div>
              {renderPlayerList(searchResults, "search")}
            </>
          )}
        </section>
      </div>
    </div>
  );
});
