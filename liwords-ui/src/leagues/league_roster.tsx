import React, { useState, useMemo, useEffect } from "react";
import { Input, Table, Tag, Tooltip } from "antd";
import type { SortOrder } from "antd/es/table/interface";
import {
  StarOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  MinusOutlined,
} from "@ant-design/icons";
import { useQuery } from "@connectrpc/connect-query";
import {
  getLeagueRoster,
  getPlayerLeagueH2H,
} from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import {
  LeagueRosterPlayer,
  LeagueRosterSeason,
  H2HRecord,
} from "../gen/api/proto/league_service/league_service_pb";
import { StandingResult } from "../gen/api/proto/ipc/league_pb";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { endReasonLabel } from "./player_game_history_modal";

type Props = {
  leagueId: string;
  currentUserId?: string;
  activeSeasonNumber?: number;
  defaultSortSeason?: number;
  onJumpToSeason: (seasonNumber: number, divisionNumber: number) => void;
};

const resultIcon = (result: StandingResult) => {
  switch (result) {
    case StandingResult.RESULT_CHAMPION:
      return <StarOutlined style={{ color: "#d4af37" }} />;
    case StandingResult.RESULT_PROMOTED:
      return <ArrowUpOutlined style={{ color: "#52c41a" }} />;
    case StandingResult.RESULT_RELEGATED:
      return <ArrowDownOutlined style={{ color: "#ff4d4f" }} />;
    case StandingResult.RESULT_STAYED:
      return <MinusOutlined style={{ color: "#8c8c8c" }} />;
    default:
      return null;
  }
};

const resultLabel = (result: StandingResult): string | null => {
  switch (result) {
    case StandingResult.RESULT_CHAMPION:
      return "Champion";
    case StandingResult.RESULT_PROMOTED:
      return "Promoted";
    case StandingResult.RESULT_RELEGATED:
      return "Relegated";
    case StandingResult.RESULT_STAYED:
      return "Stayed";
    default:
      return null;
  }
};

const formatSeason = (
  season: LeagueRosterSeason | undefined,
  compRank?: number,
  hideResult?: boolean,
  h2hTooltip?: string,
) => {
  if (!season) return <span className="roster-empty">—</span>;
  if (season.divisionNumber === 0) {
    return <Tag color="lime">Registered</Tag>;
  }
  const record = `${season.wins}-${season.losses}${season.draws ? `-${season.draws}` : ""}`;
  const spread = season.spread > 0 ? `+${season.spread}` : `${season.spread}`;
  const displayRank = compRank ?? season.rank;
  const label = hideResult ? null : resultLabel(season.result);
  const seasonLine = label
    ? `${record} (${spread}) · ${label}`
    : `${record} (${spread})`;
  const tooltip = h2hTooltip ? (
    <>
      {seasonLine}
      <br />
      H2H: {h2hTooltip}
    </>
  ) : (
    seasonLine
  );
  return (
    <Tooltip title={tooltip}>
      <span className="roster-season" style={{ cursor: "pointer" }}>
        <Tag
          color={
            season.divisionNumber === 1
              ? "gold"
              : season.divisionNumber % 2 === 0
                ? "blue"
                : "green"
          }
        >
          D{season.divisionNumber}
        </Tag>
        {displayRank > 0 && <span className="roster-rank">#{displayRank}</span>}
        {!hideResult && resultIcon(season.result)}
      </span>
    </Tooltip>
  );
};

const formatH2H = (record: H2HRecord | undefined) => {
  if (!record) return <span className="roster-empty">—</span>;
  const { wins, losses, draws, spread } = record;
  const spreadStr = spread > 0 ? `+${spread}` : `${spread}`;
  const wld = `${wins}-${losses}${draws ? `-${draws}` : ""}`;
  return (
    <Tooltip title={`${wld} (${spreadStr})`}>
      <span
        style={{
          fontSize: "12px",
          color:
            wins > losses ? "#52c41a" : wins < losses ? "#ff4d4f" : undefined,
        }}
      >
        {wld}
      </span>
    </Tooltip>
  );
};

// A player's best-ever achievement, used by the Peak column.
//   season: the LeagueRosterSeason it was achieved in.
//   displayRank: the rank shown in the cell (competition/tie-aware rank for
//     placed seasons, 0 for an unplaced/registered-only season).
//   tied: whether that finish was shared with another player (sole otherwise).
type PeakCandidate = {
  season: LeagueRosterSeason;
  displayRank: number;
  tied: boolean;
};

// Compare two peak candidates, ordered best -> worst. Returns negative when
// `a` is the better peak. Key (best first):
//   1. higher division (lower divisionNumber; 0 = unplaced = worst)
//   2. higher finish (lower displayRank)
//   3. a sole finish beats a tied one
//   4. earlier season (lower seasonNumber)
const peakSeasonBetter = (a: PeakCandidate, b: PeakCandidate): number => {
  // Division: unplaced (0) is worse than any placed division.
  const da = a.season.divisionNumber === 0 ? Infinity : a.season.divisionNumber;
  const db = b.season.divisionNumber === 0 ? Infinity : b.season.divisionNumber;
  if (da !== db) return da - db; // lower division number is better
  const bothUnplaced = da === Infinity; // && db === Infinity
  if (!bothUnplaced) {
    // Higher finish (lower rank) is better.
    if (a.displayRank !== b.displayRank) return a.displayRank - b.displayRank;
    // A sole finish beats a tied one.
    if (a.tied !== b.tied) return a.tied ? 1 : -1;
  }
  // Earlier season is the final tiebreak.
  return a.season.seasonNumber - b.season.seasonNumber;
};

// Compare two players by their status in a given season.
// Order: placed (by division ASC, rank ASC) -> registered -> absent.
const seasonCompare = (
  a: LeagueRosterPlayer,
  b: LeagueRosterPlayer,
  seasonNumber: number,
): number => {
  const sa = a.seasons.find((x) => x.seasonNumber === seasonNumber);
  const sb = b.seasons.find((x) => x.seasonNumber === seasonNumber);
  // Absent sorts last
  if (!sa && !sb) return 0;
  if (!sa) return 1;
  if (!sb) return -1;
  // Registered but unplaced sorts after placed
  if (sa.divisionNumber === 0 && sb.divisionNumber !== 0) return 1;
  if (sa.divisionNumber !== 0 && sb.divisionNumber === 0) return -1;
  // Both placed: by division, then rank
  if (sa.divisionNumber !== sb.divisionNumber)
    return sa.divisionNumber < sb.divisionNumber ? -1 : 1;
  if (sa.rank !== sb.rank) return sa.rank < sb.rank ? -1 : 1;
  return 0;
};

export const LeagueRoster: React.FC<Props> = ({
  leagueId,
  currentUserId,
  activeSeasonNumber,
  defaultSortSeason,
  onJumpToSeason,
}) => {
  const [search, setSearch] = useState("");
  const [h2hUserId, setH2hUserId] = useState("");
  const [h2hUsername, setH2hUsername] = useState("");
  const [sortKey, setSortKey] = useState<string | null>(
    defaultSortSeason ? `s${defaultSortSeason}` : null,
  );
  const [sortOrder, setSortOrder] = useState<SortOrder>(
    defaultSortSeason ? "ascend" : null,
  );
  const { data, isLoading } = useQuery(getLeagueRoster, {
    leagueId,
  });

  // Auto-show logged-in user's H2H if they're a participant
  useEffect(() => {
    if (currentUserId && data?.players && !h2hUserId) {
      const isParticipant = data.players.some(
        (p) => p.userId === currentUserId,
      );
      if (isParticipant) {
        setH2hUserId(currentUserId);
      }
    }
  }, [currentUserId, data?.players, h2hUserId]);

  // Clear h2h selection if the target player is no longer in the data
  useEffect(() => {
    if (h2hUserId && data?.players) {
      const stillExists = data.players.some((p) => p.userId === h2hUserId);
      if (!stillExists) {
        setH2hUserId("");
        setH2hUsername("");
      }
    }
  }, [data?.players, h2hUserId]);

  // Fetch h2h data for the selected player (defaults to logged-in user)
  const { data: h2hData } = useQuery(
    getPlayerLeagueH2H,
    {
      userId: h2hUserId,
      leagueId,
    },
    { enabled: !!h2hUserId },
  );

  // Build a map of opponent UUID -> H2HRecord
  const h2hMap = useMemo(() => {
    const map = new Map<string, H2HRecord>();
    if (h2hData?.records) {
      for (const record of h2hData.records) {
        map.set(record.opponentUserId, record);
      }
    }
    return map;
  }, [h2hData?.records]);

  // Build map: "opponentUserId:seasonNumber" -> game details[]
  const h2hSeasonMap = useMemo(() => {
    const map = new Map<
      string,
      Array<{
        won: boolean;
        draw: boolean;
        playerScore: number;
        opponentScore: number;
        gameEndReason: number;
      }>
    >();
    if (h2hData?.records) {
      for (const record of h2hData.records) {
        for (const game of record.seasonGames) {
          const key = `${record.opponentUserId}:${game.seasonNumber}`;
          if (!map.has(key)) map.set(key, []);
          map.get(key)!.push({
            won: game.won,
            draw: game.draw,
            playerScore: game.playerScore,
            opponentScore: game.opponentScore,
            gameEndReason: game.gameEndReason,
          });
        }
      }
    }
    return map;
  }, [h2hData?.records]);

  // Precompute competition ranks per (season, division) group.
  //   rank: "seasonNumber:userId" -> competition (tie-aware) rank number.
  //   tied: set of "seasonNumber:userId" whose competition rank is shared
  //         with another player in the same (season, division) group, i.e.
  //         a tied finish rather than a sole one. Players in a group with no
  //         games played are treated as tied (no sole finish to claim yet).
  const compRankInfo = useMemo(() => {
    const map = new Map<string, number>();
    const tied = new Set<string>();
    if (!data?.players) return { rank: map, tied };

    // Group entries by (season, division)
    const groups = new Map<
      string,
      Array<{ userId: string; points: number; spread: number; rank: number }>
    >();
    for (const player of data.players) {
      for (const season of player.seasons) {
        if (season.divisionNumber === 0) continue;
        const key = `${season.seasonNumber}:${season.divisionNumber}`;
        if (!groups.has(key)) groups.set(key, []);
        groups.get(key)!.push({
          userId: player.userId,
          points: season.wins * 2 + season.draws,
          spread: season.spread,
          rank: season.rank,
        });
      }
    }

    // Compute competition rank per group (sorted by server rank).
    for (const [groupKey, entries] of groups.entries()) {
      const seasonNumber = groupKey.split(":")[0];
      // No games played yet: nobody holds a sole finish.
      if (entries.every((e) => e.points === 0 && e.spread === 0)) {
        for (const e of entries) tied.add(`${seasonNumber}:${e.userId}`);
        continue;
      }
      entries.sort((a, b) => a.rank - b.rank);
      let currentRank = 1;
      // Count players per competition rank to flag shared (tied) finishes.
      const rankCounts = new Map<number, number>();
      for (let i = 0; i < entries.length; i++) {
        if (
          i > 0 &&
          (entries[i].points !== entries[i - 1].points ||
            entries[i].spread !== entries[i - 1].spread)
        ) {
          currentRank = i + 1;
        }
        map.set(`${seasonNumber}:${entries[i].userId}`, currentRank);
        rankCounts.set(currentRank, (rankCounts.get(currentRank) ?? 0) + 1);
      }
      for (const entry of entries) {
        const cr = map.get(`${seasonNumber}:${entry.userId}`)!;
        if ((rankCounts.get(cr) ?? 0) > 1) {
          tied.add(`${seasonNumber}:${entry.userId}`);
        }
      }
    }

    return { rank: map, tied };
  }, [data?.players]);
  const compRankMap = compRankInfo.rank;

  // Compute each player's peak (best-ever) season across all their seasons.
  // Key: userId -> { season, displayRank, tied }. Players with no seasons are
  // absent from the map (treated as worst for sorting / shown as em-dash).
  const peakMap = useMemo(() => {
    const map = new Map<
      string,
      { season: LeagueRosterSeason; displayRank: number; tied: boolean }
    >();
    if (!data?.players) return map;
    for (const player of data.players) {
      let best:
        | { season: LeagueRosterSeason; displayRank: number; tied: boolean }
        | undefined;
      for (const season of player.seasons) {
        const key = `${season.seasonNumber}:${player.userId}`;
        const displayRank =
          season.divisionNumber === 0
            ? 0
            : (compRankMap.get(key) ?? season.rank);
        const tied = compRankInfo.tied.has(key);
        const cand = { season, displayRank, tied };
        if (!best || peakSeasonBetter(cand, best) < 0) {
          best = cand;
        }
      }
      if (best) map.set(player.userId, best);
    }
    return map;
  }, [data?.players, compRankMap, compRankInfo.tied]);

  const filteredPlayers = useMemo(() => {
    if (!data?.players) return [];
    if (!search) return data.players;
    const q = search.toLowerCase();
    return data.players.filter((p: LeagueRosterPlayer) =>
      p.username.toLowerCase().includes(q),
    );
  }, [data?.players, search]);

  const seasonNumbers = useMemo(
    () => [...(data?.seasonNumbers ?? [])].reverse(),
    [data?.seasonNumbers],
  );

  const columns = [
    {
      title: "Player",
      key: "username",
      fixed: "left" as const,
      width: 140,
      sorter: (a: LeagueRosterPlayer, b: LeagueRosterPlayer) =>
        a.username.localeCompare(b.username),
      sortDirections: ["ascend", "descend"] as SortOrder[],
      sortOrder: sortKey === "username" ? sortOrder : undefined,
      render: (_: unknown, record: LeagueRosterPlayer) => (
        <UsernameWithContext
          username={record.username}
          userID={record.userId}
          omitSendMessage
          omitFriend
          omitBlock
          infoText="View H2H"
          handleInfoText={() => {
            setH2hUserId(record.userId);
            setH2hUsername(record.username);
          }}
        />
      ),
    },
    // H2H column - show when viewing any player's h2h
    ...(h2hUserId
      ? [
          {
            title: (
              <Tooltip
                title={
                  h2hUserId === currentUserId
                    ? "Your head-to-head record in league games"
                    : `Head-to-head record for ${h2hUsername}`
                }
              >
                <span>
                  H2H
                  {h2hUserId !== currentUserId && h2hUsername && (
                    <span style={{ fontSize: 10, opacity: 0.7 }}>
                      {" "}
                      ({h2hUsername})
                    </span>
                  )}
                </span>
              </Tooltip>
            ),
            key: "h2h",
            width: h2hUserId !== currentUserId ? 120 : 70,
            render: (_: unknown, record: LeagueRosterPlayer) => {
              if (record.userId === h2hUserId) return null;
              return formatH2H(h2hMap.get(record.userId));
            },
            sorter: (a: LeagueRosterPlayer, b: LeagueRosterPlayer) => {
              const aRec = h2hMap.get(a.userId);
              const bRec = h2hMap.get(b.userId);
              if (!aRec && !bRec) return 0;
              if (!aRec) return -1;
              if (!bRec) return 1;
              const aDiff = aRec.wins - aRec.losses;
              const bDiff = bRec.wins - bRec.losses;
              if (aDiff !== bDiff) return aDiff - bDiff;
              if (aRec.spread !== bRec.spread) return aRec.spread - bRec.spread;
              const aEnc = aRec.wins + aRec.losses + aRec.draws;
              const bEnc = bRec.wins + bRec.losses + bRec.draws;
              if (aEnc !== bEnc) return aEnc - bEnc;
              return b.username
                .toLowerCase()
                .localeCompare(a.username.toLowerCase());
            },
            sortDirections: ["descend", "ascend"] as SortOrder[],
            sortOrder: sortKey === "h2h" ? sortOrder : undefined,
          },
        ]
      : []),
    {
      title: <Tooltip title="Number of seasons played">#S</Tooltip>,
      key: "count",
      width: 40,
      render: (_: unknown, record: LeagueRosterPlayer) => record.seasons.length,
      sorter: (a: LeagueRosterPlayer, b: LeagueRosterPlayer) =>
        a.seasons.length - b.seasons.length,
      sortDirections: ["descend", "ascend"] as SortOrder[],
      sortOrder: sortKey === "count" ? sortOrder : undefined,
    },
    {
      title: (
        <Tooltip title="Best-ever finish across all seasons (highest division, then best rank; a sole rank beats a tied one)">
          Peak
        </Tooltip>
      ),
      key: "peak",
      width: 130,
      render: (_: unknown, record: LeagueRosterPlayer) => {
        const peak = peakMap.get(record.userId);
        if (!peak) return <span className="roster-empty">—</span>;
        return (
          <span className="roster-peak">
            {formatSeason(
              peak.season,
              compRankMap.get(`${peak.season.seasonNumber}:${record.userId}`),
            )}
            <span style={{ fontSize: 11, opacity: 0.65, marginLeft: 2 }}>
              ·S{peak.season.seasonNumber}
            </span>
          </span>
        );
      },
      // Ascending order is worst -> best, so the default (first click) descending
      // sort surfaces the best peaks first.
      sorter: (a: LeagueRosterPlayer, b: LeagueRosterPlayer) => {
        const pa = peakMap.get(a.userId);
        const pb = peakMap.get(b.userId);
        // A player with any peak outranks one with none.
        if (!pa && !pb) {
          return b.username
            .toLowerCase()
            .localeCompare(a.username.toLowerCase());
        }
        if (!pa) return -1;
        if (!pb) return 1;
        const cmp = peakSeasonBetter(pa, pb);
        if (cmp !== 0) return -cmp;
        // Equal peaks: alphabetical under the default descending sort.
        return b.username.toLowerCase().localeCompare(a.username.toLowerCase());
      },
      sortDirections: ["descend", "ascend"] as SortOrder[],
      sortOrder: sortKey === "peak" ? sortOrder : undefined,
    },
    ...seasonNumbers.map((sn: number) => ({
      title: `S${sn}`,
      key: `s${sn}`,
      width: 120,
      render: (_: unknown, record: LeagueRosterPlayer) => {
        const season = record.seasons.find(
          (s: LeagueRosterSeason) => s.seasonNumber === sn,
        );
        if (!season) return <span className="roster-empty">—</span>;
        const seasonGames =
          h2hUserId && record.userId !== h2hUserId
            ? h2hSeasonMap.get(`${record.userId}:${sn}`)
            : undefined;
        const h2hDot = h2hUserId ? (
          <span
            style={{
              color: "#2d6a9e",
              fontSize: 8,
              marginRight: 2,
              visibility: seasonGames ? "visible" : "hidden",
            }}
          >
            ●
          </span>
        ) : null;
        const gameTooltip = seasonGames
          ? seasonGames
              .map((g) => {
                // From the H2H player's perspective (matches H2H column)
                const result = g.draw ? "D" : g.won ? "W" : "L";
                const reason = endReasonLabel(g.gameEndReason as GameEndReason);
                const suffix = reason ? ` (${reason})` : "";
                return `${result} ${g.playerScore}-${g.opponentScore}${suffix}`;
              })
              .join(", ")
          : undefined;
        return (
          <span onClick={() => onJumpToSeason(sn, season.divisionNumber)}>
            {h2hDot}
            {formatSeason(
              season,
              compRankMap.get(`${sn}:${record.userId}`),
              sn === activeSeasonNumber,
              gameTooltip,
            )}
          </span>
        );
      },
      sorter: (a: LeagueRosterPlayer, b: LeagueRosterPlayer) =>
        seasonCompare(a, b, sn) ||
        a.username.toLowerCase().localeCompare(b.username.toLowerCase()),
      sortDirections: ["ascend", "descend"] as SortOrder[],
      sortOrder: sortKey === `s${sn}` ? sortOrder : undefined,
    })),
  ];

  return (
    <div className="league-roster">
      <Input
        placeholder="Search player..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        style={{ marginBottom: 12, maxWidth: 300 }}
        allowClear
      />
      <Table
        columns={columns}
        dataSource={filteredPlayers}
        rowKey="userId"
        loading={isLoading}
        pagination={{ pageSize: 45, showSizeChanger: false }}
        size="small"
        scroll={{ x: "max-content" }}
        showSorterTooltip={false}
        onChange={(_pagination, _filters, sorter) => {
          if (!Array.isArray(sorter)) {
            setSortKey((sorter.columnKey as string) ?? null);
            setSortOrder(sorter.order ?? null);
          }
        }}
      />
    </div>
  );
};
