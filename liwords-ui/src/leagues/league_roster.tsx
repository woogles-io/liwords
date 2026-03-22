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
import { UsernameWithContext } from "../shared/usernameWithContext";

type Props = {
  leagueId: string;
  currentUserId?: string;
  onJumpToSeason: (seasonNumber: number, divisionNumber: number) => void;
};

const resultIcon = (result: StandingResult) => {
  switch (result) {
    case StandingResult.RESULT_CHAMPION:
      return (
        <Tooltip title="Champion">
          <StarOutlined style={{ color: "#d4af37" }} />
        </Tooltip>
      );
    case StandingResult.RESULT_PROMOTED:
      return (
        <Tooltip title="Promoted">
          <ArrowUpOutlined style={{ color: "#52c41a" }} />
        </Tooltip>
      );
    case StandingResult.RESULT_RELEGATED:
      return (
        <Tooltip title="Relegated">
          <ArrowDownOutlined style={{ color: "#ff4d4f" }} />
        </Tooltip>
      );
    case StandingResult.RESULT_STAYED:
      return (
        <Tooltip title="Stayed">
          <MinusOutlined style={{ color: "#8c8c8c" }} />
        </Tooltip>
      );
    default:
      return null;
  }
};

const formatSeason = (
  season: LeagueRosterSeason | undefined,
  compRank?: number,
) => {
  if (!season) return <span className="roster-empty">—</span>;
  if (season.divisionNumber === 0) {
    return <Tag color="lime">Registered</Tag>;
  }
  const record = `${season.wins}-${season.losses}${season.draws ? `-${season.draws}` : ""}`;
  const spread = season.spread > 0 ? `+${season.spread}` : `${season.spread}`;
  const displayRank = compRank ?? season.rank;
  return (
    <Tooltip title={`${record} (${spread})`}>
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
        {resultIcon(season.result)}
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

// Compare two players by their status in a given season.
// Order: placed (by division ASC, rank ASC) → registered → absent.
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
  onJumpToSeason,
}) => {
  const [search, setSearch] = useState("");
  const [h2hUserId, setH2hUserId] = useState("");
  const [h2hUsername, setH2hUsername] = useState("");
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

  // Precompute competition ranks per (season, division) group.
  // Key: "seasonNumber:userId" → competition rank number.
  const compRankMap = useMemo(() => {
    const map = new Map<string, number>();
    if (!data?.players) return map;

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
    // Skip groups where no games have been played.
    for (const [groupKey, entries] of groups.entries()) {
      if (entries.every((e) => e.points === 0 && e.spread === 0)) continue;
      const seasonNumber = groupKey.split(":")[0];
      entries.sort((a, b) => a.rank - b.rank);
      let currentRank = 1;
      for (let i = 0; i < entries.length; i++) {
        if (
          i > 0 &&
          (entries[i].points !== entries[i - 1].points ||
            entries[i].spread !== entries[i - 1].spread)
        ) {
          currentRank = i + 1;
        }
        map.set(`${seasonNumber}:${entries[i].userId}`, currentRank);
      }
    }

    return map;
  }, [data?.players]);

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
              const aVal = aRec ? aRec.wins - aRec.losses : -9999;
              const bVal = bRec ? bRec.wins - bRec.losses : -9999;
              return aVal - bVal;
            },
            sortDirections: ["descend", "ascend"] as SortOrder[],
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
        return (
          <span onClick={() => onJumpToSeason(sn, season.divisionNumber)}>
            {formatSeason(season, compRankMap.get(`${sn}:${record.userId}`))}
          </span>
        );
      },
      sorter: (a: LeagueRosterPlayer, b: LeagueRosterPlayer) =>
        seasonCompare(a, b, sn) ||
        a.username.toLowerCase().localeCompare(b.username.toLowerCase()),
      sortDirections: ["ascend", "descend"] as SortOrder[],
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
        pagination={{ pageSize: 50, showSizeChanger: false }}
        size="small"
        scroll={{ x: "max-content" }}
        showSorterTooltip={false}
      />
    </div>
  );
};
