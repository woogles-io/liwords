import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router";
import { Card, Carousel } from "antd";
import { TopBar } from "../navigation/topbar";
import { PettableAvatar, PlayerAvatar } from "../shared/player_avatar";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { moderateUser } from "../mod/moderate";
import { DisplayFlag } from "../shared/display_flag";
import { useLoginStateStoreContext } from "../store/store";
import "./profile.scss";
import { BioCard } from "./bio";
import { lexiconCodeToProfileRatingName } from "../shared/lexica";
import { VariantIcon } from "../shared/variant_icons";
import moment from "moment";
import { GameCard } from "./gameCard";
import { GamesHistoryCard } from "./games_history";
import {
  GameEndReason,
  GameInfoResponse,
} from "../gen/api/proto/ipc/omgwords_pb";
import { flashError, useClient } from "../utils/hooks/connect";
import { ProfileService } from "../gen/api/proto/user_service/user_service_pb";
import { GameMetadataService } from "../gen/api/proto/game_service/game_service_pb";
import { BroadcastGamesResponse_BroadcastGame } from "../gen/api/proto/omgwords_service/omgwords_pb";
import { GameEventService } from "../gen/api/proto/omgwords_service/omgwords_pb";
import { AnnotatedGamesHistoryCard } from "./annotated_games_history";
import { UserCollectionsCard } from "./user_collections";
import variables from "../base.module.scss";
import { useQuery } from "@connectrpc/connect-query";
import { getBadgesMetadata } from "../gen/api/proto/user_service/user_service-ProfileService_connectquery";
import { Badge } from "./badge";
import { DisplayUserOrganizations } from "./organizations";
const { screenSizeTablet } = variables;

type Rating = {
  r: number;
  rd: number;
  v: number;
  ts: number;
};

type ProfileRatings = { [variant: string]: Rating };

type StatItem = {
  n?: string; // name
  t: number; // total
  a?: Array<number>; // averages
};

type Stats = {
  i1: string; // us
  i2: string; // opp
  d1: { [key: string]: StatItem }; // us
  d2: { [key: string]: StatItem }; // opp
  n: Array<StatItem>; // notable
};

type ProfileStats = {
  [variant: string]: Stats;
};

type VariantCardProps = {
  variant: string;
  ratings: Rating | null;
  stats: Stats;
};

const VariantCard = React.memo((props: VariantCardProps) => {
  const { ratings, stats, variant } = props;
  const gamesPlayed = stats.d1.Games.t;
  const rating = ratings?.r.toFixed(0);
  const highGame = stats.d1["High Game"].t;
  const averageGame = (stats.d1["Score"].t / gamesPlayed).toFixed(0);
  const averageTurn = (stats.d1["Score"].t / stats.d1["Turns"].t).toFixed(0);
  const bestPlay = stats.d1["High Turn"].t.toFixed(0);
  const bingos = stats.d1["Bingos"].t.toFixed(0);
  const lastPlayed =
    ratings?.ts && moment(new Date(ratings?.ts * 1000)).format("MMM DD, YYYY");
  return (
    <Card className="variant-stats" title={variantToName(variant)}>
      {rating && <h3 className="rating">{rating}</h3>}
      {rating && <p className="rating-date">on {lastPlayed}</p>}
      {gamesPlayed && (
        <h4 className="stat-item">
          {gamesPlayed}
          <span className="label">games played</span>
        </h4>
      )}
      {averageGame && (
        <h4 className="stat-item">
          {averageGame}
          <span className="label">average game</span>
        </h4>
      )}
      {averageTurn && (
        <h4 className="stat-item">
          {averageTurn}
          <span className="label">average play</span>
        </h4>
      )}
      {highGame && (
        <h4 className="stat-item">
          {highGame}
          <span className="label">high game</span>
        </h4>
      )}
      {bestPlay && (
        <h4 className="stat-item">
          {bestPlay}
          <span className="label">best play</span>
        </h4>
      )}
      {bingos && (
        <h4 className="stat-item">
          {bingos}
          <span className="label">bingos</span>
        </h4>
      )}
    </Card>
  );
});

type AggregateStatsProps = {
  stats: ProfileStats | null;
};

const AggregateStatsCard = React.memo((props: AggregateStatsProps) => {
  const mergeStats = (statsMaps: { [key: string]: StatItem }[]) => {
    const ret: { [key: string]: StatItem } = {};
    statsMaps.forEach((s) => {
      for (const k in s) {
        if (ret[k]) {
          ret[k] = { t: s[k].t + ret[k].t };
        } else {
          ret[k] = { t: s[k].t };
        }
      }
    });
    return ret;
  };
  const { stats } = props;
  if (!stats) {
    return null;
  }
  const player = Object.values(stats).map((a) => a.d1);
  const totals = mergeStats(player);
  const gamesPlayed = totals["Games"]?.t || 0;
  const points = totals["Score"]?.t || 0;
  const wins = totals["Wins"]?.t || 0;
  const bingos = totals["Bingos"]?.t || 0;
  const challenges = totals["Challenges Won"]?.t || 0;
  const phonies =
    totals["Challenged Phonies"]?.t + totals["Unchallenged Phonies"]?.t;
  return (
    <div className="aggregate-stats">
      <div className="aggregate-item games">
        <h4>{gamesPlayed}</h4>
        <p>games played</p>
      </div>
      <div className="aggregate-item win">
        <h4>{wins}</h4>
        <p>games won</p>
      </div>
      <div className="aggregate-item score">
        <h4>{points}</h4>
        <p>points scored</p>
      </div>
      <div className="aggregate-item bingos">
        <h4>{bingos}</h4>
        <p>bingos played</p>
      </div>
      <div className="aggregate-item phonies">
        <h4>{phonies}</h4>
        <p>phonies played</p>
      </div>
      <div className="aggregate-item challenges">
        <h4>{challenges}</h4>
        <p>challenges won</p>
      </div>
    </div>
  );
});

const variantToName = (variant: string) => {
  const arr = variant.split(".");
  let lex = arr[0];
  lex = lexiconCodeToProfileRatingName(lex);

  const timectrl = {
    ultrablitz: "Ultra-Blitz!",
    blitz: "Blitz",
    rapid: "Rapid",
    regular: "Regular",
    corres: arr[1] === "puzzle" ? "Puzzle" : "Correspondence",
  }[arr[2] as "ultrablitz" | "blitz" | "rapid" | "regular" | "corres"]; // cmon typescript

  return (
    <>
      <VariantIcon vcode={arr[1]} /> {`${lex} (${timectrl})`}
    </>
  );
};

export const PlayerProfile = React.memo(() => {
  const gamesPageSize = 24;
  const annotatedPageSize = 20;
  const { loginState } = useLoginStateStoreContext();
  const { username } = useParams();
  const navigate = useNavigate();
  const viewer = loginState.loggedIn ? loginState.username : undefined;
  if (viewer && !username) {
    navigate(`/profile/${viewer}`, { replace: true });
  }
  const location = useLocation();
  // Show username's profile
  const [ratings, setRatings] = useState<ProfileRatings | null>(null);
  const [stats, setStats] = useState<ProfileStats | null>(null);
  const [userID, setUserID] = useState("");
  const [userFetched, setUserFetched] = useState(false);
  const [fullName, setFullName] = useState("");
  // const [avatarUrl, setAvatarUrl] = useState('');
  const [avatarsEditable, setAvatarsEditable] = useState(false);
  const [bio, setBio] = useState("");
  const [showGameTable, setShowGameTable] = useState(false);
  const [countryCode, setCountryCode] = useState("");
  const [bioLoaded, setBioLoaded] = useState(false);
  const [badges, setBadges] = useState<string[]>([]);
  const [recentGames, setRecentGames] = useState<{
    numGames: number;
    offset: number;
    array: Array<GameInfoResponse>;
  }>({ numGames: gamesPageSize, offset: 0, array: [] });
  const [recentAnnotatedGames, setRecentAnnotatedGames] = useState<
    Array<BroadcastGamesResponse_BroadcastGame>
  >([]);
  const [hasMoreAnnotatedGames, setHasMoreAnnotatedGames] = useState(true);
  const [recentGamesOffset, setRecentGamesOffset] = useState(0);
  const [recentAnnotatedGamesOffset, setRecentAnnotatedGamesOffset] =
    useState(0);
  const [missingBirthdate, setMissingBirthdate] = useState(true); // always true except for self
  const profileClient = useClient(ProfileService);
  const gameMetadataClient = useClient(GameMetadataService);
  const gameEventClient = useClient(GameEventService);
  const { data: badgeMetadata } = useQuery(getBadgesMetadata);

  const checkWide = useMemo(
    () => window.matchMedia(`(min-width: ${screenSizeTablet}px)`),
    [],
  );
  const [isWide, setIsWide] = useState(checkWide.matches);
  useEffect(() => {
    const handler = (evt: MediaQueryListEvent) => {
      setIsWide(evt.matches);
    };
    checkWide.addEventListener("change", handler);
    return () => {
      checkWide.removeEventListener("change", handler);
    };
  }, [checkWide]);
  const eltsPerRow = isWide ? 4 : 2;

  useEffect(() => {
    if (!username) {
      return;
    }

    const getProfile = async () => {
      try {
        const resp = await profileClient.getProfile({ username });
        setUserFetched(true);
        setMissingBirthdate(!resp.birthDate);
        setRatings(JSON.parse(resp.ratingsJson).Data);
        setStats(JSON.parse(resp.statsJson).Data);
        setUserID(resp.userId);
        setCountryCode(resp.countryCode);
        setFullName(resp.fullName);
        // setAvatarUrl(resp.avatarUrl);
        setAvatarsEditable(resp.avatarsEditable);
        setBio(resp.about);
        setBioLoaded(true);
        setBadges(resp.badgeCodes);
      } catch (e) {
        setUserFetched(true);
        flashError(e);
      }
    };

    getProfile();
  }, [username, location.pathname, profileClient]);

  const [queriedRecentGamesOffset, setQueriedRecentGamesOffset] =
    useState(recentGamesOffset);
  const reentrancyCheck = useRef<Record<string, never>>(undefined);

  useEffect(() => {
    if (!username) {
      return;
    }
    const hiddenObject = {}; // allocate a new thing every time
    reentrancyCheck.current = hiddenObject;
    (async () => {
      try {
        let queriedOffset = queriedRecentGamesOffset;
        const resp = await gameMetadataClient.getRecentGames({
          username,
          numGames: gamesPageSize,
          offset: queriedOffset,
        });

        // XXX: connect should support AbortController, but I'm not sure
        // what to do here.
        // Outdated axios does not support fetch()-compatible AbortController.
        if (reentrancyCheck.current !== hiddenObject) return;
        // If the array is empty and it is not the first page,
        // use binary search to find the last page with content.
        if (!resp.gameInfo.length && queriedOffset > 0) {
          // The maximum valid page number is before the empty page retrieved.
          const maxGuess = Math.max(
            Math.floor(queriedOffset / gamesPageSize - 1),
            0,
          );
          let guessBit = 1;
          while (guessBit < maxGuess) guessBit *= 2;
          let guess = 0;
          for (; guessBit >= 1; guessBit /= 2) {
            const newGuess = guess + guessBit;
            if (newGuess <= maxGuess) {
              const resp2 = await gameMetadataClient.getRecentGames({
                username,
                numGames: gamesPageSize,
                offset: newGuess * gamesPageSize,
              });

              if (reentrancyCheck.current !== hiddenObject) return;
              if (resp2.gameInfo.length) {
                // This is within range.
                guess = newGuess;
              }
            }
          }
          queriedOffset = guess * gamesPageSize;
        }
        if (queriedRecentGamesOffset !== queriedOffset) {
          // This will re-fetch that last page.
          setRecentGamesOffset(queriedOffset);
          setQueriedRecentGamesOffset(queriedOffset);
        } else {
          setRecentGames({
            numGames: gamesPageSize,
            offset: queriedOffset,
            array: resp.gameInfo,
          });
        }
      } catch (e) {
        flashError(e);
      }
    })();
  }, [username, queriedRecentGamesOffset, gameMetadataClient]);

  useEffect(() => {
    // offset and numGames are int32 in the protobuf.
    const maxPage = Math.floor(((1 << 30) * 2 - 1) / gamesPageSize);
    const adjustedRecentGamesOffset = Math.max(
      Math.min(recentGamesOffset, maxPage * gamesPageSize),
      0,
    );
    if (recentGamesOffset !== adjustedRecentGamesOffset) {
      setRecentGamesOffset(adjustedRecentGamesOffset);
    } else {
      const t = setTimeout(() => {
        setQueriedRecentGamesOffset(recentGamesOffset);
      }, 500);
      return () => {
        clearTimeout(t);
      };
    }
  }, [recentGamesOffset]);

  const fetchPrev = useCallback(() => {
    setRecentGamesOffset((r) => Math.max(r - gamesPageSize, 0));
  }, []);
  // Unbounded. It is possible to paginate many empty pages behind what exists by clicking Next many times rapidly.
  const fetchNext = useCallback(() => {
    setRecentGamesOffset((r) => r + gamesPageSize);
  }, []);

  const fetchPrevAnnotatedGames = useCallback(() => {
    setRecentAnnotatedGamesOffset((r) => Math.max(r - annotatedPageSize, 0));
  }, []);
  const fetchNextAnnotatedGames = useCallback(() => {
    setRecentAnnotatedGamesOffset((r) => r + annotatedPageSize);
  }, []);

  const handleChangePageNumber = useCallback(
    (value: number | string | null) => {
      if (
        value != null &&
        String(value) === String(Math.floor(Number(value)))
      ) {
        const valueNum = Math.max(1, Math.floor(Number(value)));
        setRecentGamesOffset((valueNum - 1) * gamesPageSize);
      }
    },
    [],
  );

  useEffect(() => {
    if (!userID) {
      return;
    }
    (async () => {
      try {
        const resp = await gameEventClient.getGamesForEditor({
          userId: userID,
          limit: annotatedPageSize,
          offset: recentAnnotatedGamesOffset,
        });
        setRecentAnnotatedGames(resp.games);
        // If we got fewer games than requested, there are no more
        setHasMoreAnnotatedGames(resp.games.length === annotatedPageSize);
      } catch (e) {
        console.log(e);
      }
    })();
  }, [gameEventClient, userID, recentAnnotatedGamesOffset]);

  const player = {
    fullName: fullName,
    userId: userID, // for name-based avatar initial to work
  };
  const avatarEditable = avatarsEditable && viewer === username;

  const variantCards = useMemo(() => {
    const data = [];
    for (const variant in stats) {
      if (stats.hasOwnProperty(variant)) {
        data.push({
          variant: variant,
          ratings: ratings ? ratings[variant] : null,
          stats: stats[variant],
        });
      }
    }
    const ret = data
      .sort((a, b) => {
        return (b.stats?.d1["Games"]?.t || 0) - (a.stats?.d1["Games"]?.t || 0);
      })
      .map((d) => {
        return <VariantCard key={d.variant} {...d} />;
      });
    return ret;
  }, [ratings, stats]);

  const puzzleCards = useMemo(() => {
    const data = [];
    for (const variant in ratings) {
      if (ratings.hasOwnProperty(variant) && variant.includes("puzzle")) {
        data.push({
          variant: variant,
          ratings: ratings[variant],
        });
      }
    }
    if (!ratings) {
      return;
    }
    const ret = data.map((v) => {
      const rating = v.ratings?.r.toFixed(0);
      const lastPlayed =
        v.ratings?.ts &&
        moment(new Date(v.ratings?.ts * 1000)).format("MMM DD, YYYY");
      return (
        <Card
          className="puzzle-stats"
          title={variantToName(v.variant)}
          key={v.variant}
        >
          {rating && <h3 className="rating">{rating}</h3>}
          {rating && <p className="rating-date">on {lastPlayed}</p>}
        </Card>
      );
    });
    return ret;
  }, [ratings]);

  const gameCards = useMemo(() => {
    if (!recentGames?.array) {
      return [];
    }
    const ret = recentGames?.array
      ?.filter(
        (g) => g.players?.length && g.gameEndReason !== GameEndReason.CANCELLED,
      )
      .map((g) => <GameCard game={g} key={g.gameId} userID={userID} />);
    return ret;
  }, [recentGames, userID]);

  const emptyCards = (n: number, f: (n: number) => boolean) => {
    const ret = [];
    while (f(n)) {
      ret.push(<div key={`empty-${n}`} />);
      ++n;
    }
    return ret;
  };

  return (
    <>
      <TopBar />
      {userID && (
        <div className="profile">
          <header>
            <div>
              <div className="user-info">
                <PettableAvatar>
                  <PlayerAvatar
                    player={player}
                    editable={avatarEditable}
                    username={username}
                  />
                  <h3>
                    {username ? (
                      <UsernameWithContext
                        omitProfileLink
                        omitSendMessage
                        fullName={fullName}
                        includeFlag
                        username={username}
                        userID={userID}
                        showModTools
                        moderate={moderateUser}
                        omitBadges
                      />
                    ) : (
                      <span className="user">
                        <span>{fullName}</span>
                        <DisplayFlag countryCode={countryCode} />
                      </span>
                    )}
                  </h3>
                </PettableAvatar>
              </div>
              {!(missingBirthdate && viewer === username) && (
                <BioCard bio={bio} bioLoaded={bioLoaded} />
              )}
              {badges.map((b) => (
                <div key={b} style={{ marginBottom: 16 }}>
                  <Badge name={b} width={36} />
                  <span style={{ marginLeft: 8 }}>
                    {badgeMetadata?.badges[b]}
                  </span>
                </div>
              ))}
              {username && <DisplayUserOrganizations username={username} />}
              {missingBirthdate && viewer === username && (
                <div className="bio">
                  <Link to={"/settings/personal"}>
                    Let us know your birthdate
                  </Link>{" "}
                  to share your bio and details
                </div>
              )}
            </div>
            <AggregateStatsCard stats={stats} />
          </header>
          {variantCards?.length > 0 && (
            <>
              <h2>Game Ratings</h2>
              <Carousel
                arrows
                className="variant-items"
                slidesToScroll={eltsPerRow}
                slidesToShow={eltsPerRow}
                swipeToSlide
                swipe
                infinite={false}
              >
                {variantCards}
                {emptyCards(
                  variantCards.length,
                  (n) => n % (n <= eltsPerRow ? eltsPerRow : n) !== 0,
                )}
              </Carousel>
            </>
          )}
          {!!puzzleCards?.length && (
            <>
              <h2>Puzzle Ratings</h2>
              <Carousel
                arrows
                className="puzzle-items"
                slidesToScroll={eltsPerRow}
                slidesToShow={eltsPerRow}
                swipeToSlide
                swipe
                infinite={false}
              >
                {puzzleCards}
                {emptyCards(
                  puzzleCards.length,
                  (n) => n % (n <= eltsPerRow ? eltsPerRow : n) !== 0,
                )}
              </Carousel>
            </>
          )}
          {!!gameCards?.length && (
            <>
              <h2>Recent games</h2>
              <Carousel
                arrows
                className="game-items"
                slidesToScroll={eltsPerRow}
                slidesToShow={eltsPerRow}
                rows={gameCards.length > eltsPerRow ? 2 : 1}
                swipeToSlide
                swipe
                infinite={false}
              >
                {gameCards}
                {emptyCards(
                  gameCards.length,
                  (n) => n % (n <= 2 * eltsPerRow ? eltsPerRow : n) !== 0,
                )}
              </Carousel>
            </>
          )}
          <p
            className="show-games-toggle"
            onClick={() => {
              setShowGameTable((x) => !x);
            }}
          >
            {!showGameTable ? "Show game table" : "Hide game table"}
          </p>
          {username && showGameTable && (
            <GamesHistoryCard
              games={recentGames.array}
              username={username}
              userID={userID}
              fetchPrev={recentGamesOffset > 0 ? fetchPrev : undefined}
              fetchNext={
                recentGames.array.length < recentGames.numGames
                  ? undefined
                  : fetchNext
              }
              currentOffset={recentGames.offset}
              currentPageSize={recentGames.numGames}
              desiredOffset={recentGamesOffset}
              desiredPageSize={gamesPageSize}
              onChangePageNumber={handleChangePageNumber}
            />
          )}

          {username && userID && (
            <UserCollectionsCard
              userUuid={userID}
              isOwnProfile={loginState.userID === userID}
            />
          )}

          {username &&
            !(
              recentAnnotatedGamesOffset === 0 && !recentAnnotatedGames?.length
            ) && (
              <AnnotatedGamesHistoryCard
                games={recentAnnotatedGames}
                fetchPrev={
                  recentAnnotatedGamesOffset > 0
                    ? fetchPrevAnnotatedGames
                    : undefined
                }
                fetchNext={
                  hasMoreAnnotatedGames ? fetchNextAnnotatedGames : undefined
                }
                loggedInUserID={loginState.userID}
                showAnnotator={false}
              />
            )}
        </div>
      )}

      {userFetched && !userID && username && (
        <div className="not-found">User not found.</div>
      )}
      {!username && (
        <div className="not-found">Login or register to view your profile.</div>
      )}
    </>
  );
});
